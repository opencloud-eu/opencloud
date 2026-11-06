// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package natsjsregistry_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go-micro.dev/v4/registry"

	"github.com/opencloud-eu/opencloud/pkg/natsjsregistry"
)

var _ = Describe("NewRegistryMicro", func() {
	BeforeEach(func() {
		GinkgoT().Setenv("MICRO_REGISTRY_ADDRESS", "")
		GinkgoT().Setenv("MICRO_REGISTRY_AUTH_USERNAME", "env-user")
		GinkgoT().Setenv("MICRO_REGISTRY_AUTH_PASSWORD", "env-secret")
		DeferCleanup(natsjsregistry.SetMicroOptions)
	})

	It("reads the auth from the env vars without micro options", func() {
		reg := natsjsregistry.NewRegistryMicro()

		user, password := natsjsregistry.GetAuth(reg.Options())
		Expect(user).To(Equal("env-user"))
		Expect(password).To(Equal("env-secret"))
	})

	It("uses the micro options", func() {
		natsjsregistry.SetMicroOptions(
			registry.Addrs("nats:9233"),
			natsjsregistry.Authenticate("user", "secret"),
		)

		reg := natsjsregistry.NewRegistryMicro()

		Expect(reg.Options().Addrs).To(Equal([]string{"nats:9233"}))
		user, password := natsjsregistry.GetAuth(reg.Options())
		Expect(user).To(Equal("user"))
		Expect(password).To(Equal("secret"))
	})

	It("prefers the options passed to it", func() {
		natsjsregistry.SetMicroOptions(registry.Addrs("nats:9233"))

		reg := natsjsregistry.NewRegistryMicro(registry.Addrs("other:9233"))

		Expect(reg.Options().Addrs).To(Equal([]string{"other:9233"}))
	})
})
