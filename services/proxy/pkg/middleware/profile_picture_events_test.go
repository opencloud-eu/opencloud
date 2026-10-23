package middleware

import (
	"net/http"
	"testing"

	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	ocevents "github.com/opencloud-eu/opencloud/pkg/events"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/oidc"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/user/backend"
	backendmocks "github.com/opencloud-eu/opencloud/services/proxy/pkg/user/backend/mocks"
	rolemocks "github.com/opencloud-eu/opencloud/services/proxy/pkg/userroles/mocks"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	microevents "go-micro.dev/v4/events"
)

type profilePictureEventPublisher struct {
	events []any
}

func (p *profilePictureEventPublisher) Publish(_ string, event any, _ ...microevents.PublishOption) error {
	p.events = append(p.events, event)
	return nil
}

func TestAccountResolverPublishesProfilePictureSync(t *testing.T) {
	const pictureURL = "https://idp.example.com/avatar.png"
	for _, tt := range []struct {
		name         string
		newSession   bool
		pictureClaim string
		picture      any
		accountError error
		wantEvents   int
		wantSync     bool
	}{
		{name: "new session syncs without autoprovisioning", newSession: true, pictureClaim: "picture", picture: pictureURL, wantEvents: 2, wantSync: true},
		{name: "ordinary request does not sync", pictureClaim: "picture", picture: pictureURL},
		{name: "disabled claim does not sync", newSession: true, picture: pictureURL, wantEvents: 1},
		{name: "missing claim does not sync", newSession: true, pictureClaim: "picture", wantEvents: 1},
		{name: "empty claim does not sync", newSession: true, pictureClaim: "picture", picture: "", wantEvents: 1},
		{name: "nonstring claim does not sync", newSession: true, pictureClaim: "picture", picture: 42, wantEvents: 1},
		{name: "disabled account does not sync", newSession: true, pictureClaim: "picture", picture: pictureURL, accountError: backend.ErrAccountDisabled},
	} {
		t.Run(tt.name, func(t *testing.T) {
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
				assert.Equal(t, "access-token", r.Header.Get(revactx.TokenHeader))
			}))
			req, recorder := mockRequest(map[string]any{
				oidc.Iss:               testIdP,
				oidc.PreferredUsername: "foo",
				"picture":              tt.picture,
			})
			req = req.WithContext(oidc.NewContextSessionFlag(req.Context(), tt.newSession))

			handler.ServeHTTP(recorder, req)

			require.Len(t, publisher.events, tt.wantEvents)
			if tt.wantEvents > 0 {
				_, ok := publisher.events[0].(events.UserSignedIn)
				assert.True(t, ok, "the normal sign-in event is still published")
			}
			if tt.wantSync {
				event, ok := publisher.events[1].(ocevents.UserProfilePictureSyncRequested)
				require.True(t, ok)
				assert.Equal(t, user.Id, event.Executant)
				assert.Equal(t, pictureURL, event.PictureURL)
				assert.NotNil(t, event.Timestamp)
			}
			if tt.accountError != nil {
				assert.False(t, nextCalled)
				assert.Equal(t, http.StatusUnauthorized, recorder.Code)
			} else {
				assert.True(t, nextCalled)
				assert.Equal(t, http.StatusOK, recorder.Code)
			}
			userProvider.AssertExpectations(t)
			userProvider.AssertNotCalled(t, "CreateUserFromClaims", mock.Anything, mock.Anything)
			userProvider.AssertNotCalled(t, "UpdateUserIfNeeded", mock.Anything, mock.Anything, mock.Anything)
			userProvider.AssertNotCalled(t, "SyncGroupMemberships", mock.Anything, mock.Anything, mock.Anything)
			roleAssigner.AssertExpectations(t)
		})
	}
}
