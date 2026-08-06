package service

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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("NotificationFilter", func() {
	var (
		testLogger = log.NewLogger()
		vs         *settingsmocks.ValueService
		ulf        userlogFilter
	)

	BeforeEach(func() {
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

	Describe("execute", func() {
		It("handles executants", func() {
			vs.On("GetValueByUniqueIdentifiers", mock.Anything, mock.Anything).Return(nil, nil)

			Expect(ulf.execute(context.TODO(), events.Event{}, &user.UserId{OpaqueId: "executant"}, []string{"foo"})).To(ConsistOf("foo"))
		})
		It("handles connection errors", func() {
			vs.On("GetValueByUniqueIdentifiers", mock.Anything, mock.Anything).Return(nil, errors.New("no connection to ValueService"))

			Expect(ulf.execute(context.TODO(), events.Event{Event: events.ShareCreated{}}, nil, []string{"foo"})).To(BeEmpty())
		})
		It("handles no setting", func() {
			vs.On("GetValueByUniqueIdentifiers", mock.Anything, mock.Anything).Return(&settings.GetValueResponse{}, nil)

			Expect(ulf.execute(context.TODO(), events.Event{Event: events.ShareCreated{}}, nil, []string{"foo"})).To(BeEmpty())
		})
		It("handles nil response", func() {
			vs.On("GetValueByUniqueIdentifiers", mock.Anything, mock.Anything).Return(nil, nil)

			Expect(ulf.execute(context.TODO(), events.Event{Event: events.ShareCreated{}}, nil, []string{"foo"})).To(BeEmpty())
		})
		It("handles events that can not be disabled", func() {
			ulf.valueClient = setupMockValueService(true)

			Expect(ulf.execute(context.TODO(), events.Event{Event: events.BytesReceived{}}, nil, []string{"foo"})).To(ConsistOf("foo"))
		})

		It("handles ShareCreated events", func() {
			ulf.valueClient = setupMockValueService(true)

			Expect(ulf.execute(context.TODO(), events.Event{Event: events.ShareCreated{}}, nil, []string{"foo"})).To(ConsistOf("foo"))
		})

		It("handles ShareRemoved events", func() {
			ulf.valueClient = setupMockValueService(true)

			Expect(ulf.execute(context.TODO(), events.Event{Event: events.ShareRemoved{}}, nil, []string{"foo"})).To(ConsistOf("foo"))
		})

		It("handles ShareExpired events", func() {
			ulf.valueClient = setupMockValueService(true)

			Expect(ulf.execute(context.TODO(), events.Event{Event: events.ShareExpired{}}, nil, []string{"foo"})).To(ConsistOf("foo"))
		})

		It("handles SpaceShared enabled", func() {
			ulf.valueClient = setupMockValueService(true)

			Expect(ulf.execute(context.TODO(), events.Event{Event: events.SpaceShared{}}, nil, []string{"foo"})).To(ConsistOf("foo"))
		})

		It("handles SpaceUnshared enabled", func() {
			ulf.valueClient = setupMockValueService(true)

			Expect(ulf.execute(context.TODO(), events.Event{Event: events.SpaceUnshared{}}, nil, []string{"foo"})).To(ConsistOf("foo"))
		})

		It("handles SpaceMembershipExpired enabled", func() {
			ulf.valueClient = setupMockValueService(true)

			Expect(ulf.execute(context.TODO(), events.Event{Event: events.SpaceMembershipExpired{}}, nil, []string{"foo"})).To(ConsistOf("foo"))
		})

		It("handles SpaceDisabled enabled", func() {
			ulf.valueClient = setupMockValueService(true)

			Expect(ulf.execute(context.TODO(), events.Event{Event: events.SpaceDisabled{}}, nil, []string{"foo"})).To(ConsistOf("foo"))
		})

		It("handles SpaceDeleted enabled", func() {
			ulf.valueClient = setupMockValueService(true)

			Expect(ulf.execute(context.TODO(), events.Event{Event: events.SpaceDeleted{}}, nil, []string{"foo"})).To(ConsistOf("foo"))
		})

		It("handles ShareCreated disabled", func() {
			ulf.valueClient = setupMockValueService(false)

			Expect(ulf.execute(context.TODO(), events.Event{Event: events.ShareCreated{}}, nil, []string{"foo"})).To(BeEmpty())
		})

		It("handles ShareRemoved disabled", func() {
			ulf.valueClient = setupMockValueService(false)

			Expect(ulf.execute(context.TODO(), events.Event{Event: events.ShareRemoved{}}, nil, []string{"foo"})).To(BeEmpty())
		})

		It("handles ShareExpired disabled", func() {
			ulf.valueClient = setupMockValueService(false)

			Expect(ulf.execute(context.TODO(), events.Event{Event: events.ShareExpired{}}, nil, []string{"foo"})).To(BeEmpty())
		})

		It("handles SpaceShared disabled", func() {
			ulf.valueClient = setupMockValueService(false)

			Expect(ulf.execute(context.TODO(), events.Event{Event: events.SpaceShared{}}, nil, []string{"foo"})).To(BeEmpty())
		})

		It("handles SpaceUnshared disabled", func() {
			ulf.valueClient = setupMockValueService(false)

			Expect(ulf.execute(context.TODO(), events.Event{Event: events.SpaceUnshared{}}, nil, []string{"foo"})).To(BeEmpty())
		})

		It("handles SpaceMembershipExpired disabled", func() {
			ulf.valueClient = setupMockValueService(false)

			Expect(ulf.execute(context.TODO(), events.Event{Event: events.SpaceMembershipExpired{}}, nil, []string{"foo"})).To(BeEmpty())
		})

		It("handles SpaceDisabled disabled", func() {
			ulf.valueClient = setupMockValueService(false)

			Expect(ulf.execute(context.TODO(), events.Event{Event: events.SpaceDisabled{}}, nil, []string{"foo"})).To(BeEmpty())
		})

		It("handles SpaceDeleted disabled", func() {
			ulf.valueClient = setupMockValueService(false)

			Expect(ulf.execute(context.TODO(), events.Event{Event: events.SpaceDeleted{}}, nil, []string{"foo"})).To(BeEmpty())
		})

		It("handles PostprocessingStepFinished disabled", func() {
			ulf.valueClient = setupMockValueService(false)

			Expect(ulf.execute(context.TODO(), events.Event{Event: events.PostprocessingStepFinished{}}, nil, []string{"foo"})).To(BeEmpty())
		})
	})
})
