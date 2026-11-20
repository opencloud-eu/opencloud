package command

import (
	"bytes"
	"crypto/tls"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/opencloud-eu/opencloud/pkg/config"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	gatewayconfig "github.com/opencloud-eu/opencloud/services/gateway/pkg/config"
	graphconfig "github.com/opencloud-eu/opencloud/services/graph/pkg/config"
)

const (
	adminRoleID      = "1ecb83a2-d7b2-4b67-9d81-3d3e4f8a9b0c"
	userRoleID       = "d7beeea8-8ff4-4b0e-9d7e-1c1d3c3bd0e1"
	spaceAdminRoleID = "9f4f8f3f-3e34-4a1a-9a7e-26f1b0a0a3b1"
)

func testRoleNames() map[string]string {
	return map[string]string{
		adminRoleID:      "Admin",
		userRoleID:       "User",
		spaceAdminRoleID: "Space Admin",
	}
}

func countPtr(c int) *int {
	return &c
}

// testUser builds a user with the given last successful sign-in and role
// assignments. A nil lastSignIn means the user never signed in.
func testUser(name string, lastSignIn *time.Time, roleIDs ...string) libregraph.User {
	u := libregraph.User{DisplayName: name, OnPremisesSamAccountName: name}
	if lastSignIn != nil {
		u.SetSignInActivity(libregraph.SignInActivity{LastSuccessfulSignInDateTime: lastSignIn})
	}
	for _, id := range roleIDs {
		u.AppRoleAssignments = append(u.AppRoleAssignments, libregraph.AppRoleAssignment{AppRoleId: id})
	}
	return u
}

func TestCountUsersByRole(t *testing.T) {
	activeSince := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)

	recent := activeSince.Add(24 * time.Hour)
	onCutoff := activeSince
	stale := activeSince.Add(-time.Second)

	testCases := []struct {
		name                 string
		users                []libregraph.User
		expectedRoles        []roleCount
		expectedActive       *int
		expectedWithSignInfo int
	}{
		{
			name:                 "no users",
			users:                nil,
			expectedRoles:        []roleCount{},
			expectedActive:       countPtr(0),
			expectedWithSignInfo: 0,
		},
		{
			name: "roles are ordered by count, then by name",
			users: []libregraph.User{
				testUser("einstein", &recent, userRoleID),
				testUser("marie", &recent, userRoleID),
				testUser("moss", &recent, spaceAdminRoleID),
				testUser("admin", &recent, adminRoleID),
			},
			expectedRoles: []roleCount{
				{Role: "User", Users: 2, SignedIn: countPtr(2)},
				{Role: "Admin", Users: 1, SignedIn: countPtr(1)},
				{Role: "Space Admin", Users: 1, SignedIn: countPtr(1)},
			},
			expectedActive:       countPtr(4),
			expectedWithSignInfo: 4,
		},
		{
			name: "users that never signed in still count towards their role",
			users: []libregraph.User{
				testUser("einstein", &recent, userRoleID),
				testUser("marie", nil, userRoleID),
			},
			expectedRoles:        []roleCount{{Role: "User", Users: 2, SignedIn: countPtr(1)}},
			expectedActive:       countPtr(1),
			expectedWithSignInfo: 1,
		},
		{
			name: "users outside the window still count towards their role",
			users: []libregraph.User{
				testUser("einstein", &onCutoff, userRoleID),
				testUser("marie", &stale, userRoleID),
			},
			expectedRoles:        []roleCount{{Role: "User", Users: 2, SignedIn: countPtr(1)}},
			expectedActive:       countPtr(1),
			expectedWithSignInfo: 2,
		},
		{
			name: "sign-in counts are unknown when no sign-in was ever recorded",
			users: []libregraph.User{
				testUser("einstein", nil, userRoleID),
				testUser("admin", nil, adminRoleID),
			},
			expectedRoles: []roleCount{
				{Role: "Admin", Users: 1, SignedIn: nil},
				{Role: "User", Users: 1, SignedIn: nil},
			},
			expectedActive:       nil,
			expectedWithSignInfo: 0,
		},
		{
			name:                 "users without a role land in their own bucket",
			users:                []libregraph.User{testUser("einstein", &recent)},
			expectedRoles:        []roleCount{{Role: _reportNoRole, Users: 1, SignedIn: countPtr(1)}},
			expectedActive:       countPtr(1),
			expectedWithSignInfo: 1,
		},
		{
			name:  "users with multiple roles are counted for each of them",
			users: []libregraph.User{testUser("einstein", &recent, adminRoleID, spaceAdminRoleID)},
			expectedRoles: []roleCount{
				{Role: "Admin", Users: 1, SignedIn: countPtr(1)},
				{Role: "Space Admin", Users: 1, SignedIn: countPtr(1)},
			},
			expectedActive:       countPtr(1),
			expectedWithSignInfo: 1,
		},
		{
			name:                 "unknown roles are reported by their id",
			users:                []libregraph.User{testUser("einstein", &recent, "some-unknown-role-id")},
			expectedRoles:        []roleCount{{Role: "some-unknown-role-id", Users: 1, SignedIn: countPtr(1)}},
			expectedActive:       countPtr(1),
			expectedWithSignInfo: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			roles, active, withSignInfo := countUsersByRole(tc.users, testRoleNames(), &activeSince)

			if tableCount(active) != tableCount(tc.expectedActive) {
				t.Errorf("expected %s active users, got %s", tableCount(tc.expectedActive), tableCount(active))
			}
			if withSignInfo != tc.expectedWithSignInfo {
				t.Errorf("expected %d users with sign-in data, got %d", tc.expectedWithSignInfo, withSignInfo)
			}
			if len(roles) != len(tc.expectedRoles) {
				t.Fatalf("expected %d roles, got %d", len(tc.expectedRoles), len(roles))
			}
			for i, expected := range tc.expectedRoles {
				got := roles[i]
				if got.Role != expected.Role || got.Users != expected.Users ||
					tableCount(got.SignedIn) != tableCount(expected.SignedIn) {
					t.Errorf("role %d: expected %s/%d/%s, got %s/%d/%s",
						i, expected.Role, expected.Users, tableCount(expected.SignedIn),
						got.Role, got.Users, tableCount(got.SignedIn))
				}
			}
		})
	}
}

func TestRenderUserReport(t *testing.T) {
	report := userReport{
		GeneratedAt:         time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC),
		Days:                31,
		ActiveSince:         time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC),
		TotalUsers:          5,
		ActiveUsers:         countPtr(3),
		UsersWithSignInData: 4,
		SignInReported:      true,
		Roles: []roleCount{
			{Role: "User", Users: 3, SignedIn: countPtr(2)},
			{Role: "Admin", Users: 1, SignedIn: countPtr(1)},
		},
	}

	testCases := []struct {
		format   string
		expected []string
	}{
		{
			format: _reportOutputTable,
			expected: []string{
				"Sign-in window: last 31 days (since 2026-09-08T12:00:00Z)",
				"Users in total: 5",
				"Signed in within the last 31 days: 3",
				"Users with a recorded sign-in: 4",
				"Role", "Users", "Admin", "User",
			},
		},
		{
			format: _reportOutputJSON,
			expected: []string{
				`"activeSince": "2026-09-08T12:00:00Z"`,
				`"totalUsers": 5`,
				`"activeUsers": 3`,
				`"usersWithSignInData": 4`,
				`"role": "Admin"`,
				`"users": 1`,
				`"signedIn": 2`,
			},
		},
		{
			format: _reportOutputCSV,
			expected: []string{
				"days,activeSince,totalUsers,activeUsers,usersWithSignInData,role,users,signedIn\n" +
					"31,2026-09-08T12:00:00Z,5,3,4,User,3,2\n" +
					"31,2026-09-08T12:00:00Z,5,3,4,Admin,1,1\n",
			},
		},
		{
			format: _reportOutputYAML,
			expected: []string{
				"days: 31", "totalUsers: 5", "activeUsers: 3", "usersWithSignInData: 4",
				"- role: User", "users: 3", "signedIn: 2",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.format, func(t *testing.T) {
			var buf bytes.Buffer
			if err := renderUserReport(&buf, report, tc.format); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// collapse the padding the table format uses to align its labels
			actual := regexp.MustCompile(` +`).ReplaceAllString(buf.String(), " ")
			for _, expected := range tc.expected {
				if !strings.Contains(actual, expected) {
					t.Errorf("expected output to contain %q, got:\n%s", expected, buf.String())
				}
			}
		})
	}
}

// TestRenderUserReportNobodySignedInRecently makes sure the role grouping is
// still reported in full when nobody signed in within the reported window, and
// that a genuine 0 stays a 0.
func TestRenderUserReportNobodySignedInRecently(t *testing.T) {
	report := userReport{
		GeneratedAt:         time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC),
		Days:                1,
		ActiveSince:         time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC),
		TotalUsers:          5,
		ActiveUsers:         countPtr(0),
		UsersWithSignInData: 5,
		SignInReported:      true,
		Roles: []roleCount{
			{Role: "User", Users: 4, SignedIn: countPtr(0)},
			{Role: "Admin", Users: 1, SignedIn: countPtr(0)},
		},
	}

	testCases := []struct {
		format   string
		expected []string
	}{
		{
			format:   _reportOutputTable,
			expected: []string{"Signed in within the last 1 day: 0", "User", "Admin"},
		},
		{format: _reportOutputJSON, expected: []string{`"activeUsers": 0`, `"users": 4`}},
		{format: _reportOutputYAML, expected: []string{"activeUsers: 0", "users: 4"}},
		{
			format: _reportOutputCSV,
			expected: []string{
				"days,activeSince,totalUsers,activeUsers,usersWithSignInData,role,users,signedIn\n" +
					"1,2026-10-08T12:00:00Z,5,0,5,User,4,0\n" +
					"1,2026-10-08T12:00:00Z,5,0,5,Admin,1,0\n",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.format, func(t *testing.T) {
			var buf bytes.Buffer
			if err := renderUserReport(&buf, report, tc.format); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			actual := regexp.MustCompile(` +`).ReplaceAllString(buf.String(), " ")
			for _, expected := range tc.expected {
				if !strings.Contains(actual, expected) {
					t.Errorf("expected output to contain %q, got:\n%s", expected, buf.String())
				}
			}
			if strings.Contains(actual, "N/A") {
				t.Errorf("did not expect an N/A count, got:\n%s", buf.String())
			}
			if strings.Contains(actual, "Warning:") {
				t.Errorf("did not expect a warning, got:\n%s", buf.String())
			}
		})
	}
}

// TestRenderUserReportWithoutUsers makes sure the csv output keeps carrying the
// report context when there is not a single user to group.
func TestRenderUserReportWithoutUsers(t *testing.T) {
	report := userReport{
		GeneratedAt:    time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC),
		Days:           31,
		ActiveSince:    time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC),
		ActiveUsers:    countPtr(0),
		SignInReported: true,
		Roles:          []roleCount{},
	}

	var buf bytes.Buffer
	if err := renderUserReport(&buf, report, _reportOutputCSV); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "days,activeSince,totalUsers,activeUsers,usersWithSignInData,role,users,signedIn\n" +
		"31,2026-09-08T12:00:00Z,0,0,0,,0,0\n"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}

// TestRenderUserReportWithoutSignInData covers deployments that never record a
// sign-in, e.g. because the identity backend is read only. The users are still
// grouped by role there, but the sign-in counts are not available and must not
// be rendered as 0.
func TestRenderUserReportWithoutSignInData(t *testing.T) {
	report := userReport{
		GeneratedAt:         time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC),
		Days:                31,
		ActiveSince:         time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC),
		TotalUsers:          6,
		ActiveUsers:         nil,
		UsersWithSignInData: 0,
		SignInReported:      true,
		Roles: []roleCount{
			{Role: "User", Users: 5, SignedIn: nil},
			{Role: "Admin", Users: 1, SignedIn: nil},
		},
	}

	testCases := []struct {
		format      string
		expected    []string
		notExpected []string
	}{
		{
			format: _reportOutputTable,
			expected: []string{
				"Signed in within the last 31 days: N/A",
				"Warning: not a single user has a recorded sign-in",
				// the grouping has to be there regardless of the missing data
				"Users in total: 6", "User", "Admin", "N/A",
			},
		},
		{
			format:      _reportOutputJSON,
			expected:    []string{`"activeUsers": null`, `"signedIn": null`, `"usersWithSignInData": 0`},
			notExpected: []string{`"activeUsers": 0`, `"signedIn": 0`},
		},
		{
			format:      _reportOutputYAML,
			expected:    []string{"activeUsers: null", "signedIn: null"},
			notExpected: []string{"activeUsers: 0", "signedIn: 0"},
		},
		{
			format: _reportOutputCSV,
			expected: []string{
				"days,activeSince,totalUsers,activeUsers,usersWithSignInData,role,users,signedIn\n" +
					"31,2026-09-08T12:00:00Z,6,,0,User,5,\n" +
					"31,2026-09-08T12:00:00Z,6,,0,Admin,1,\n",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.format, func(t *testing.T) {
			var buf bytes.Buffer
			if err := renderUserReport(&buf, report, tc.format); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			actual := regexp.MustCompile(` +`).ReplaceAllString(buf.String(), " ")
			for _, expected := range tc.expected {
				if !strings.Contains(actual, expected) {
					t.Errorf("expected output to contain %q, got:\n%s", expected, buf.String())
				}
			}
			for _, notExpected := range tc.notExpected {
				if strings.Contains(actual, notExpected) {
					t.Errorf("expected output not to contain %q, got:\n%s", notExpected, buf.String())
				}
			}
		})
	}
}

// TestCountUsersByRoleWithoutWindow covers the default: without a window
// the sign-in data is not looked at at all, even on an instance that records
// it, and the users are still grouped by their role.
func TestCountUsersByRoleWithoutWindow(t *testing.T) {
	recent := time.Now().UTC().Add(-time.Hour)
	users := []libregraph.User{
		testUser("alan", &recent, userRoleID),
		testUser("lynn", nil, userRoleID),
		testUser("dennis", &recent, adminRoleID),
		testUser("mary", nil),
	}

	roles, active, withSignInfo := countUsersByRole(users, testRoleNames(), nil)

	if active != nil {
		t.Errorf("expected no count of active users, got %d", *active)
	}
	if withSignInfo != 0 {
		t.Errorf("expected no sign-in data to be counted, got %d", withSignInfo)
	}

	expectedRoles := []roleCount{
		{Role: "User", Users: 2},
		{Role: _reportNoRole, Users: 1},
		{Role: "Admin", Users: 1},
	}
	if len(roles) != len(expectedRoles) {
		t.Fatalf("expected %d roles, got %d", len(expectedRoles), len(roles))
	}
	for i, expected := range expectedRoles {
		got := roles[i]
		if got.Role != expected.Role || got.Users != expected.Users {
			t.Errorf("role %d: expected %s/%d, got %s/%d",
				i, expected.Role, expected.Users, got.Role, got.Users)
		}
		if got.SignedIn != nil {
			t.Errorf("role %s: expected no sign-in count, got %d", got.Role, *got.SignedIn)
		}
	}
}

// TestRenderUserReportWithoutSignIn makes sure the default leaves every
// trace of the sign-in window out of the output, rather than reporting it as
// null or N/A.
func TestRenderUserReportWithoutSignIn(t *testing.T) {
	report := userReport{
		GeneratedAt: time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC),
		TotalUsers:  6,
		Roles: []roleCount{
			{Role: "User", Users: 4},
			{Role: "Admin", Users: 2},
		},
	}

	testCases := []struct {
		format      string
		expected    []string
		notExpected []string
	}{
		{
			format:      _reportOutputTable,
			expected:    []string{"Users in total: 6", "Role", "Users", "User", "Admin"},
			notExpected: []string{"Sign-in window", "Signed in within", "recorded sign-in", "N/A", "Warning"},
		},
		{
			format:      _reportOutputJSON,
			expected:    []string{`"totalUsers": 6`, `"role": "User"`, `"users": 4`},
			notExpected: []string{"signedIn", "activeUsers", "activeSince", "usersWithSignInData", `"days"`, "null"},
		},
		{
			format:      _reportOutputCSV,
			expected:    []string{"totalUsers,role,users\n6,User,4\n6,Admin,2\n"},
			notExpected: []string{"signedIn", "activeUsers", "activeSince", "usersWithSignInData"},
		},
		{
			format:      _reportOutputYAML,
			expected:    []string{"totalUsers: 6", "- role: User", "users: 4"},
			notExpected: []string{"signedIn", "activeUsers", "activeSince", "usersWithSignInData", "days:"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.format, func(t *testing.T) {
			var buf bytes.Buffer
			if err := renderUserReport(&buf, report, tc.format); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// collapse the padding the table format uses to align its labels
			actual := regexp.MustCompile(` +`).ReplaceAllString(buf.String(), " ")
			for _, expected := range tc.expected {
				if !strings.Contains(actual, expected) {
					t.Errorf("expected output to contain %q, got:\n%s", expected, buf.String())
				}
			}
			for _, notExpected := range tc.notExpected {
				if strings.Contains(actual, notExpected) {
					t.Errorf("expected output not to contain %q, got:\n%s", notExpected, buf.String())
				}
			}
		})
	}
}

// TestRenderUserReportWithoutSignInWithoutUsers keeps the csv total readable
// when there is not a single user to group.
func TestRenderUserReportWithoutSignInWithoutUsers(t *testing.T) {
	report := userReport{
		GeneratedAt: time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC),
		Roles:       []roleCount{},
	}

	var buf bytes.Buffer
	if err := renderUserReport(&buf, report, _reportOutputCSV); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "totalUsers,role,users\n0,,0\n"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}

// TestNodeProtocol covers the TLS resolution of a registry node. The registry
// reports "http" for every HTTP service, so only the use_tls metadata tells a
// TLS listener apart from a plain one.
func TestNodeProtocol(t *testing.T) {
	testCases := []struct {
		name     string
		metadata map[string]string
		expected string
	}{
		{
			name:     "plain http service",
			metadata: map[string]string{"protocol": "http"},
			expected: "http",
		},
		{
			name:     "http service with TLS enabled",
			metadata: map[string]string{"protocol": "http", "use_tls": "true"},
			expected: "https",
		},
		{
			name:     "use_tls wins over the reported protocol",
			metadata: map[string]string{"protocol": "grpc", "use_tls": "true"},
			expected: "https",
		},
		{
			name:     "use_tls false is plain",
			metadata: map[string]string{"protocol": "http", "use_tls": "false"},
			expected: "http",
		},
		{
			name:     "missing protocol falls back to http",
			metadata: map[string]string{},
			expected: "http",
		},
		{
			name:     "missing protocol with TLS enabled",
			metadata: map[string]string{"use_tls": "true"},
			expected: "https",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := nodeProtocol(tc.metadata); got != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}

// TestReportCommandFlags pins the command line surface: the shorthands must
// stay stable for scripts, and a new flag must not silently take one that is
// already in use.
func TestReportCommandFlags(t *testing.T) {
	cmd := ReportCommand(&config.Config{})

	testCases := []struct {
		name      string
		shorthand string
		defValue  string
	}{
		{name: _reportDaysFlagName, shorthand: "d", defValue: "31"},
		{name: _reportOutputFlagName, shorthand: "o", defValue: _reportOutputTable},
		{name: _reportTimeoutFlagName, shorthand: "t", defValue: "5m0s"},
		{name: _reportSignInFlagName, shorthand: "", defValue: "false"},
		{name: _reportInsecureFlagName, shorthand: "", defValue: "false"},
	}

	seen := map[string]string{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			flag := cmd.Flags().Lookup(tc.name)
			if flag == nil {
				t.Fatalf("flag --%s is not registered", tc.name)
			}
			if flag.Shorthand != tc.shorthand {
				t.Errorf("expected shorthand %q for --%s, got %q", tc.shorthand, tc.name, flag.Shorthand)
			}
			if flag.DefValue != tc.defValue {
				t.Errorf("expected default %q for --%s, got %q", tc.defValue, tc.name, flag.DefValue)
			}
		})
		if tc.shorthand != "" {
			if other, taken := seen[tc.shorthand]; taken {
				t.Errorf("shorthand -%s is used by both --%s and --%s", tc.shorthand, other, tc.name)
			}
			seen[tc.shorthand] = tc.name
		}
	}
}

// TestSignInUnavailable covers when --with-sign-in may be used at all.
// OpenCloud stores the last sign-in on the user entry in LDAP, so a read-only
// or non-LDAP backend records nothing and the flag has to be refused rather
// than producing a column of N/A.
func TestSignInUnavailable(t *testing.T) {
	testCases := []struct {
		name         string
		backend      string
		writeEnabled bool
		available    bool
		contains     string
	}{
		{
			name:         "writable ldap backend",
			backend:      "ldap",
			writeEnabled: true,
			available:    true,
		},
		{
			name:         "read-only ldap backend",
			backend:      "ldap",
			writeEnabled: false,
			available:    false,
			contains:     "OC_LDAP_SERVER_WRITE_ENABLED",
		},
		{
			name:         "cs3 backend never records sign-ins",
			backend:      "cs3",
			writeEnabled: true,
			available:    false,
			contains:     "cs3",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{Graph: &graphconfig.Config{}}
			cfg.Graph.Identity.Backend = tc.backend
			cfg.Graph.Identity.LDAP.WriteEnabled = tc.writeEnabled

			reason := signInUnavailable(cfg)
			if tc.available {
				if reason != "" {
					t.Errorf("expected sign-ins to be available, got refusal %q", reason)
				}
				return
			}
			if reason == "" {
				t.Fatal("expected a refusal, got none")
			}
			if !strings.Contains(reason, tc.contains) {
				t.Errorf("expected the refusal to mention %q, got %q", tc.contains, reason)
			}
		})
	}
}

// TestNewGraphHTTPClient pins the TLS floor and makes sure the certificate
// check is only skipped on request. It also guards the transport clone: the
// defaults for proxy support and HTTP/2 must survive, which a bare
// &http.Transport{} would silently drop.
func TestNewGraphHTTPClient(t *testing.T) {
	testCases := []struct {
		name     string
		insecure bool
	}{
		{name: "verifies certificates by default", insecure: false},
		{name: "skips verification when asked", insecure: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := newGraphHTTPClient(reportOptions{timeout: 42 * time.Second, insecure: tc.insecure})

			if client.Timeout != 42*time.Second {
				t.Errorf("expected the timeout to be carried over, got %s", client.Timeout)
			}

			transport, ok := client.Transport.(*http.Transport)
			if !ok {
				t.Fatalf("expected an *http.Transport, got %T", client.Transport)
			}
			if transport.TLSClientConfig.MinVersion != tls.VersionTLS12 {
				t.Errorf("expected TLS 1.2 as the minimum, got %#x", transport.TLSClientConfig.MinVersion)
			}
			if transport.TLSClientConfig.InsecureSkipVerify != tc.insecure {
				t.Errorf("expected InsecureSkipVerify to be %v, got %v",
					tc.insecure, transport.TLSClientConfig.InsecureSkipVerify)
			}

			// cloned from the default transport, not built from scratch
			if transport.Proxy == nil {
				t.Error("expected the default proxy support to survive the clone")
			}
			if !transport.ForceAttemptHTTP2 {
				t.Error("expected HTTP/2 to stay enabled after the clone")
			}
		})
	}
}

// TestConfiguredGraphURL covers the fallback used when the service registry
// holds no entry, e.g. a deployment running with MICRO_REGISTRY=memory.
func TestConfiguredGraphURL(t *testing.T) {
	newCfg := func(addr, root string, tlsEnabled bool) *config.Config {
		cfg := &config.Config{
			Commons: &shared.Commons{
				HTTPServiceTLS: shared.HTTPServiceTLS{Enabled: tlsEnabled},
			},
			Graph: &graphconfig.Config{},
		}
		cfg.Graph.HTTP.Addr = addr
		cfg.Graph.HTTP.Root = root
		return cfg
	}

	testCases := []struct {
		name     string
		cfg      *config.Config
		expected string
	}{
		{
			name:     "plain http",
			cfg:      newCfg("127.0.0.1:9120", "/graph", false),
			expected: "http://127.0.0.1:9120/graph",
		},
		{
			name:     "https when the http services serve TLS",
			cfg:      newCfg("127.0.0.1:9120", "/graph", true),
			expected: "https://127.0.0.1:9120/graph",
		},
		{
			name:     "no address configured leaves it to the caller",
			cfg:      newCfg("", "/graph", false),
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := configuredGraphURL(tc.cfg); got != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}

// TestConfiguredGatewayAddress covers the dns:/// fallback target. A wildcard
// bind address has to become loopback: the report runs next to the gateway, and
// 0.0.0.0 is not a dialable destination.
func TestConfiguredGatewayAddress(t *testing.T) {
	newCfg := func(addr string) *config.Config {
		cfg := &config.Config{Gateway: &gatewayconfig.Config{}}
		cfg.Gateway.GRPC.Addr = addr
		return cfg
	}

	testCases := []struct {
		name     string
		cfg      *config.Config
		expected string
	}{
		{name: "loopback stays", cfg: newCfg("127.0.0.1:9142"), expected: "dns:///127.0.0.1:9142"},
		{name: "ipv4 wildcard becomes loopback", cfg: newCfg("0.0.0.0:9142"), expected: "dns:///127.0.0.1:9142"},
		{name: "ipv6 wildcard becomes loopback", cfg: newCfg("[::]:9142"), expected: "dns:///127.0.0.1:9142"},
		{name: "empty host becomes loopback", cfg: newCfg(":9142"), expected: "dns:///127.0.0.1:9142"},
		{name: "a real host is kept", cfg: newCfg("gateway.internal:9142"), expected: "dns:///gateway.internal:9142"},
		{name: "no address configured", cfg: newCfg(""), expected: ""},
		{name: "not a host:port", cfg: newCfg("eu.opencloud.api.gateway"), expected: ""},
		{name: "no gateway config at all", cfg: &config.Config{}, expected: ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := configuredGatewayAddress(tc.cfg); got != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}
