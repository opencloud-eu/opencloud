package svc

import (
	"fmt"
	"net/http"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"

	"github.com/opencloud-eu/opencloud/services/webdav/pkg/thumbnail"
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
			setDriveItemThumbnails(&items[i], infos[i], g.config.Commons.OpenCloudURL, g.sourceTypes)
		}
	}
}

func setDriveItemThumbnails(item *libregraph.DriveItem, res *provider.ResourceInfo, baseURL string, sourceTypes thumbnail.SourceTypes) {
	if set := previewThumbnailSet(res, baseURL, sourceTypes); set != nil {
		item.SetThumbnails([]libregraph.ThumbnailSet{*set})
	}
}

// previewThumbnailSet returns nil when no preview can be produced for the resource.
func previewThumbnailSet(res *provider.ResourceInfo, baseURL string, sourceTypes thumbnail.SourceTypes) *libregraph.ThumbnailSet {
	if !sourceTypes.HasPreview(res) {
		return nil
	}

	itemID := storagespace.FormatResourceID(res.GetId())
	set := thumbnailSetFor(baseURL, itemID)
	if w, h := thumbnail.SourceDimensions(res); w > 0 && h > 0 {
		set.Source = previewThumbnail(previewBaseURL(baseURL, itemID), w, h)
	}
	return set
}

// thumbnailSetFor builds the urls of the WebDAV preview endpoint.
func thumbnailSetFor(baseURL, itemID string) *libregraph.ThumbnailSet {
	base := previewBaseURL(baseURL, itemID)
	return &libregraph.ThumbnailSet{
		Small:  previewThumbnail(base, thumbnailBoxSmall, thumbnailBoxSmall),
		Medium: previewThumbnail(base, thumbnailBoxMedium, thumbnailBoxMedium),
		Large:  previewThumbnail(base, thumbnailBoxLarge, thumbnailBoxLarge),
	}
}

func previewBaseURL(baseURL, itemID string) string {
	return fmt.Sprintf("%s/dav/spaces/%s?scalingup=0&preview=1&processor=thumbnail", baseURL, itemID)
}

func previewThumbnail(base string, width, height int32) *libregraph.Thumbnail {
	url := fmt.Sprintf("%s&x=%d&y=%d", base, width, height)
	return &libregraph.Thumbnail{Url: &url}
}

// setShareThumbnails works off the driveItem, the share listings have no resource
// info. The id comes separately, a received share carries it on its remote item.
func setShareThumbnails(item *libregraph.DriveItem, itemID, baseURL string, sourceTypes thumbnail.SourceTypes) {
	mimeType := item.GetFile().MimeType
	if itemID == "" || mimeType == nil || !sourceTypes.Supports(*mimeType) {
		return
	}
	item.SetThumbnails([]libregraph.ThumbnailSet{*thumbnailSetFor(baseURL, itemID)})
}
