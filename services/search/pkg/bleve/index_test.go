package bleve_test

import (
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/services/search/pkg/bleve"
)

var _ = Describe("Index", func() {
	Describe("NewIndex", func() {
		It("puts the index into a directory of its own generation", func() {
			root := GinkgoT().TempDir()

			index, err := bleve.NewIndex(root)
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(index.Close)

			Expect(index.Name()).To(Equal(filepath.Join(root, "bleve-v2")))
			Expect(filepath.Join(root, "bleve-v2")).To(BeADirectory())
		})

		It("opens the index that is already there", func() {
			root := GinkgoT().TempDir()

			index, err := bleve.NewIndex(root)
			Expect(err).ToNot(HaveOccurred())
			Expect(index.Close()).To(Succeed())

			reopened, err := bleve.NewIndex(root)
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(reopened.Close)

			Expect(reopened.Name()).To(Equal(filepath.Join(root, "bleve-v2")))
		})
	})
})
