package opensearch_test

import (
	"encoding/json"
	"testing"

	opensearchgoAPI "github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"github.com/stretchr/testify/assert"

	searchMessage "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/test"
)

func TestSearchHitToSearchMessageMatch(t *testing.T) {
	resource := opensearchtest.Testdata.Resources.File
	resource.MimeType = "audio/anything"

	hit := opensearchgoAPI.SearchHit{
		Score:  1.1,
		Source: json.RawMessage(opensearchtest.JSONMustMarshal(t, resource)),
	}
	match, err := opensearch.SearchHitToSearchMessageMatch(hit)
	assert.NoError(t, err)
	assert.Equal(t, hit.Score, match.Score)
	assert.Equal(t, resource.Name, match.Entity.Name)

	t.Run("converts the audio field to the expected type", func(t *testing.T) {
		// searchMessage.Audio contains int64, int32 ... values that are converted to strings by the JSON marshaler,
		// so we need to convert the resource.Audio to align the expectations for the JSON comparison.
		audio, err := opensearch.Convert[*searchMessage.Audio](resource.Audio)
		assert.NoError(t, err)

		assert.Equal(t, resource.Audio.Bitrate, match.Entity.Audio.Bitrate)
		assert.JSONEq(t, opensearchtest.JSONMustMarshal(t, audio), opensearchtest.JSONMustMarshal(t, match.Entity.Audio))
	})
}
