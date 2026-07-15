package convert_test

import (
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	opensearchgoAPI "github.com/opensearch-project/opensearch-go/v4/opensearchapi"

	"github.com/opencloud-eu/opencloud/pkg/conversions"
	searchMessage "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/convert"
	opensearchtest "github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/test"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

// jsonMarshal marshals data to a JSON string, failing the running spec on error.
func jsonMarshal(data any) string {
	GinkgoHelper()
	b, err := json.Marshal(data)
	Expect(err).ToNot(HaveOccurred())
	return string(b)
}

var _ = Describe("OpenSearchHitToMatch", func() {
	var (
		resource search.Resource
		hit      opensearchgoAPI.SearchHit
		mtime    time.Time
		match    *searchMessage.Match
		err      error
	)

	BeforeEach(func() {
		resource = opensearchtest.Testdata.Resources.File
		resource.MimeType = "audio/mpeg"
		mtime = time.Date(2025, 7, 24, 15, 15, 1, 0, time.UTC)
		resource.Mtime = &mtime
		resource.Favorites = []string{"cbf24bce-3e6e-4d9e-a2a2-cbf24bce3e6e"}

		hit = opensearchgoAPI.SearchHit{
			Score:  1.1,
			Source: json.RawMessage(jsonMarshal(resource)),
			Highlight: map[string][]string{
				"Content": {"first match", "second match"},
			},
		}

		match, err = convert.OpenSearchHitToMatch(hit)
		Expect(err).ToNot(HaveOccurred())
	})

	It("maps the score", func() {
		Expect(match.Score).To(Equal(hit.Score))
	})

	It("maps all resource fields to the entity", func() {
		entity := match.Entity
		Expect(entity).ToNot(BeNil())

		// reference (derived from RootID) and path
		Expect(entity.Ref.ResourceId.StorageId).To(Equal("1"))
		Expect(entity.Ref.ResourceId.SpaceId).To(Equal("1"))
		Expect(entity.Ref.ResourceId.OpaqueId).To(Equal("1"))
		Expect(entity.Ref.Path).To(Equal(resource.Path))

		// resource id
		Expect(entity.Id.StorageId).To(Equal("1"))
		Expect(entity.Id.SpaceId).To(Equal("1"))
		Expect(entity.Id.OpaqueId).To(Equal("3"))

		// parent id
		Expect(entity.ParentId.StorageId).To(Equal("1"))
		Expect(entity.ParentId.SpaceId).To(Equal("1"))
		Expect(entity.ParentId.OpaqueId).To(Equal("2"))

		// scalar fields
		Expect(entity.Name).To(Equal(resource.Name))
		Expect(entity.Size).To(Equal(resource.Size))
		Expect(entity.Type).To(Equal(resource.Type))
		Expect(entity.MimeType).To(Equal(resource.MimeType))
		Expect(entity.Deleted).To(Equal(resource.Deleted))
		Expect(entity.Tags).To(Equal(resource.Tags))
		Expect(entity.Favorites).To(Equal(resource.Favorites))

		// highlights are joined together
		Expect(entity.Highlights).To(Equal("first match; second match"))

		// last modified time is parsed from the Mtime
		Expect(entity.LastModifiedTime).ToNot(BeNil())
		Expect(entity.LastModifiedTime.Seconds).To(Equal(mtime.Unix()))
	})

	It("converts the media metadata to the expected types", func() {
		// searchMessage.Audio contains int64, int32 ... values that are converted to strings by the JSON marshaler,
		// so we need to convert the resource fields to align the expectations for the JSON comparison.
		expectedAudio, err := conversions.To[*searchMessage.Audio](resource.Audio)
		Expect(err).ToNot(HaveOccurred())
		Expect(match.Entity.Audio.Bitrate).To(Equal(resource.Audio.Bitrate))
		Expect(jsonMarshal(match.Entity.Audio)).To(MatchJSON(jsonMarshal(expectedAudio)))

		expectedImage, err := conversions.To[*searchMessage.Image](resource.Image)
		Expect(err).ToNot(HaveOccurred())
		Expect(jsonMarshal(match.Entity.Image)).To(MatchJSON(jsonMarshal(expectedImage)))

		expectedLocation, err := conversions.To[*searchMessage.GeoCoordinates](resource.Location)
		Expect(err).ToNot(HaveOccurred())
		Expect(jsonMarshal(match.Entity.Location)).To(MatchJSON(jsonMarshal(expectedLocation)))

		expectedPhoto, err := conversions.To[*searchMessage.Photo](resource.Photo)
		Expect(err).ToNot(HaveOccurred())
		Expect(jsonMarshal(match.Entity.Photo)).To(MatchJSON(jsonMarshal(expectedPhoto)))
	})
})
