package activitylog_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestActivitylog(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Activitylog Suite")
}
