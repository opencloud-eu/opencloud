package search_test

import (
	"context"
	"time"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	nserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

var _ = Describe("SkippedSpaces", func() {
	var (
		ctx     context.Context
		kv      jetstream.KeyValue
		skipped *search.SkippedSpaces

		spaceID = &provider.StorageSpaceId{OpaqueId: "storage$space"}
	)

	BeforeEach(func() {
		ctx = context.Background()

		srv, err := nserver.NewServer(&nserver.Options{
			DontListen: true,
			JetStream:  true,
			StoreDir:   GinkgoT().TempDir(),
			NoLog:      true,
			NoSigs:     true,
		})
		Expect(err).ToNot(HaveOccurred())
		go srv.Start()
		Expect(srv.ReadyForConnections(5 * time.Second)).To(BeTrue())
		DeferCleanup(srv.Shutdown)

		nc, err := nats.Connect("", nats.InProcessServer(srv))
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(nc.Close)

		js, err := jetstream.New(nc)
		Expect(err).ToNot(HaveOccurred())
		kv, err = js.CreateOrUpdateKeyValue(ctx, jetstream.KeyValueConfig{Bucket: search.SkippedSpacesBucket})
		Expect(err).ToNot(HaveOccurred())

		skipped = search.NewSkippedSpaces(kv)
	})

	isMarked := func(id *provider.StorageSpaceId) (bool, bool) {
		marked, force, err := skipped.IsMarked(id)
		Expect(err).ToNot(HaveOccurred())
		return marked, force
	}

	It("reports unknown spaces as not marked", func() {
		marked, force := isMarked(spaceID)
		Expect(marked).To(BeFalse())
		Expect(force).To(BeFalse())
	})

	It("marks a space for a shallow reindex", func() {
		Expect(skipped.Mark(spaceID, false)).To(Succeed())

		marked, force := isMarked(spaceID)
		Expect(marked).To(BeTrue())
		Expect(force).To(BeFalse())
	})

	It("marks a space for a forced reindex", func() {
		Expect(skipped.Mark(spaceID, true)).To(Succeed())

		marked, force := isMarked(spaceID)
		Expect(marked).To(BeTrue())
		Expect(force).To(BeTrue())
	})

	It("does not downgrade a forced mark to a shallow one", func() {
		Expect(skipped.Mark(spaceID, true)).To(Succeed())
		Expect(skipped.Mark(spaceID, false)).To(Succeed())

		_, force := isMarked(spaceID)
		Expect(force).To(BeTrue())
	})

	It("upgrades a shallow mark to a forced one", func() {
		Expect(skipped.Mark(spaceID, false)).To(Succeed())
		Expect(skipped.Mark(spaceID, true)).To(Succeed())

		_, force := isMarked(spaceID)
		Expect(force).To(BeTrue())
	})

	It("unmarks a space", func() {
		Expect(skipped.Mark(spaceID, true)).To(Succeed())
		Expect(skipped.Unmark(spaceID)).To(Succeed())

		marked, force := isMarked(spaceID)
		Expect(marked).To(BeFalse())
		Expect(force).To(BeFalse())
	})

	It("does not fail when unmarking an unknown space", func() {
		Expect(skipped.Unmark(spaceID)).To(Succeed())
	})

	It("keeps spaces apart", func() {
		other := &provider.StorageSpaceId{OpaqueId: "storage$other"}
		Expect(skipped.Mark(spaceID, true)).To(Succeed())

		marked, _ := isMarked(other)
		Expect(marked).To(BeFalse())
	})

	It("returns an error for invalid space ids", func() {
		invalid := &provider.StorageSpaceId{}
		Expect(skipped.Mark(invalid, false)).ToNot(Succeed())
		Expect(skipped.Unmark(invalid)).ToNot(Succeed())
		_, _, err := skipped.IsMarked(invalid)
		Expect(err).To(HaveOccurred())
	})

	DescribeTable("is a no-op without a bucket",
		func(s *search.SkippedSpaces) {
			Expect(s.Mark(spaceID, true)).To(Succeed())
			Expect(s.Unmark(spaceID)).To(Succeed())
			marked, force, err := s.IsMarked(spaceID)
			Expect(err).ToNot(HaveOccurred())
			Expect(marked).To(BeFalse())
			Expect(force).To(BeFalse())
		},
		Entry("nil bucket", search.NewSkippedSpaces(nil)),
		Entry("nil tracker", (*search.SkippedSpaces)(nil)),
	)
})
