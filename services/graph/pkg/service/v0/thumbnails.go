package svc

import (
	"fmt"
	"net/http"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"

	"github.com/opencloud-eu/opencloud/pkg/conversions"
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

// previewThumbnailSet returns nil when no preview can be produced for the resource.
func previewThumbnailSet(res *provider.ResourceInfo, baseURL string) *libregraph.ThumbnailSet {
	if !thumbnail.HasPreview(res) {
		return nil
	}

	itemID := storagespace.FormatResourceID(res.GetId())
	set := thumbnailSetFor(baseURL, itemID)
	// only the source carries exact dimensions, the boxes above are requests
	if w, h := previewSourceDimensions(res); w > 0 && h > 0 {
		url := fmt.Sprintf("%s&x=%d&y=%d", previewBaseURL(baseURL, itemID), w, h)
		set.Source = &libregraph.Thumbnail{Url: &url, Width: &w, Height: &h}
	}
	return set
}

// previewSourceDimensions: audio cover from oc.preview, images from the image facet.
func previewSourceDimensions(res *provider.ResourceInfo) (int32, int32) {
	if w, h := thumbnail.PreviewDimensions(res); w > 0 && h > 0 {
		return w, h
	}
	meta := res.GetArbitraryMetadata().GetMetadata()
	w := conversions.StringToInt32(meta["libre.graph.image.width"], 0)
	h := conversions.StringToInt32(meta["libre.graph.image.height"], 0)
	return w, h
}

// thumbnailSetFor builds the urls of the WebDAV preview endpoint.
func thumbnailSetFor(baseURL, itemID string) *libregraph.ThumbnailSet {
	base := previewBaseURL(baseURL, itemID)
	return &libregraph.ThumbnailSet{
		Small:  previewThumbnail(base, thumbnailBoxSmall),
		Medium: previewThumbnail(base, thumbnailBoxMedium),
		Large:  previewThumbnail(base, thumbnailBoxLarge),
	}
}

func previewBaseURL(baseURL, itemID string) string {
	return fmt.Sprintf("%s/dav/spaces/%s?scalingup=0&preview=1&processor=thumbnail", baseURL, itemID)
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
