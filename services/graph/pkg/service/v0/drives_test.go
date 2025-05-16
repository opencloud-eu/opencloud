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
