// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package registry_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/pkg/registry"
	"github.com/opencloud-eu/opencloud/pkg/shared"
)

var _ = Describe("Registry config", func() {
	BeforeEach(func() {
		GinkgoT().Setenv("MICRO_REGISTRY", "")
		GinkgoT().Setenv("MICRO_REGISTRY_ADDRESS", "")
		DeferCleanup(registry.ResetConfigure)
	})

	It("defaults to the local nats-js-kv registry", func() {
		cfg := registry.GetConfig()
		Expect(cfg.Type).To(Equal("nats-js-kv"))
		Expect(cfg.Addresses).To(Equal([]string{"127.0.0.1:9233"}))
		Expect(cfg.Username).To(BeEmpty())
		Expect(cfg.Password).To(BeEmpty())
	})

	It("reads the env vars when Configure was not called", func() {
		GinkgoT().Setenv("MICRO_REGISTRY", "memory")
		GinkgoT().Setenv("MICRO_REGISTRY_ADDRESS", "nats-1:9233,nats-2:9233")

		cfg := registry.GetConfig()
		Expect(cfg.Type).To(Equal("memory"))
		Expect(cfg.Addresses).To(Equal([]string{"nats-1:9233", "nats-2:9233"}))
	})

	It("uses the configured values", func() {
		registry.Configure(shared.Registry{
			Addresses:    []string{"nats:9233"},
			AuthUsername: "user",
			AuthPassword: "secret",
		})

		cfg := registry.GetConfig()
		Expect(cfg.Type).To(Equal("nats-js-kv"))
		Expect(cfg.Addresses).To(Equal([]string{"nats:9233"}))
		Expect(cfg.Username).To(Equal("user"))
		Expect(cfg.Password).To(Equal("secret"))
	})

	It("keeps the defaults for values that are not configured", func() {
		registry.Configure(shared.Registry{})

		cfg := registry.GetConfig()
		Expect(cfg.Type).To(Equal("nats-js-kv"))
		Expect(cfg.Addresses).To(Equal([]string{"127.0.0.1:9233"}))
	})

	It("lets options override the configured values", func() {
		registry.Configure(shared.Registry{Type: "nats-js-kv"})

		Expect(registry.GetConfig(registry.Inmemory()).Type).To(Equal("memory"))
	})
})
