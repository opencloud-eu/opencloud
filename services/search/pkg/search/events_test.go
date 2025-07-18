package search_test

import (
	"sync/atomic"

	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/search/pkg/config"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
	searchMocks "github.com/opencloud-eu/opencloud/services/search/pkg/search/mocks"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/events/raw"
	rawMocks "github.com/opencloud-eu/reva/v2/pkg/events/raw/mocks"
	"github.com/stretchr/testify/mock"
)

var _ = DescribeTable("events",
	func(mcks []string, e any, asyncUploads bool) {
		var (
			s     = &searchMocks.Searcher{}
			calls atomic.Int32
		)

		stream := rawMocks.NewStream(GinkgoT())
		ch := make(chan raw.Event, 1)
		stream.EXPECT().Consume(mock.Anything, mock.Anything).Return((<-chan raw.Event)(ch), nil)

		search.HandleEvents(s, stream, &config.Config{
			Events: config.Events{
				AsyncUploads: asyncUploads,
			},
		}, nil, log.NewLogger())

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
	Entry("ItemTrashed", []string{"TrashItem", "IndexSpace"}, events.ItemTrashed{}, false),
	Entry("ItemMoved", []string{"MoveItem", "IndexSpace"}, events.ItemMoved{}, false),
	Entry("ItemRestored", []string{"RestoreItem", "IndexSpace"}, events.ItemRestored{}, false),
	Entry("ContainerCreated", []string{"IndexSpace"}, events.ContainerCreated{}, false),
	Entry("FileTouched", []string{"IndexSpace"}, events.FileTouched{}, false),
	Entry("FileVersionRestored", []string{"IndexSpace"}, events.FileVersionRestored{}, false),
	Entry("TagsAdded", []string{"UpsertItem"}, events.TagsAdded{}, false),
	Entry("TagsRemoved", []string{"UpsertItem"}, events.TagsRemoved{}, false),
	Entry("FileUploaded", []string{"IndexSpace"}, events.FileUploaded{}, false),
	Entry("UploadReady", []string{"IndexSpace"}, events.UploadReady{ExecutingUser: &userv1beta1.User{}}, true),
)
