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

package ocdav

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/errors"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/net"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/prop"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/propfind"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/spacelookup"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/permission"
	rstatus "github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/rs/zerolog"
)

func (s *svc) handlePathProppatch(w http.ResponseWriter, r *http.Request, ns string) (status int, err error) {
	ctx, span := appctx.GetTracerProvider(r.Context()).Tracer(tracerName).Start(r.Context(), "proppatch")
	defer span.End()

	fn := path.Join(ns, r.URL.Path)

	sublog := appctx.GetLogger(ctx).With().Str("path", fn).Logger()

	pp, status, err := readProppatch(r.Body)
	if err != nil {
		return status, err
	}

	space, rpcStatus, err := spacelookup.LookUpStorageSpaceForPath(ctx, s.gatewaySelector, fn)
	switch {
	case err != nil:
		return http.StatusInternalServerError, err
	case rpcStatus.Code == rpc.Code_CODE_ABORTED:
		return http.StatusPreconditionFailed, errtypes.NewErrtypeFromStatus(rpcStatus)
	case rpcStatus.Code != rpc.Code_CODE_OK:
		return rstatus.HTTPStatusFromCode(rpcStatus.Code), errtypes.NewErrtypeFromStatus(rpcStatus)
	}

	client, err := s.gatewaySelector.Next()
	if err != nil {
		return http.StatusInternalServerError, errtypes.InternalError(err.Error())
	}
	// check if resource exists
	statReq := &provider.StatRequest{Ref: spacelookup.MakeRelativeReference(space, fn, false)}
	statRes, err := client.Stat(ctx, statReq)
	switch {
	case err != nil:
		return http.StatusInternalServerError, err
	case statRes.Status.Code == rpc.Code_CODE_ABORTED:
		return http.StatusPreconditionFailed, errtypes.NewErrtypeFromStatus(statRes.Status)
	case statRes.Status.Code != rpc.Code_CODE_OK:
		return rstatus.HTTPStatusFromCode(rpcStatus.Code), errtypes.NewErrtypeFromStatus(statRes.Status)
	}

	acceptedProps, removedProps, ok := s.handleProppatch(ctx, w, r, spacelookup.MakeRelativeReference(space, fn, false), pp, sublog)
	if !ok {
		// handleProppatch handles responses in error cases so return 0
		return 0, nil
	}

	nRef := strings.TrimPrefix(fn, ns)
	nRef = path.Join(ctx.Value(net.CtxKeyBaseURI).(string), nRef)
	if statRes.Info.Type == provider.ResourceType_RESOURCE_TYPE_CONTAINER {
		nRef += "/"
	}

	s.handleProppatchResponse(ctx, w, r, acceptedProps, removedProps, nRef, sublog)
	return 0, nil
}

func (s *svc) handleSpacesProppatch(w http.ResponseWriter, r *http.Request, spaceID string) (status int, err error) {
	ctx, span := appctx.GetTracerProvider(r.Context()).Tracer(tracerName).Start(r.Context(), "spaces_proppatch")
	defer span.End()

	sublog := appctx.GetLogger(ctx).With().Str("path", r.URL.Path).Str("spaceid", spaceID).Logger()

	pp, status, err := readProppatch(r.Body)
	if err != nil {
		return status, err
	}

	ref, err := spacelookup.MakeStorageSpaceReference(spaceID, r.URL.Path)
	if err != nil {
		return http.StatusBadRequest, err
	}

	acceptedProps, removedProps, ok := s.handleProppatch(ctx, w, r, &ref, pp, sublog)
	if !ok {
		// handleProppatch handles responses in error cases so return 0
		return 0, nil
	}

	nRef := path.Join(spaceID, r.URL.Path)
	nRef = path.Join(ctx.Value(net.CtxKeyBaseURI).(string), nRef)

	s.handleProppatchResponse(ctx, w, r, acceptedProps, removedProps, nRef, sublog)
	return 0, nil
}

func (s *svc) handleProppatch(ctx context.Context, w http.ResponseWriter, r *http.Request, ref *provider.Reference, patches []Proppatch, log zerolog.Logger) ([]xml.Name, []xml.Name, bool) {

	rreq := &provider.UnsetArbitraryMetadataRequest{
		Ref:                   ref,
		ArbitraryMetadataKeys: []string{""},
		LockId:                requestLock(r),
	}
	sreq := &provider.SetArbitraryMetadataRequest{
		Ref: ref,
		ArbitraryMetadata: &provider.ArbitraryMetadata{
			Metadata: map[string]string{},
		},
		LockId: requestLock(r),
	}

	acceptedProps := []xml.Name{}
	removedProps := []xml.Name{}

	client, err := s.gatewaySelector.Next()
	if err != nil {
		log.Error().Err(err).Msg("error selecting next gateway client")
		w.WriteHeader(http.StatusInternalServerError)
		return nil, nil, false
	}
	for i := range patches {
		if len(patches[i].Props) < 1 {
			continue
		}
		for j := range patches[i].Props {
			propNameXML := patches[i].Props[j].XMLName
			// don't use path.Join. It removes the double slash! concatenate with a /
			key := fmt.Sprintf("%s/%s", patches[i].Props[j].XMLName.Space, patches[i].Props[j].XMLName.Local)
			value := string(patches[i].Props[j].InnerXML)
			remove := patches[i].Remove
			// boolean flags may be "set" to false as well
			if s.isBooleanProperty(key) {
				// Make boolean properties either "0" or "1"
				value = s.as0or1(value)
				if value == "0" {
					remove = true
				}
			}
			// Webdav spec requires the operations to be executed in the order
			// specified in the PROPPATCH request
			// http://www.webdav.org/specs/rfc2518.html#rfc.section.8.2
			// FIXME: batch this somehow
			if remove {
				rreq.ArbitraryMetadataKeys[0] = key
				res, err := client.UnsetArbitraryMetadata(ctx, rreq)
				if err != nil {
					log.Error().Err(err).Msg("error sending a grpc UnsetArbitraryMetadata request")
					w.WriteHeader(http.StatusInternalServerError)
					return nil, nil, false
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
						sRes, err := client.Stat(ctx, &provider.StatRequest{Ref: ref})
						if err != nil {
							log.Error().Err(err).Msg("error performing stat grpc request")
							w.WriteHeader(http.StatusInternalServerError)
							return nil, nil, false
						}
						if sRes.Status.Code != rpc.Code_CODE_OK {
							// return not found error so we do not leak existence of a file
							// TODO hide permission failed for users without access in every kind of request
							// TODO should this be done in the driver?
							status = http.StatusNotFound
						}
					}
					if status == http.StatusNotFound {
						m = "Resource not found" // mimic the oc10 error message
					}
					w.WriteHeader(status)
					b, err := errors.Marshal(status, m, "", "")
					errors.HandleWebdavError(&log, w, b, err)
					return nil, nil, false
				}
				if key == "http://owncloud.org/ns/favorite" {
					statRes, err := client.Stat(ctx, &provider.StatRequest{Ref: ref})
					if err != nil {
						w.WriteHeader(http.StatusInternalServerError)
						return nil, nil, false
					}
					currentUser := ctxpkg.ContextMustGetUser(ctx)
					ok, err := utils.CheckPermission(ctx, permission.WriteFavorites, client)
					if err != nil {
						log.Error().Err(err).Msg("error checking permission")
						w.WriteHeader(http.StatusInternalServerError)
						return nil, nil, false
					}
					if !ok {
						log.Info().Interface("user", currentUser).Msg("user not allowed to unset favorite")
						w.WriteHeader(http.StatusForbidden)
						return nil, nil, false
					}
					err = s.favoritesManager.UnsetFavorite(ctx, currentUser.Id, statRes.Info)
					if err != nil {
						w.WriteHeader(http.StatusInternalServerError)
						return nil, nil, false
					}
				}
				removedProps = append(removedProps, propNameXML)
			} else {
				sreq.ArbitraryMetadata.Metadata[key] = value
				res, err := client.SetArbitraryMetadata(ctx, sreq)
				if err != nil {
					log.Error().Err(err).Str("key", key).Str("value", value).Msg("error sending a grpc SetArbitraryMetadata request")
					w.WriteHeader(http.StatusInternalServerError)
					return nil, nil, false
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
						sRes, err := client.Stat(ctx, &provider.StatRequest{Ref: ref})
						if err != nil {
							log.Error().Err(err).Msg("error performing stat grpc request")
							w.WriteHeader(http.StatusInternalServerError)
							return nil, nil, false
						}
						if sRes.Status.Code != rpc.Code_CODE_OK {
							// return not found error so we don't leak existence of a file
							// TODO hide permission failed for users without access in every kind of request
							// TODO should this be done in the driver?
							status = http.StatusNotFound
						}
					}
					if status == http.StatusNotFound {
						m = "Resource not found" // mimic the oc10 error message
					}
					w.WriteHeader(status)
					b, err := errors.Marshal(status, m, "", "")
					errors.HandleWebdavError(&log, w, b, err)
					return nil, nil, false
				}

				acceptedProps = append(acceptedProps, propNameXML)
				delete(sreq.ArbitraryMetadata.Metadata, key)

				if key == "http://owncloud.org/ns/favorite" {
					statRes, err := client.Stat(ctx, &provider.StatRequest{Ref: ref})
					if err != nil || statRes.Info == nil {
						w.WriteHeader(http.StatusInternalServerError)
						return nil, nil, false
					}
					currentUser := ctxpkg.ContextMustGetUser(ctx)
					ok, err := utils.CheckPermission(ctx, permission.WriteFavorites, client)
					if err != nil {
						log.Error().Err(err).Msg("error checking permission")
						w.WriteHeader(http.StatusInternalServerError)
						return nil, nil, false
					}
					if !ok {
						log.Info().Interface("user", currentUser).Msg("user not allowed to set favorite")
						w.WriteHeader(http.StatusForbidden)
						return nil, nil, false
					}
					err = s.favoritesManager.SetFavorite(ctx, currentUser.Id, statRes.Info)
					if err != nil {
						w.WriteHeader(http.StatusInternalServerError)
						return nil, nil, false
					}
				}
			}
		}
		// FIXME: in case of error, need to set all properties back to the original state,
		// and return the error in the matching propstat block, if applicable
		// http://www.webdav.org/specs/rfc2518.html#rfc.section.8.2
	}

	return acceptedProps, removedProps, true
}

func (s *svc) handleProppatchResponse(ctx context.Context, w http.ResponseWriter, r *http.Request, acceptedProps, removedProps []xml.Name, path string, log zerolog.Logger) {
	propRes, err := s.formatProppatchResponse(ctx, acceptedProps, removedProps, path)
	if err != nil {
		log.Error().Err(err).Msg("error formatting proppatch response")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set(net.HeaderDav, "1, 3, extended-mkcol")
	w.Header().Set(net.HeaderContentType, "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusMultiStatus)
	if _, err := w.Write(propRes); err != nil {
		log.Err(err).Msg("error writing response")
	}
}

func (s *svc) formatProppatchResponse(ctx context.Context, acceptedProps []xml.Name, removedProps []xml.Name, ref string) ([]byte, error) {
	responses := make([]propfind.ResponseXML, 0, 1)
	response := propfind.ResponseXML{
		Href:     net.EncodePath(ref),
		Propstat: []propfind.PropstatXML{},
	}

	if len(acceptedProps) > 0 {
		propstatBody := []prop.PropertyXML{}
		for i := range acceptedProps {
			propstatBody = append(propstatBody, prop.EscapedNS(acceptedProps[i].Space, acceptedProps[i].Local, ""))
		}
		response.Propstat = append(response.Propstat, propfind.PropstatXML{
			Status: "HTTP/1.1 200 OK",
			Prop:   propstatBody,
		})
	}

	if len(removedProps) > 0 {
		propstatBody := []prop.PropertyXML{}
		for i := range removedProps {
			propstatBody = append(propstatBody, prop.EscapedNS(removedProps[i].Space, removedProps[i].Local, ""))
		}
		response.Propstat = append(response.Propstat, propfind.PropstatXML{
			Status: "HTTP/1.1 204 No Content",
			Prop:   propstatBody,
		})
	}

	responses = append(responses, response)
	responsesXML, err := xml.Marshal(&responses)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0" encoding="utf-8"?><d:multistatus xmlns:d="DAV:" `)
	buf.WriteString(`xmlns:s="http://sabredav.org/ns" xmlns:oc="http://owncloud.org/ns">`)
	buf.Write(responsesXML)
	buf.WriteString(`</d:multistatus>`)
	return buf.Bytes(), nil
}

func (s *svc) isBooleanProperty(prop string) bool {
	// TODO add other properties we know to be boolean?
	return prop == net.PropOcFavorite
}

func (s *svc) as0or1(val string) string {
	switch strings.TrimSpace(val) {
	case "false":
		return "0"
	case "":
		return "0"
	case "0":
		return "0"
	case "no":
		return "0"
	case "off":
		return "0"
	}
	return "1"
}

// Proppatch describes a property update instruction as defined in RFC 4918.
// See http://www.webdav.org/specs/rfc4918.html#METHOD_PROPPATCH
type Proppatch struct {
	// Remove specifies whether this patch removes properties. If it does not
	// remove them, it sets them.
	Remove bool
	// Props contains the properties to be set or removed.
	Props []prop.PropertyXML
}

// http://www.webdav.org/specs/rfc4918.html#ELEMENT_prop (for proppatch)
type proppatchProps []prop.PropertyXML

// UnmarshalXML appends the property names and values enclosed within start
// to ps.
//
// An xml:lang attribute that is defined either on the DAV:prop or property
// name XML element is propagated to the property's Lang field.
//
// UnmarshalXML returns an error if start does not contain any properties or if
// property values contain syntactically incorrect XML.
func (ps *proppatchProps) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	lang := xmlLang(start, "")
	for {
		t, err := prop.Next(d)
		if err != nil {
			return err
		}
		switch elem := t.(type) {
		case xml.EndElement:
			if len(*ps) == 0 {
				return fmt.Errorf("%s must not be empty", start.Name.Local)
			}
			return nil
		case xml.StartElement:
			p := prop.PropertyXML{}
			err = d.DecodeElement(&p, &elem)
			if err != nil {
				return err
			}
			// special handling for the lang property
			p.Lang = xmlLang(t.(xml.StartElement), lang)
			*ps = append(*ps, p)
		}
	}
}

// http://www.webdav.org/specs/rfc4918.html#ELEMENT_set
// http://www.webdav.org/specs/rfc4918.html#ELEMENT_remove
type setRemove struct {
	XMLName xml.Name
	Lang    string         `xml:"xml:lang,attr,omitempty"`
	Prop    proppatchProps `xml:"DAV: prop"`
}

// http://www.webdav.org/specs/rfc4918.html#ELEMENT_propertyupdate
type propertyupdate struct {
	XMLName   xml.Name    `xml:"DAV: propertyupdate"`
	Lang      string      `xml:"xml:lang,attr,omitempty"`
	SetRemove []setRemove `xml:",any"`
}

func readProppatch(r io.Reader) (patches []Proppatch, status int, err error) {
	var pu propertyupdate
	if err = xml.NewDecoder(r).Decode(&pu); err != nil {
		return nil, http.StatusBadRequest, err
	}
	for _, op := range pu.SetRemove {
		remove := false
		switch op.XMLName {
		case xml.Name{Space: net.NsDav, Local: "set"}:
			// No-op.
		case xml.Name{Space: net.NsDav, Local: "remove"}:
			for _, p := range op.Prop {
				if len(p.InnerXML) > 0 {
					return nil, http.StatusBadRequest, errors.ErrInvalidProppatch
				}
			}
			remove = true
		default:
			return nil, http.StatusBadRequest, errors.ErrInvalidProppatch
		}
		patches = append(patches, Proppatch{Remove: remove, Props: op.Prop})
	}
	return patches, 0, nil
}

var xmlLangName = xml.Name{Space: "http://www.w3.org/XML/1998/namespace", Local: "lang"}

func xmlLang(s xml.StartElement, d string) string {
	for _, attr := range s.Attr {
		if attr.Name == xmlLangName {
			return attr.Value
		}
	}
	return d
}
