package middleware

import (
	"net/http"

	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ocevents "github.com/opencloud-eu/opencloud/pkg/events"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/oidc"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/user/backend"
	backendmocks "github.com/opencloud-eu/opencloud/services/proxy/pkg/user/backend/mocks"
	rolemocks "github.com/opencloud-eu/opencloud/services/proxy/pkg/userroles/mocks"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/stretchr/testify/mock"
	microevents "go-micro.dev/v4/events"
)

type profilePictureEventPublisher struct {
	events []any
}

func (p *profilePictureEventPublisher) Publish(_ string, event any, _ ...microevents.PublishOption) error {
	p.events = append(p.events, event)
	return nil
}

const pictureURL = "https://idp.example.com/avatar.png"

type profilePictureEventCase struct {
	newSession   bool
	pictureClaim string
	picture      any
	accountError error
	wantEvents   int
	wantSync     bool
}

var _ = DescribeTable("profile picture sync events from the account resolver", func(tt profilePictureEventCase) {
	user := &userv1beta1.User{
		Id:       &userv1beta1.UserId{Idp: testIdP, OpaqueId: "profile-picture-user"},
		Username: "foo",
	}
	userProvider := &backendmocks.UserBackend{}
	userProvider.On("GetUserByClaims", mock.Anything, "username", "foo").Return(user, "access-token", tt.accountError).Once()
	roleAssigner := &rolemocks.UserRoleAssigner{}
	if tt.accountError == nil {
		roleAssigner.On("UpdateUserRoleAssignment", mock.Anything, user, mock.Anything).Return(user, nil).Once()
	}
	publisher := &profilePictureEventPublisher{}
	nextCalled := false
	handler := AccountResolver(
		Logger(log.NopLogger()),
		UserProvider(userProvider),
		UserRoleAssigner(roleAssigner),
		UserOIDCClaim(oidc.PreferredUsername),
		UserCS3Claim("username"),
		PictureClaim(tt.pictureClaim),
		AutoprovisionAccounts(false),
		EventsPublisher(publisher),
	)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		Expect(r.Header.Get(revactx.TokenHeader)).To(Equal("access-token"))
	}))
	req, recorder := mockRequest(map[string]any{
		oidc.Iss:               testIdP,
		oidc.PreferredUsername: "foo",
		"picture":              tt.picture,
	})
	req = req.WithContext(oidc.NewContextSessionFlag(req.Context(), tt.newSession))

	handler.ServeHTTP(recorder, req)

	Expect(publisher.events).To(HaveLen(tt.wantEvents))
	if tt.wantEvents > 0 {
		_, ok := publisher.events[0].(events.UserSignedIn)
		Expect(ok).To(BeTrue(), "the normal sign-in event is still published")
	}
	if tt.wantSync {
		event, ok := publisher.events[1].(ocevents.UserProfilePictureSyncRequested)
		Expect(ok).To(BeTrue())
		Expect(event.Executant).To(Equal(user.Id))
		Expect(event.PictureURL).To(Equal(pictureURL))
		Expect(event.Timestamp).NotTo(BeNil())
	}
	if tt.accountError != nil {
		Expect(nextCalled).To(BeFalse())
		Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
	} else {
		Expect(nextCalled).To(BeTrue())
		Expect(recorder.Code).To(Equal(http.StatusOK))
	}
	userProvider.AssertExpectations(GinkgoT())
	userProvider.AssertNotCalled(GinkgoT(), "CreateUserFromClaims", mock.Anything, mock.Anything)
	userProvider.AssertNotCalled(GinkgoT(), "UpdateUserIfNeeded", mock.Anything, mock.Anything, mock.Anything)
	userProvider.AssertNotCalled(GinkgoT(), "SyncGroupMemberships", mock.Anything, mock.Anything, mock.Anything)
	roleAssigner.AssertExpectations(GinkgoT())
},
	Entry("new session syncs without autoprovisioning", profilePictureEventCase{newSession: true, pictureClaim: "picture", picture: pictureURL, wantEvents: 2, wantSync: true}),
	Entry("ordinary request does not sync", profilePictureEventCase{pictureClaim: "picture", picture: pictureURL}),
	Entry("disabled claim does not sync", profilePictureEventCase{newSession: true, picture: pictureURL, wantEvents: 1}),
	Entry("missing claim does not sync", profilePictureEventCase{newSession: true, pictureClaim: "picture", wantEvents: 1}),
	Entry("empty claim does not sync", profilePictureEventCase{newSession: true, pictureClaim: "picture", picture: "", wantEvents: 1}),
	Entry("nonstring claim does not sync", profilePictureEventCase{newSession: true, pictureClaim: "picture", picture: 42, wantEvents: 1}),
	Entry("disabled account does not sync", profilePictureEventCase{newSession: true, pictureClaim: "picture", picture: pictureURL, accountError: backend.ErrAccountDisabled}),
)
