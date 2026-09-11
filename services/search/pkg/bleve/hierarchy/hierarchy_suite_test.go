package hierarchy_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestHierarchy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "hierarchy tokenizer")
}
