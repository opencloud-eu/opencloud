package announcement_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAnnouncement(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Announcement Suite")
}
