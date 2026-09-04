package svc

import (
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/services/webdav/pkg/generator"
)

var _ = Describe("ParseMaxInputFileSize", func() {
	It("should return 0 for empty string", func() {
		size, err := generator.ParseMaxInputFileSize("")
		Expect(err).ToNot(HaveOccurred())
		Expect(size).To(Equal(uint64(0)))
	})

	It("should parse plain bytes", func() {
		size, err := generator.ParseMaxInputFileSize("1234")
		Expect(err).ToNot(HaveOccurred())
		Expect(size).To(Equal(uint64(1234)))
	})

	It("should parse KB suffix", func() {
		size, err := generator.ParseMaxInputFileSize("50KB")
		Expect(err).ToNot(HaveOccurred())
		Expect(size).To(Equal(uint64(50 * 1024)))
	})

	It("should parse kB suffix (case insensitive)", func() {
		size, err := generator.ParseMaxInputFileSize("50kB")
		Expect(err).ToNot(HaveOccurred())
		Expect(size).To(Equal(uint64(50 * 1024)))
	})

	It("should parse KiB suffix", func() {
		size, err := generator.ParseMaxInputFileSize("50KiB")
		Expect(err).ToNot(HaveOccurred())
		Expect(size).To(Equal(uint64(50 * 1024)))
	})

	It("should parse MB suffix", func() {
		size, err := generator.ParseMaxInputFileSize("5MB")
		Expect(err).ToNot(HaveOccurred())
		Expect(size).To(Equal(uint64(5 * 1024 * 1024)))
	})

	It("should parse MiB suffix", func() {
		size, err := generator.ParseMaxInputFileSize("5MiB")
		Expect(err).ToNot(HaveOccurred())
		Expect(size).To(Equal(uint64(5 * 1024 * 1024)))
	})

	It("should parse GB suffix", func() {
		size, err := generator.ParseMaxInputFileSize("2GB")
		Expect(err).ToNot(HaveOccurred())
		Expect(size).To(Equal(uint64(2 * 1024 * 1024 * 1024)))
	})

	It("should parse GiB suffix", func() {
		size, err := generator.ParseMaxInputFileSize("2GiB")
		Expect(err).ToNot(HaveOccurred())
		Expect(size).To(Equal(uint64(2 * 1024 * 1024 * 1024)))
	})

	It("should handle leading/trailing whitespace", func() {
		size, err := generator.ParseMaxInputFileSize("  5MB  ")
		Expect(err).ToNot(HaveOccurred())
		Expect(size).To(Equal(uint64(5 * 1024 * 1024)))
	})

	It("should handle whitespace around suffix", func() {
		size, err := generator.ParseMaxInputFileSize("5 MB")
		Expect(err).ToNot(HaveOccurred())
		Expect(size).To(Equal(uint64(5 * 1024 * 1024)))
	})

	It("should return error for invalid input", func() {
		_, err := generator.ParseMaxInputFileSize("abc")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid file size"))
	})

	It("should return error for negative number", func() {
		_, err := generator.ParseMaxInputFileSize("-5MB")
		Expect(err).To(HaveOccurred())
	})
})

// testEnvVarFallback simulates go-micro's ";" separator pattern for env vars.
func testEnvVarFallback(names string) string {
	for _, n := range strings.Split(names, ";") {
		if v := os.Getenv(strings.TrimSpace(n)); v != "" {
			return v
		}
	}
	return ""
}

var _ = Describe("envVarFallback", func() {
	BeforeEach(func() {
		os.Unsetenv("WEBDAV_THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE")
		os.Unsetenv("THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE")
	})

	AfterEach(func() {
		os.Unsetenv("WEBDAV_THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE")
		os.Unsetenv("THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE")
	})

	It("should return empty string when neither env var is set", func() {
		val := testEnvVarFallback("WEBDAV_THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE;THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE")
		Expect(val).To(Equal(""))
	})

	It("should return primary env var value when only primary is set", func() {
		os.Setenv("WEBDAV_THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE", "50MB")
		val := testEnvVarFallback("WEBDAV_THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE;THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE")
		Expect(val).To(Equal("50MB"))
	})

	It("should return fallback env var value when only fallback is set", func() {
		os.Setenv("THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE", "100MB")
		val := testEnvVarFallback("WEBDAV_THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE;THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE")
		Expect(val).To(Equal("100MB"))
	})

	It("should prefer primary env var over fallback when both are set", func() {
		os.Setenv("WEBDAV_THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE", "50MB")
		os.Setenv("THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE", "100MB")
		val := testEnvVarFallback("WEBDAV_THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE;THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE")
		Expect(val).To(Equal("50MB"))
	})

	It("should parse fallback value correctly through ParseMaxInputFileSize", func() {
		os.Setenv("THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE", "2GB")
		val := testEnvVarFallback("WEBDAV_THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE;THUMBNAILS_MAX_INPUT_IMAGE_FILE_SIZE")
		size, err := generator.ParseMaxInputFileSize(val)
		Expect(err).ToNot(HaveOccurred())
		Expect(size).To(Equal(uint64(2 * 1024 * 1024 * 1024)))
	})
})

var _ = Describe("ExtensionInfo", func() {
	It("should map gif extension correctly", func() {
		info := generator.GetExtensionInfo("gif")
		Expect(info.OutputFormat).To(Equal("gif"))
		Expect(info.ContentType).To(Equal("image/gif"))
	})

	It("should map png extension correctly", func() {
		info := generator.GetExtensionInfo("png")
		Expect(info.OutputFormat).To(Equal("png"))
		Expect(info.ContentType).To(Equal("image/png"))
	})

	It("should map jpeg extension correctly", func() {
		info := generator.GetExtensionInfo("jpeg")
		Expect(info.OutputFormat).To(Equal("jpeg"))
		Expect(info.ContentType).To(Equal("image/jpeg"))
	})

	It("should map jpg to jpeg output format", func() {
		info := generator.GetExtensionInfo("jpg")
		Expect(info.OutputFormat).To(Equal("jpeg"))
		Expect(info.ContentType).To(Equal("image/jpeg"))
	})

	It("should return default info for unknown extension", func() {
		info := generator.GetExtensionInfo("webp")
		Expect(info.OutputFormat).To(Equal("jpeg"))
		Expect(info.ContentType).To(Equal("image/jpeg"))
	})

	It("should return default info for empty extension", func() {
		info := generator.GetExtensionInfo("")
		Expect(info.OutputFormat).To(Equal("jpeg"))
		Expect(info.ContentType).To(Equal("image/jpeg"))
	})
})
