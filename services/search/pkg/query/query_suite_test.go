package query_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestQuery(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Query Suite")
}
