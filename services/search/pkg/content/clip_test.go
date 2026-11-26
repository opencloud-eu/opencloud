package content_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/search/pkg/clip"
	conf "github.com/opencloud-eu/opencloud/services/search/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/services/search/pkg/content"
	contentMocks "github.com/opencloud-eu/opencloud/services/search/pkg/content/mocks"
)

// clipResponse renders the immich-ml /predict response for a vector of n dims.
func clipResponse(n int) string {
	vec := make([]float32, n)
	for i := range vec {
		vec[i] = float32(i) / float32(n)
	}
	inner, _ := json.Marshal(vec)
	outer, _ := json.Marshal(map[string]string{"clip": string(inner)})
	return string(outer)
}

var _ = Describe("Clip", func() {
	var (
		srv        *httptest.Server
		dims       int
		failNext   atomic.Bool
		imageCalls atomic.Int32
		inner      *contentMocks.Extractor
		retriever  *contentMocks.Retriever
	)

	BeforeEach(func() {
		dims = content.ImageVectorDims
		failNext.Store(false)
		imageCalls.Store(0)
		srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if req.URL.Path != "/predict" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			if failNext.Load() {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			Expect(req.ParseMultipartForm(1 << 20)).To(Succeed())
			if _, _, err := req.FormFile("image"); err == nil {
				imageCalls.Add(1)
			}
			_, _ = w.Write([]byte(clipResponse(dims)))
		}))

		inner = &contentMocks.Extractor{}
		retriever = &contentMocks.Retriever{}
	})

	AfterEach(func() {
		srv.Close()
	})

	newClip := func() (*content.Clip, error) {
		cfg := conf.DefaultConfig()
		cfg.Extractor.Clip.URL = srv.URL

		client := clip.NewClient(srv.URL, cfg.Extractor.Clip.Model, 0)
		extractor, err := content.NewClipExtractor(inner, client, nil, log.NewLogger(), cfg)
		if err != nil {
			return nil, err
		}
		extractor.Retriever = retriever
		return extractor, nil
	}

	imageResource := func(size uint64) *provider.ResourceInfo {
		return &provider.ResourceInfo{
			Type:     provider.ResourceType_RESOURCE_TYPE_FILE,
			Name:     "photo.jpg",
			MimeType: "image/jpeg",
			Size:     size,
		}
	}

	Describe("NewClipExtractor", func() {
		It("fails when the inference service is unreachable", func() {
			srv.Close()
			_, err := newClip()
			Expect(err).To(HaveOccurred())
		})

		It("fails when the model dimensionality does not match the schema", func() {
			dims = 768
			_, err := newClip()
			Expect(err).To(MatchError(ContainSubstring("768")))
		})
	})

	Describe("Extract", func() {
		var extractor *content.Clip

		BeforeEach(func() {
			var err error
			extractor, err = newClip()
			Expect(err).ToNot(HaveOccurred())

			inner.On("Extract", mock.Anything, mock.Anything).Return(content.Document{Name: "photo.jpg"}, nil)
			retriever.On("Retrieve", mock.Anything, mock.Anything).Return(io.NopCloser(strings.NewReader("fakeimagebytes")), nil)
		})

		It("adds a vector to image documents", func() {
			doc, err := extractor.Extract(context.TODO(), imageResource(1024))
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.Name).To(Equal("photo.jpg"))
			Expect(doc.ImageVector).To(HaveLen(content.ImageVectorDims))
			Expect(imageCalls.Load()).To(Equal(int32(1)))
		})

		It("skips non-image files", func() {
			doc, err := extractor.Extract(context.TODO(), &provider.ResourceInfo{
				Type:     provider.ResourceType_RESOURCE_TYPE_FILE,
				Name:     "doc.pdf",
				MimeType: "application/pdf",
				Size:     1024,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.ImageVector).To(BeNil())
			Expect(imageCalls.Load()).To(Equal(int32(0)))
		})

		It("skips directories", func() {
			doc, err := extractor.Extract(context.TODO(), &provider.ResourceInfo{
				Type:     provider.ResourceType_RESOURCE_TYPE_CONTAINER,
				MimeType: "image/jpeg",
				Size:     1024,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.ImageVector).To(BeNil())
		})

		It("skips images above the size limit", func() {
			doc, err := extractor.Extract(context.TODO(), imageResource(51*1024*1024))
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.ImageVector).To(BeNil())
			Expect(imageCalls.Load()).To(Equal(int32(0)))
		})

		It("indexes without a vector when inference fails", func() {
			failNext.Store(true)
			doc, err := extractor.Extract(context.TODO(), imageResource(1024))
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.Name).To(Equal("photo.jpg"))
			Expect(doc.ImageVector).To(BeNil())
		})

		It("indexes without a vector when retrieval fails", func() {
			failingRetriever := &contentMocks.Retriever{}
			failingRetriever.On("Retrieve", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("boom"))
			extractor.Retriever = failingRetriever

			doc, err := extractor.Extract(context.TODO(), imageResource(1024))
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.ImageVector).To(BeNil())
		})

		It("propagates inner extractor errors", func() {
			failingInner := &contentMocks.Extractor{}
			failingInner.On("Extract", mock.Anything, mock.Anything).Return(content.Document{}, fmt.Errorf("inner boom"))
			cfg := conf.DefaultConfig()
			cfg.Extractor.Clip.URL = srv.URL
			client := clip.NewClient(srv.URL, cfg.Extractor.Clip.Model, 0)
			failing, err := content.NewClipExtractor(failingInner, client, nil, log.NewLogger(), cfg)
			Expect(err).ToNot(HaveOccurred())
			failing.Retriever = retriever

			_, err = failing.Extract(context.TODO(), imageResource(1024))
			Expect(err).To(MatchError(ContainSubstring("inner boom")))
		})
	})
})
