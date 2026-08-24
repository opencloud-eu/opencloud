package consumer

import (
	"testing"

	mRegistry "go-micro.dev/v4/registry"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/pkg/registry"
)

func init() {
	r := registry.GetRegistry(registry.Inmemory())
	service := registry.BuildGRPCService("eu.opencloud.api.gateway", "", "", "")
	service.Nodes = []*mRegistry.Node{{
		Address: "any",
	}}

	_ = r.Register(service)
}

func TestConsumer(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Userlog consumer Suite")
}
