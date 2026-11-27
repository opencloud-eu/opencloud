package content_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

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
			rmetaStatus   int
			rmetaHangs    bool
			versionHangs  bool
			rmetaPath     string
			rmetaBytes    int
			srv           *httptest.Server
			tika          *content.Tika
		)

		BeforeEach(func() {
			body = ""
			language = ""
			tika4Language = false
			version = ""
			fullResponse = ""
			rmetaStatus = 0
			rmetaHangs = false
			versionHangs = false
			srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				out := ""
				switch req.URL.Path {
				case "/version":
					if versionHangs {
						<-req.Context().Done()
						return
					}
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
				case "/rmeta/text", "/rmeta/ignore":
					rmetaPath = req.URL.Path
					if rmetaHangs {
						// the server notices the client leaving only once the body is read
						_, _ = io.Copy(io.Discard, req.Body)
						<-req.Context().Done()
						return
					}
					received, _ := io.ReadAll(req.Body)
					rmetaBytes = len(received)
					if rmetaStatus != 0 {
						w.WriteHeader(rmetaStatus)
						return
					}
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

		It("fails when tika returns no metadata for a file with content", func() {
			fullResponse = "[]"

			_, err := tika.Extract(context.TODO(), &provider.ResourceInfo{
				Type: provider.ResourceType_RESOURCE_TYPE_FILE,
				Name: "report.pdf",
				Size: 1,
			})
			Expect(err).To(MatchError(content.ErrUnparsable))
			Expect(err).To(MatchError(ContainSubstring("report.pdf")))
		})

		DescribeTable("fails when tika could not parse the file",
			func(response string) {
				fullResponse = response

				doc, err := tika.Extract(context.TODO(), &provider.ResourceInfo{
					Type: provider.ResourceType_RESOURCE_TYPE_FILE,
					Name: "scan.png",
					Size: 1,
				})
				Expect(err).To(MatchError(content.ErrUnparsable))
				Expect(err).To(MatchError(And(ContainSubstring("scan.png"), ContainSubstring("bad exit value 1"))))
				Expect(err.Error()).ToNot(ContainSubstring("at org.apache"))
				Expect(doc.Name).To(Equal("scan.png"))
			},
			Entry("tika 4", `[{"tk:content": "", "tk:exception:container-exception": "org.apache.tika.exception.TikaException: TesseractOCRParser bad exit value 1\n\tat org.apache.tika.parser.ocr.TesseractOCRParser.doOCR"}]`),
			Entry("tika 3", `[{"X-TIKA:content": "", "X-TIKA:EXCEPTION:container_exception": "org.apache.tika.exception.TikaException: TesseractOCRParser bad exit value 1\n\tat org.apache.tika.parser.ocr.TesseractOCRParser.doOCR"}]`),
		)

		DescribeTable("tells an unreachable tika from a request it rejects",
			func(status int, unreachable bool) {
				rmetaStatus = status

				_, err := tika.Extract(context.TODO(), &provider.ResourceInfo{
					Type: provider.ResourceType_RESOURCE_TYPE_FILE,
					Size: 1,
				})
				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, content.ErrUnreachable)).To(Equal(unreachable))
			},
			Entry("server error", http.StatusServiceUnavailable, true),
			Entry("too many requests", http.StatusTooManyRequests, true),
			Entry("unprocessable file", http.StatusUnprocessableEntity, false),
		)

		It("counts a tika that does not answer in time as a failure of the file", func() {
			rmetaHangs = true
			tika.Timeout = 10 * time.Millisecond

			_, err := tika.Extract(context.TODO(), &provider.ResourceInfo{
				Type: provider.ResourceType_RESOURCE_TYPE_FILE,
				Size: 1,
			})
			Expect(err).To(MatchError(context.DeadlineExceeded))
			Expect(errors.Is(err, content.ErrUnreachable)).To(BeFalse())
		})

		It("counts a tika that does not even answer its version as unreachable", func() {
			DeferCleanup(content.SetProbeTimeout(10 * time.Millisecond))
			rmetaHangs, versionHangs = true, true
			tika.Timeout = 10 * time.Millisecond

			_, err := tika.Extract(context.TODO(), &provider.ResourceInfo{
				Type: provider.ResourceType_RESOURCE_TYPE_FILE,
				Size: 1,
			})
			Expect(err).To(MatchError(context.DeadlineExceeded))
			Expect(errors.Is(err, content.ErrUnreachable)).To(BeTrue())
		})

		Context("with a file over the content extraction size limit", func() {
			oversized := &provider.ResourceInfo{Type: provider.ResourceType_RESOURCE_TYPE_FILE, Size: 8_000_000}

			BeforeEach(func() {
				body = strings.Repeat("x", 100)
				tika.ContentExtractionSizeLimit = 10
				retriever := &contentMocks.Retriever{}
				retriever.On("Retrieve", mock.Anything, mock.Anything, mock.Anything).Return(io.NopCloser(strings.NewReader(body)), nil)
				tika.Retriever = retriever
			})

			It("extracts the metadata from the head of the file, without the content", func() {
				fullResponse = `[{"Content-Type": "image/jpeg", "tiff:Make": "Google", "tiff:ImageWidth": "4080", "tiff:ImageLength": "3072", "tk:content": "not asked for"}]`

				doc, err := tika.Extract(context.TODO(), oversized)
				Expect(err).ToNot(HaveOccurred())
				Expect(rmetaPath).To(Equal("/rmeta/ignore"))
				Expect(rmetaBytes).To(Equal(10))
				Expect(doc.Photo.CameraMake).To(Equal(libregraph.PtrString("Google")))
				Expect(doc.Image.Width).To(Equal(libregraph.PtrInt32(4080)))
				Expect(doc.Content).To(BeEmpty())
			})

			It("keeps the metadata when tika fails on the cut", func() {
				fullResponse = `[{"Content-Type": "image/jpeg", "tiff:ImageWidth": "4080", "tk:exception:container-exception": "org.apache.tika.exception.TikaException: TesseractOCRParser bad exit value 1 err msg: Premature end of JPEG file"}]`

				doc, err := tika.Extract(context.TODO(), oversized)
				Expect(err).ToNot(HaveOccurred())
				Expect(doc.Image.Width).To(Equal(libregraph.PtrInt32(4080)))
			})

			It("does not count a head tika rejects as a failure of the file", func() {
				rmetaStatus = http.StatusUnprocessableEntity

				_, err := tika.Extract(context.TODO(), oversized)
				Expect(err).ToNot(HaveOccurred())
			})

			It("still reports an unreachable tika", func() {
				rmetaStatus = http.StatusServiceUnavailable

				_, err := tika.Extract(context.TODO(), oversized)
				Expect(err).To(MatchError(content.ErrUnreachable))
			})
		})

		DescribeTable("sends only files with content to tika",
			func(info *provider.ResourceInfo, remote bool) {
				Expect(tika.Remote(info)).To(Equal(remote))
			},
			Entry("file", &provider.ResourceInfo{Type: provider.ResourceType_RESOURCE_TYPE_FILE, Size: 1}, true),
			Entry("empty file", &provider.ResourceInfo{Type: provider.ResourceType_RESOURCE_TYPE_FILE}, false),
			Entry("file over the limit", &provider.ResourceInfo{Type: provider.ResourceType_RESOURCE_TYPE_FILE, Size: math.MaxInt64}, true),
			Entry("folder", &provider.ResourceInfo{Type: provider.ResourceType_RESOURCE_TYPE_CONTAINER, Size: 1}, false),
		)

		It("probes a tika that answers its version", func() {
			Expect(tika.Probe(context.TODO())).To(Succeed())
		})

		It("probes a tika that hangs as unreachable", func() {
			DeferCleanup(content.SetProbeTimeout(10 * time.Millisecond))
			versionHangs = true

			Expect(tika.Probe(context.TODO())).To(MatchError(content.ErrUnreachable))
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

		It("takes facets from the main document, not from an embedded resource", func() {
			// metas[0] is the file (audio), metas[1] its embedded cover art. The
			// cover must not give the track an image facet.
			fullResponse = `[{"Content-Type": "audio/mpeg", "dc:title": "Sucker", "tk:content": "lyrics"}, {"Content-Type": "image/jpeg", "tiff:ImageWidth": "500"}]`

			doc, err := tika.Extract(context.TODO(), &provider.ResourceInfo{
				Type: provider.ResourceType_RESOURCE_TYPE_FILE,
				Size: 1,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.Audio).ToNot(BeNil())
			Expect(doc.Audio.Title).To(Equal(libregraph.PtrString("Sucker")))
			Expect(doc.Image).To(BeNil())
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

		It("keeps the motion photo facet when tika emits the video", func() {
			fullResponse = `[{"Camera:MotionPhoto": "1", "Camera:MotionPhotoVersion": "1"}, {"Content-Type": "video/mp4", "Content-Length": "40"}]`

			doc, err := tika.Extract(context.TODO(), &provider.ResourceInfo{
				Type: provider.ResourceType_RESOURCE_TYPE_FILE,
				Size: 100,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.MotionPhoto).ToNot(BeNil())
			Expect(doc.MotionPhoto.VideoSize).To(Equal(libregraph.PtrInt64(40)))
		})

		It("drops the motion photo facet when the advertised video is gone", func() {
			fullResponse = `[{"Camera:MotionPhoto": "1", "Camera:MotionPhotoVersion": "1"}]`

			doc, err := tika.Extract(context.TODO(), &provider.ResourceInfo{
				Type: provider.ResourceType_RESOURCE_TYPE_FILE,
				Size: 100,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.MotionPhoto).To(BeNil())
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

		DescribeTable("cuts the extracted text at the text limit",
			func(response string, limit uint64, want string) {
				fullResponse = response
				tika.ContentExtractionTextLimit = limit

				doc, err := tika.Extract(context.TODO(), &provider.ResourceInfo{
					Type: provider.ResourceType_RESOURCE_TYPE_FILE,
					Size: 1,
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(doc.Content).To(Equal(want))
			},
			Entry("text within the limit", `[{"tk:content": "one two"}]`, uint64(7), "one two"),
			Entry("no limit", `[{"tk:content": "one two"}]`, uint64(0), "one two"),
			Entry("before the cut word", `[{"tk:content": "one two three"}]`, uint64(10), "one two"),
			Entry("at a space", `[{"tk:content": "one two three"}]`, uint64(7), "one two"),
			Entry("the joined text of all meta entries", `[{"tk:content": "one two"}, {"tk:content": "three"}]`, uint64(9), "one two"),
			Entry("without splitting a character", `[{"tk:content": "grüße"}]`, uint64(3), "gr"),
			Entry("a long stretch without a space", `[{"tk:content": "a `+strings.Repeat("x", 200)+`"}]`, uint64(150), "a "+strings.Repeat("x", 148)),
		)
	})
})
