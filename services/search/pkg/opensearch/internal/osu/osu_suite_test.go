package osu_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestOSUGeo(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OSU Geo Suite")
}
