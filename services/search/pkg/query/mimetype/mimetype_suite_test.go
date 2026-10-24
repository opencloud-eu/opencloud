package mimetype_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMimetype(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Mimetype Suite")
}
