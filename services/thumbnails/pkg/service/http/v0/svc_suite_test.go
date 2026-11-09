package svc

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSvc(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Thumbnails Push Endpoint Suite")
}
