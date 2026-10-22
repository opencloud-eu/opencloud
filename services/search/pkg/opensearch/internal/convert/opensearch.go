package convert

import (
	"fmt"
	"strings"

	opensearchgoAPI "github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/opencloud-eu/reva/v2/pkg/storagespace"

	"github.com/opencloud-eu/opencloud/pkg/conversions"
	searchMessage "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
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
			Size:      resource.Size,
			Type:      resource.Type,
			MimeType:  resource.MimeType,
			Deleted:   resource.Deleted,
			Tags:      resource.Tags,
			Favorites: resource.Favorites,
			Highlights: func() string {
				contentHighlights, ok := hit.Highlight["Content"]
				if !ok {
					return ""
				}

				return strings.Join(contentHighlights[:], "; ")
			}(),
			Audio:       mapping.ToProto[searchMessage.Audio](resource.Audio),
			Image:       mapping.ToProto[searchMessage.Image](resource.Image),
			Location:    mapping.ToProto[searchMessage.GeoCoordinates](resource.Location),
			Photo:       mapping.ToProto[searchMessage.Photo](resource.Photo),
			Video:       mapping.ToProto[searchMessage.Video](resource.Video),
			MotionPhoto: mapping.ToProto[searchMessage.MotionPhoto](resource.MotionPhoto),
			LivePhoto:   mapping.ToProto[searchMessage.LivePhoto](resource.LivePhoto),
		},
	}

	if resource.Mtime != nil {
		match.Entity.LastModifiedTime = timestamppb.New(*resource.Mtime)
	}

	return match, nil
}
