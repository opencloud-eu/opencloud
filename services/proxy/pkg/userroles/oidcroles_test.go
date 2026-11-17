package userroles

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	cs3 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config"
)

func TestExtractRolesArray(t *testing.T) {
	byt := []byte(`{"roles":["a","b"]}`)

	claims := map[string]any{}
	err := json.Unmarshal(byt, &claims)
	if err != nil {
		t.Fatal(err)
	}

	roles, err := extractRoles("roles", claims)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := roles["a"]; !ok {
		t.Fatal("must contain 'a'")
	}
	if _, ok := roles["b"]; !ok {
		t.Fatal("must contain 'b'")
	}
}

func TestExtractRolesString(t *testing.T) {
	byt := []byte(`{"roles":"a"}`)

	claims := map[string]any{}
	err := json.Unmarshal(byt, &claims)
	if err != nil {
		t.Fatal(err)
	}

	roles, err := extractRoles("roles", claims)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := roles["a"]; !ok {
		t.Fatal("must contain 'a'")
	}
}

func TestExtractRolesPathArray(t *testing.T) {
	byt := []byte(`{"sub":{"roles":["a","b"]}}`)

	claims := map[string]any{}
	err := json.Unmarshal(byt, &claims)
	if err != nil {
		t.Fatal(err)
	}

	roles, err := extractRoles("sub.roles", claims)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := roles["a"]; !ok {
		t.Fatal("must contain 'a'")
	}
	if _, ok := roles["b"]; !ok {
		t.Fatal("must contain 'b'")
	}
}

func TestExtractRolesPathString(t *testing.T) {
	byt := []byte(`{"sub":{"roles":"a"}}`)

	claims := map[string]any{}
	err := json.Unmarshal(byt, &claims)
	if err != nil {
		t.Fatal(err)
	}

	roles, err := extractRoles("sub.roles", claims)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := roles["a"]; !ok {
		t.Fatal("must contain 'a'")
	}
}

func TestExtractEscapedRolesPathString(t *testing.T) {
	byt := []byte(`{"sub.roles":"a"}`)

	claims := map[string]any{}
	err := json.Unmarshal(byt, &claims)
	if err != nil {
		t.Fatal(err)
	}

	roles, err := extractRoles("sub\\.roles", claims)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := roles["a"]; !ok {
		t.Fatal("must contain 'a'")
	}
}

func TestNoRoles(t *testing.T) {
	byt := []byte(`{"sub":{"foo":"a"}}`)

	claims := map[string]any{}
	err := json.Unmarshal(byt, &claims)
	if err != nil {
		t.Fatal(err)
	}

	roles, err := extractRoles("sub.roles", claims)
	if err == nil {
		t.Fatal("must not find a role")
	}
	if len(roles) != 0 {
		t.Fatal("length of roles mut be 0")
	}
}

func TestUpdateUserRoleAssignmentClaimErrors(t *testing.T) {
	// pre-fill the role name cache so no settings service is needed
	roleNameToID.lock.Lock()
	roleNameToID.roleNameToID = map[string]string{"admin": "admin-id", "user": "user-id"}
	roleNameToID.lastRead = time.Now()
	roleNameToID.lock.Unlock()
	t.Cleanup(func() {
		roleNameToID.lock.Lock()
		roleNameToID.roleNameToID = nil
		roleNameToID.lastRead = time.Time{}
		roleNameToID.lock.Unlock()
	})

	ra := NewOIDCRoleAssigner(
		WithLogger(log.NewLogger()),
		WithRolesClaim("organization.roles"),
		WithRoleMapping([]config.RoleMapping{{RoleName: "admin", ClaimValue: "oc-admin"}}),
	)
	user := &cs3.User{Id: &cs3.UserId{OpaqueId: "user-id"}}

	tests := []struct {
		name   string
		claims map[string]any
	}{
		{"claim missing", map[string]any{"sub": "x"}},
		{"claim is null", map[string]any{"organization": map[string]any{"roles": nil}}},
		{"claim is empty", map[string]any{"organization": map[string]any{"roles": []any{}}}},
		{"claim has a non-string role", map[string]any{"organization": map[string]any{"roles": []any{1}}}},
		{"claim maps to no role", map[string]any{"organization": map[string]any{"roles": []any{"other"}}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ra.UpdateUserRoleAssignment(context.Background(), user, tc.claims)
			if got != nil {
				t.Fatal("must not return a user")
			}
			if !errors.Is(err, ErrRoleClaim) {
				t.Fatalf("want ErrRoleClaim, got %v", err)
			}
		})
	}
}
