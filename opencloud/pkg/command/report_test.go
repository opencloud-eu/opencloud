package command

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
	"time"

	libregraph "github.com/opencloud-eu/libre-graph-api-go"
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

func count(c int) *int {
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
			expectedActive:       count(0),
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
				{Role: "User", Users: 2, SignedIn: count(2)},
				{Role: "Admin", Users: 1, SignedIn: count(1)},
				{Role: "Space Admin", Users: 1, SignedIn: count(1)},
			},
			expectedActive:       count(4),
			expectedWithSignInfo: 4,
		},
		{
			name: "users that never signed in still count towards their role",
			users: []libregraph.User{
				testUser("einstein", &recent, userRoleID),
				testUser("marie", nil, userRoleID),
			},
			expectedRoles:        []roleCount{{Role: "User", Users: 2, SignedIn: count(1)}},
			expectedActive:       count(1),
			expectedWithSignInfo: 1,
		},
		{
			name: "users outside the window still count towards their role",
			users: []libregraph.User{
				testUser("einstein", &onCutoff, userRoleID),
				testUser("marie", &stale, userRoleID),
			},
			expectedRoles:        []roleCount{{Role: "User", Users: 2, SignedIn: count(1)}},
			expectedActive:       count(1),
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
			expectedRoles:        []roleCount{{Role: _reportNoRole, Users: 1, SignedIn: count(1)}},
			expectedActive:       count(1),
			expectedWithSignInfo: 1,
		},
		{
			name:  "users with multiple roles are counted for each of them",
			users: []libregraph.User{testUser("einstein", &recent, adminRoleID, spaceAdminRoleID)},
			expectedRoles: []roleCount{
				{Role: "Admin", Users: 1, SignedIn: count(1)},
				{Role: "Space Admin", Users: 1, SignedIn: count(1)},
			},
			expectedActive:       count(1),
			expectedWithSignInfo: 1,
		},
		{
			name:                 "unknown roles are reported by their id",
			users:                []libregraph.User{testUser("einstein", &recent, "some-unknown-role-id")},
			expectedRoles:        []roleCount{{Role: "some-unknown-role-id", Users: 1, SignedIn: count(1)}},
			expectedActive:       count(1),
			expectedWithSignInfo: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			roles, active, withSignInfo := countUsersByRole(tc.users, testRoleNames(), activeSince)

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
		ActiveUsers:         count(3),
		UsersWithSignInData: 4,
		Roles: []roleCount{
			{Role: "User", Users: 3, SignedIn: count(2)},
			{Role: "Admin", Users: 1, SignedIn: count(1)},
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
		ActiveUsers:         count(0),
		UsersWithSignInData: 5,
		Roles: []roleCount{
			{Role: "User", Users: 4, SignedIn: count(0)},
			{Role: "Admin", Users: 1, SignedIn: count(0)},
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
		GeneratedAt: time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC),
		Days:        31,
		ActiveSince: time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC),
		ActiveUsers: count(0),
		Roles:       []roleCount{},
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
