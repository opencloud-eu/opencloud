package content_test

import (
	"time"

	"github.com/opencloud-eu/opencloud/pkg/conversions"

	"context"
	"encoding/json"

	storageProvider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	cs3Types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/search/pkg/content"
)

var _ = Describe("Basic", func() {
	var (
		basic  content.Extractor
		logger = log.NewLogger()
		ctx    = context.TODO()
	)

	BeforeEach(func() {
		basic, _ = content.NewBasicExtractor(logger)
	})

	Describe("extract", func() {
		It("basic fields", func() {
			ri := &storageProvider.ResourceInfo{
				Name:     "bar.pdf",
				Path:     "./foo/bar.pdf",
				Size:     1024,
				MimeType: "application/pdf",
			}

			doc, err := basic.Extract(ctx, ri)

			Expect(err).To(BeNil())
			Expect(doc).ToNot(BeNil())
			Expect(doc.Name).To(Equal(ri.Name))
			Expect(doc.Size).To(Equal(ri.Size))
			Expect(doc.MimeType).To(Equal(ri.MimeType))
		})

		It("adds tags", func() {
			for _, data := range []struct {
				tags   string
				expect []string
			}{
				{tags: "", expect: []string{}},
				{tags: ",,,", expect: []string{}},
				{tags: ",foo,,", expect: []string{"foo"}},
				{tags: ",foo,,bar,", expect: []string{"foo", "bar"}},
			} {
				ri := &storageProvider.ResourceInfo{
					ArbitraryMetadata: &storageProvider.ArbitraryMetadata{
						Metadata: map[string]string{
							"tags": data.tags,
						},
					},
				}

				doc, err := basic.Extract(ctx, ri)
				Expect(err).To(BeNil())
				Expect(doc).ToNot(BeNil())
				Expect(doc.Tags).To(Equal(data.expect))
			}
		})

		It("RFC3339 mtime", func() {
			for _, data := range []struct {
				second uint64
				expect *time.Time
			}{
				{second: 4000, expect: conversions.ToPointer(time.Unix(4000, 0).UTC())},
				{second: 3000, expect: conversions.ToPointer(time.Unix(3000, 0).UTC())},
				{},
			} {
				ri := &storageProvider.ResourceInfo{}

				if data.second != 0 {
					ri.Mtime = &cs3Types.Timestamp{Seconds: data.second}
				}

				doc, err := basic.Extract(ctx, ri)
				Expect(err).To(BeNil())
				Expect(doc).ToNot(BeNil())
				Expect(doc.Mtime).To(Equal(data.expect))
			}
		})

		It("extracts favorites", func() {
			favorites := []string{"foo", "bar"}
			favBytes, _ := json.Marshal(favorites)

			ri := &storageProvider.ResourceInfo{
				Opaque: &cs3Types.Opaque{
					Map: map[string]*cs3Types.OpaqueEntry{
						"favorites": {
							Decoder: "json",
							Value:   favBytes,
						},
					},
				},
			}

			doc, err := basic.Extract(ctx, ri)
			Expect(err).To(BeNil())
			Expect(doc).ToNot(BeNil())
			Expect(doc.Favorites).To(Equal(favorites))
		})
	})
})
