package aggs_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAggs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Aggs Suite")
}
