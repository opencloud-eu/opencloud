package groupware

import (
	"slices"
	"testing"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/structs"
	"github.com/stretchr/testify/require"
)

func TestSanitizeEmail(t *testing.T) {
	email := jmap.Email{
		Subject: "test",
		BodyValues: map[string]jmap.EmailBodyValue{
			"koze92I1": {
				Value: `<a onblur="alert(secret)" href="http://www.cyberdyne.com">Cyberdyne</a>`,
			},
			"zee7urae": {
				Value: `Hello. <a onblur="hack()" href="file:///download.exe">Click here</a> for AI slop.`,
			},
		},
		HtmlBody: []jmap.EmailBodyPart{
			{
				PartId: "koze92I1",
				Type:   "text/html",
				Size:   71,
			},
			{
				PartId: "zee7urae",
				Type:   "text/html",
				Size:   81,
			},
		},
	}

	g := &Groupware{sanitize: true}
	req := Request{g: g}

	safe, err := req.sanitizeEmail(email)

	require := require.New(t)
	require.Nil(err)
	require.Equal(`<a href="http://www.cyberdyne.com" rel="nofollow">Cyberdyne</a>`, safe.BodyValues["koze92I1"].Value)
	require.Equal(63, safe.HtmlBody[0].Size)
	require.Equal(`Hello. Click here for AI slop.`, safe.BodyValues["zee7urae"].Value)
	require.Equal(30, safe.HtmlBody[1].Size)
}

func TestSortMailboxes(t *testing.T) {
	mailboxes := []jmap.Mailbox{
		{Id: "a", Name: "Other"},
		{Id: "b", Role: jmap.JmapMailboxRoleSent, Name: "Sent"},
		{Id: "c", Name: "Zebras"},
		{Id: "d", Role: jmap.JmapMailboxRoleInbox, Name: "Inbox"},
		{Id: "e", Name: "Appraisal"},
		{Id: "f", Name: "Zealots", SortOrder: -10},
	}
	slices.SortFunc(mailboxes, compareMailboxes)
	names := structs.Map(mailboxes, func(m jmap.Mailbox) string { return m.Name })
	require := require.New(t)
	require.Equal([]string{"Zealots", "Inbox", "Sent", "Appraisal", "Other", "Zebras"}, names)
}
