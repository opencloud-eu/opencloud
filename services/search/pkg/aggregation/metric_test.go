package aggregation_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/aggregation"
)

var _ = Describe("Observe", func() {
	It("keeps every accumulator, whatever the kind", func() {
		m := &searchsvc.Metric{Kind: searchsvc.MetricKind_METRIC_KIND_MAX}
		for _, v := range []float64{1982, 1971, 2001} {
			aggregation.Observe(m, v)
		}
		Expect(m.GetCount()).To(Equal(int64(3)))
		Expect(m.GetSum()).To(Equal(5954.0))
		Expect(m.GetMin()).To(Equal(1971.0))
		Expect(m.GetMax()).To(Equal(2001.0))
		Expect(m.Value).To(BeNil())
	})

	It("takes the first value as minimum and maximum, a negative one too", func() {
		m := &searchsvc.Metric{}
		aggregation.Observe(m, -3)
		Expect(m.GetMin()).To(Equal(-3.0))
		Expect(m.GetMax()).To(Equal(-3.0))
	})
})
