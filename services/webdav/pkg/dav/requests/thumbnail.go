package requests

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	providerv1beta1 "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/go-chi/chi/v5"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"

	"github.com/opencloud-eu/opencloud/services/webdav/pkg/constants"
)

const (
	// DefaultWidth defines the default width of a thumbnail
	DefaultWidth = 32
	// DefaultHeight defines the default height of a thumbnail
	DefaultHeight = 32
)

// ThumbnailRequest combines all parameters provided when requesting a thumbnail.
type ThumbnailRequest struct {
	// Ref is the CS3 reference to the source file. It carries both the space root
	// (ResourceId) and the relative path, mirroring reva's ocdav split of
	// space-ID-based requests from path-only requests:
	//   - Space-ID requests (dav/spaces/{id}/...): ResourceId is the parsed space
	//     ID and Path is the URL-relative path. The workflow uses it directly.
	//   - Path-only requests (dav/files/{user}/..., /webdav/..., public links):
	//     ResourceId is nil and Path is an absolute CS3 path. The workflow resolves
	//     the space root via a lookup (which also works for public links).
	Ref *providerv1beta1.Reference
	// The file name of the source file including the extension
	Filename string
	// The requested width of the thumbnail
	Width int32
	// The requested height of the thumbnail
	Height int32
	// Indicates which image processor to use
	Processor string
	// Aspect is the legacy ownCloud "a" flag when the client sent it: true wants
	// the aspect ratio preserved, false wants the box filled. It is nil when the
	// client said nothing, and then nothing is reported as contradicted either.
	Aspect *bool
	// Identifier is the username from /dav/files/{user}/... when present. The
	// workflow resolves it via GetUserByClaim so the file is looked up in that
	// user's home; empty for space-ID, /webdav/ and public-link requests.
	Identifier string
}

// ParseThumbnailRequest extracts all required parameters from a http request.
func ParseThumbnailRequest(r *http.Request) (*ThumbnailRequest, error) {
	ctx := r.Context()

	fp := ctx.Value(constants.ContextKeyPath).(string)

	var (
		ref        *providerv1beta1.Reference
		identifier string
	)
	if v := ctx.Value(constants.ContextKeyID); v != nil {
		id := addMissingStorageID(v.(string))
		if strings.Contains(id, "$") {
			// Space-ID based request (dav/spaces/{id}/...): the URL carries the
			// space root, so build the reference directly from the parsed ID.
			rid, err := storagespace.ParseID(id)
			if err != nil {
				return nil, fmt.Errorf("parse space id %q: %w", id, err)
			}
			ref = &providerv1beta1.Reference{
				ResourceId: &rid,
				Path:       utils.MakeRelativePath(fp),
			}
		} else {
			// The identifier is a username (dav/files/{username}/...); the workflow
			// resolves it to that user's home path via GetUserByClaim.
			ref = &providerv1beta1.Reference{Path: fp}
			identifier = id
		}
	}

	if token := chi.URLParam(r, "token"); token != "" {
		ref = &providerv1beta1.Reference{Path: path.Join("/public", token, strings.TrimLeft(fp, "/"))}
	} else if ref == nil {
		// Path-only request (/webdav/...): the absolute CS3 path is carried in
		// ContextKeyPath; the workflow resolves the space root.
		ref = &providerv1beta1.Reference{Path: fp}
	}

	q := r.URL.Query()
	width, height, err := parseDimensions(q)
	if err != nil {
		return nil, err
	}
	processor := q.Get("processor")
	if (width == 0 || height == 0) && processor != "" && cropsToBox(processor) {
		// A processor that was asked for by name and crops has no target for an
		// unconstrained side. Without a processor the axis decides and the
		// workflow fits instead. The wording is the legacy one because clients
		// assert on it.
		name := "width"
		if width != 0 {
			name = "height"
		}
		return nil, fmt.Errorf("Cannot set %s of 0 or smaller!", name)
	}

	return &ThumbnailRequest{
		Ref:        ref,
		Filename:   filepath.Base(fp),
		Width:      int32(width),
		Height:     int32(height),
		Processor:  processor,
		Aspect:     aspect(q),
		Identifier: identifier,
	}, nil
}

// addMissingStorageID fills in the shares storage provider ID when a share mount
// space ID is missing it, so the parsed reference routes to the right provider.
func addMissingStorageID(id string) string {
	rid := &providerv1beta1.ResourceId{}
	rid.StorageId, rid.SpaceId, rid.OpaqueId, _ = storagespace.SplitID(id)
	if rid.StorageId == "" && rid.SpaceId == utils.ShareStorageSpaceID {
		rid.StorageId = utils.ShareStorageProviderID
	}
	return storagespace.FormatResourceID(rid)
}

// parseDimensions reads the requested box. An omitted axis is left
// unconstrained, so a client that knows only one side can ask for it alone,
// while an explicit zero stays the error it has always been. Only when both are
// absent does the legacy default apply, which keeps the dimensionless OC10 call
// working.
func parseDimensions(q url.Values) (int64, int64, error) {
	x, y := q.Get("x"), q.Get("y")
	if x == "" && y == "" {
		return DefaultWidth, DefaultHeight, nil
	}
	width, err := parseDimension(x, "width")
	if err != nil {
		return 0, 0, err
	}
	height, err := parseDimension(y, "height")
	if err != nil {
		return 0, 0, err
	}
	return width, height, nil
}

func parseDimension(d, name string) (int64, error) {
	if d == "" {
		return 0, nil
	}
	result, err := strconv.ParseInt(d, 10, 32)
	if err != nil || result <= 0 {
		// The error message doesn't fit but for OC10 API compatibility reasons we have to set this.
		return 0, fmt.Errorf("Cannot set %s of 0 or smaller!", name)
	}
	return result, nil
}

// aspect reads the legacy "a" flag, nil when the client did not send it.
func aspect(q url.Values) *bool {
	if !q.Has("a") {
		return nil
	}

	preserve := q.Get("a") != "0"

	return &preserve
}

// cropsToBox reports whether the processor produces exactly the requested box,
// which an unconstrained axis cannot describe.
func cropsToBox(processor string) bool {
	switch strings.ToLower(strings.TrimSpace(processor)) {
	case "fit", "fit-in":
		return false
	default:
		return true
	}
}
