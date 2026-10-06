package filtertoken_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestFilterToken(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Filter Token Suite")
}
