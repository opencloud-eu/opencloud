package convert

import (
	"fmt"
	"strings"
	"time"

	opensearchgoAPI "github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/opencloud-eu/reva/v2/pkg/storagespace"

	"github.com/opencloud-eu/opencloud/pkg/conversions"
	searchMessage "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

func OpenSearchHitToMatch(hit opensearchgoAPI.SearchHit) (*searchMessage.Match, error) {
	resource, err := conversions.To[search.Resource](hit.Source)
	if err != nil {
		return nil, fmt.Errorf("failed to convert hit source: %w", err)
	}

	resourceRootID, err := storagespace.ParseID(resource.RootID)
	if err != nil {
		return nil, err
	}

	resourceID, err := storagespace.ParseID(resource.ID)
	if err != nil {
		return nil, err
	}

	resourceParentID, _ := storagespace.ParseID(resource.ParentID)

	match := &searchMessage.Match{
		Score: hit.Score,
		Entity: &searchMessage.Entity{
			Ref: &searchMessage.Reference{
				ResourceId: &searchMessage.ResourceID{
					StorageId: resourceRootID.GetStorageId(),
					SpaceId:   resourceRootID.GetSpaceId(),
					OpaqueId:  resourceRootID.GetOpaqueId(),
				},
				Path: resource.Path,
			},
			Id: &searchMessage.ResourceID{
				StorageId: resourceID.GetStorageId(),
				SpaceId:   resourceID.GetSpaceId(),
				OpaqueId:  resourceID.GetOpaqueId(),
			},
			Name: resource.Name,
			ParentId: &searchMessage.ResourceID{
				StorageId: resourceParentID.GetStorageId(),
				SpaceId:   resourceParentID.GetSpaceId(),
				OpaqueId:  resourceParentID.GetOpaqueId(),
			},
			Size:     resource.Size,
			Type:     resource.Type,
			MimeType: resource.MimeType,
			Deleted:  resource.Deleted,
			Tags:     resource.Tags,
			Highlights: func() string {
				contentHighlights, ok := hit.Highlight["Content"]
				if !ok {
					return ""
				}

				return strings.Join(contentHighlights[:], "; ")
			}(),
			Audio: func() *searchMessage.Audio {
				if !strings.HasPrefix(resource.MimeType, "audio/") {
					return nil
				}

				audio, _ := conversions.To[*searchMessage.Audio](resource.Audio)
				return audio
			}(),
			Image: func() *searchMessage.Image {
				image, _ := conversions.To[*searchMessage.Image](resource.Image)
				return image
			}(),
			Location: func() *searchMessage.GeoCoordinates {
				geoCoordinates, _ := conversions.To[*searchMessage.GeoCoordinates](resource.Location)
				return geoCoordinates
			}(),
			Photo: func() *searchMessage.Photo {
				photo, _ := conversions.To[*searchMessage.Photo](resource.Photo)
				return photo
			}(),
		},
	}

	if mtime, err := time.Parse(time.RFC3339, resource.Mtime); err == nil {
		match.Entity.LastModifiedTime = &timestamppb.Timestamp{Seconds: mtime.Unix(), Nanos: int32(mtime.Nanosecond())}
	}

	return match, nil
}
