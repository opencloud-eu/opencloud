// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package bleve_test

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/blevesearch/bleve/v2/index/scorch"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/search/pkg/bleve"
	"github.com/opencloud-eu/opencloud/services/search/pkg/content"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

var _ = Describe("zapx segment buffer", func() {
	It("covers every segment version bleve can open", func() {
		Expect(scorch.SupportedSegmentTypes()).To(ConsistOf("zap"))
		Expect(bleve.SegmentBufferVersions()).To(ConsistOf(scorch.SupportedSegmentTypeVersions("zap")))
	})

	It("does not size the next segment after a large document", func() {
		// zapx keeps the last estimate in a sync.Pool, which hands it back only
		// on the same P
		DeferCleanup(runtime.GOMAXPROCS, runtime.GOMAXPROCS(1))

		idx, _, err := bleve.NewIndex(GinkgoT().TempDir(), log.NopLogger())
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(idx.Close)

		resource := func(id, text string) search.Resource {
			return search.Resource{
				ID:       "s$p!" + id,
				RootID:   "s$p!p",
				ParentID: "s$p!p",
				Path:     "./" + id,
				Type:     uint64(1),
				Document: content.Document{Name: id, Content: text},
			}
		}
		upsert := func(r search.Resource) {
			b, err := bleve.NewBatch(idx, 1)
			Expect(err).ToNot(HaveOccurred())
			Expect(b.Upsert(r.ID, r)).To(Succeed())
			Expect(b.Push()).To(Succeed())
		}

		// about 1 MB of distinct words, alone in its segment
		words := make([]string, 0, 100_000)
		for i := range cap(words) {
			words = append(words, fmt.Sprintf("word%d", i))
		}
		upsert(resource("large", strings.Join(words, " ")))

		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		upsert(resource("small", "a few words"))
		runtime.ReadMemStats(&after)

		// zapx would reserve a hundred times the large segment, some 500 MB
		Expect(after.TotalAlloc - before.TotalAlloc).To(BeNumerically("<", 50<<20))
	})
})
