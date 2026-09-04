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
	// The file extension
	Extension string
	// The requested width of the thumbnail
	Width int32
	// The requested height of the thumbnail
	Height int32
	// Indicates which image processor to use
	Processor string
	// Aspect reports whether the client wants the aspect ratio preserved (the
	// legacy ownCloud "a" flag: a=1/absent -> preserve, a=0 -> fill the box).
	Aspect bool
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
		id := v.(string)
		if strings.Contains(id, "$") {
			// Space-ID based request (dav/spaces/{id}/...): the URL carries the
			// space root, so build the reference directly from the parsed ID.
			rid, err := storagespace.ParseID(addMissingStorageID(id))
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

	return &ThumbnailRequest{
		Ref:        ref,
		Filename:   filepath.Base(fp),
		Extension:  filepath.Ext(fp),
		Width:      int32(width),
		Height:     int32(height),
		Processor:  q.Get("processor"),
		Aspect:     q.Get("a") != "0",
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

func parseDimensions(q url.Values) (int64, int64, error) {
	width, err := parseDimension(q.Get("x"), "width", DefaultWidth)
	if err != nil {
		return 0, 0, err
	}
	height, err := parseDimension(q.Get("y"), "height", DefaultHeight)
	if err != nil {
		return 0, 0, err
	}
	return width, height, nil
}

func parseDimension(d, name string, defaultValue int64) (int64, error) {
	if d == "" {
		return defaultValue, nil
	}
	result, err := strconv.ParseInt(d, 10, 32)
	if err != nil || result < 1 {
		// The error message doesn't fit but for OC10 API compatibility reasons we have to set this.
		return 0, fmt.Errorf("Cannot set %s of 0 or smaller!", name)
	}
	return result, nil
}
