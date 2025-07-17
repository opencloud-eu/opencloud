package svc

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	merrors "go-micro.dev/v4/errors"
	"go-micro.dev/v4/metadata"

	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/tags"
	"github.com/opencloud-eu/reva/v2/pkg/utils"

	searchmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/search/v0"
	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/thumbnails/pkg/thumbnail"
	"github.com/opencloud-eu/opencloud/services/webdav/pkg/constants"
	"github.com/opencloud-eu/opencloud/services/webdav/pkg/net"
	"github.com/opencloud-eu/opencloud/services/webdav/pkg/prop"
	"github.com/opencloud-eu/opencloud/services/webdav/pkg/propfind"
)

const (
	elementNameSearchFiles = "search-files"
	// TODO elementNameFilterFiles = "filter-files"
)

// Search is the endpoint for retrieving search results for REPORT requests
func (g Webdav) Search(w http.ResponseWriter, r *http.Request) {
	logger := g.log.SubloggerWithRequestID(r.Context())
	rep, err := readReport(r.Body)
	if err != nil {
		renderError(w, r, errBadRequest(err.Error()))
		logger.Debug().Err(err).Msg("error reading report")
		return
	}

	if rep.SearchFiles == nil {
		renderError(w, r, errBadRequest("missing search-files tag"))
		logger.Debug().Err(err).Msg("error reading report")
		return
	}

	t := r.Header.Get(revactx.TokenHeader)
	ctx := revactx.ContextSetToken(r.Context(), t)
	ctx = metadata.Set(ctx, revactx.TokenHeader, t)

	req := &searchsvc.SearchRequest{
		Query:    rep.SearchFiles.Search.Pattern,
		PageSize: int32(rep.SearchFiles.Search.Limit),
	}

	// Limit search to the according space when searching /dav/spaces/
	if strings.HasPrefix(r.URL.Path, "/dav/spaces") {
		space := strings.TrimPrefix(r.URL.Path, "/dav/spaces/")
		rid, err := storagespace.ParseID(space)
		if err != nil {
			logger.Debug().Err(err).Msg("error parsing the space id for filtering")
		} else {
			req.Ref = &searchmsg.Reference{
				ResourceId: &searchmsg.ResourceID{
					StorageId: rid.StorageId,
					SpaceId:   rid.SpaceId,
					OpaqueId:  rid.SpaceId,
				},
			}
		}
	}

	rsp, err := g.searchClient.Search(ctx, req)
	if err != nil {
		e := merrors.Parse(err.Error())
		switch e.Code {
		case http.StatusBadRequest:
			renderError(w, r, errBadRequest(e.Detail))
		default:
			renderError(w, r, errInternalError(err.Error()))
		}
		logger.Error().Err(err).Msg("could not get search results")
		return
	}

	g.sendSearchResponse(rsp, w, r)
}

func (g Webdav) sendSearchResponse(rsp *searchsvc.SearchResponse, w http.ResponseWriter, r *http.Request) {
	logger := g.log.SubloggerWithRequestID(r.Context())
	responsesXML, err := multistatusResponse(r.Context(), g.config.OpenCloudPublicURL, rsp.Matches)
	if err != nil {
		logger.Error().Err(err).Msg("error formatting propfind")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set(net.HeaderDav, "1, 3, extended-mkcol")
	w.Header().Set(net.HeaderContentType, "application/xml; charset=utf-8")
	if len(rsp.Matches) > 0 {
		w.Header().Set(net.HeaderContentRange, fmt.Sprintf("rows 0-%d/%d", len(rsp.Matches)-1, rsp.TotalMatches))
	}
	w.WriteHeader(http.StatusMultiStatus)
	if _, err := w.Write(responsesXML); err != nil {
		logger.Err(err).Msg("error writing response")
	}
}

// multistatusResponse converts a list of matches into a multistatus response string
func multistatusResponse(ctx context.Context, publicURL string, matches []*searchmsg.Match) ([]byte, error) {
	responses := make([]*propfind.ResponseXML, 0, len(matches))
	for i := range matches {
		res, err := matchToPropResponse(ctx, publicURL, matches[i])
		if err != nil {
			return nil, err
		}
		responses = append(responses, res)
	}

	msr := propfind.NewMultiStatusResponseXML()
	msr.Responses = responses
	msg, err := xml.Marshal(msr)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func matchToPropResponse(ctx context.Context, publicURL string, match *searchmsg.Match) (*propfind.ResponseXML, error) {
	// unfortunately, search uses own versions of ResourceId and Ref. So we need to assert them here
	var (
		ref string
		err error
	)

	// to copy PROPFIND behaviour, we need to deliver different ids
	// for shares it needs to be sharestorageproviderid!shareid
	// for other spaces it needs to be storageproviderid$spaceid
	switch match.Entity.Ref.ResourceId.StorageId {
	default:
		ref, err = storagespace.FormatReference(&provider.Reference{
			ResourceId: &provider.ResourceId{
				StorageId: match.Entity.Ref.ResourceId.StorageId,
				SpaceId:   match.Entity.Ref.ResourceId.SpaceId,
			},
			Path: match.Entity.Ref.Path,
		})
	case utils.ShareStorageProviderID:
		ref, err = storagespace.FormatReference(&provider.Reference{
			ResourceId: &provider.ResourceId{
				//StorageId: match.Entity.Ref.ResourceId.StorageId,
				SpaceId:  match.Entity.Ref.ResourceId.SpaceId,
				OpaqueId: match.Entity.Ref.ResourceId.OpaqueId,
			},
			Path: match.Entity.Ref.Path,
		})
	}
	if err != nil {
		return nil, err
	}
	response := propfind.ResponseXML{
		Href:     net.EncodePath(path.Join("/remote.php/dav/spaces/", ref)),
		Propstat: []propfind.PropstatXML{},
	}

	propstatOK := propfind.PropstatXML{
		Status: "HTTP/1.1 200 OK",
		Prop:   []prop.PropertyXML{},
	}

	propstatOK.Prop = append(propstatOK.Prop, prop.Escaped("oc:fileid", storagespace.FormatResourceID(&provider.ResourceId{
		StorageId: match.Entity.Id.StorageId,
		SpaceId:   match.Entity.Id.SpaceId,
		OpaqueId:  match.Entity.Id.OpaqueId,
	})))
	if match.Entity.ParentId != nil {
		propstatOK.Prop = append(propstatOK.Prop, prop.Escaped("oc:file-parent", storagespace.FormatResourceID(&provider.ResourceId{
			StorageId: match.Entity.ParentId.StorageId,
			SpaceId:   match.Entity.ParentId.SpaceId,
			OpaqueId:  match.Entity.ParentId.OpaqueId,
		})))
	}
	if match.Entity.Ref.ResourceId.StorageId == utils.ShareStorageProviderID {
		propstatOK.Prop = append(propstatOK.Prop, prop.Escaped("oc:shareid", match.Entity.Ref.ResourceId.OpaqueId))
		propstatOK.Prop = append(propstatOK.Prop, prop.Escaped("oc:shareroot", match.Entity.ShareRootName))
		propstatOK.Prop = append(propstatOK.Prop, prop.Escaped("oc:remote-item-id", storagespace.FormatResourceID(&provider.ResourceId{
			StorageId: match.Entity.GetRemoteItemId().GetStorageId(),
			SpaceId:   match.Entity.GetRemoteItemId().GetSpaceId(),
			OpaqueId:  match.Entity.GetRemoteItemId().GetOpaqueId(),
		})))
	}
	propstatOK.Prop = append(propstatOK.Prop, prop.Escaped("oc:name", match.Entity.Name))
	propstatOK.Prop = append(propstatOK.Prop, prop.Escaped("d:getlastmodified", match.Entity.LastModifiedTime.AsTime().Format(constants.RFC1123)))
	propstatOK.Prop = append(propstatOK.Prop, prop.Escaped("oc:permissions", match.Entity.Permissions))
	propstatOK.Prop = append(propstatOK.Prop, prop.Escaped("oc:highlights", match.Entity.Highlights))
	propstatOK.Prop = append(propstatOK.Prop, prop.Escaped("d:getcontenttype", match.Entity.MimeType))
	_, isSupportedMimeType := thumbnail.SupportedMimeTypes[match.Entity.MimeType]
	if isSupportedMimeType {
		propstatOK.Prop = append(propstatOK.Prop, prop.Escaped("oc:has-preview", "1"))
	} else {
		propstatOK.Prop = append(propstatOK.Prop, prop.Escaped("oc:has-preview", "0"))
	}

	t := tags.New(match.Entity.Tags...)
	propstatOK.Prop = append(propstatOK.Prop, prop.Escaped("oc:tags", t.AsList()))

	// those seem empty - bug?
	propstatOK.Prop = append(propstatOK.Prop, prop.Escaped("d:getetag", match.Entity.Etag))

	size := strconv.FormatUint(match.Entity.Size, 10)
	if match.Entity.Type == uint64(provider.ResourceType_RESOURCE_TYPE_CONTAINER) {
		propstatOK.Prop = append(propstatOK.Prop, prop.Raw("d:resourcetype", "<d:collection/>"))
		propstatOK.Prop = append(propstatOK.Prop, prop.Escaped("oc:size", size))
	} else {
		propstatOK.Prop = append(propstatOK.Prop,
			prop.Escaped("d:resourcetype", ""),
			prop.Escaped("d:getcontentlength", size),
		)
	}
	score := strconv.FormatFloat(float64(match.Score), 'f', -1, 64)
	propstatOK.Prop = append(propstatOK.Prop, prop.Escaped("oc:score", score))

	if privateURL, err := url.Parse(publicURL); err == nil && match.Entity.Id != nil {
		privateURL.Path = path.Join(privateURL.Path, "f", storagespace.FormatResourceID(&provider.ResourceId{
			StorageId: match.Entity.Id.StorageId,
			SpaceId:   match.Entity.Id.SpaceId,
			OpaqueId:  match.Entity.Id.OpaqueId,
		}))
		propstatOK.Prop = append(propstatOK.Prop, prop.Escaped("oc:privatelink", privateURL.String()))
	}

	if len(propstatOK.Prop) > 0 {
		response.Propstat = append(response.Propstat, propstatOK)
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

type report struct {
	SearchFiles *reportSearchFiles
	// FilterFiles TODO add this for tag based search
	FilterFiles *reportFilterFiles `xml:"filter-files"`
}
type reportSearchFiles struct {
	XMLName xml.Name                `xml:"search-files"`
	Lang    string                  `xml:"xml:lang,attr,omitempty"`
	Prop    Props                   `xml:"DAV: prop"`
	Search  reportSearchFilesSearch `xml:"search"`
}
type reportSearchFilesSearch struct {
	Pattern string `xml:"pattern"`
	Limit   int    `xml:"limit"`
	Offset  int    `xml:"offset"`
}

type reportFilterFiles struct {
	XMLName xml.Name               `xml:"filter-files"`
	Lang    string                 `xml:"xml:lang,attr,omitempty"`
	Prop    Props                  `xml:"DAV: prop"`
	Rules   reportFilterFilesRules `xml:"filter-rules"`
}

type reportFilterFilesRules struct {
	Favorite  bool `xml:"favorite"`
	SystemTag int  `xml:"systemtag"`
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

func readReport(r io.Reader) (rep *report, err error) {
	decoder := xml.NewDecoder(r)
	rep = &report{}
	for {
		t, err := decoder.Token()
		if err == io.EOF {
			// io.EOF is a successful end
			return rep, nil
		}
		if err != nil {
			return nil, err
		}

		if v, ok := t.(xml.StartElement); ok {
			if v.Name.Local == elementNameSearchFiles {
				var repSF reportSearchFiles
				err = decoder.DecodeElement(&repSF, &v)
				if err != nil {
					return nil, err
				}
				rep.SearchFiles = &repSF
				/*
					} else if v.Name.Local == elementNameFilterFiles {
						var repFF reportFilterFiles
						err = decoder.DecodeElement(&repFF, &v)
						if err != nil {
							return nil, http.StatusBadRequest, err
						}
						rep.FilterFiles = &repFF
				*/
			}
		}
	}
}
