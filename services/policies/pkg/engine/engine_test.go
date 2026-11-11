package engine_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	pMessage "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/policies/v0"
	"github.com/opencloud-eu/opencloud/services/policies/pkg/engine"
)

var _ = Describe("Engine", func() {
	DescribeTable("NewEnvironmentFromPB",
		func(incomingStage pMessage.Stage, outgoinStage engine.Stage) {
			pEnv := &pMessage.Environment{
				Stage: incomingStage,
			}

			env, err := engine.NewEnvironmentFromPB(pEnv)
			Expect(err).ToNot(HaveOccurred())

			Expect(env.Stage).To(Equal(outgoinStage))
		},
		Entry("http stage", pMessage.Stage_STAGE_HTTP, engine.StageHTTP),
		Entry("pp stage", pMessage.Stage_STAGE_PP, engine.StagePP),
	)

	It("NewEnvironmentFromPB keeps the request lengths", func() {
		pEnv := &pMessage.Environment{
			Stage: pMessage.Stage_STAGE_HTTP,
			Request: &pMessage.Request{
				Method:        "POST",
				Path:          "/remote.php/dav/spaces/some-space-id",
				ContentLength: 1024,
				UploadLength:  20000000000,
			},
		}

		env, err := engine.NewEnvironmentFromPB(pEnv)
		Expect(err).ToNot(HaveOccurred())

		Expect(env.Request.Method).To(Equal("POST"))
		Expect(env.Request.ContentLength).To(Equal(uint64(1024)))
		Expect(env.Request.UploadLength).To(Equal(uint64(20000000000)))
	})
})
