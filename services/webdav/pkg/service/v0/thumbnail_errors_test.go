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

	DescribeTable("a file without a thumbnail is not a server error",
		func(step string) {
			// clients cache a 404 as "no preview"; a 500 makes them retry forever
			rr := httptest.NewRecorder()
			g.handleWorkflowError(rr, httptest.NewRequest(http.MethodGet, "/song.mp3", nil), workflow.ErrNoThumbnail, tr, logger, step)

			Expect(rr.Code).To(Equal(http.StatusNotFound))
		},
		Entry("on GET", "thumbnail"),
		Entry("on HEAD", "thumbnail head check"),
	)
})
