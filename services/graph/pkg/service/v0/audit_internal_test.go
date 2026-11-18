package svc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/go-chi/chi/v5"
	"github.com/opencloud-eu/reva/v3/pkg/audit"
	revactx "github.com/opencloud-eu/reva/v3/pkg/ctx"
	"github.com/opencloud-eu/reva/v3/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuditEventsForDomainEvents(t *testing.T) {
	for _, tc := range []struct {
		ev       any
		def      *audit.Definition
		targetID string
	}{
		{events.UserCreated{UserID: "u-1"}, audit.UserCreate, "u-1"},
		{events.UserDeleted{UserID: "u-1"}, audit.UserDelete, "u-1"},
		{events.UserSoftDeleted{UserID: "u-1"}, audit.UserSoftDelete, "u-1"},
		{events.GroupCreated{GroupID: "g-1"}, audit.GroupCreate, "g-1"},
		{events.GroupDeleted{GroupID: "g-1"}, audit.GroupDelete, "g-1"},
		{events.GroupMemberAdded{GroupID: "g-1", UserID: "u-1"}, audit.GroupMemberAdd, "g-1"},
		{events.GroupMemberRemoved{GroupID: "g-1", UserID: "u-1"}, audit.GroupMemberRemove, "g-1"},
	} {
		evs := auditEvents(tc.ev)
		require.Len(t, evs, 1, tc.def.Name)
		assert.Equal(t, tc.def, evs[0].Definition)
		assert.Equal(t, tc.targetID, evs[0].Target.ID)
	}

	assert.Empty(t, auditEvents(events.SpaceCreated{}), "events graph doesn't own are audited elsewhere")
}

func TestAuditEventsForUserFeatures(t *testing.T) {
	old := "old@example.com"
	oldRole := "role-user"
	evs := auditEvents(events.UserFeatureChanged{
		UserID: "u-1",
		Features: []events.UserFeature{
			{Name: "email", Value: "new@example.com", OldValue: &old},
			{Name: "displayname", Value: "Alice"},
			{Name: "accountEnabled", Value: "false"},
			{Name: "passwordChanged"},
			{Name: "roleChanged", Value: "role-admin", OldValue: &oldRole},
		},
	})

	var names []string
	for _, ev := range evs {
		names = append(names, ev.Definition.Name)
		assert.Equal(t, "u-1", ev.Target.ID)
	}
	assert.ElementsMatch(t, []string{
		"opencloud.user.disable",
		"opencloud.user.password.reset",
		"opencloud.app_role.assign",
		"opencloud.user.update",
	}, names)

	update := evs[len(evs)-1]
	assert.Equal(t, map[string]any{"email": "new@example.com", "displayname": "Alice"}, update.Change.After)
	assert.Equal(t, map[string]any{"email": "old@example.com"}, update.Change.Before)

	evs = auditEvents(events.UserFeatureChanged{UserID: "u-1", Features: []events.UserFeature{{Name: "password", Value: "***"}}})
	require.Len(t, evs, 1)
	assert.Equal(t, audit.UserPasswordChange, evs[0].Definition)
	assert.Nil(t, evs[0].Change, "no password values in the record")

	evs = auditEvents(events.UserFeatureChanged{UserID: "u-1", Features: []events.UserFeature{{Name: "accountEnabled", Value: "true"}}})
	require.Len(t, evs, 1)
	assert.Equal(t, audit.UserEnable, evs[0].Definition)
}

type recordingEmitter struct{ events []audit.Event }

func (r *recordingEmitter) Emit(_ context.Context, ev audit.Event) error {
	r.events = append(r.events, ev)
	return nil
}

func TestAuditRoutes(t *testing.T) {
	rec := &recordingEmitter{}
	g := Graph{auditEmitter: rec}
	denyAll := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusForbidden) })
	}
	ok := func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) }
	notFound := func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) }

	router := chi.NewRouter()
	router.Route("/users/{userID}", func(r chi.Router) {
		r.With(g.auditOnError(audit.AppRoleAssign, "userID"), denyAll).Post("/appRoleAssignments", ok)
		r.With(g.auditOnError(audit.UserDelete, "userID")).Delete("/", notFound)
		r.With(g.auditOnError(audit.UserUpdate, "userID")).Patch("/", ok)
		r.With(g.auditAlways(audit.PersonalDataExport, "userID")).Post("/exportPersonalData", ok)
	})

	do := func(method, path string) {
		router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(method, path, nil))
	}

	do(http.MethodPost, "/users/u-1/appRoleAssignments")
	require.Len(t, rec.events, 1, "a refused role assignment is audited")
	assert.Equal(t, audit.AppRoleAssign, rec.events[0].Definition)
	assert.Equal(t, audit.OutcomeDenied, rec.events[0].Outcome)
	assert.Equal(t, "permission_denied", rec.events[0].Reason)
	assert.Equal(t, "u-1", rec.events[0].Target.ID)

	do(http.MethodDelete, "/users/u-2")
	require.Len(t, rec.events, 2)
	assert.Equal(t, audit.OutcomeFailure, rec.events[1].Outcome)
	assert.Equal(t, "not_found", rec.events[1].Reason)

	do(http.MethodPatch, "/users/u-1")
	assert.Len(t, rec.events, 2, "successes come from publishEvent, not from the route")

	do(http.MethodPost, "/users/u-1/exportPersonalData")
	require.Len(t, rec.events, 3, "the export has no domain event, its success is audited here")
	assert.Equal(t, audit.PersonalDataExport, rec.events[2].Definition)
	assert.Equal(t, audit.OutcomeSuccess, rec.events[2].Outcome)
}

func TestAuditRoutesDeniedOnly(t *testing.T) {
	rec := &recordingEmitter{}
	g := Graph{auditEmitter: rec}
	status := http.StatusCreated
	router := chi.NewRouter()
	router.With(g.auditOnDenied(audit.SpaceCreate, "")).Post("/drives", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(status) })

	for _, status = range []int{http.StatusCreated, http.StatusBadRequest, http.StatusInternalServerError} {
		router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/drives", nil))
	}
	assert.Empty(t, rec.events, "the storage provider reports successes and failures")

	status = http.StatusForbidden
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/drives", nil))
	require.Len(t, rec.events, 1, "graph refused it before any CS3 call")
	assert.Equal(t, audit.SpaceCreate, rec.events[0].Definition)
	assert.Equal(t, audit.OutcomeDenied, rec.events[0].Outcome)
}

func TestPersonalDataExportForAnotherUserIsDenied(t *testing.T) {
	rec := &recordingEmitter{}
	g := Graph{auditEmitter: rec}
	router := chi.NewRouter()
	// RequireOwnUserID answers 400 for other users
	router.With(g.auditAlways(audit.PersonalDataExport, "userID")).Post("/users/{userID}/exportPersonalData", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})
	ctx := revactx.ContextSetUser(context.Background(), &userpb.User{Id: &userpb.UserId{OpaqueId: "alan"}})
	do := func(userID string) {
		r := httptest.NewRequest(http.MethodPost, "/users/"+userID+"/exportPersonalData", nil).WithContext(ctx)
		router.ServeHTTP(httptest.NewRecorder(), r)
	}

	do("admin")
	require.Len(t, rec.events, 1)
	assert.Equal(t, audit.OutcomeDenied, rec.events[0].Outcome)
	assert.Equal(t, "permission_denied", rec.events[0].Reason)

	do("alan")
	require.Len(t, rec.events, 2)
	assert.Equal(t, audit.OutcomeFailure, rec.events[1].Outcome, "a bad request for oneself stays a failure")
}
