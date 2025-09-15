package event_test

import (
	"testing"

	"github.com/opencloud-eu/opencloud/pkg/registry"
	mRegistry "go-micro.dev/v4/registry"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func init() {
	r := registry.GetRegistry(registry.Inmemory())
	service := registry.BuildGRPCService("eu.opencloud.api.gateway", "", "", "")
	service.Nodes = []*mRegistry.Node{{
		Address: "any",
	}}

	_ = r.Register(service)
}
func TestEvent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Event Suite")
}
