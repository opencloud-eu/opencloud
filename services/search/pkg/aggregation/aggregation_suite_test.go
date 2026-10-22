package aggregation_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAggregation(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Aggregation Suite")
}
