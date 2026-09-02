package content_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/stretchr/testify/mock"

	"github.com/opencloud-eu/opencloud/pkg/log"
	conf "github.com/opencloud-eu/opencloud/services/search/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/services/search/pkg/content"
	contentMocks "github.com/opencloud-eu/opencloud/services/search/pkg/content/mocks"
)

var _ = Describe("Tika", func() {
	Describe("extract", func() {
		var (
			body          string
			fullResponse  string
			language      string
			tika4Language bool
			version       string
			srv           *httptest.Server
			tika          *content.Tika
		)

		BeforeEach(func() {
			body = ""
			language = ""
			tika4Language = false
			version = ""
			fullResponse = ""
			srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				out := ""
				switch req.URL.Path {
				case "/version":
					out = version
				case "/language":
					if tika4Language {
						out = language
					} else {
						w.WriteHeader(http.StatusNotFound)
						return
					}
				case "/language/string":
					out = language
				case "/rmeta/text":
					if fullResponse != "" {
						out = fullResponse
					} else {
						out = fmt.Sprintf(`[{"X-TIKA:content":"%s"}]`, body)
					}
				}

				_, _ = w.Write([]byte(out))
			}))

			cfg := conf.DefaultConfig()
			cfg.Extractor.Tika.TikaURL = srv.URL
			cfg.Extractor.Tika.CleanStopWords = true

			var err error
			tika, err = content.NewTikaExtractor(nil, log.NewLogger(), cfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(tika).ToNot(BeNil())

			retriever := &contentMocks.Retriever{}
			retriever.On("Retrieve", mock.Anything, mock.Anything, mock.Anything).Return(io.NopCloser(strings.NewReader(body)), nil)

			tika.Retriever = retriever
		})

		AfterEach(func() {
			srv.Close()
		})

		It("skips non file resources", func() {
			doc, err := tika.Extract(context.TODO(), &provider.ResourceInfo{})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.Content).To(Equal(""))
		})

		It("adds content", func() {
			body = "any body"

			doc, err := tika.Extract(context.TODO(), &provider.ResourceInfo{
				Type: provider.ResourceType_RESOURCE_TYPE_FILE,
				Size: 1,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.Content).To(Equal(body))
		})

		It("adds the title", func() {
			fullResponse = `[{"dc:title": "quarterly report", "X-TIKA:content": "some data"}]`

			doc, err := tika.Extract(context.TODO(), &provider.ResourceInfo{
				Type: provider.ResourceType_RESOURCE_TYPE_FILE,
				Size: 1,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.Title).To(Equal("quarterly report"))
		})

		It("adds the content of a tika 4", func() {
			fullResponse = `[{"tk:content": "some data"}]`

			doc, err := tika.Extract(context.TODO(), &provider.ResourceInfo{
				Type: provider.ResourceType_RESOURCE_TYPE_FILE,
				Size: 1,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.Content).To(Equal("some data"))
		})

		It("adds the title of an older tika", func() {
			fullResponse = `[{"title": "quarterly report"}]`

			doc, err := tika.Extract(context.TODO(), &provider.ResourceInfo{
				Type: provider.ResourceType_RESOURCE_TYPE_FILE,
				Size: 1,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.Title).To(Equal("quarterly report"))
		})

		It("removes stop words with a tika 4 language endpoint", func() {
			body = "body to test stop words!!! against almost everyone"
			language = "en"
			tika4Language = true

			doc, err := tika.Extract(context.TODO(), &provider.ResourceInfo{
				Type: provider.ResourceType_RESOURCE_TYPE_FILE,
				Size: 1,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.Content).To(Equal("body test stop words!!!"))
		})

		It("ignores zero exposure time", func() {
			fullResponse = `[{"exif:ExposureTime": "0.0", "exif:FNumber": "2.0"}]`

			doc, err := tika.Extract(context.TODO(), &provider.ResourceInfo{
				Type: provider.ResourceType_RESOURCE_TYPE_FILE,
				Size: 1,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.Photo).ToNot(BeNil())
			Expect(doc.Photo.FNumber).To(Equal(libregraph.PtrFloat64(2)))
			Expect(doc.Photo.ExposureDenominator).To(BeNil())
		})

		It("removes stop words", func() {
			body = "body to test stop words!!! against almost everyone"
			language = "en"

			doc, err := tika.Extract(context.TODO(), &provider.ResourceInfo{
				Type: provider.ResourceType_RESOURCE_TYPE_FILE,
				Size: 1,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.Content).To(Equal("body test stop words!!!"))
		})

		It("keeps the audio facet when an embedded resource follows", func() {
			fullResponse = `[{"Content-Type": "audio/mpeg", "dc:title": "Sucker", "tk:content": "lyrics"}, {"Content-Type": "image/jpeg", "tiff:ImageWidth": "500"}]`

			doc, err := tika.Extract(context.TODO(), &provider.ResourceInfo{
				Type: provider.ResourceType_RESOURCE_TYPE_FILE,
				Size: 1,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.Audio).ToNot(BeNil())
			Expect(doc.Audio.Title).To(Equal(libregraph.PtrString("Sucker")))
			Expect(doc.Image).ToNot(BeNil())
		})

		It("adds no audio facet to non-audio documents", func() {
			fullResponse = `[{"Content-Type": "application/pdf", "dc:title": "quarterly report"}]`

			doc, err := tika.Extract(context.TODO(), &provider.ResourceInfo{
				Type: provider.ResourceType_RESOURCE_TYPE_FILE,
				Size: 1,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.Audio).To(BeNil())
			Expect(doc.Title).To(Equal("quarterly report"))
		})

		It("prefers the tika 4 content key over the legacy one", func() {
			fullResponse = `[{"tk:content": "new", "X-TIKA:content": "old"}]`

			doc, err := tika.Extract(context.TODO(), &provider.ResourceInfo{
				Type: provider.ResourceType_RESOURCE_TYPE_FILE,
				Size: 1,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.Content).To(Equal("new"))
		})

		It("joins the content of all meta entries", func() {
			fullResponse = `[{"tk:content": "one"}, {"X-TIKA:content": "two"}]`

			doc, err := tika.Extract(context.TODO(), &provider.ResourceInfo{
				Type: provider.ResourceType_RESOURCE_TYPE_FILE,
				Size: 1,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.Content).To(Equal("one two"))
		})

		It("keeps stop words", func() {
			body = "body to test stop words!!! against almost everyone"
			language = "en"

			tika.CleanStopWords = false
			doc, err := tika.Extract(context.TODO(), &provider.ResourceInfo{
				Type: provider.ResourceType_RESOURCE_TYPE_FILE,
				Size: 1,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.Content).To(Equal(body))
		})
	})
})
