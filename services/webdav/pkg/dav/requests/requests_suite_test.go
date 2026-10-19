package requests_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRequests(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Requests Suite")
}
