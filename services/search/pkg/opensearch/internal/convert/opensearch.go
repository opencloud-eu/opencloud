package convert

import (
	"fmt"
	"strings"

	opensearchgoAPI "github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/opencloud-eu/reva/v2/pkg/storagespace"

	"github.com/opencloud-eu/opencloud/pkg/conversions"
	searchMessage "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

// copyFacet converts a typed pointer from the indexed shape (libregraph) to
// the protobuf shape via conversions.To. Returns nil when src is nil so the
// enclosing Match.Entity field stays nil.
func copyFacet[Dst, Src any](src *Src) *Dst {
	if src == nil {
		return nil
	}
	dst, _ := conversions.To[*Dst](src)
	return dst
}

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
			Audio:       copyFacet[searchMessage.Audio](resource.Audio),
			Image:       copyFacet[searchMessage.Image](resource.Image),
			Location:    copyFacet[searchMessage.GeoCoordinates](resource.Location),
			Photo:       copyFacet[searchMessage.Photo](resource.Photo),
			Video:       copyFacet[searchMessage.Video](resource.Video),
			MotionPhoto: copyFacet[searchMessage.MotionPhoto](resource.MotionPhoto),
		},
	}

	if resource.Mtime != nil {
		match.Entity.LastModifiedTime = timestamppb.New(*resource.Mtime)
	}

	return match, nil
}
