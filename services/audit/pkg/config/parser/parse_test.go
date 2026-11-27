// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package parser_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/services/audit/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/services/audit/pkg/config/parser"
)

var _ = Describe("ParseConfig", func() {
	BeforeEach(func() {
		GinkgoT().Setenv("OC_CONFIG_DIR", GinkgoT().TempDir())
	})

	// an installation that only adds the audit service keeps its legacy audit log after an upgrade,
	// the variable that makes the services emit audit events also switches to the audit stream
	DescribeTable("switches between the legacy audit log and the audit stream",
		func(env map[string]string, stream bool) {
			for k, v := range env {
				GinkgoT().Setenv(k, v)
			}

			cfg := defaults.DefaultConfig()
			Expect(parser.ParseConfig(cfg)).To(Succeed())
			Expect(cfg.Stream.Enabled).To(Equal(stream))
		},
		Entry("legacy audit log by default", map[string]string{}, false),
		Entry("audit stream with OC_AUDIT_ENABLED", map[string]string{
			"OC_AUDIT_ENABLED": "true",
		}, true),
		Entry("audit stream with AUDIT_STREAM_ENABLED", map[string]string{
			"AUDIT_STREAM_ENABLED": "true",
		}, true),
		Entry("AUDIT_STREAM_ENABLED wins when both are set", map[string]string{
			"OC_AUDIT_ENABLED":     "true",
			"AUDIT_STREAM_ENABLED": "false",
		}, false),
	)
})
