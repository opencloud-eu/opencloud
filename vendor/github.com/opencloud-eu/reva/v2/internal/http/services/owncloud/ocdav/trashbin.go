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
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/otel/codes"

	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/config"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/errors"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/net"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/prop"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/propfind"
	"github.com/opencloud-eu/reva/v2/internal/http/services/owncloud/ocdav/spacelookup"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"

	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"

	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	rstatus "github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

// TrashbinHandler handles trashbin requests
type TrashbinHandler struct {
	gatewaySvc                  string
	namespace                   string
	allowPropfindDepthInfinitiy bool
}

func (h *TrashbinHandler) init(c *config.Config) error {
	h.gatewaySvc = c.GatewaySvc
	h.namespace = path.Join("/", c.FilesNamespace)
	h.allowPropfindDepthInfinitiy = c.AllowPropfindDepthInfinitiy
	return nil
}

// Handler handles requests
func (h *TrashbinHandler) Handler(s *svc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := appctx.GetLogger(ctx)

		if r.Method == http.MethodOptions {
			s.handleOptions(w, r)
			return
		}

		var username string
		username, r.URL.Path = splitSpaceAndKey(r.URL.Path)
		if username == "" {
			// listing is disabled, no auth will change that
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		user, ok := ctxpkg.ContextGetUser(ctx)
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if user.Username != username {
			log.Debug().Str("username", username).Interface("user", user).Msg("trying to read another users trash")
			// listing other users trash is forbidden, no auth will change that
			// do not leak existence of space and return 404
			w.WriteHeader(http.StatusNotFound)
			b, err := errors.Marshal(http.StatusNotFound, "not found", "", "")
			if err != nil {
				log.Error().Msgf("error marshaling xml response: %s", b)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			_, err = w.Write(b)
			if err != nil {
				log.Error().Msgf("error writing xml response: %s", b)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			return
		}

		useLoggedInUser := true
		ns, newPath, err := s.ApplyLayout(ctx, h.namespace, useLoggedInUser, r.URL.Path)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			b, err := errors.Marshal(http.StatusNotFound, fmt.Sprintf("could not get storage for %s", r.URL.Path), "", "")
			errors.HandleWebdavError(appctx.GetLogger(r.Context()), w, b, err)
		}
		r.URL.Path = newPath

		basePath := path.Join(ns, newPath)
		space, rpcstatus, err := spacelookup.LookUpStorageSpaceForPath(ctx, s.gatewaySelector, basePath)
		switch {
		case err != nil:
			log.Error().Err(err).Str("path", basePath).Msg("failed to look up storage space")
			w.WriteHeader(http.StatusInternalServerError)
			return
		case rpcstatus.Code != rpc.Code_CODE_OK:
			httpStatus := rstatus.HTTPStatusFromCode(rpcstatus.Code)
			w.WriteHeader(httpStatus)
			b, err := errors.Marshal(httpStatus, rpcstatus.Message, "", "")
			errors.HandleWebdavError(log, w, b, err)
			return
		}
		ref := spacelookup.MakeRelativeReference(space, ".", false)

		// key will be a base64 encoded cs3 path, it uniquely identifies a trash item with an opaque id and an optional path
		key := r.URL.Path

		switch r.Method {
		case MethodPropfind:
			h.listTrashbin(w, r, s, ref, user.Username, key)
		case MethodMove:
			if key == "" {
				http.Error(w, "501 Not implemented", http.StatusNotImplemented)
				break
			}
			// find path in url relative to trash base
			trashBase := ctx.Value(net.CtxKeyBaseURI).(string)
			baseURI := path.Join(path.Dir(trashBase), "files", username)

			dh := r.Header.Get(net.HeaderDestination)
			dst, err := net.ParseDestination(baseURI, dh)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			p := path.Join(ns, dst)
			// The destination can be in another space. E.g. the 'Shares Jail'.
			space, rpcstatus, err := spacelookup.LookUpStorageSpaceForPath(ctx, s.gatewaySelector, p)
			if err != nil {
				log.Error().Err(err).Str("path", p).Msg("failed to look up destination storage space")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if rpcstatus.Code != rpc.Code_CODE_OK {
				httpStatus := rstatus.HTTPStatusFromCode(rpcstatus.Code)
				w.WriteHeader(httpStatus)
				b, err := errors.Marshal(httpStatus, rpcstatus.Message, "", "")
				errors.HandleWebdavError(log, w, b, err)
				return
			}
			dstRef := spacelookup.MakeRelativeReference(space, p, false)

			log.Debug().Str("key", key).Str("dst", dst).Msg("restore")
			h.restore(w, r, s, ref, dstRef, key)
		case http.MethodDelete:
			h.delete(w, r, s, ref, key)
		default:
			http.Error(w, "501 Not implemented", http.StatusNotImplemented)
		}
	})
}

func (h *TrashbinHandler) getDepth(r *http.Request) (net.Depth, error) {
	dh := r.Header.Get(net.HeaderDepth)
	depth, err := net.ParseDepth(dh)
	if err != nil || depth == net.DepthInfinity && !h.allowPropfindDepthInfinitiy {
		return "", errors.ErrInvalidDepth
	}
	return depth, nil
}

func (h *TrashbinHandler) listTrashbin(w http.ResponseWriter, r *http.Request, s *svc, ref *provider.Reference, refBase, key string) {
	ctx, span := appctx.GetTracerProvider(r.Context()).Tracer(tracerName).Start(r.Context(), "list_trashbin")
	defer span.End()

	sublog := appctx.GetLogger(ctx).With().Logger()

	depth, err := h.getDepth(r)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Invalid Depth header value")
		span.SetAttributes(semconv.HTTPResponseStatusCodeKey.Int(http.StatusBadRequest))
		sublog.Debug().Str("depth", r.Header.Get(net.HeaderDepth)).Msg(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		m := fmt.Sprintf("Invalid Depth header value: %v", r.Header.Get(net.HeaderDepth))
		b, err := errors.Marshal(http.StatusBadRequest, m, "", "")
		errors.HandleWebdavError(&sublog, w, b, err)
		return
	}

	pf, status, err := propfind.ReadPropfind(r.Body)
	if err != nil {
		sublog.Debug().Err(err).Msg("error reading propfind request")
		w.WriteHeader(status)
		return
	}

	if key == "" && depth == net.DepthZero {
		// we are listing the trash root, but without children
		// so we just fake a root element without actually querying the gateway
		rootHref := path.Join(refBase, key)
		propRes, err := h.formatTrashPropfind(ctx, s, ref.ResourceId.SpaceId, refBase, rootHref, &pf, nil, true)
		if err != nil {
			sublog.Error().Err(err).Msg("error formatting propfind")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set(net.HeaderDav, "1, 3, extended-mkcol")
		w.Header().Set(net.HeaderContentType, "application/xml; charset=utf-8")
		w.WriteHeader(http.StatusMultiStatus)
		_, err = w.Write(propRes)
		if err != nil {
			sublog.Error().Err(err).Msg("error writing body")
			return
		}
		return
	}

	if depth == net.DepthOne && key != "" && !strings.HasSuffix(key, "/") {
		// when a key is provided and the depth is 1 we need to append a / to the key to list the children
		key += "/"
	}

	client, err := s.gatewaySelector.Next()
	if err != nil {
		sublog.Error().Err(err).Msg("error selecting next gateway client")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// ask gateway for recycle items
	getRecycleRes, err := client.ListRecycle(ctx, &provider.ListRecycleRequest{Ref: ref, Key: key})
	if err != nil {
		sublog.Error().Err(err).Msg("error calling ListRecycle")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if getRecycleRes.Status.Code != rpc.Code_CODE_OK {
		httpStatus := rstatus.HTTPStatusFromCode(getRecycleRes.Status.Code)
		w.WriteHeader(httpStatus)
		b, err := errors.Marshal(httpStatus, getRecycleRes.Status.Message, "", "")
		errors.HandleWebdavError(&sublog, w, b, err)
		return
	}

	items := getRecycleRes.RecycleItems

	if depth == net.DepthInfinity {
		var stack []string
		// check sub-containers in reverse order and add them to the stack
		// the reversed order here will produce a more logical sorting of results
		for i := len(items) - 1; i >= 0; i-- {
			// for i := range res.Infos {
			if items[i].Type == provider.ResourceType_RESOURCE_TYPE_CONTAINER {
				stack = append(stack, items[i].Key+"/") // fetch children of the item
			}
		}

		for len(stack) > 0 {
			key := stack[len(stack)-1]
			getRecycleRes, err := client.ListRecycle(ctx, &provider.ListRecycleRequest{Ref: ref, Key: key})
			if err != nil {
				sublog.Error().Err(err).Msg("error calling ListRecycle")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			if getRecycleRes.Status.Code != rpc.Code_CODE_OK {
				httpStatus := rstatus.HTTPStatusFromCode(getRecycleRes.Status.Code)
				w.WriteHeader(httpStatus)
				b, err := errors.Marshal(httpStatus, getRecycleRes.Status.Message, "", "")
				errors.HandleWebdavError(&sublog, w, b, err)
				return
			}
			items = append(items, getRecycleRes.RecycleItems...)

			stack = stack[:len(stack)-1]
			// check sub-containers in reverse order and add them to the stack
			// the reversed order here will produce a more logical sorting of results
			for i := len(getRecycleRes.RecycleItems) - 1; i >= 0; i-- {
				// for i := range res.Infos {
				if getRecycleRes.RecycleItems[i].Type == provider.ResourceType_RESOURCE_TYPE_CONTAINER {
					stack = append(stack, getRecycleRes.RecycleItems[i].Key)
				}
			}
		}
	}

	rootHref := path.Join(refBase, key)
	propRes, err := h.formatTrashPropfind(ctx, s, ref.ResourceId.SpaceId, refBase, rootHref, &pf, items, depth != net.DepthZero)
	if err != nil {
		sublog.Error().Err(err).Msg("error formatting propfind")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set(net.HeaderDav, "1, 3, extended-mkcol")
	w.Header().Set(net.HeaderContentType, "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusMultiStatus)
	_, err = w.Write(propRes)
	if err != nil {
		sublog.Error().Err(err).Msg("error writing body")
		return
	}
}

func (h *TrashbinHandler) formatTrashPropfind(ctx context.Context, s *svc, spaceID, refBase, rootHref string, pf *propfind.XML, items []*provider.RecycleItem, fakeRoot bool) ([]byte, error) {
	responses := make([]*propfind.ResponseXML, 0, len(items)+1)
	if fakeRoot {
		responses = append(responses, &propfind.ResponseXML{
			Href: net.EncodePath(path.Join(ctx.Value(net.CtxKeyBaseURI).(string), rootHref) + "/"), // url encode response.Href TODO
			Propstat: []propfind.PropstatXML{
				{
					Status: "HTTP/1.1 200 OK",
					Prop: []prop.PropertyXML{
						prop.Raw("d:resourcetype", "<d:collection/>"),
					},
				},
				{
					Status: "HTTP/1.1 404 Not Found",
					Prop: []prop.PropertyXML{
						prop.NotFound("oc:trashbin-original-filename"),
						prop.NotFound("oc:trashbin-original-location"),
						prop.NotFound("oc:trashbin-delete-datetime"),
						prop.NotFound("d:getcontentlength"),
					},
				},
			},
		})
	}

	for i := range items {
		res, err := h.itemToPropResponse(ctx, s, spaceID, refBase, pf, items[i])
		if err != nil {
			return nil, err
		}
		responses = append(responses, res)
	}
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

// itemToPropResponse needs to create a listing that contains a key and destination
// the key is the name of an entry in the trash listing
// for now we need to limit trash to the users home, so we can expect all trash keys to have the home storage as the opaque id
func (h *TrashbinHandler) itemToPropResponse(ctx context.Context, s *svc, spaceID, refBase string, pf *propfind.XML, item *provider.RecycleItem) (*propfind.ResponseXML, error) {

	baseURI := ctx.Value(net.CtxKeyBaseURI).(string)
	ref := path.Join(baseURI, refBase, item.GetKey())
	if item.GetType() == provider.ResourceType_RESOURCE_TYPE_CONTAINER {
		ref += "/"
	}

	response := propfind.ResponseXML{
		Href:     net.EncodePath(ref), // url encode response.Href
		Propstat: []propfind.PropstatXML{},
	}

	// TODO(jfd): if the path we list here is taken from the ListRecycle request we rely on the gateway to prefix it with the mount point

	t := utils.TSToTime(item.GetDeletionTime()).UTC()
	dTime := t.Format(time.RFC1123Z)
	size := strconv.FormatUint(item.GetSize(), 10)

	// when allprops has been requested
	if pf.Allprop != nil {
		// return all known properties
		propstatOK := propfind.PropstatXML{
			Status: "HTTP/1.1 200 OK",
			Prop:   []prop.PropertyXML{},
		}
		// yes this is redundant, can be derived from oc:trashbin-original-location which contains the full path, clients should not fetch it
		propstatOK.Prop = append(propstatOK.Prop, prop.Escaped("oc:trashbin-original-filename", path.Base(item.GetRef().GetPath())))
		propstatOK.Prop = append(propstatOK.Prop, prop.Escaped("oc:trashbin-original-location", strings.TrimPrefix(item.GetRef().GetPath(), "/")))
		propstatOK.Prop = append(propstatOK.Prop, prop.Escaped("oc:trashbin-delete-timestamp", strconv.FormatUint(item.GetDeletionTime().GetSeconds(), 10)))
		propstatOK.Prop = append(propstatOK.Prop, prop.Escaped("oc:trashbin-delete-datetime", dTime))
		if item.GetType() == provider.ResourceType_RESOURCE_TYPE_CONTAINER {
			propstatOK.Prop = append(propstatOK.Prop, prop.Raw("d:resourcetype", "<d:collection/>"))
			propstatOK.Prop = append(propstatOK.Prop, prop.Raw("oc:size", size))
		} else {
			propstatOK.Prop = append(propstatOK.Prop,
				prop.Escaped("d:resourcetype", ""),
				prop.Escaped("d:getcontentlength", size),
			)
		}
		propstatOK.Prop = append(propstatOK.Prop, prop.Escaped("oc:spaceid", spaceID))
		response.Propstat = append(response.Propstat, propstatOK)
	} else {
		// otherwise return only the requested properties
		propstatOK := propfind.PropstatXML{
			Status: "HTTP/1.1 200 OK",
			Prop:   []prop.PropertyXML{},
		}
		propstatNotFound := propfind.PropstatXML{
			Status: "HTTP/1.1 404 Not Found",
			Prop:   []prop.PropertyXML{},
		}
		for i := range pf.Prop {
			switch pf.Prop[i].Space {
			case net.NsOwncloud:
				switch pf.Prop[i].Local {
				case "oc:size":
					if item.GetType() == provider.ResourceType_RESOURCE_TYPE_CONTAINER {
						propstatOK.Prop = append(propstatOK.Prop, prop.Escaped("oc:size", size))
					} else {
						propstatNotFound.Prop = append(propstatNotFound.Prop, prop.NotFound("oc:size"))
					}
				case "trashbin-original-filename":
					// yes this is redundant, can be derived from oc:trashbin-original-location which contains the full path, clients should not fetch it
					propstatOK.Prop = append(propstatOK.Prop, prop.Escaped("oc:trashbin-original-filename", path.Base(item.GetRef().GetPath())))
				case "trashbin-original-location":
					// TODO (jfd) double check and clarify the cs3 spec what the Key is about and if Path is only the folder that contains the file or if it includes the filename
					propstatOK.Prop = append(propstatOK.Prop, prop.Escaped("oc:trashbin-original-location", strings.TrimPrefix(item.GetRef().GetPath(), "/")))
				case "trashbin-delete-datetime":
					propstatOK.Prop = append(propstatOK.Prop, prop.Escaped("oc:trashbin-delete-datetime", dTime))
				case "trashbin-delete-timestamp":
					propstatOK.Prop = append(propstatOK.Prop, prop.Escaped("oc:trashbin-delete-timestamp", strconv.FormatUint(item.GetDeletionTime().GetSeconds(), 10)))
				case "spaceid":
					propstatOK.Prop = append(propstatOK.Prop, prop.Escaped("oc:spaceid", spaceID))
				default:
					propstatNotFound.Prop = append(propstatNotFound.Prop, prop.NotFound("oc:"+pf.Prop[i].Local))
				}
			case net.NsDav:
				switch pf.Prop[i].Local {
				case "getcontentlength":
					if item.GetType() == provider.ResourceType_RESOURCE_TYPE_CONTAINER {
						propstatNotFound.Prop = append(propstatNotFound.Prop, prop.NotFound("d:getcontentlength"))
					} else {
						propstatOK.Prop = append(propstatOK.Prop, prop.Escaped("d:getcontentlength", size))
					}
				case "resourcetype":
					if item.GetType() == provider.ResourceType_RESOURCE_TYPE_CONTAINER {
						propstatOK.Prop = append(propstatOK.Prop, prop.Raw("d:resourcetype", "<d:collection/>"))
					} else {
						propstatOK.Prop = append(propstatOK.Prop, prop.Raw("d:resourcetype", ""))
						// redirectref is another option
					}
				case "getcontenttype":
					if item.GetType() == provider.ResourceType_RESOURCE_TYPE_CONTAINER {
						propstatOK.Prop = append(propstatOK.Prop, prop.Raw("d:getcontenttype", "httpd/unix-directory"))
					} else {
						propstatNotFound.Prop = append(propstatNotFound.Prop, prop.NotFound("d:getcontenttype"))
					}
				default:
					propstatNotFound.Prop = append(propstatNotFound.Prop, prop.NotFound("d:"+pf.Prop[i].Local))
				}
			default:
				// TODO (jfd) lookup shortname for unknown namespaces?
				propstatNotFound.Prop = append(propstatNotFound.Prop, prop.NotFound(pf.Prop[i].Space+":"+pf.Prop[i].Local))
			}
		}
		response.Propstat = append(response.Propstat, propstatOK, propstatNotFound)
	}

	return &response, nil
}

func (h *TrashbinHandler) restore(w http.ResponseWriter, r *http.Request, s *svc, ref, dst *provider.Reference, key string) {
	ctx, span := appctx.GetTracerProvider(r.Context()).Tracer(tracerName).Start(r.Context(), "restore")
	defer span.End()

	sublog := appctx.GetLogger(ctx).With().Logger()

	oh := r.Header.Get(net.HeaderOverwrite)

	overwrite, err := net.ParseOverwrite(oh)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	client, err := s.gatewaySelector.Next()
	if err != nil {
		sublog.Error().Err(err).Msg("error selecting next gateway client")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	dstStatReq := &provider.StatRequest{Ref: dst}
	dstStatRes, err := client.Stat(ctx, dstStatReq)
	if err != nil {
		sublog.Error().Err(err).Msg("error sending grpc stat request")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if dstStatRes.Status.Code != rpc.Code_CODE_OK && dstStatRes.Status.Code != rpc.Code_CODE_NOT_FOUND {
		errors.HandleErrorStatus(&sublog, w, dstStatRes.Status)
		return
	}

	// Restoring to a non-existent location is not supported by the WebDAV spec. The following block ensures the target
	// restore location exists, and if it doesn't returns a conflict error code.
	if dstStatRes.Status.Code == rpc.Code_CODE_NOT_FOUND && isNested(dst.Path) {
		parentRef := &provider.Reference{ResourceId: dst.ResourceId, Path: utils.MakeRelativePath(path.Dir(dst.Path))}
		parentStatReq := &provider.StatRequest{Ref: parentRef}

		parentStatResponse, err := client.Stat(ctx, parentStatReq)
		if err != nil {
			sublog.Error().Err(err).Msg("error sending grpc stat request")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if parentStatResponse.Status.Code == rpc.Code_CODE_NOT_FOUND {
			// 409 if intermediate dir is missing, see https://tools.ietf.org/html/rfc4918#section-9.8.5
			w.WriteHeader(http.StatusConflict)
			return
		}
	}

	successCode := http.StatusCreated // 201 if new resource was created, see https://tools.ietf.org/html/rfc4918#section-9.9.4
	if dstStatRes.Status.Code == rpc.Code_CODE_OK {
		successCode = http.StatusNoContent // 204 if target already existed, see https://tools.ietf.org/html/rfc4918#section-9.9.4

		if !overwrite {
			sublog.Warn().Bool("overwrite", overwrite).Msg("dst already exists")
			w.WriteHeader(http.StatusPreconditionFailed) // 412, see https://tools.ietf.org/html/rfc4918#section-9.9.4
			b, err := errors.Marshal(
				http.StatusPreconditionFailed,
				"The destination node already exists, and the overwrite header is set to false",
				net.HeaderOverwrite,
				"",
			)
			errors.HandleWebdavError(&sublog, w, b, err)
			return
		}
		// delete existing tree
		delReq := &provider.DeleteRequest{Ref: dst}
		delRes, err := client.Delete(ctx, delReq)
		if err != nil {
			sublog.Error().Err(err).Msg("error sending grpc delete request")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if delRes.Status.Code != rpc.Code_CODE_OK && delRes.Status.Code != rpc.Code_CODE_NOT_FOUND {
			errors.HandleErrorStatus(&sublog, w, delRes.Status)
			return
		}
	}

	req := &provider.RestoreRecycleItemRequest{
		Ref:        ref,
		Key:        key,
		RestoreRef: dst,
	}

	res, err := client.RestoreRecycleItem(ctx, req)
	if err != nil {
		sublog.Error().Err(err).Msg("error sending a grpc restore recycle item request")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if res.Status.Code != rpc.Code_CODE_OK {
		if res.Status.Code == rpc.Code_CODE_PERMISSION_DENIED {
			w.WriteHeader(http.StatusForbidden)
			b, err := errors.Marshal(http.StatusForbidden, "Permission denied to restore", "", "")
			errors.HandleWebdavError(&sublog, w, b, err)
		}
		errors.HandleErrorStatus(&sublog, w, res.Status)
		return
	}

	dstStatRes, err = client.Stat(ctx, dstStatReq)
	if err != nil {
		sublog.Error().Err(err).Msg("error sending grpc stat request")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if dstStatRes.Status.Code != rpc.Code_CODE_OK {
		errors.HandleErrorStatus(&sublog, w, dstStatRes.Status)
		return
	}

	info := dstStatRes.Info
	w.Header().Set(net.HeaderContentType, info.MimeType)
	w.Header().Set(net.HeaderETag, info.Etag)
	w.Header().Set(net.HeaderOCFileID, storagespace.FormatResourceID(info.Id))
	w.Header().Set(net.HeaderOCETag, info.Etag)

	w.WriteHeader(successCode)
}

// delete has only a key
func (h *TrashbinHandler) delete(w http.ResponseWriter, r *http.Request, s *svc, ref *provider.Reference, key string) {
	ctx, span := appctx.GetTracerProvider(r.Context()).Tracer(tracerName).Start(r.Context(), "erase")
	defer span.End()

	sublog := appctx.GetLogger(ctx).With().Interface("reference", ref).Str("key", key).Logger()

	req := &provider.PurgeRecycleRequest{
		Ref: ref,
		Key: key,
	}

	client, err := s.gatewaySelector.Next()
	if err != nil {
		sublog.Error().Err(err).Msg("error selecting next gateway client")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	res, err := client.PurgeRecycle(ctx, req)
	if err != nil {
		sublog.Error().Err(err).Msg("error sending a grpc restore recycle item request")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	switch res.Status.Code {
	case rpc.Code_CODE_OK:
		w.WriteHeader(http.StatusNoContent)
	case rpc.Code_CODE_NOT_FOUND:
		sublog.Debug().Interface("status", res.Status).Msg("resource not found")
		w.WriteHeader(http.StatusConflict)
		m := fmt.Sprintf("key %s not found", key)
		b, err := errors.Marshal(http.StatusConflict, m, "", "")
		errors.HandleWebdavError(&sublog, w, b, err)
	case rpc.Code_CODE_PERMISSION_DENIED:
		w.WriteHeader(http.StatusForbidden)
		var m string
		if key == "" {
			m = "Permission denied to purge recycle"
		} else {
			m = "Permission denied to delete"
		}
		b, err := errors.Marshal(http.StatusForbidden, m, "", "")
		errors.HandleWebdavError(&sublog, w, b, err)
	default:
		errors.HandleErrorStatus(&sublog, w, res.Status)
	}
}

func isNested(p string) bool {
	dir, _ := path.Split(p)
	return dir != "/" && dir != "./"
}
