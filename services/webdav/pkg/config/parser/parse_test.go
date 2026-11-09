// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package parser_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	thumbnailsdefaults "github.com/opencloud-eu/opencloud/services/thumbnails/pkg/config/defaults"
	thumbnailsparser "github.com/opencloud-eu/opencloud/services/thumbnails/pkg/config/parser"
	"github.com/opencloud-eu/opencloud/services/webdav/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/services/webdav/pkg/config/parser"
)

var _ = Describe("ParseConfig", func() {
	BeforeEach(func() {
		GinkgoT().Setenv("OC_CONFIG_DIR", GinkgoT().TempDir())
	})

	// a body webdav allows and the generator refuses is downloaded and posted for nothing
	DescribeTable("reads the maximum input file size the way the thumbnails service does",
		func(env map[string]string, want string) {
			for k, v := range env {
				GinkgoT().Setenv(k, v)
			}

			cfg := defaults.DefaultConfig()
			Expect(parser.ParseConfig(cfg)).To(Succeed())

			generator := thumbnailsdefaults.DefaultConfig()
			Expect(thumbnailsparser.ParseConfig(generator)).To(Succeed())

			Expect(cfg.Thumbnails.MaxInputFileSize).To(Equal(want))
			Expect(generator.Thumbnail.MaxInputImageFileSize).To(Equal(want))
		},
		Entry("from the shared name", map[string]string{
			"OC_THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE": "100MB",
		}, "100MB"),
		Entry("from the old name", map[string]string{
			"THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE": "20MB",
		}, "20MB"),
		Entry("from the old name when both are set", map[string]string{
			"THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE":    "20MB",
			"OC_THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE": "100MB",
		}, "20MB"),
	)
})
