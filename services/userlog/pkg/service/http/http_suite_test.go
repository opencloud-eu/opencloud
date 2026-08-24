package http_test

import (
	"testing"

	mRegistry "go-micro.dev/v4/registry"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

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

func TestHttp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Userlog http Suite")
}
