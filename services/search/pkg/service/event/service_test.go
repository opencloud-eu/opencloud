package event_test

import (
	"context"
	"slices"
	"sync"
	"sync/atomic"

	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
	searchMocks "github.com/opencloud-eu/opencloud/services/search/pkg/search/mocks"
	"github.com/opencloud-eu/opencloud/services/search/pkg/service/event"
	"github.com/opencloud-eu/reva/v3/pkg/events"
	"github.com/opencloud-eu/reva/v3/pkg/events/raw"
	rawMocks "github.com/opencloud-eu/reva/v3/pkg/events/raw/mocks"
	"github.com/stretchr/testify/mock"
)

var _ = DescribeTable("event",
	func(mcks []string, e any, asyncUploads bool) {
		var (
			s     = &searchMocks.Searcher{}
			calls atomic.Int32
		)

		stream := rawMocks.NewStream(GinkgoT())
		ch := make(chan raw.Event, 1)
		stream.EXPECT().Consume(mock.Anything, mock.Anything).Return((<-chan raw.Event)(ch), nil)

		event, err := event.New(context.Background(), stream, log.NewLogger(), nil, nil, s, search.NewSkippedSpaces(nil), 50, 1, asyncUploads)
		Expect(err).NotTo(HaveOccurred())

		go func() {
			err := event.Run()
			Expect(err).NotTo(HaveOccurred())
		}()
		defer event.Close()

		for _, mck := range mcks {
			s.On(mck, mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
				calls.Add(1)
			})
		}

		ch <- raw.Event{
			Event: events.Event{Event: e},
		}

		Eventually(func() int {
			return int(calls.Load())
		}, "2s").Should(Equal(len(mcks)))
	},
	Entry("SpaceDeleted", []string{"PurgeSpace"}, events.SpaceDeleted{}, false),
	Entry("ItemPurged", []string{"PurgeItem"}, events.ItemPurged{}, false),
	Entry("TrashbinPurged", []string{"PurgeDeleted"}, events.TrashbinPurged{}, false),
	Entry("LabelAdded", []string{"UpsertItem"}, events.LabelAdded{}, false),
	Entry("LabelRemoved", []string{"UpsertItem"}, events.LabelRemoved{}, false),
	Entry("ItemTrashed", []string{"TrashItem", "IndexSpace"}, events.ItemTrashed{}, false),
	Entry("ItemMoved", []string{"MoveItem", "IndexSpace"}, events.ItemMoved{}, false),
	Entry("ItemRestored", []string{"RestoreItem", "IndexSpace"}, events.ItemRestored{}, false),
	Entry("ContainerCreated", []string{"IndexSpace"}, events.ContainerCreated{}, false),
	Entry("FileTouched", []string{"IndexSpace"}, events.FileTouched{}, false),
	Entry("FileVersionRestored", []string{"IndexSpace"}, events.FileVersionRestored{}, false),
	Entry("TagsAdded", []string{"UpsertItem", "IndexSpace"}, events.TagsAdded{}, false),
	Entry("TagsRemoved", []string{"UpsertItem", "IndexSpace"}, events.TagsRemoved{}, false),
	Entry("FileUploaded", []string{"IndexSpace"}, events.FileUploaded{}, false),
	Entry("UploadReady", []string{"IndexSpace"}, events.UploadReady{ExecutingUser: &userv1beta1.User{}}, true),
)

var _ = It("does not index the space of a failed upload", func() {
	var (
		s       = &searchMocks.Searcher{}
		mu      sync.Mutex
		indexed []string
	)
	indexedSpaces := func() []string {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(indexed)
	}

	stream := rawMocks.NewStream(GinkgoT())
	ch := make(chan raw.Event, 2)
	stream.EXPECT().Consume(mock.Anything, mock.Anything).Return((<-chan raw.Event)(ch), nil)

	svc, err := event.New(context.Background(), stream, log.NewLogger(), nil, nil, s, search.NewSkippedSpaces(nil), 50, 1, true)
	Expect(err).NotTo(HaveOccurred())
	go func() {
		defer GinkgoRecover()
		Expect(svc.Run()).To(Succeed())
	}()
	defer svc.Close()

	s.On("IndexSpace", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		mu.Lock()
		defer mu.Unlock()
		indexed = append(indexed, args.Get(0).(*provider.StorageSpaceId).GetOpaqueId())
	})

	ref := func(spaceID string) *provider.Reference {
		return &provider.Reference{ResourceId: &provider.ResourceId{StorageId: "storageid", SpaceId: spaceID}}
	}
	// the successful upload of another space marks when the failed one would have been indexed
	ch <- raw.Event{Event: events.Event{Event: events.UploadReady{ExecutingUser: &userv1beta1.User{}, FileRef: ref("failed"), Failed: true}}}
	ch <- raw.Event{Event: events.Event{Event: events.UploadReady{ExecutingUser: &userv1beta1.User{}, FileRef: ref("succeeded")}}}

	Eventually(indexedSpaces, "2s").Should(ContainElement("storageid$succeeded"))
	Consistently(indexedSpaces, "200ms").Should(Equal([]string{"storageid$succeeded"}))
})
