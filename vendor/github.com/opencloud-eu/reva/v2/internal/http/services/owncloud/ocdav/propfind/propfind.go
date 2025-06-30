// Copyright 2018-2021 CERN
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// In applying this license, CERN does not waive the privileges and immunities
// granted to it by virtue of its status as an Intergovernmental Organization
// or submit itself to any jurisdiction.

package propfind

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	link "github.com/cs3org/go-cs3apis/cs3/sharing/link/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	typesv1beta1 "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/iancoleman/strcase"
	"github.com/opencloud-eu/opencloud/services/thumbnails/pkg/thumbnail"
	"github.com/opencloud-eu/reva/v2/internal/grpc/services/storageprovider"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/config"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/errors"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/net"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/prop"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/spacelookup"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/conversions"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/publicshare"
	rstatus "github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/rhttp/router"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

const (
	tracerName = "ocdav"
)

// these keys are used to lookup in ArbitraryMetadata, generated prop names are lowercased
var (
	audioKeys = []string{
		"album",
		"albumArtist",
		"artist",
		"bitrate",
		"composers",
		"copyright",
		"disc",
		"discCount",
		"duration",
		"genre",
		"hasDrm",
		"isVariableBitrate",
		"title",
		"track",
		"trackCount",
		"year",
	}
	locationKeys = []string{
		"altitude",
		"latitude",
		"longitude",
	}
	imageKeys = []string{
		"width",
		"height",
	}
	photoKeys = []string{
		"cameraMake",
		"cameraModel",
		"exposureDenominator",
		"exposureNumerator",
		"fNumber",
		"focalLength",
		"iso",
		"orientation",
		"takenDateTime",
	}
)

type countingReader struct {
	n int
	r io.Reader
}

// Props represents properties related to a resource
// http://www.webdav.org/specs/rfc4918.html#ELEMENT_prop (for propfind)
type Props []xml.Name

// XML holds the xml representation of a propfind
// http://www.webdav.org/specs/rfc4918.html#ELEMENT_propfind
type XML struct {
	XMLName  xml.Name  `xml:"DAV: propfind"`
	Allprop  *struct{} `xml:"DAV: allprop"`
	Propname *struct{} `xml:"DAV: propname"`
	Prop     Props     `xml:"DAV: prop"`
	Include  Props     `xml:"DAV: include"`
}

// PropstatXML holds the xml representation of a propfind response
// http://www.webdav.org/specs/rfc4918.html#ELEMENT_propstat
type PropstatXML struct {
	// Prop requires DAV: to be the default namespace in the enclosing
	// XML. This is due to the standard encoding/xml package currently
	// not honoring namespace declarations inside a xmltag with a
	// parent element for anonymous slice elements.
	// Use of multistatusWriter takes care of this.
	Prop                []prop.PropertyXML `xml:"d:prop>_ignored_"`
	Status              string             `xml:"d:status"`
	Error               *errors.ErrorXML   `xml:"d:error"`
	ResponseDescription string             `xml:"d:responsedescription,omitempty"`
}

// ResponseXML holds the xml representation of a propfind response
type ResponseXML struct {
	XMLName             xml.Name         `xml:"d:response"`
	Href                string           `xml:"d:href"`
	Propstat            []PropstatXML    `xml:"d:propstat"`
	Status              string           `xml:"d:status,omitempty"`
	Error               *errors.ErrorXML `xml:"d:error"`
	ResponseDescription string           `xml:"d:responsedescription,omitempty"`
}

// MultiStatusResponseXML holds the xml representation of a multistatus propfind response
type MultiStatusResponseXML struct {
	XMLName xml.Name `xml:"d:multistatus"`
	XmlnsS  string   `xml:"xmlns:s,attr,omitempty"`
	XmlnsD  string   `xml:"xmlns:d,attr,omitempty"`
	XmlnsOC string   `xml:"xmlns:oc,attr,omitempty"`

	Responses []*ResponseXML `xml:"d:response"`
}

// ResponseUnmarshalXML is a workaround for https://github.com/golang/go/issues/13400
type ResponseUnmarshalXML struct {
	XMLName             xml.Name               `xml:"response"`
	Href                string                 `xml:"href"`
	Propstat            []PropstatUnmarshalXML `xml:"propstat"`
	Status              string                 `xml:"status,omitempty"`
	Error               *errors.ErrorXML       `xml:"d:error"`
	ResponseDescription string                 `xml:"responsedescription,omitempty"`
}

// MultiStatusResponseUnmarshalXML is a workaround for https://github.com/golang/go/issues/13400
type MultiStatusResponseUnmarshalXML struct {
	XMLName xml.Name `xml:"multistatus"`
	XmlnsS  string   `xml:"xmlns:s,attr,omitempty"`
	XmlnsD  string   `xml:"xmlns:d,attr,omitempty"`
	XmlnsOC string   `xml:"xmlns:oc,attr,omitempty"`

	Responses []*ResponseUnmarshalXML `xml:"response"`
}

// PropstatUnmarshalXML is a workaround for https://github.com/golang/go/issues/13400
type PropstatUnmarshalXML struct {
	// Prop requires DAV: to be the default namespace in the enclosing
	// XML. This is due to the standard encoding/xml package currently
	// not honoring namespace declarations inside a xmltag with a
	// parent element for anonymous slice elements.
	// Use of multistatusWriter takes care of this.
	Prop                []*prop.PropertyXML `xml:"prop"`
	Status              string              `xml:"status"`
	Error               *errors.ErrorXML    `xml:"d:error"`
	ResponseDescription string              `xml:"responsedescription,omitempty"`
}

// spaceData is used to remember the space for a resource info
type spaceData struct {
	Ref       *provider.Reference
	SpaceType string
}

// NewMultiStatusResponseXML returns a preconfigured instance of MultiStatusResponseXML
func NewMultiStatusResponseXML() *MultiStatusResponseXML {
	return &MultiStatusResponseXML{
		XmlnsD:  "DAV:",
		XmlnsS:  "http://sabredav.org/ns",
		XmlnsOC: "http://owncloud.org/ns",
	}
}

// Handler handles propfind requests
type Handler struct {
	PublicURL string
	selector  pool.Selectable[gateway.GatewayAPIClient]
	c         *config.Config
}

// NewHandler returns a new PropfindHandler instance
func NewHandler(publicURL string, selector pool.Selectable[gateway.GatewayAPIClient], c *config.Config) *Handler {
	return &Handler{
		PublicURL: publicURL,
		selector:  selector,
		c:         c,
	}
}

// HandlePathPropfind handles a path based propfind request
// ns is the namespace that is prefixed to the path in the cs3 namespace
func (p *Handler) HandlePathPropfind(w http.ResponseWriter, r *http.Request, ns string) {
	ctx, span := appctx.GetTracerProvider(r.Context()).Tracer(tracerName).Start(r.Context(), fmt.Sprintf("%s %v", r.Method, r.URL.Path))
	defer span.End()

	fn := path.Join(ns, r.URL.Path) // TODO do we still need to jail if we query the registry about the spaces?

	sublog := appctx.GetLogger(ctx).With().Str("path", fn).Logger()
	dh := r.Header.Get(net.HeaderDepth)

	depth, err := net.ParseDepth(dh)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Invalid Depth header value")
		span.SetAttributes(semconv.HTTPStatusCodeKey.Int(http.StatusBadRequest))
		sublog.Debug().Str("depth", dh).Msg(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		m := fmt.Sprintf("Invalid Depth header value: %v", dh)
		b, err := errors.Marshal(http.StatusBadRequest, m, "", "")
		errors.HandleWebdavError(&sublog, w, b, err)
		return
	}

	if depth == net.DepthInfinity && !p.c.AllowPropfindDepthInfinitiy {
		span.RecordError(errors.ErrInvalidDepth)
		span.SetStatus(codes.Error, "DEPTH: infinity is not supported")
		span.SetAttributes(semconv.HTTPStatusCodeKey.Int(http.StatusBadRequest))
		sublog.Debug().Str("depth", dh).Msg(errors.ErrInvalidDepth.Error())
		w.WriteHeader(http.StatusBadRequest)
		m := fmt.Sprintf("Invalid Depth header value: %v", dh)
		b, err := errors.Marshal(http.StatusBadRequest, m, "", "")
		errors.HandleWebdavError(&sublog, w, b, err)
		return
	}

	pf, status, err := ReadPropfind(r.Body)
	if err != nil {
		sublog.Debug().Err(err).Msg("error reading propfind request")
		w.WriteHeader(status)
		return
	}

	// retrieve a specific storage space
	client, err := p.selector.Next()
	if err != nil {
		sublog.Error().Err(err).Msg("error retrieving a gateway service client")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// TODO look up all spaces and request the root_info in the field mask
	spaces, rpcStatus, err := spacelookup.LookUpStorageSpacesForPathWithChildren(ctx, client, fn)
	if err != nil {
		sublog.Error().Err(err).Msg("error sending a grpc request")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if rpcStatus.Code != rpc.Code_CODE_OK {
		errors.HandleErrorStatus(&sublog, w, rpcStatus)
		return
	}

	resourceInfos, sendTusHeaders, ok := p.getResourceInfos(ctx, w, r, pf, spaces, fn, depth, sublog)
	if !ok {
		// getResourceInfos handles responses in case of an error so we can just return here.
		return
	}
	p.propfindResponse(ctx, w, r, ns, pf, sendTusHeaders, resourceInfos, sublog)
}

// HandleSpacesPropfind handles a spaces based propfind request
func (p *Handler) HandleSpacesPropfind(w http.ResponseWriter, r *http.Request, spaceID string) {
	ctx, span := appctx.GetTracerProvider(r.Context()).Tracer(tracerName).Start(r.Context(), "spaces_propfind")
	defer span.End()

	sublog := appctx.GetLogger(ctx).With().Str("path", r.URL.Path).Str("spaceid", spaceID).Logger()
	dh := r.Header.Get(net.HeaderDepth)

	depth, err := net.ParseDepth(dh)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Invalid Depth header value")
		span.SetAttributes(semconv.HTTPStatusCodeKey.Int(http.StatusBadRequest))
		sublog.Debug().Str("depth", dh).Msg(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		m := fmt.Sprintf("Invalid Depth header value: %v", dh)
		b, err := errors.Marshal(http.StatusBadRequest, m, "", "")
		errors.HandleWebdavError(&sublog, w, b, err)
		return
	}

	if depth == net.DepthInfinity && !p.c.AllowPropfindDepthInfinitiy {
		span.RecordError(errors.ErrInvalidDepth)
		span.SetStatus(codes.Error, "DEPTH: infinity is not supported")
		span.SetAttributes(semconv.HTTPStatusCodeKey.Int(http.StatusBadRequest))
		sublog.Debug().Str("depth", dh).Msg(errors.ErrInvalidDepth.Error())
		w.WriteHeader(http.StatusBadRequest)
		m := fmt.Sprintf("Invalid Depth header value: %v", dh)
		b, err := errors.Marshal(http.StatusBadRequest, m, "", "")
		errors.HandleWebdavError(&sublog, w, b, err)
		return
	}

	pf, status, err := ReadPropfind(r.Body)
	if err != nil {
		sublog.Debug().Err(err).Msg("error reading propfind request")
		w.WriteHeader(status)
		return
	}

	ref, err := spacelookup.MakeStorageSpaceReference(spaceID, r.URL.Path)
	if err != nil {
		sublog.Debug().Msg("invalid space id")
		w.WriteHeader(http.StatusBadRequest)
		m := fmt.Sprintf("Invalid space id: %v", spaceID)
		b, err := errors.Marshal(http.StatusBadRequest, m, "", "")
		errors.HandleWebdavError(&sublog, w, b, err)
		return
	}

	client, err := p.selector.Next()
	if err != nil {
		sublog.Error().Err(err).Msg("error getting grpc client")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	metadataKeys, _ := metadataKeys(pf)

	// stat the reference and request the space in the field mask
	res, err := client.Stat(ctx, &provider.StatRequest{
		Ref:                   &ref,
		ArbitraryMetadataKeys: metadataKeys,
		FieldMask:             &fieldmaskpb.FieldMask{Paths: []string{"*"}}, // TODO use more sophisticated filter? we don't need all space properties, afaict only the spacetype
	})
	if err != nil {
		sublog.Error().Err(err).Msg("error getting grpc client")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if res.Status.Code != rpc.Code_CODE_OK {
		status := rstatus.HTTPStatusFromCode(res.Status.Code)
		if res.Status.Code == rpc.Code_CODE_ABORTED {
			// aborted is used for etag an lock mismatches, which translates to 412
			// in case a real Conflict response is needed, the calling code needs to send the header
			status = http.StatusPreconditionFailed
		}
		m := res.Status.Message
		if res.Status.Code == rpc.Code_CODE_PERMISSION_DENIED {
			// check if user has access to resource
			sRes, err := client.Stat(ctx, &provider.StatRequest{Ref: &provider.Reference{ResourceId: ref.GetResourceId()}})
			if err != nil {
				sublog.Error().Err(err).Msg("error performing stat grpc request")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if sRes.Status.Code != rpc.Code_CODE_OK {
				// return not found error so we do not leak existence of a space
				status = http.StatusNotFound
			}
		}
		if status == http.StatusNotFound {
			m = "Resource not found" // mimic the oc10 error message
		}
		w.WriteHeader(status)
		b, err := errors.Marshal(status, m, "", "")
		errors.HandleWebdavError(&sublog, w, b, err)
		return
	}
	var space *provider.StorageSpace
	if res.Info.Space == nil {
		sublog.Debug().Msg("stat did not include a space, executing an additional lookup request")
		// fake a space root
		space = &provider.StorageSpace{
			Id: &provider.StorageSpaceId{OpaqueId: spaceID},
			Opaque: &typesv1beta1.Opaque{
				Map: map[string]*typesv1beta1.OpaqueEntry{
					"path": {
						Decoder: "plain",
						Value:   []byte("/"),
					},
				},
			},
			Root:     ref.ResourceId,
			RootInfo: res.Info,
		}
	}

	res.Info.Path = r.URL.Path

	resourceInfos := []*provider.ResourceInfo{
		res.Info,
	}
	if res.Info.Type == provider.ResourceType_RESOURCE_TYPE_CONTAINER && depth != net.DepthZero {
		childInfos, ok := p.getSpaceResourceInfos(ctx, w, r, pf, &ref, r.URL.Path, depth, sublog)
		if !ok {
			// getResourceInfos handles responses in case of an error so we can just return here.
			return
		}
		resourceInfos = append(resourceInfos, childInfos...)
	}

	// prefix space id to paths
	for i := range resourceInfos {
		resourceInfos[i].Path = path.Join("/", spaceID, resourceInfos[i].Path)
		// add space to info so propfindResponse can access space type
		if resourceInfos[i].Space == nil {
			resourceInfos[i].Space = space
		}
	}

	sendTusHeaders := true
	// let clients know this collection supports tus.io POST requests to start uploads
	if res.Info.Type == provider.ResourceType_RESOURCE_TYPE_CONTAINER {
		if res.Info.Opaque != nil {
			_, ok := res.Info.Opaque.Map["disable_tus"]
			sendTusHeaders = !ok
		}
	}

	p.propfindResponse(ctx, w, r, "", pf, sendTusHeaders, resourceInfos, sublog)
}

func (p *Handler) propfindResponse(ctx context.Context, w http.ResponseWriter, r *http.Request, namespace string, pf XML, sendTusHeaders bool, resourceInfos []*provider.ResourceInfo, log zerolog.Logger) {
	ctx, span := appctx.GetTracerProvider(r.Context()).Tracer(tracerName).Start(ctx, "propfind_response")
	defer span.End()

	var linkshares map[string]struct{}
	// public link access does not show share-types
	// oc:share-type is not part of an allprops response
	if namespace != "/public" {
		// only fetch this if property was queried
		for _, prop := range pf.Prop {
			if prop.Space == net.NsOwncloud && (prop.Local == "share-types" || prop.Local == "permissions") {
				filters := make([]*link.ListPublicSharesRequest_Filter, 0, len(resourceInfos))
				for i := range resourceInfos {
					// FIXME this is expensive
					// the filters array grow by one for every file in a folder
					// TODO store public links as grants on the storage, reassembling them here is too costly
					// we can then add the filter if the file has share-types=3 in the opaque,
					// same as user / group shares for share indicators
					filters = append(filters, publicshare.ResourceIDFilter(resourceInfos[i].Id))
				}
				client, err := p.selector.Next()
				if err != nil {
					log.Error().Err(err).Msg("error getting grpc client")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				listResp, err := client.ListPublicShares(ctx, &link.ListPublicSharesRequest{Filters: filters})
				if err == nil {
					linkshares = make(map[string]struct{}, len(listResp.Share))
					for i := range listResp.Share {
						linkshares[listResp.Share[i].ResourceId.OpaqueId] = struct{}{}
					}
				} else {
					log.Error().Err(err).Msg("propfindResponse: couldn't list public shares")
					span.SetStatus(codes.Error, err.Error())
				}
				break
			}
		}
	}

	prefer := net.ParsePrefer(r.Header.Get(net.HeaderPrefer))
	returnMinimal := prefer[net.HeaderPreferReturn] == "minimal"

	propRes, err := MultistatusResponse(ctx, &pf, resourceInfos, p.PublicURL, namespace, linkshares, returnMinimal)
	if err != nil {
		log.Error().Err(err).Msg("error formatting propfind")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set(net.HeaderDav, "1, 3, extended-mkcol")
	w.Header().Set(net.HeaderContentType, "application/xml; charset=utf-8")
	if sendTusHeaders {
		w.Header().Add(net.HeaderAccessControlExposeHeaders, net.HeaderTusResumable)
		w.Header().Add(net.HeaderAccessControlExposeHeaders, net.HeaderTusVersion)
		w.Header().Add(net.HeaderAccessControlExposeHeaders, net.HeaderTusExtension)
		w.Header().Set(net.HeaderAccessControlExposeHeaders, strings.Join(w.Header().Values(net.HeaderAccessControlExposeHeaders), ", "))
		w.Header().Set(net.HeaderTusResumable, "1.0.0")
		w.Header().Set(net.HeaderTusVersion, "1.0.0")
		w.Header().Set(net.HeaderTusExtension, "creation, creation-with-upload, checksum, expiration")
	}
	w.Header().Add(net.HeaderVary, net.HeaderPrefer)
	w.Header().Set(net.HeaderVary, strings.Join(w.Header().Values(net.HeaderVary), ", "))
	if returnMinimal {
		w.Header().Set(net.HeaderPreferenceApplied, "return=minimal")
	}

	w.WriteHeader(http.StatusMultiStatus)
	if _, err := w.Write(propRes); err != nil {
		log.Err(err).Msg("error writing response")
	}
}

// TODO this is just a stat -> rename
func (p *Handler) statSpace(ctx context.Context, ref *provider.Reference, metadataKeys, fieldMaskPaths []string) (*provider.ResourceInfo, *rpc.Status, error) {
	client, err := p.selector.Next()
	if err != nil {
		return nil, nil, err
	}
	req := &provider.StatRequest{
		Ref:                   ref,
		ArbitraryMetadataKeys: metadataKeys,
		FieldMask:             &fieldmaskpb.FieldMask{Paths: fieldMaskPaths},
	}
	res, err := client.Stat(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	return res.GetInfo(), res.GetStatus(), nil
}

func (p *Handler) getResourceInfos(ctx context.Context, w http.ResponseWriter, r *http.Request, pf XML, spaces []*provider.StorageSpace, requestPath string, depth net.Depth, log zerolog.Logger) ([]*provider.ResourceInfo, bool, bool) {
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "get_resource_infos")
	span.SetAttributes(attribute.KeyValue{Key: "requestPath", Value: attribute.StringValue(requestPath)})
	span.SetAttributes(attribute.KeyValue{Key: "depth", Value: attribute.StringValue(depth.String())})
	defer span.End()

	metadataKeys, fieldMaskPaths := metadataKeys(pf)

	// we need to stat all spaces to aggregate the root etag, mtime and size
	// TODO cache per space (hah, no longer per user + per space!)
	var (
		err                 error
		rootInfo            *provider.ResourceInfo
		mostRecentChildInfo *provider.ResourceInfo
		aggregatedChildSize uint64
		spaceMap            = make(map[*provider.ResourceInfo]spaceData, len(spaces))
	)
	for _, space := range spaces {
		spacePath := ""
		if spacePath = utils.ReadPlainFromOpaque(space.Opaque, "path"); spacePath == "" {
			continue // not mounted
		}
		if space.RootInfo == nil {
			spaceRef, err := spacelookup.MakeStorageSpaceReference(space.Id.OpaqueId, ".")
			if err != nil {
				continue
			}
			info, status, err := p.statSpace(ctx, &spaceRef, metadataKeys, fieldMaskPaths)
			if err != nil || status.GetCode() != rpc.Code_CODE_OK {
				continue
			}
			space.RootInfo = info
		}

		// TODO separate stats to the path or to the children, after statting all children update the mtime/etag
		// TODO get mtime, and size from space as well, so we no longer have to stat here? would require sending the requested metadata keys as well
		// root should be a ResourceInfo so it can contain the full stat, not only the id ... do we even need spaces then?
		// metadata keys could all be prefixed with "root." to indicate we want more than the root id ...
		// TODO can we reuse the space.rootinfo?
		spaceRef := spacelookup.MakeRelativeReference(space, requestPath, false)
		var info *provider.ResourceInfo
		if spaceRef.Path == "." && utils.ResourceIDEqual(spaceRef.ResourceId, space.Root) {
			info = space.RootInfo
		} else {
			var status *rpc.Status
			info, status, err = p.statSpace(ctx, spaceRef, metadataKeys, fieldMaskPaths)
			if err != nil || status.GetCode() != rpc.Code_CODE_OK {
				continue
			}
		}

		// adjust path
		info.Path = filepath.Join(spacePath, spaceRef.Path)
		info.Name = filepath.Base(info.Path)

		spaceMap[info] = spaceData{Ref: spaceRef, SpaceType: space.SpaceType}

		if rootInfo == nil && requestPath == info.Path {
			rootInfo = info
		} else if requestPath != spacePath && strings.HasPrefix(spacePath, requestPath) { // Check if the space is a child of the requested path
			// aggregate child metadata
			aggregatedChildSize += info.Size
			if mostRecentChildInfo == nil {
				mostRecentChildInfo = info
				continue
			}
			if mostRecentChildInfo.Mtime == nil || (info.Mtime != nil && utils.TSToUnixNano(info.Mtime) > utils.TSToUnixNano(mostRecentChildInfo.Mtime)) {
				mostRecentChildInfo = info
			}
		}
	}

	if len(spaceMap) == 0 || rootInfo == nil {
		// TODO if we have children invent node on the fly
		w.WriteHeader(http.StatusNotFound)
		m := "Resource not found"
		b, err := errors.Marshal(http.StatusNotFound, m, "", "")
		errors.HandleWebdavError(&log, w, b, err)
		return nil, false, false
	}
	if mostRecentChildInfo != nil {
		if rootInfo.Mtime == nil || (mostRecentChildInfo.Mtime != nil && utils.TSToUnixNano(mostRecentChildInfo.Mtime) > utils.TSToUnixNano(rootInfo.Mtime)) {
			rootInfo.Mtime = mostRecentChildInfo.Mtime
			if mostRecentChildInfo.Etag != "" {
				rootInfo.Etag = mostRecentChildInfo.Etag
			}
		}
		if rootInfo.Etag == "" {
			rootInfo.Etag = mostRecentChildInfo.Etag
		}
	}

	// add size of children
	rootInfo.Size += aggregatedChildSize

	resourceInfos := []*provider.ResourceInfo{
		rootInfo, // PROPFIND always includes the root resource
	}

	if rootInfo.Type == provider.ResourceType_RESOURCE_TYPE_FILE || depth == net.DepthZero {
		// If the resource is a file then it can't have any children so we can
		// stop here.
		return resourceInfos, true, true
	}

	childInfos := map[string]*provider.ResourceInfo{}
	for spaceInfo, spaceData := range spaceMap {
		switch {
		case spaceInfo.Type != provider.ResourceType_RESOURCE_TYPE_CONTAINER && depth != net.DepthInfinity:
			addChild(childInfos, spaceInfo, requestPath, rootInfo)

		case spaceInfo.Type == provider.ResourceType_RESOURCE_TYPE_CONTAINER && depth == net.DepthOne:
			switch {
			case strings.HasPrefix(requestPath, spaceInfo.Path) && spaceData.SpaceType != "virtual":
				client, err := p.selector.Next()
				if err != nil {
					log.Error().Err(err).Msg("error getting grpc client")
					w.WriteHeader(http.StatusInternalServerError)
					return nil, false, false
				}
				req := &provider.ListContainerRequest{
					Ref:                   spaceData.Ref,
					ArbitraryMetadataKeys: metadataKeys,
				}
				res, err := client.ListContainer(ctx, req)
				if err != nil {
					log.Error().Err(err).Msg("error sending list container grpc request")
					w.WriteHeader(http.StatusInternalServerError)
					return nil, false, false
				}

				if res.Status.Code != rpc.Code_CODE_OK {
					log.Debug().Interface("status", res.Status).Msg("List Container not ok, skipping")
					continue
				}
				for _, info := range res.Infos {
					info.Path = path.Join(requestPath, info.Path)
				}
				resourceInfos = append(resourceInfos, res.Infos...)
			case strings.HasPrefix(spaceInfo.Path, requestPath): // space is a deep child of the requested path
				addChild(childInfos, spaceInfo, requestPath, rootInfo)
			}

		case depth == net.DepthInfinity:
			// use a stack to explore sub-containers breadth-first
			if spaceInfo != rootInfo {
				resourceInfos = append(resourceInfos, spaceInfo)
			}
			stack := []*provider.ResourceInfo{spaceInfo}
			for len(stack) != 0 {
				info := stack[0]
				stack = stack[1:]

				if info.Type != provider.ResourceType_RESOURCE_TYPE_CONTAINER || spaceData.SpaceType == "virtual" {
					continue
				}
				client, err := p.selector.Next()
				if err != nil {
					log.Error().Err(err).Msg("error getting grpc client")
					w.WriteHeader(http.StatusInternalServerError)
					return nil, false, false
				}
				req := &provider.ListContainerRequest{
					Ref: &provider.Reference{
						ResourceId: spaceInfo.Id,
						// TODO here we cut of the path that we added after stating the space above
						Path: utils.MakeRelativePath(strings.TrimPrefix(info.Path, spaceInfo.Path)),
					},
					ArbitraryMetadataKeys: metadataKeys,
				}
				res, err := client.ListContainer(ctx, req) // FIXME public link depth infinity -> "gateway: could not find provider: gateway: error calling ListStorageProviders: rpc error: code = PermissionDenied desc = auth: core access token is invalid"
				if err != nil {
					log.Error().Err(err).Interface("info", info).Msg("error sending list container grpc request")
					w.WriteHeader(http.StatusInternalServerError)
					return nil, false, false
				}
				if res.Status.Code != rpc.Code_CODE_OK {
					log.Debug().Interface("status", res.Status).Msg("List Container not ok, skipping")
					continue
				}

				// check sub-containers in reverse order and add them to the stack
				// the reversed order here will produce a more logical sorting of results
				for i := len(res.Infos) - 1; i >= 0; i-- {
					// add path to resource
					res.Infos[i].Path = filepath.Join(info.Path, res.Infos[i].Path)
					if res.Infos[i].Type == provider.ResourceType_RESOURCE_TYPE_CONTAINER {
						stack = append(stack, res.Infos[i])
					}
				}

				resourceInfos = append(resourceInfos, res.Infos...)
				// TODO: stream response to avoid storing too many results in memory
				// we can do that after having stated the root.
			}
		}
	}

	if rootInfo.Type == provider.ResourceType_RESOURCE_TYPE_CONTAINER {
		// now add all aggregated child infos
		for _, childInfo := range childInfos {
			resourceInfos = append(resourceInfos, childInfo)
		}
	}

	sendTusHeaders := true
	// let clients know this collection supports tus.io POST requests to start uploads
	if rootInfo.Type == provider.ResourceType_RESOURCE_TYPE_CONTAINER {
		if rootInfo.Opaque != nil {
			_, ok := rootInfo.Opaque.Map["disable_tus"]
			sendTusHeaders = !ok
		}
	}

	return resourceInfos, sendTusHeaders, true
}

func (p *Handler) getSpaceResourceInfos(ctx context.Context, w http.ResponseWriter, r *http.Request, pf XML, ref *provider.Reference, requestPath string, depth net.Depth, log zerolog.Logger) ([]*provider.ResourceInfo, bool) {
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "get_space_resource_infos")
	span.SetAttributes(attribute.KeyValue{Key: "requestPath", Value: attribute.StringValue(requestPath)})
	span.SetAttributes(attribute.KeyValue{Key: "depth", Value: attribute.StringValue(depth.String())})
	defer span.End()

	client, err := p.selector.Next()
	if err != nil {
		log.Error().Err(err).Msg("error getting grpc client")
		w.WriteHeader(http.StatusInternalServerError)
		return nil, false
	}

	metadataKeys, _ := metadataKeys(pf)

	resourceInfos := []*provider.ResourceInfo{}

	req := &provider.ListContainerRequest{
		Ref:                   ref,
		ArbitraryMetadataKeys: metadataKeys,
		FieldMask:             &fieldmaskpb.FieldMask{Paths: []string{"*"}}, // TODO use more sophisticated filter
	}
	res, err := client.ListContainer(ctx, req)
	if err != nil {
		log.Error().Err(err).Msg("error sending list container grpc request")
		w.WriteHeader(http.StatusInternalServerError)
		return nil, false
	}

	if res.Status.Code != rpc.Code_CODE_OK {
		log.Debug().Interface("status", res.Status).Msg("List Container not ok, skipping")
		w.WriteHeader(http.StatusInternalServerError)
		return nil, false
	}
	for _, info := range res.Infos {
		info.Path = path.Join(requestPath, info.Path)
	}
	resourceInfos = append(resourceInfos, res.Infos...)

	if depth == net.DepthInfinity {
		// use a stack to explore sub-containers breadth-first
		stack := resourceInfos
		for len(stack) != 0 {
			info := stack[0]
			stack = stack[1:]

			if info.Type != provider.ResourceType_RESOURCE_TYPE_CONTAINER /*|| space.SpaceType == "virtual"*/ {
				continue
			}
			req := &provider.ListContainerRequest{
				Ref: &provider.Reference{
					ResourceId: info.Id,
					Path:       ".",
				},
				ArbitraryMetadataKeys: metadataKeys,
			}
			res, err := client.ListContainer(ctx, req) // FIXME public link depth infinity -> "gateway: could not find provider: gateway: error calling ListStorageProviders: rpc error: code = PermissionDenied desc = auth: core access token is invalid"
			if err != nil {
				log.Error().Err(err).Interface("info", info).Msg("error sending list container grpc request")
				w.WriteHeader(http.StatusInternalServerError)
				return nil, false
			}
			if res.Status.Code != rpc.Code_CODE_OK {
				log.Debug().Interface("status", res.Status).Msg("List Container not ok, skipping")
				continue
			}

			// check sub-containers in reverse order and add them to the stack
			// the reversed order here will produce a more logical sorting of results
			for i := len(res.Infos) - 1; i >= 0; i-- {
				// add path to resource
				res.Infos[i].Path = filepath.Join(info.Path, res.Infos[i].Path)
				res.Infos[i].Path = utils.MakeRelativePath(res.Infos[i].Path)
				if res.Infos[i].Type == provider.ResourceType_RESOURCE_TYPE_CONTAINER {
					stack = append(stack, res.Infos[i])
				}
			}

			resourceInfos = append(resourceInfos, res.Infos...)
			// TODO: stream response to avoid storing too many results in memory
			// we can do that after having stated the root.
		}
	}

	return resourceInfos, true
}

func metadataKeysWithPrefix(prefix string, keys []string) []string {
	fullKeys := []string{}
	for _, key := range keys {
		fullKeys = append(fullKeys, fmt.Sprintf("%s.%s", prefix, key))
	}
	return fullKeys
}

// metadataKeys splits the propfind properties into arbitrary metadata and ResourceInfo field mask paths
func metadataKeys(pf XML) ([]string, []string) {

	var metadataKeys []string
	var fieldMaskKeys []string

	if pf.Allprop != nil {
		// TODO this changes the behavior and returns all properties if allprops has been set,
		// but allprops should only return some default properties
		// see https://tools.ietf.org/html/rfc4918#section-9.1
		// the description of arbitrary_metadata_keys in https://cs3org.github.io/cs3apis/#cs3.storage.provider.v1beta1.ListContainerRequest an others may need clarification
		// tracked in https://github.com/cs3org/cs3apis/issues/104
		metadataKeys = append(metadataKeys, "*")
		fieldMaskKeys = append(fieldMaskKeys, "*")
	} else {
		metadataKeys = make([]string, 0, len(pf.Prop))
		fieldMaskKeys = make([]string, 0, len(pf.Prop))
		for i := range pf.Prop {
			if requiresExplicitFetching(&pf.Prop[i]) {
				key := metadataKeyOf(&pf.Prop[i])
				switch key {
				case "share-types":
					fieldMaskKeys = append(fieldMaskKeys, key)
				case "http://owncloud.org/ns/audio":
					metadataKeys = append(metadataKeys, metadataKeysWithPrefix("libre.graph.audio", audioKeys)...)
				case "http://owncloud.org/ns/location":
					metadataKeys = append(metadataKeys, metadataKeysWithPrefix("libre.graph.location", locationKeys)...)
				case "http://owncloud.org/ns/image":
					metadataKeys = append(metadataKeys, metadataKeysWithPrefix("libre.graph.image", imageKeys)...)
				case "http://owncloud.org/ns/photo":
					metadataKeys = append(metadataKeys, metadataKeysWithPrefix("libre.graph.photo", photoKeys)...)
				default:
					metadataKeys = append(metadataKeys, key)
				}

			}
		}
	}
	return metadataKeys, fieldMaskKeys
}

func addChild(childInfos map[string]*provider.ResourceInfo,
	spaceInfo *provider.ResourceInfo,
	requestPath string,
	rootInfo *provider.ResourceInfo,
) {
	if spaceInfo == rootInfo {
		return // already accounted for
	}

	childPath := strings.TrimPrefix(spaceInfo.Path, requestPath)
	childName, tail := router.ShiftPath(childPath)
	if tail != "/" {
		spaceInfo.Type = provider.ResourceType_RESOURCE_TYPE_CONTAINER
		spaceInfo.Checksum = nil
		// TODO unset opaque checksum
	}
	spaceInfo.Path = path.Join(requestPath, childName)
	if existingChild, ok := childInfos[childName]; ok {
		// aggregate size
		childInfos[childName].Size += spaceInfo.Size
		// use most recent child
		if existingChild.Mtime == nil || (spaceInfo.Mtime != nil && utils.TSToUnixNano(spaceInfo.Mtime) > utils.TSToUnixNano(existingChild.Mtime)) {
			childInfos[childName].Mtime = spaceInfo.Mtime
			childInfos[childName].Etag = spaceInfo.Etag
		}
		// only update fileid if the resource is a direct child
		if tail == "/" {
			childInfos[childName].Id = spaceInfo.Id
		}
	} else {
		childInfos[childName] = spaceInfo
	}
}

func requiresExplicitFetching(n *xml.Name) bool {
	switch n.Space {
	case net.NsDav:
		switch n.Local {
		case "quota-available-bytes", "quota-used-bytes", "lockdiscovery":
			//  A <DAV:allprop> PROPFIND request SHOULD NOT return DAV:quota-available-bytes and DAV:quota-used-bytes
			// from https://www.rfc-editor.org/rfc/rfc4331.html#section-2
			return true
		default:
			return false
		}
	case net.NsOwncloud:
		switch n.Local {
		case "favorite", "share-types", "checksums", "size", "tags", "audio", "location", "image", "photo":
			return true
		default:
			return false
		}
	case net.NsOCS:
		return false
	}
	return true
}

// ReadPropfind extracts and parses the propfind XML information from a Reader
// from https://github.com/golang/net/blob/e514e69ffb8bc3c76a71ae40de0118d794855992/webdav/xml.go#L178-L205
func ReadPropfind(r io.Reader) (pf XML, status int, err error) {
	c := countingReader{r: r}
	if err = xml.NewDecoder(&c).Decode(&pf); err != nil {
		if err == io.EOF {
			if c.n == 0 {
				// An empty body means to propfind allprop.
				// http://www.webdav.org/specs/rfc4918.html#METHOD_PROPFIND
				return XML{Allprop: new(struct{})}, 0, nil
			}
			err = errors.ErrInvalidPropfind
		}
		return XML{}, http.StatusBadRequest, err
	}

	if pf.Allprop == nil && pf.Include != nil {
		return XML{}, http.StatusBadRequest, errors.ErrInvalidPropfind
	}
	if pf.Allprop != nil && (pf.Prop != nil || pf.Propname != nil) {
		return XML{}, http.StatusBadRequest, errors.ErrInvalidPropfind
	}
	if pf.Prop != nil && pf.Propname != nil {
		return XML{}, http.StatusBadRequest, errors.ErrInvalidPropfind
	}
	if pf.Propname == nil && pf.Allprop == nil && pf.Prop == nil {
		// jfd: I think <d:prop></d:prop> is perfectly valid ... treat it as allprop
		return XML{Allprop: new(struct{})}, 0, nil
	}
	return pf, 0, nil
}

// MultistatusResponse converts a list of resource infos into a multistatus response string
func MultistatusResponse(ctx context.Context, pf *XML, mds []*provider.ResourceInfo, publicURL, ns string, linkshares map[string]struct{}, returnMinimal bool) ([]byte, error) {
	g, ctx := errgroup.WithContext(ctx)

	type work struct {
		position int
		info     *provider.ResourceInfo
	}
	type result struct {
		position int
		info     *ResponseXML
	}
	workChan := make(chan work, len(mds))
	resultChan := make(chan result, len(mds))

	// Distribute work
	g.Go(func() error {
		defer close(workChan)
		for i, md := range mds {
			select {
			case workChan <- work{position: i, info: md}:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	})

	// Spawn workers that'll concurrently work the queue
	numWorkers := 50
	if len(mds) < numWorkers {
		numWorkers = len(mds)
	}
	for i := 0; i < numWorkers; i++ {
		g.Go(func() error {
			for work := range workChan {
				res, err := mdToPropResponse(ctx, pf, work.info, publicURL, ns, linkshares, returnMinimal)
				if err != nil {
					return err
				}
				select {
				case resultChan <- result{position: work.position, info: res}:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			return nil
		})
	}

	// Wait for things to settle down, then close results chan
	go func() {
		_ = g.Wait() // error is checked later
		close(resultChan)
	}()

	if err := g.Wait(); err != nil {
		return nil, err
	}

	responses := make([]*ResponseXML, len(mds))
	for res := range resultChan {
		responses[res.position] = res.info
	}

	msr := NewMultiStatusResponseXML()
	msr.Responses = responses
	msg, err := xml.Marshal(msr)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

// mdToPropResponse converts the CS3 metadata into a webdav PropResponse
// ns is the CS3 namespace that needs to be removed from the CS3 path before
// prefixing it with the baseURI
func mdToPropResponse(ctx context.Context, pf *XML, md *provider.ResourceInfo, publicURL, ns string, linkshares map[string]struct{}, returnMinimal bool) (*ResponseXML, error) {
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "md_to_prop_response")
	span.SetAttributes(attribute.KeyValue{Key: "publicURL", Value: attribute.StringValue(publicURL)})
	span.SetAttributes(attribute.KeyValue{Key: "ns", Value: attribute.StringValue(ns)})
	defer span.End()

	sublog := appctx.GetLogger(ctx).With().Interface("md", md).Str("ns", ns).Logger()
	id := md.Id
	p := strings.TrimPrefix(md.Path, ns)

	baseURI := ctx.Value(net.CtxKeyBaseURI).(string)

	ref := path.Join(baseURI, p)
	if md.Type == provider.ResourceType_RESOURCE_TYPE_CONTAINER {
		ref += "/"
	}

	response := ResponseXML{
		Href:     net.EncodePath(ref),
		Propstat: []PropstatXML{},
	}

	var ls *link.PublicShare

	// -1 indicates uncalculated
	// -2 indicates unknown (default)
	// -3 indicates unlimited
	quota := net.PropQuotaUnknown
	size := strconv.FormatUint(md.Size, 10)
	var lock *provider.Lock
	shareTypes := ""
	// TODO refactor helper functions: GetOpaqueJSONEncoded(opaque, key string, *struct) err, GetOpaquePlainEncoded(opaque, key) value, err
	// or use ok like pattern and return bool?
	if md.Opaque != nil && md.Opaque.Map != nil {
		if md.Opaque.Map["link-share"] != nil && md.Opaque.Map["link-share"].Decoder == "json" {
			ls = &link.PublicShare{}
			err := json.Unmarshal(md.Opaque.Map["link-share"].Value, ls)
			if err != nil {
				sublog.Error().Err(err).Msg("could not unmarshal link json")
			}
		}
		if quota = utils.ReadPlainFromOpaque(md.Opaque, "quota"); quota == "" {
			quota = net.PropQuotaUnknown
		}
		if md.Opaque.Map["lock"] != nil && md.Opaque.Map["lock"].Decoder == "json" {
			lock = &provider.Lock{}
			err := json.Unmarshal(md.Opaque.Map["lock"].Value, lock)
			if err != nil {
				sublog.Error().Err(err).Msg("could not unmarshal locks json")
			}
		}
		shareTypes = utils.ReadPlainFromOpaque(md.Opaque, "share-types")
	}
	role := conversions.RoleFromResourcePermissions(md.PermissionSet, ls != nil)

	if md.Space != nil && md.Space.SpaceType != "grant" && utils.ResourceIDEqual(md.Space.Root, id) {
		// a space root is never shared
		shareTypes = ""
	}
	var wdp string
	isPublic := ls != nil
	isShared := shareTypes != "" && !net.IsCurrentUserOwnerOrManager(ctx, md.Owner, md)
	if md.PermissionSet != nil {
		wdp = role.WebDAVPermissions(
			md.Type == provider.ResourceType_RESOURCE_TYPE_CONTAINER,
			isShared,
			false,
			isPublic,
		)
	}

	// replace fileid of /public/{token} mountpoint with grant fileid
	if ls != nil && id != nil && id.SpaceId == utils.PublicStorageSpaceID && id.OpaqueId == ls.Token {
		id = ls.ResourceId
	}

	propstatOK := PropstatXML{
		Status: "HTTP/1.1 200 OK",
		Prop:   []prop.PropertyXML{},
	}
	propstatNotFound := PropstatXML{
		Status: "HTTP/1.1 404 Not Found",
		Prop:   []prop.PropertyXML{},
	}

	appendToOK := func(p ...prop.PropertyXML) {
		propstatOK.Prop = append(propstatOK.Prop, p...)
	}
	appendToNotFound := func(p ...prop.PropertyXML) {
		propstatNotFound.Prop = append(propstatNotFound.Prop, p...)
	}
	if returnMinimal {
		appendToNotFound = func(p ...prop.PropertyXML) {}
	}

	appendMetadataProp := func(metadata map[string]string, tagNamespace string, name string, metadataPrefix string, keys []string) {
		content := strings.Builder{}
		for _, key := range keys {
			kebabCaseKey := strcase.ToKebab(key)
			if v, ok := metadata[fmt.Sprintf("%s.%s", metadataPrefix, key)]; ok {
				content.WriteString("<")
				content.WriteString(tagNamespace)
				content.WriteString(":")
				content.WriteString(kebabCaseKey)
				content.WriteString(">")
				content.Write(prop.Escaped("", v).InnerXML)
				content.WriteString("</")
				content.WriteString(tagNamespace)
				content.WriteString(":")
				content.WriteString(kebabCaseKey)
				content.WriteString(">")
			}
		}

		propName := fmt.Sprintf("%s:%s", tagNamespace, name)
		if content.Len() > 0 {
			appendToOK(prop.Raw(propName, content.String()))
		} else {
			appendToNotFound(prop.NotFound(propName))
		}
	}

	// when allprops has been requested
	if pf.Allprop != nil {
		// return all known properties

		if id != nil {
			sid := storagespace.FormatResourceID(id)
			appendToOK(
				prop.Escaped("oc:id", sid),
				prop.Escaped("oc:fileid", sid),
				prop.Escaped("oc:spaceid", storagespace.FormatStorageID(id.StorageId, id.SpaceId)),
			)
		}

		if md.ParentId != nil {
			appendToOK(prop.Escaped("oc:file-parent", storagespace.FormatResourceID(md.ParentId)))
		} else {
			appendToNotFound(prop.NotFound("oc:file-parent"))
		}

		// we need to add the shareid if possible - the only way to extract it here is to parse it from the path
		if ref, err := storagespace.ParseReference(strings.TrimPrefix(p, "/")); err == nil && ref.GetResourceId().GetSpaceId() == utils.ShareStorageSpaceID {
			appendToOK(prop.Raw("oc:shareid", ref.GetResourceId().GetOpaqueId()))
		}

		if md.Name != "" {
			appendToOK(prop.Escaped("oc:name", md.Name))
			appendToOK(prop.Escaped("d:displayname", md.Name))
		}

		if md.Etag != "" {
			// etags must be enclosed in double quotes and cannot contain them.
			// See https://tools.ietf.org/html/rfc7232#section-2.3 for details
			// TODO(jfd) handle weak tags that start with 'W/'
			appendToOK(prop.Escaped("d:getetag", quoteEtag(md.Etag)))
		}

		if md.PermissionSet != nil {
			appendToOK(prop.Escaped("oc:permissions", wdp))
		}

		// always return size, well nearly always ... public link shares are a little weird
		if md.Type == provider.ResourceType_RESOURCE_TYPE_CONTAINER {
			appendToOK(prop.Raw("d:resourcetype", "<d:collection/>"))
			if ls == nil {
				appendToOK(prop.Escaped("oc:size", size))
			}
			// A <DAV:allprop> PROPFIND request SHOULD NOT return DAV:quota-available-bytes and DAV:quota-used-bytes
			// from https://www.rfc-editor.org/rfc/rfc4331.html#section-2
			// appendToOK(prop.NewProp("d:quota-used-bytes", size))
			// appendToOK(prop.NewProp("d:quota-available-bytes", quota))
		} else {
			appendToOK(
				prop.Escaped("d:resourcetype", ""),
				prop.Escaped("d:getcontentlength", size),
			)
			if md.MimeType != "" {
				appendToOK(prop.Escaped("d:getcontenttype", md.MimeType))
			}
		}
		// Finder needs the getLastModified property to work.
		if md.Mtime != nil {
			t := utils.TSToTime(md.Mtime).UTC()
			lastModifiedString := t.Format(net.RFC1123)
			appendToOK(prop.Escaped("d:getlastmodified", lastModifiedString))
		}

		// stay bug compatible with oc10, see https://github.com/owncloud/core/pull/38304#issuecomment-762185241
		var checksums strings.Builder
		if md.Checksum != nil {
			checksums.WriteString("<oc:checksum>")
			checksums.WriteString(strings.ToUpper(string(storageprovider.GRPC2PKGXS(md.Checksum.Type))))
			checksums.WriteString(":")
			checksums.WriteString(md.Checksum.Sum)
		}
		if md.Opaque != nil {
			if e, ok := md.Opaque.Map["md5"]; ok {
				if checksums.Len() == 0 {
					checksums.WriteString("<oc:checksum>MD5:")
				} else {
					checksums.WriteString(" MD5:")
				}
				checksums.Write(e.Value)
			}
			if e, ok := md.Opaque.Map["adler32"]; ok {
				if checksums.Len() == 0 {
					checksums.WriteString("<oc:checksum>ADLER32:")
				} else {
					checksums.WriteString(" ADLER32:")
				}
				checksums.Write(e.Value)
			}
		}
		if checksums.Len() > 0 {
			checksums.WriteString("</oc:checksum>")
			appendToOK(prop.Raw("oc:checksums", checksums.String()))
		}

		if k := md.GetArbitraryMetadata().GetMetadata(); k != nil {
			propstatOK.Prop = append(propstatOK.Prop, prop.Raw("oc:tags", k["tags"]))
			appendMetadataProp(k, "oc", "audio", "libre.graph.audio", audioKeys)
			appendMetadataProp(k, "oc", "location", "libre.graph.location", locationKeys)
			appendMetadataProp(k, "oc", "image", "libre.graph.image", imageKeys)
			appendMetadataProp(k, "oc", "photo", "libre.graph.photo", photoKeys)
		}

		if md.Type == provider.ResourceType_RESOURCE_TYPE_CONTAINER {
			// directories have no preview
			appendToNotFound(prop.NotFound("oc:has-preview"))
		} else if md.MimeType != "" {
			hasPreview(md, appendToOK)
		}

		// ls do not report any properties as missing by default
		if ls == nil {
			// favorites from arbitrary metadata
			if k := md.GetArbitraryMetadata(); k == nil {
				appendToOK(prop.Raw("oc:favorite", "0"))
			} else if amd := k.GetMetadata(); amd == nil {
				appendToOK(prop.Raw("oc:favorite", "0"))
			} else if v, ok := amd[net.PropOcFavorite]; ok && v != "" {
				appendToOK(prop.Escaped("oc:favorite", v))
			} else {
				appendToOK(prop.Raw("oc:favorite", "0"))
			}
		}

		if lock != nil {
			appendToOK(prop.Raw("d:lockdiscovery", activeLocks(&sublog, lock)))
		}
		// TODO return other properties ... but how do we put them in a namespace?
	} else {
		// otherwise return only the requested properties
		for i := range pf.Prop {
			switch pf.Prop[i].Space {
			case net.NsOwncloud:
				switch pf.Prop[i].Local {
				// TODO(jfd): maybe phoenix and the other clients can just use this id as an opaque string?
				// I tested the desktop client and phoenix to annotate which properties are requestted, see below cases
				case "fileid": // phoenix only
					if id != nil {
						appendToOK(prop.Escaped("oc:fileid", storagespace.FormatResourceID(id)))
					} else {
						appendToNotFound(prop.NotFound("oc:fileid"))
					}
				case "id": // desktop client only
					if id != nil {
						appendToOK(prop.Escaped("oc:id", storagespace.FormatResourceID(id)))
					} else {
						appendToNotFound(prop.NotFound("oc:id"))
					}
				case "file-parent":
					if md.ParentId != nil {
						appendToOK(prop.Escaped("oc:file-parent", storagespace.FormatResourceID(md.ParentId)))
					} else {
						appendToNotFound(prop.NotFound("oc:file-parent"))
					}
				case "spaceid":
					if id != nil {
						appendToOK(prop.Escaped("oc:spaceid", storagespace.FormatStorageID(id.StorageId, id.SpaceId)))
					} else {
						appendToNotFound(prop.Escaped("oc:spaceid", ""))
					}
				case "permissions": // both
					// oc:permissions take several char flags to indicate the permissions the user has on this node:
					// D = delete
					// NV = update (renameable moveable)
					// W = update (files only)
					// CK = create (folders only)
					// S = Shared
					// R = Shareable (Reshare)
					// M = Mounted
					// in contrast, the ocs:share-permissions further down below indicate clients the maximum permissions that can be granted
					appendToOK(prop.Escaped("oc:permissions", wdp))
				case "public-link-permission": // only on a share root node
					if ls != nil && md.PermissionSet != nil {
						appendToOK(prop.Escaped("oc:public-link-permission", role.OCSPermissions().String()))
					} else {
						appendToNotFound(prop.NotFound("oc:public-link-permission"))
					}
				case "public-link-item-type": // only on a share root node
					if ls != nil {
						if md.Type == provider.ResourceType_RESOURCE_TYPE_CONTAINER {
							appendToOK(prop.Raw("oc:public-link-item-type", "folder"))
						} else {
							appendToOK(prop.Raw("oc:public-link-item-type", "file"))
							// redirectref is another option
						}
					} else {
						appendToNotFound(prop.NotFound("oc:public-link-item-type"))
					}
				case "public-link-share-datetime":
					if ls != nil && ls.Mtime != nil {
						t := utils.TSToTime(ls.Mtime).UTC() // TODO or ctime?
						shareTimeString := t.Format(net.RFC1123)
						appendToOK(prop.Escaped("oc:public-link-share-datetime", shareTimeString))
					} else {
						appendToNotFound(prop.NotFound("oc:public-link-share-datetime"))
					}
				case "public-link-share-owner":
					if ls != nil && ls.Owner != nil {
						if net.IsCurrentUserOwnerOrManager(ctx, ls.Owner, nil) {
							u := ctxpkg.ContextMustGetUser(ctx)
							appendToOK(prop.Escaped("oc:public-link-share-owner", u.Username))
						} else {
							u, _ := ctxpkg.ContextGetUser(ctx)
							sublog.Error().Interface("share", ls).Interface("user", u).Msg("the current user in the context should be the owner of a public link share")
							appendToNotFound(prop.NotFound("oc:public-link-share-owner"))
						}
					} else {
						appendToNotFound(prop.NotFound("oc:public-link-share-owner"))
					}
				case "public-link-expiration":
					if ls != nil && ls.Expiration != nil {
						t := utils.TSToTime(ls.Expiration).UTC()
						expireTimeString := t.Format(net.RFC1123)
						appendToOK(prop.Escaped("oc:public-link-expiration", expireTimeString))
					} else {
						appendToNotFound(prop.NotFound("oc:public-link-expiration"))
					}
				case "size": // phoenix only
					// TODO we cannot find out if md.Size is set or not because ints in go default to 0
					// TODO what is the difference to d:quota-used-bytes (which only exists for collections)?
					// oc:size is available on files and folders and behaves like d:getcontentlength or d:quota-used-bytes respectively
					// The hasPrefix is a workaround to make children of the link root show a size if they have 0 bytes
					if ls == nil || strings.HasPrefix(p, "/"+ls.Token+"/") {
						appendToOK(prop.Escaped("oc:size", size))
					} else {
						// link share root collection has no size
						appendToNotFound(prop.NotFound("oc:size"))
					}
				case "owner-id": // phoenix only
					if md.Owner != nil {
						if net.IsCurrentUserOwnerOrManager(ctx, md.Owner, md) {
							u := ctxpkg.ContextMustGetUser(ctx)
							appendToOK(prop.Escaped("oc:owner-id", u.Username))
						} else {
							sublog.Debug().Msg("TODO fetch user username")
							appendToNotFound(prop.NotFound("oc:owner-id"))
						}
					} else {
						appendToNotFound(prop.NotFound("oc:owner-id"))
					}
				case "favorite": // phoenix only
					// TODO: can be 0 or 1?, in oc10 it is present or not
					// TODO: read favorite via separate call? that would be expensive? I hope it is in the md
					// TODO: this boolean favorite property is so horribly wrong ... either it is presont, or it is not ... unless ... it is possible to have a non binary value ... we need to double check
					if ls == nil {
						if k := md.GetArbitraryMetadata(); k == nil {
							appendToOK(prop.Raw("oc:favorite", "0"))
						} else if amd := k.GetMetadata(); amd == nil {
							appendToOK(prop.Raw("oc:favorite", "0"))
						} else if v, ok := amd[net.PropOcFavorite]; ok && v != "" {
							appendToOK(prop.Raw("oc:favorite", "1"))
						} else {
							appendToOK(prop.Raw("oc:favorite", "0"))
						}
					} else {
						// link share root collection has no favorite
						appendToNotFound(prop.NotFound("oc:favorite"))
					}
				case "checksums": // desktop ... not really ... the desktop sends the OC-Checksum header

					// stay bug compatible with oc10, see https://github.com/owncloud/core/pull/38304#issuecomment-762185241
					var checksums strings.Builder
					if md.Checksum != nil {
						checksums.WriteString("<oc:checksum>")
						checksums.WriteString(strings.ToUpper(string(storageprovider.GRPC2PKGXS(md.Checksum.Type))))
						checksums.WriteString(":")
						checksums.WriteString(md.Checksum.Sum)
					}
					if md.Opaque != nil {
						if e, ok := md.Opaque.Map["md5"]; ok {
							if checksums.Len() == 0 {
								checksums.WriteString("<oc:checksum>MD5:")
							} else {
								checksums.WriteString(" MD5:")
							}
							checksums.Write(e.Value)
						}
						if e, ok := md.Opaque.Map["adler32"]; ok {
							if checksums.Len() == 0 {
								checksums.WriteString("<oc:checksum>ADLER32:")
							} else {
								checksums.WriteString(" ADLER32:")
							}
							checksums.Write(e.Value)
						}
					}
					if checksums.Len() > 13 {
						checksums.WriteString("</oc:checksum>")
						appendToOK(prop.Raw("oc:checksums", checksums.String()))
					} else {
						appendToNotFound(prop.NotFound("oc:checksums"))
					}
				case "share-types": // used to render share indicators to share owners
					var types strings.Builder

					sts := strings.Split(shareTypes, ",")
					for _, shareType := range sts {
						switch shareType {
						case "1": // provider.GranteeType_GRANTEE_TYPE_USER
							types.WriteString("<oc:share-type>" + strconv.Itoa(int(conversions.ShareTypeUser)) + "</oc:share-type>")
						case "2": // provider.GranteeType_GRANTEE_TYPE_GROUP
							types.WriteString("<oc:share-type>" + strconv.Itoa(int(conversions.ShareTypeGroup)) + "</oc:share-type>")
						default:
							sublog.Debug().Interface("shareType", shareType).Msg("unknown share type, ignoring")
						}
					}

					if id != nil {
						if _, ok := linkshares[id.OpaqueId]; ok {
							types.WriteString("<oc:share-type>3</oc:share-type>")
						}
					}

					if types.Len() != 0 {
						appendToOK(prop.Raw("oc:share-types", types.String()))
					} else {
						appendToNotFound(prop.NotFound("oc:" + pf.Prop[i].Local))
					}
				case "owner-display-name": // phoenix only
					if md.Owner != nil {
						if net.IsCurrentUserOwnerOrManager(ctx, md.Owner, md) {
							u := ctxpkg.ContextMustGetUser(ctx)
							appendToOK(prop.Escaped("oc:owner-display-name", u.DisplayName))
						} else {
							sublog.Debug().Msg("TODO fetch user displayname")
							appendToNotFound(prop.NotFound("oc:owner-display-name"))
						}
					} else {
						appendToNotFound(prop.NotFound("oc:owner-display-name"))
					}
				case "downloadURL": // desktop
					if isPublic && md.Type == provider.ResourceType_RESOURCE_TYPE_FILE {
						var path string
						if !ls.PasswordProtected {
							path = p
						} else {
							expiration := time.Unix(int64(ls.Signature.SignatureExpiration.Seconds), int64(ls.Signature.SignatureExpiration.Nanos))
							var sb strings.Builder

							sb.WriteString(p)
							sb.WriteString("?signature=")
							sb.WriteString(ls.Signature.Signature)
							sb.WriteString("&expiration=")
							sb.WriteString(url.QueryEscape(expiration.Format(time.RFC3339)))

							path = sb.String()
						}
						appendToOK(prop.Escaped("oc:downloadURL", publicURL+baseURI+path))
					} else {
						appendToNotFound(prop.NotFound("oc:" + pf.Prop[i].Local))
					}
				case "privatelink":
					privateURL, err := url.Parse(publicURL)
					if err == nil && id != nil {
						privateURL.Path = path.Join(privateURL.Path, "f", storagespace.FormatResourceID(id))
						propstatOK.Prop = append(propstatOK.Prop, prop.Escaped("oc:privatelink", privateURL.String()))
					} else {
						propstatNotFound.Prop = append(propstatNotFound.Prop, prop.NotFound("oc:privatelink"))
					}
				case "signature-auth":
					if isPublic {
						// We only want to add the attribute to the root of the propfind.
						if strings.HasSuffix(p, ls.Token) && ls.Signature != nil {
							expiration := time.Unix(int64(ls.Signature.SignatureExpiration.Seconds), int64(ls.Signature.SignatureExpiration.Nanos))
							var sb strings.Builder
							sb.WriteString("<oc:signature>")
							sb.WriteString(ls.Signature.Signature)
							sb.WriteString("</oc:signature>")
							sb.WriteString("<oc:expiration>")
							sb.WriteString(expiration.Format(time.RFC3339))
							sb.WriteString("</oc:expiration>")

							appendToOK(prop.Raw("oc:signature-auth", sb.String()))
						} else {
							appendToNotFound(prop.NotFound("oc:signature-auth"))
						}
					}
				case "tags":
					if k := md.GetArbitraryMetadata().GetMetadata(); k != nil {
						propstatOK.Prop = append(propstatOK.Prop, prop.Raw("oc:tags", k["tags"]))
					}
				case "audio":
					if k := md.GetArbitraryMetadata().GetMetadata(); k != nil {
						appendMetadataProp(k, "oc", "audio", "libre.graph.audio", audioKeys)
					}
				case "location":
					if k := md.GetArbitraryMetadata().GetMetadata(); k != nil {
						appendMetadataProp(k, "oc", "location", "libre.graph.location", locationKeys)
					}
				case "image":
					if k := md.GetArbitraryMetadata().GetMetadata(); k != nil {
						appendMetadataProp(k, "oc", "image", "libre.graph.image", imageKeys)
					}
				case "photo":
					if k := md.GetArbitraryMetadata().GetMetadata(); k != nil {
						appendMetadataProp(k, "oc", "photo", "libre.graph.photo", photoKeys)
					}
				case "name":
					appendToOK(prop.Escaped("oc:name", md.Name))
				case "shareid":
					if ref, err := storagespace.ParseReference(strings.TrimPrefix(p, "/")); err == nil && ref.GetResourceId().GetSpaceId() == utils.ShareStorageSpaceID {
						appendToOK(prop.Raw("oc:shareid", ref.GetResourceId().GetOpaqueId()))
					}
				case "dDC": // desktop
					fallthrough
				case "data-fingerprint": // desktop
					// used by admins to indicate a backup has been restored,
					// can only occur on the root node
					// server implementation in https://github.com/owncloud/core/pull/24054
					// see https://doc.owncloud.com/server/admin_manual/configuration/server/occ_command.html#maintenance-commands
					// TODO(jfd): double check the client behavior with reva on backup restore
					fallthrough
				case "has-preview":
					if md.Type == provider.ResourceType_RESOURCE_TYPE_CONTAINER {
						// directories have no preview
						appendToNotFound(prop.NotFound("oc:has-preview"))
					} else if md.MimeType != "" {
						hasPreview(md, appendToOK)
					}
				default:
					appendToNotFound(prop.NotFound("oc:" + pf.Prop[i].Local))
				}
			case net.NsDav:
				switch pf.Prop[i].Local {
				case "getetag": // both
					if md.Etag != "" {
						appendToOK(prop.Escaped("d:getetag", quoteEtag(md.Etag)))
					} else {
						appendToNotFound(prop.NotFound("d:getetag"))
					}
				case "getcontentlength": // both
					// see everts stance on this https://stackoverflow.com/a/31621912, he points to http://tools.ietf.org/html/rfc4918#section-15.3
					// > Purpose: Contains the Content-Length header returned by a GET without accept headers.
					// which only would make sense when eg. rendering a plain HTML filelisting when GETing a collection,
					// which is not the case ... so we don't return it on collections. OpenCloud has oc:size for that
					// TODO we cannot find out if md.Size is set or not because ints in go default to 0
					if md.Type == provider.ResourceType_RESOURCE_TYPE_CONTAINER {
						appendToNotFound(prop.NotFound("d:getcontentlength"))
					} else {
						appendToOK(prop.Escaped("d:getcontentlength", size))
					}
				case "resourcetype": // both
					if md.Type == provider.ResourceType_RESOURCE_TYPE_CONTAINER {
						appendToOK(prop.Raw("d:resourcetype", "<d:collection/>"))
					} else {
						appendToOK(prop.Raw("d:resourcetype", ""))
						// redirectref is another option
					}
				case "getcontenttype": // phoenix
					if md.Type == provider.ResourceType_RESOURCE_TYPE_CONTAINER {
						// directories have no contenttype
						appendToNotFound(prop.NotFound("d:getcontenttype"))
					} else if md.MimeType != "" {
						appendToOK(prop.Escaped("d:getcontenttype", md.MimeType))
					}
				case "getlastmodified": // both
					// TODO we cannot find out if md.Mtime is set or not because ints in go default to 0
					if md.Mtime != nil {
						t := utils.TSToTime(md.Mtime).UTC()
						lastModifiedString := t.Format(net.RFC1123)
						appendToOK(prop.Escaped("d:getlastmodified", lastModifiedString))
					} else {
						appendToNotFound(prop.NotFound("d:getlastmodified"))
					}
				case "quota-used-bytes": // RFC 4331
					if md.Type == provider.ResourceType_RESOURCE_TYPE_CONTAINER {
						// always returns the current usage,
						// in oc10 there seems to be a bug that makes the size in webdav differ from the one in the user properties, not taking shares into account
						// in OpenCloud we plan to always mak the quota a property of the storage space
						appendToOK(prop.Escaped("d:quota-used-bytes", size))
					} else {
						appendToNotFound(prop.NotFound("d:quota-used-bytes"))
					}
				case "quota-available-bytes": // RFC 4331
					if md.Type == provider.ResourceType_RESOURCE_TYPE_CONTAINER {
						// oc10 returns -3 for unlimited, -2 for unknown, -1 for uncalculated
						appendToOK(prop.Escaped("d:quota-available-bytes", quota))
					} else {
						appendToNotFound(prop.NotFound("d:quota-available-bytes"))
					}
				case "lockdiscovery": // http://www.webdav.org/specs/rfc2518.html#PROPERTY_lockdiscovery
					if lock == nil {
						appendToNotFound(prop.NotFound("d:lockdiscovery"))
					} else {
						appendToOK(prop.Raw("d:lockdiscovery", activeLocks(&sublog, lock)))
					}
				default:
					appendToNotFound(prop.NotFound("d:" + pf.Prop[i].Local))
				}
			case net.NsOCS:
				switch pf.Prop[i].Local {
				// ocs:share-permissions indicate clients the maximum permissions that can be granted:
				// 1 = read
				// 2 = write (update)
				// 4 = create
				// 8 = delete
				// 16 = share
				// shared files can never have the create or delete permission bit set
				case "share-permissions":
					if md.PermissionSet != nil {
						perms := role.OCSPermissions()
						// shared files cant have the create or delete permission set
						if md.Type == provider.ResourceType_RESOURCE_TYPE_FILE {
							perms &^= conversions.PermissionCreate
							perms &^= conversions.PermissionDelete
						}
						appendToOK(prop.EscapedNS(pf.Prop[i].Space, pf.Prop[i].Local, perms.String()))
					}
				default:
					appendToNotFound(prop.NotFound("d:" + pf.Prop[i].Local))
				}
			default:
				// handle custom properties
				if k := md.GetArbitraryMetadata(); k == nil {
					appendToNotFound(prop.NotFoundNS(pf.Prop[i].Space, pf.Prop[i].Local))
				} else if amd := k.GetMetadata(); amd == nil {
					appendToNotFound(prop.NotFoundNS(pf.Prop[i].Space, pf.Prop[i].Local))
				} else if v, ok := amd[metadataKeyOf(&pf.Prop[i])]; ok && v != "" {
					appendToOK(prop.EscapedNS(pf.Prop[i].Space, pf.Prop[i].Local, v))
				} else {
					appendToNotFound(prop.NotFoundNS(pf.Prop[i].Space, pf.Prop[i].Local))
				}
			}
		}
	}

	if status := utils.ReadPlainFromOpaque(md.Opaque, "status"); status == "processing" {
		response.Propstat = append(response.Propstat, PropstatXML{
			Status: "HTTP/1.1 425 TOO EARLY",
			Prop:   propstatOK.Prop,
		})
		return &response, nil
	}

	if len(propstatOK.Prop) > 0 {
		response.Propstat = append(response.Propstat, propstatOK)
	}
	if len(propstatNotFound.Prop) > 0 {
		response.Propstat = append(response.Propstat, propstatNotFound)
	}

	return &response, nil
}

func hasPreview(md *provider.ResourceInfo, appendToOK func(p ...prop.PropertyXML)) {
	_, match := thumbnail.SupportedMimeTypes[md.MimeType]
	if match {
		appendToOK(prop.Escaped("oc:has-preview", "1"))
	} else {
		appendToOK(prop.Escaped("oc:has-preview", "0"))
	}
}

func activeLocks(log *zerolog.Logger, lock *provider.Lock) string {
	if lock == nil || lock.Type == provider.LockType_LOCK_TYPE_INVALID {
		return ""
	}
	expiration := "Infinity"
	if lock.Expiration != nil {
		now := uint64(time.Now().Unix())
		// Should we hide expired locks here? No.
		//
		// If the timeout expires, then the lock SHOULD be removed.  In this
		// case the server SHOULD act as if an UNLOCK method was executed by the
		// server on the resource using the lock token of the timed-out lock,
		// performed with its override authority.
		//
		// see https://datatracker.ietf.org/doc/html/rfc4918#section-6.6
		if lock.Expiration.Seconds >= now {
			expiration = "Second-" + strconv.FormatUint(lock.Expiration.Seconds-now, 10)
		} else {
			expiration = "Second-0"
		}
	}

	// xml.Encode cannot render emptytags like <d:write/>, see https://github.com/golang/go/issues/21399
	var activelocks strings.Builder
	activelocks.WriteString("<d:activelock>")
	// webdav locktype write | transaction
	switch lock.Type {
	case provider.LockType_LOCK_TYPE_EXCL:
		fallthrough
	case provider.LockType_LOCK_TYPE_WRITE:
		activelocks.WriteString("<d:locktype><d:write/></d:locktype>")
	}
	// webdav lockscope exclusive, shared, or local
	switch lock.Type {
	case provider.LockType_LOCK_TYPE_EXCL:
		fallthrough
	case provider.LockType_LOCK_TYPE_WRITE:
		activelocks.WriteString("<d:lockscope><d:exclusive/></d:lockscope>")
	case provider.LockType_LOCK_TYPE_SHARED:
		activelocks.WriteString("<d:lockscope><d:shared/></d:lockscope>")
	}
	// we currently only support depth infinity
	activelocks.WriteString("<d:depth>Infinity</d:depth>")

	if lock.User != nil || lock.AppName != "" {
		activelocks.WriteString("<d:owner>")

		if lock.User != nil {
			// TODO oc10 uses displayname and email, needs a user lookup
			activelocks.WriteString(prop.Escape(lock.User.OpaqueId + "@" + lock.User.Idp))
		}
		if lock.AppName != "" {
			if lock.User != nil {
				activelocks.WriteString(" via ")
			}
			activelocks.WriteString(prop.Escape(lock.AppName))
		}
		activelocks.WriteString("</d:owner>")
	}

	if un := utils.ReadPlainFromOpaque(lock.Opaque, "lockownername"); un != "" {
		activelocks.WriteString("<oc:ownername>")
		activelocks.WriteString(un)
		activelocks.WriteString("</oc:ownername>")
	}
	if lt := utils.ReadPlainFromOpaque(lock.Opaque, "locktime"); lt != "" {
		activelocks.WriteString("<oc:locktime>")
		activelocks.WriteString(lt)
		activelocks.WriteString("</oc:locktime>")
	}
	activelocks.WriteString("<d:timeout>")
	activelocks.WriteString(expiration)
	activelocks.WriteString("</d:timeout>")
	if lock.LockId != "" {
		activelocks.WriteString("<d:locktoken><d:href>")
		activelocks.WriteString(prop.Escape(lock.LockId))
		activelocks.WriteString("</d:href></d:locktoken>")
	}
	// lockroot is only used when setting the lock
	activelocks.WriteString("</d:activelock>")
	return activelocks.String()
}

// be defensive about wrong encoded etags
func quoteEtag(etag string) string {
	if strings.HasPrefix(etag, "W/") {
		return `W/"` + strings.Trim(etag[2:], `"`) + `"`
	}
	return `"` + strings.Trim(etag, `"`) + `"`
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += n
	return n, err
}

func metadataKeyOf(n *xml.Name) string {
	switch n.Local {
	case "quota-available-bytes":
		return "quota"
	case "share-types", "tags", "lockdiscovery":
		return n.Local
	default:
		return fmt.Sprintf("%s/%s", n.Space, n.Local)
	}
}

// UnmarshalXML appends the property names enclosed within start to pn.
//
// It returns an error if start does not contain any properties or if
// properties contain values. Character data between properties is ignored.
func (pn *Props) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for {
		t, err := prop.Next(d)
		if err != nil {
			return err
		}
		switch e := t.(type) {
		case xml.EndElement:
			// jfd: I think <d:prop></d:prop> is perfectly valid ... treat it as allprop
			/*
				if len(*pn) == 0 {
					return fmt.Errorf("%s must not be empty", start.Name.Local)
				}
			*/
			return nil
		case xml.StartElement:
			t, err = prop.Next(d)
			if err != nil {
				return err
			}
			if _, ok := t.(xml.EndElement); !ok {
				return fmt.Errorf("unexpected token %T", t)
			}
			*pn = append(*pn, e.Name)
		}
	}
}
