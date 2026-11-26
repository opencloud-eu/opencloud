package svc

import (
	"testing"
	"time"

	"github.com/CiscoM31/godata"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type sortTest struct {
	Drives       []*libregraph.Drive
	Query        godata.GoDataRequest
	DrivesSorted []*libregraph.Drive
}

var time1 = time.Date(2022, 02, 02, 15, 00, 00, 00, time.UTC)
var time2 = time.Date(2022, 02, 03, 15, 00, 00, 00, time.UTC)
var time3, time5, time6 *time.Time
var time4 = time.Date(2022, 02, 05, 15, 00, 00, 00, time.UTC)
var drives = []*libregraph.Drive{
	drive("3", "project", "Admin", time3),
	drive("1", "project", "Alan", &time1),
	drive("2", "project", "Mary", &time2),
	drive("4", "project", "Margaret", &time4),
}
var drivesLong = append(drives, []*libregraph.Drive{
	drive("5", "project", "bob", time5),
	drive("6", "project", "alice", time6),
}...)

var sortTests = []sortTest{
	{
		Drives: drives,
		Query: godata.GoDataRequest{
			Query: &godata.GoDataQuery{
				OrderBy: &godata.GoDataOrderByQuery{
					OrderByItems: []*godata.OrderByItem{
						{Field: &godata.Token{Value: "name"}, Order: "asc"},
					},
				},
			},
		},
		DrivesSorted: []*libregraph.Drive{
			drive("3", "project", "Admin", time3),
			drive("1", "project", "Alan", &time1),
			drive("4", "project", "Margaret", &time4),
			drive("2", "project", "Mary", &time2),
		},
	},
	{
		Drives: drives,
		Query: godata.GoDataRequest{
			Query: &godata.GoDataQuery{
				OrderBy: &godata.GoDataOrderByQuery{
					OrderByItems: []*godata.OrderByItem{
						{Field: &godata.Token{Value: "name"}, Order: _sortDescending},
					},
				},
			},
		},
		DrivesSorted: []*libregraph.Drive{
			drive("2", "project", "Mary", &time2),
			drive("4", "project", "Margaret", &time4),
			drive("1", "project", "Alan", &time1),
			drive("3", "project", "Admin", time3),
		},
	},
	{
		Drives: drivesLong,
		Query: godata.GoDataRequest{
			Query: &godata.GoDataQuery{
				OrderBy: &godata.GoDataOrderByQuery{
					OrderByItems: []*godata.OrderByItem{
						{Field: &godata.Token{Value: "lastModifiedDateTime"}, Order: "asc"},
					},
				},
			},
		},
		DrivesSorted: []*libregraph.Drive{
			drive("3", "project", "Admin", time3),
			drive("6", "project", "alice", time6),
			drive("5", "project", "bob", time5),
			drive("1", "project", "Alan", &time1),
			drive("2", "project", "Mary", &time2),
			drive("4", "project", "Margaret", &time4),
		},
	},
	{
		Drives: drivesLong,
		Query: godata.GoDataRequest{
			Query: &godata.GoDataQuery{
				OrderBy: &godata.GoDataOrderByQuery{
					OrderByItems: []*godata.OrderByItem{
						{Field: &godata.Token{Value: "lastModifiedDateTime"}, Order: _sortDescending},
					},
				},
			},
		},
		DrivesSorted: []*libregraph.Drive{
			drive("4", "project", "Margaret", &time4),
			drive("2", "project", "Mary", &time2),
			drive("1", "project", "Alan", &time1),
			drive("5", "project", "bob", time5),
			drive("6", "project", "alice", time6),
			drive("3", "project", "Admin", time3),
		},
	},
}

func drive(ID string, dType string, name string, lastModified *time.Time) *libregraph.Drive {
	return &libregraph.Drive{Id: libregraph.PtrString(ID), DriveType: libregraph.PtrString(dType), Name: name, LastModifiedDateTime: lastModified, Quota: &libregraph.Quota{}}
}

// TestSort tests the available orderby queries
func TestSort(t *testing.T) {
	for _, test := range sortTests {
		sorted, err := sortSpaces(&test.Query, test.Drives)
		assert.NoError(t, err)
		assert.Equal(t, test.DrivesSorted, sorted)
	}
}

// TestSortNameNatural verifies that sorting drives by name uses a natural
// (numeric-aware) order, so "Project 10" sorts after "Project 2".
func TestSortNameNatural(t *testing.T) {
	input := []*libregraph.Drive{
		drive("a", "project", "Project 10", nil),
		drive("b", "project", "Project 2", nil),
		drive("c", "project", "Project 1", nil),
		drive("d", "project", "Project 10a", nil),
	}
	want := []string{"Project 1", "Project 2", "Project 10", "Project 10a"}
	query := godata.GoDataRequest{
		Query: &godata.GoDataQuery{
			OrderBy: &godata.GoDataOrderByQuery{
				OrderByItems: []*godata.OrderByItem{
					{Field: &godata.Token{Value: "name"}, Order: "asc"},
				},
			},
		},
	}
	sorted, err := sortSpaces(&query, input)
	require.NoError(t, err)
	got := make([]string, 0, len(sorted))
	for _, d := range sorted {
		got = append(got, d.GetName())
	}
	assert.Equal(t, want, got)

	// Descending must invert the natural order.
	query.Query.OrderBy.OrderByItems[0].Order = _sortDescending
	sortedDesc, err := sortSpaces(&query, input)
	require.NoError(t, err)
	gotDesc := make([]string, 0, len(sortedDesc))
	for _, d := range sortedDesc {
		gotDesc = append(gotDesc, d.GetName())
	}
	assert.Equal(t, []string{"Project 10a", "Project 10", "Project 2", "Project 1"}, gotDesc)
}

func TestSpaceNameValidation(t *testing.T) {
	// set max length
	_maxSpaceNameLength = 10

	testCases := []struct {
		Alias         string
		SpaceName     string
		ExpectedError error
	}{
		{"Happy Path", "Space", nil},
		{"Just not too Long", "abcdefghij", nil},
		{"Too Long", "abcdefghijk", ErrNameTooLong},
		{"Empty", "", ErrNameEmpty},
		{"Contains /", "Space/", ErrForbiddenCharacter},
		{`Contains \`, `Space\`, ErrForbiddenCharacter},
		{`Contains \\`, `Space\\`, ErrForbiddenCharacter},
		{"Contains .", "Space.", ErrForbiddenCharacter},
		{"Contains :", "Space:", ErrForbiddenCharacter},
		{"Contains ?", "Sp?ace", ErrForbiddenCharacter},
		{"Contains *", "Spa*ce", ErrForbiddenCharacter},
		{`Contains "`, `"Space"`, ErrForbiddenCharacter},
		{`Contains >`, `Sp>ce`, ErrForbiddenCharacter},
		{`Contains <`, `S<pce`, ErrForbiddenCharacter},
		{`Contains |`, `S|p|e`, ErrForbiddenCharacter},
	}

	for _, tc := range testCases {
		err := validateSpaceName(tc.SpaceName)
		require.Equal(t, tc.ExpectedError, err, tc.Alias)
	}

	// set max length back to protect other tests
	_maxSpaceNameLength = 255
}

func TestTrimSpaceName(t *testing.T) {
	// the names are written as escape sequences on purpose: written directly, these characters
	// would be invisible or look like ordinary spaces in the source code
	testCases := []struct {
		Alias     string
		SpaceName string
		Expected  string
	}{
		{"Unchanged", "Space", "Space"},
		{"Spaces around", "  Space  ", "Space"},
		{"Spaces inside are kept", "My Space", "My Space"},
		// U+00A0 no-break space, U+3000 ideographic space
		{"Unicode spaces around", "\u00a0Space\u3000", "Space"},
		// U+200B zero-width space, U+FEFF byte order mark
		{"Invisible characters around", "\u200bSpace\ufeff", "Space"},
		// U+2060 word joiner, U+2003 em space, U+200E left-to-right mark
		{"Mixed around", " \u2060\u2003Space\u200e ", "Space"},
		// U+200D zero-width joiner, as in the emoji sequence of a family
		{"Zero-width joiner inside is kept", "Team \U0001F468\u200d\U0001F469\u200d\U0001F467", "Team \U0001F468\u200d\U0001F469\u200d\U0001F467"},
		// U+200B zero-width space, U+200C zero-width non-joiner, U+00A0 no-break space
		{"Only invisible characters", "\u200b\u200c\u00a0", ""},
	}

	for _, tc := range testCases {
		require.Equal(t, tc.Expected, trimSpaceName(tc.SpaceName), tc.Alias)
	}
}
