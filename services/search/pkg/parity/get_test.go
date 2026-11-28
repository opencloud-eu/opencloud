package parity

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

// the indexer reads its retry bookkeeping back through Get, a nested object
// with a timestamp has to survive the round trip on every engine
var _ = Describe("Get", Ordered, func() {
	failed := fixtureDoc("failed.pdf", withExtractionFailure(2, "2030-01-01T12:30:00Z"))
	extracted := fixtureDoc("extracted.txt")

	var engines []testEngine
	BeforeAll(func() {
		engines = newEngines("opencloud-test-engine-parity-get", []search.Resource{failed, extracted})
	})

	for _, name := range engineNames {
		It("returns the extraction failure as written on "+name, func() {
			e := engineNamed(engines, name)
			if e.unavailable != "" {
				Skip(e.unavailable)
			}

			r, err := e.backend.Get(context.Background(), failed.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(r.ExtractionFailure).NotTo(BeNil())
			Expect(r.ExtractionFailure.Count).To(Equal(2))
			Expect(r.ExtractionFailure.RetryAt).NotTo(BeNil())
			Expect(*r.ExtractionFailure.RetryAt).To(BeTemporally("==", *failed.ExtractionFailure.RetryAt))

			r, err = e.backend.Get(context.Background(), extracted.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(r.ExtractionFailure).To(BeNil())
		})
	}
})
