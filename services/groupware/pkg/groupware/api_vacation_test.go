package groupware

import (
	"testing"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/stretchr/testify/require"
)

func TestGroupwareVacation(t *testing.T) {
	require := require.New(t)
	g, err := newGroupwareTest(t)
	require.NoError(err)
	u := g.user()

	{
		resp := jmap.VacationResponse{}
		gget(g, "get-initial-vacation", u, "/accounts/_/vacation", &resp)
		require.Equal("singleton", resp.Id)
		require.False(*resp.IsEnabled)
		require.Empty(resp.Subject)
		require.Empty(resp.TextBody)
		require.Empty(resp.HtmlBody)
		require.Nil(resp.FromDate)
		require.Nil(resp.ToDate)
	}

	{
		resp := jmap.VacationResponse{}
		req := jmap.VacationResponseChange{
			IsEnabled: ptr(true),
			Subject:   "testing",
			TextBody:  "text",
			HtmlBody:  "html",
		}
		gput(g, "set-vacation", u, "/accounts/_/vacation", req, &resp)
		require.Equal("singleton", resp.Id)
		require.True(*resp.IsEnabled)
		require.Equal("testing", resp.Subject)
		require.Equal("text", resp.TextBody)
		require.Equal("html", resp.HtmlBody)
		require.Nil(resp.FromDate)
		require.Nil(resp.ToDate)
	}

	{
		resp := jmap.VacationResponse{}
		gget(g, "get-initial-vacation-after-change", u, "/accounts/_/vacation", &resp)
		require.Equal("singleton", resp.Id)
		require.True(*resp.IsEnabled)
		require.Equal("testing", resp.Subject)
		require.Equal("text", resp.TextBody)
		require.Equal("html", resp.HtmlBody)
		require.Nil(resp.FromDate)
		require.Nil(resp.ToDate)
	}

	{
		resp := jmap.VacationResponse{}
		from, err := time.Parse(time.RFC3339, "2026-07-08T00:00:00Z")
		require.NoError(err)
		to, err := time.Parse(time.RFC3339, "2026-07-18T23:59:59.999Z")
		require.NoError(err)
		req := jmap.VacationResponseChange{
			IsEnabled: nil,
			FromDate:  &from,
			ToDate:    &to,
		}
		gput(g, "set-vacation", u, "/accounts/_/vacation", req, &resp)
		require.Equal("singleton", resp.Id)
		require.False(*resp.IsEnabled) // TODO should be true: this is a bug in Stalwart up to 0.16.12, see https://support.stalw.art/t/vacationresponse-jmap-api-isenabled-reset-to-false-whenever-properties-are-changed/992
		require.Equal("testing", resp.Subject)
		require.Equal("text", resp.TextBody)
		require.Equal("html", resp.HtmlBody)
		require.Equal("2026-07-08T00:00:00Z", resp.FromDate.Format(time.RFC3339))
		require.Equal("2026-07-18T23:59:59Z", resp.ToDate.Format(time.RFC3339)) // for some reason the .999 gets lost in Stalwart (?), but nothing critical
	}
}
