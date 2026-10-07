package svc

import (
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/webdav/pkg/dav/requests"
	"github.com/opencloud-eu/opencloud/services/webdav/pkg/thumbnail/workflow"
)

var _ = Describe("thumbnail error mapping", func() {
	var (
		g      Webdav
		tr     *requests.ThumbnailRequest
		logger log.Logger
	)

	BeforeEach(func() {
		logger = log.NopLogger()
		g = Webdav{log: logger}
		tr = &requests.ThumbnailRequest{Filename: "song.mp3"}
	})

	record := func(handle func(http.ResponseWriter, *http.Request, error, *requests.ThumbnailRequest, log.Logger), err error) int {
		rr := httptest.NewRecorder()
		handle(rr, httptest.NewRequest(http.MethodGet, "/song.mp3", nil), err, tr, logger)
		return rr.Code
	}

	DescribeTable("a file without a thumbnail is not a server error",
		func(handle func(http.ResponseWriter, *http.Request, error, *requests.ThumbnailRequest, log.Logger)) {
			// clients cache a 404 as "no preview"; a 500 makes them retry forever
			Expect(record(handle, workflow.ErrNoThumbnail)).To(Equal(http.StatusNotFound))
		},
		Entry("on GET", func(w http.ResponseWriter, r *http.Request, err error, tr *requests.ThumbnailRequest, l log.Logger) {
			g.handleWorkflowError(w, r, err, tr, l)
		}),
		Entry("on HEAD", func(w http.ResponseWriter, r *http.Request, err error, tr *requests.ThumbnailRequest, l log.Logger) {
			g.handleHeadError(w, r, err, tr, l)
		}),
	)
})
