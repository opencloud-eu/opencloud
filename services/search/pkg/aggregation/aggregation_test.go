package aggregation_test

import (
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/aggregation"
)

var _ = Describe("CheckBuckets", func() {
	// artists of an album each: twice as many buckets as artists
	artistsWithAlbum := func(artists int) []*searchsvc.AggregationResult {
		res := &searchsvc.AggregationResult{Field: "audio.artist"}
		for i := range artists {
			res.Buckets = append(res.Buckets, &searchsvc.Bucket{Key: strconv.Itoa(i), Count: 1, SubAggregations: []*searchsvc.AggregationResult{
				{Field: "audio.album", Buckets: []*searchsvc.Bucket{{Key: "Singles", Count: 1}}},
			}})
		}
		return []*searchsvc.AggregationResult{res}
	}

	artists := func(count int) []*searchsvc.AggregationResult {
		res := &searchsvc.AggregationResult{Field: "audio.artist"}
		for i := range count {
			res.Buckets = append(res.Buckets, &searchsvc.Bucket{Key: strconv.Itoa(i), Count: 1})
		}
		return []*searchsvc.AggregationResult{res}
	}

	It("counts the buckets against the limit, none is fine", func() {
		Expect(aggregation.MaxBuckets).To(Equal(65535))
		Expect(aggregation.CheckBuckets(nil)).To(Succeed())
		Expect(aggregation.CheckBuckets(artists(65535))).To(Succeed())
		Expect(aggregation.CheckBuckets(artists(65536))).To(MatchError(aggregation.ErrTooManyBuckets))
	})

	It("counts the buckets of all levels", func() {
		Expect(aggregation.CheckBuckets(artistsWithAlbum(32767))).To(Succeed(), "65534 buckets")
		Expect(aggregation.CheckBuckets(artistsWithAlbum(32768))).To(MatchError(aggregation.ErrTooManyBuckets), "65536 buckets")
	})
})
