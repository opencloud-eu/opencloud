package kql_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestGeoKQL(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "KQL Geo Suite")
}
