package service_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestV0(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "V0 Suite")
}
