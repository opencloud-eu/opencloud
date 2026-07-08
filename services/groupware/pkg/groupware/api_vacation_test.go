package groupware

import (
	"testing"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/stretchr/testify/require"
)

func TestGroupwareVacation(t *testing.T) {
	require := require.New(t)
	g, err := newGroupwareTest(t)
	require.NoError(err)

	{
		resp := jmap.VacationResponse{}
		gget("get-initial-vacation", g, "/accounts/_/vacation", &resp)
		require.Equal("singleton", resp.Id)
		require.False(resp.IsEnabled)
		require.Empty(resp.Subject)
		require.Empty(resp.TextBody)
		require.Empty(resp.HtmlBody)
		require.Nil(resp.FromDate)
		require.Nil(resp.ToDate)
	}

	{
		resp := jmap.VacationResponse{}
		req := jmap.VacationResponseChange{
			IsEnabled: true,
			Subject:   "testing",
			TextBody:  "text",
			HtmlBody:  "html",
		}
		gput("set-vacation", g, "/accounts/_/vacation", req, &resp)
		require.Equal("singleton", resp.Id)
		require.True(resp.IsEnabled)
		require.Equal("testing", resp.Subject)
		require.Equal("text", resp.TextBody)
		require.Equal("html", resp.HtmlBody)
		require.Nil(resp.FromDate)
		require.Nil(resp.ToDate)
	}
}
