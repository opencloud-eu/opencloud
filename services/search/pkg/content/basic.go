package content

import (
	"context"
	"encoding/json"

	storageProvider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/opencloud/pkg/conversions"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/reva/v2/pkg/tags"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

// Basic is the simplest Extractor implementation.
type Basic struct {
	logger log.Logger
}

// NewBasicExtractor creates a new Basic instance.
func NewBasicExtractor(logger log.Logger) (*Basic, error) {
	return &Basic{logger: logger}, nil
}

// Extract literally just rearranges the inputs and processes them into a Document.
func (b Basic) Extract(_ context.Context, ri *storageProvider.ResourceInfo) (Document, error) {
	doc := Document{
		Name:     ri.Name,
		Size:     ri.Size,
		MimeType: ri.MimeType,
	}

	if m := ri.ArbitraryMetadata.GetMetadata(); m != nil {
		if t, ok := m["tags"]; ok {
			doc.Tags = tags.New(t).AsSlice()
		}
	}

	if m := ri.Opaque.GetMap(); m != nil && m["favorites"] != nil {
		favEntry := m["favorites"]

		switch favEntry.Decoder {
		case "json":
			favorites := []string{}
			err := json.Unmarshal(favEntry.Value, &favorites)
			if err != nil {
				b.logger.Error().Err(err).Msg("failed to unmarshal favorites")
				break
			}

			doc.Favorites = favorites
		default:
			b.logger.Error().Msgf("unsupported decoder for favorites: %s", favEntry.Decoder)
		}
	}

	if ri.Mtime != nil {
		doc.Mtime = conversions.ToPointer(utils.TSToTime(ri.Mtime).UTC())
	}

	return doc, nil
}
