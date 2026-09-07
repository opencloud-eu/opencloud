package thumbnail_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestThumbnail(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Thumbnail Suite")
}
