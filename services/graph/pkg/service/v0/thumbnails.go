package svc

import (
	"fmt"
	"net/http"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"

	"github.com/opencloud-eu/opencloud/services/thumbnails/pkg/thumbnail"
)

const (
	thumbnailBoxSmall  = 36
	thumbnailBoxMedium = 48
	thumbnailBoxLarge  = 96
)

func (g Graph) setDriveItemsThumbnails(r *http.Request, items []libregraph.DriveItem, infos []*provider.ResourceInfo) {
	if !driveItemRelationExpanded(r, _expandThumbnails) {
		return
	}
	for i := range items {
		if i < len(infos) {
			setDriveItemThumbnails(&items[i], infos[i], g.config.Commons.OpenCloudURL)
		}
	}
}

func setDriveItemThumbnails(item *libregraph.DriveItem, res *provider.ResourceInfo, baseURL string) {
	if set := previewThumbnailSet(res, baseURL); set != nil {
		item.SetThumbnails([]libregraph.ThumbnailSet{*set})
	}
}

// previewThumbnailSet returns nil when the thumbnailer cannot render the resource.
func previewThumbnailSet(res *provider.ResourceInfo, baseURL string) *libregraph.ThumbnailSet {
	if !thumbnail.IsMimeTypeSupported(res.GetMimeType()) {
		return nil
	}
	return thumbnailSetFor(baseURL, storagespace.FormatResourceID(res.GetId()))
}

// thumbnailSetFor builds the urls of the WebDAV preview endpoint.
func thumbnailSetFor(baseURL, itemID string) *libregraph.ThumbnailSet {
	base := fmt.Sprintf("%s/dav/spaces/%s?scalingup=0&preview=1&processor=thumbnail", baseURL, itemID)
	return &libregraph.ThumbnailSet{
		Small:  previewThumbnail(base, thumbnailBoxSmall),
		Medium: previewThumbnail(base, thumbnailBoxMedium),
		Large:  previewThumbnail(base, thumbnailBoxLarge),
	}
}

func previewThumbnail(base string, box int32) *libregraph.Thumbnail {
	url := fmt.Sprintf("%s&x=%d&y=%d", base, box, box)
	return &libregraph.Thumbnail{Url: &url}
}

// setShareThumbnails works off the driveItem, the share listings have no resource
// info. The id comes separately, a received share carries it on its remote item.
func setShareThumbnails(item *libregraph.DriveItem, itemID, baseURL string) {
	mimeType := item.GetFile().MimeType
	if itemID == "" || mimeType == nil || !thumbnail.IsMimeTypeSupported(*mimeType) {
		return
	}
	item.SetThumbnails([]libregraph.ThumbnailSet{*thumbnailSetFor(baseURL, itemID)})
}
