package consumer

import (
	"context"

	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/opencloud-eu/opencloud/pkg/log"
	settingsmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/settings/v0"
	settings "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0"
	settingsmocks "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0/mocks"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("NotificationFilter", func() {
	var (
		testLogger = log.NewLogger()
		vs         *settingsmocks.ValueService
		ulf        userlogFilter
	)

	ginkgo.BeforeEach(func() {
		vs = &settingsmocks.ValueService{}
		ulf = userlogFilter{
			log:         testLogger,
			valueClient: vs,
		}
	})

	setupMockValueService := func(inApp bool) *settingsmocks.ValueService {
		vs := settingsmocks.ValueService{}
		vs.On("GetValueByUniqueIdentifiers", mock.Anything, mock.Anything).Return(&settings.GetValueResponse{
			Value: &settingsmsg.ValueWithIdentifier{
				Value: &settingsmsg.Value{
					Value: &settingsmsg.Value_CollectionValue{
						CollectionValue: &settingsmsg.CollectionValue{
							Values: []*settingsmsg.CollectionOption{
								{
									Key:    "in-app",
									Option: &settingsmsg.CollectionOption_BoolValue{BoolValue: inApp},
								},
							},
						},
					},
				},
			},
		}, nil)
		return &vs
	}

	ginkgo.Describe("execute", func() {
		ginkgo.It("handles executants", func() {
			vs.On("GetValueByUniqueIdentifiers", mock.Anything, mock.Anything).Return(nil, nil)

			gomega.Expect(ulf.execute(context.TODO(), events.Event{}, &user.UserId{OpaqueId: "executant"}, []string{"foo"})).To(gomega.ConsistOf("foo"))
		})
		ginkgo.It("handles connection errors", func() {
			vs.On("GetValueByUniqueIdentifiers", mock.Anything, mock.Anything).Return(nil, errors.New("no connection to ValueService"))

			gomega.Expect(ulf.execute(context.TODO(), events.Event{Event: events.ShareCreated{}}, nil, []string{"foo"})).To(gomega.BeEmpty())
		})
		ginkgo.It("handles no setting", func() {
			vs.On("GetValueByUniqueIdentifiers", mock.Anything, mock.Anything).Return(&settings.GetValueResponse{}, nil)

			gomega.Expect(ulf.execute(context.TODO(), events.Event{Event: events.ShareCreated{}}, nil, []string{"foo"})).To(gomega.BeEmpty())
		})
		ginkgo.It("handles nil response", func() {
			vs.On("GetValueByUniqueIdentifiers", mock.Anything, mock.Anything).Return(nil, nil)

			gomega.Expect(ulf.execute(context.TODO(), events.Event{Event: events.ShareCreated{}}, nil, []string{"foo"})).To(gomega.BeEmpty())
		})
		ginkgo.It("handles events that can not be disabled", func() {
			ulf.valueClient = setupMockValueService(true)

			gomega.Expect(ulf.execute(context.TODO(), events.Event{Event: events.BytesReceived{}}, nil, []string{"foo"})).To(gomega.ConsistOf("foo"))
		})

		ginkgo.It("handles ShareCreated events", func() {
			ulf.valueClient = setupMockValueService(true)

			gomega.Expect(ulf.execute(context.TODO(), events.Event{Event: events.ShareCreated{}}, nil, []string{"foo"})).To(gomega.ConsistOf("foo"))
		})

		ginkgo.It("handles ShareRemoved events", func() {
			ulf.valueClient = setupMockValueService(true)

			gomega.Expect(ulf.execute(context.TODO(), events.Event{Event: events.ShareRemoved{}}, nil, []string{"foo"})).To(gomega.ConsistOf("foo"))
		})

		ginkgo.It("handles ShareExpired events", func() {
			ulf.valueClient = setupMockValueService(true)

			gomega.Expect(ulf.execute(context.TODO(), events.Event{Event: events.ShareExpired{}}, nil, []string{"foo"})).To(gomega.ConsistOf("foo"))
		})

		ginkgo.It("handles SpaceShared enabled", func() {
			ulf.valueClient = setupMockValueService(true)

			gomega.Expect(ulf.execute(context.TODO(), events.Event{Event: events.SpaceShared{}}, nil, []string{"foo"})).To(gomega.ConsistOf("foo"))
		})

		ginkgo.It("handles SpaceUnshared enabled", func() {
			ulf.valueClient = setupMockValueService(true)

			gomega.Expect(ulf.execute(context.TODO(), events.Event{Event: events.SpaceUnshared{}}, nil, []string{"foo"})).To(gomega.ConsistOf("foo"))
		})

		ginkgo.It("handles SpaceMembershipExpired enabled", func() {
			ulf.valueClient = setupMockValueService(true)

			gomega.Expect(ulf.execute(context.TODO(), events.Event{Event: events.SpaceMembershipExpired{}}, nil, []string{"foo"})).To(gomega.ConsistOf("foo"))
		})

		ginkgo.It("handles SpaceDisabled enabled", func() {
			ulf.valueClient = setupMockValueService(true)

			gomega.Expect(ulf.execute(context.TODO(), events.Event{Event: events.SpaceDisabled{}}, nil, []string{"foo"})).To(gomega.ConsistOf("foo"))
		})

		ginkgo.It("handles SpaceDeleted enabled", func() {
			ulf.valueClient = setupMockValueService(true)

			gomega.Expect(ulf.execute(context.TODO(), events.Event{Event: events.SpaceDeleted{}}, nil, []string{"foo"})).To(gomega.ConsistOf("foo"))
		})

		ginkgo.It("handles ShareCreated disabled", func() {
			ulf.valueClient = setupMockValueService(false)

			gomega.Expect(ulf.execute(context.TODO(), events.Event{Event: events.ShareCreated{}}, nil, []string{"foo"})).To(gomega.BeEmpty())
		})

		ginkgo.It("handles ShareRemoved disabled", func() {
			ulf.valueClient = setupMockValueService(false)

			gomega.Expect(ulf.execute(context.TODO(), events.Event{Event: events.ShareRemoved{}}, nil, []string{"foo"})).To(gomega.BeEmpty())
		})

		ginkgo.It("handles ShareExpired disabled", func() {
			ulf.valueClient = setupMockValueService(false)

			gomega.Expect(ulf.execute(context.TODO(), events.Event{Event: events.ShareExpired{}}, nil, []string{"foo"})).To(gomega.BeEmpty())
		})

		ginkgo.It("handles SpaceShared disabled", func() {
			ulf.valueClient = setupMockValueService(false)

			gomega.Expect(ulf.execute(context.TODO(), events.Event{Event: events.SpaceShared{}}, nil, []string{"foo"})).To(gomega.BeEmpty())
		})

		ginkgo.It("handles SpaceUnshared disabled", func() {
			ulf.valueClient = setupMockValueService(false)

			gomega.Expect(ulf.execute(context.TODO(), events.Event{Event: events.SpaceUnshared{}}, nil, []string{"foo"})).To(gomega.BeEmpty())
		})

		ginkgo.It("handles SpaceMembershipExpired disabled", func() {
			ulf.valueClient = setupMockValueService(false)

			gomega.Expect(ulf.execute(context.TODO(), events.Event{Event: events.SpaceMembershipExpired{}}, nil, []string{"foo"})).To(gomega.BeEmpty())
		})

		ginkgo.It("handles SpaceDisabled disabled", func() {
			ulf.valueClient = setupMockValueService(false)

			gomega.Expect(ulf.execute(context.TODO(), events.Event{Event: events.SpaceDisabled{}}, nil, []string{"foo"})).To(gomega.BeEmpty())
		})

		ginkgo.It("handles SpaceDeleted disabled", func() {
			ulf.valueClient = setupMockValueService(false)

			gomega.Expect(ulf.execute(context.TODO(), events.Event{Event: events.SpaceDeleted{}}, nil, []string{"foo"})).To(gomega.BeEmpty())
		})
	})
})
