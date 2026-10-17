package parity

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/services/thumbnails/internal/imagortest"
)

// the container handles stay in process 1.
var (
	imagorURLs   = map[string]string{}
	imagorErrors = map[string]string{}
	stopImagors  []func()
)

// Only one direction is a contract: imagor must be able to replace the built-in
// generator. Where imagor accepts what the built-in rejects, nothing has to change.
func TestGeneratorParity(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Thumbnail Generator Parity Suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	found := map[string]string{}
	for _, v := range imagortest.Variants() {
		url, done, err := imagortest.Setup(context.Background(), v)
		if err != nil {
			// without docker the built-in side still runs, imagor's specs skip.
			found[v.Name] = "!" + err.Error()

			continue
		}
		stopImagors = append(stopImagors, done)
		found[v.Name] = url
	}

	payload, err := json.Marshal(found)
	Expect(err).ToNot(HaveOccurred())

	return payload
}, func(payload []byte) {
	found := map[string]string{}
	Expect(json.Unmarshal(payload, &found)).To(Succeed())

	for name, value := range found {
		if reason, missing := strings.CutPrefix(value, "!"); missing {
			imagorErrors[name] = reason

			continue
		}
		imagorURLs[name] = value
	}
})

var _ = SynchronizedAfterSuite(func() {}, func() {
	for _, stop := range stopImagors {
		stop()
	}
})
