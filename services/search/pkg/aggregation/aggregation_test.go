package aggregation_test

import (
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/aggregation"
)

var _ = Describe("CheckBuckets", func() {
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
})
