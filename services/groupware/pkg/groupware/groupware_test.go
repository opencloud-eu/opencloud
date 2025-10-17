package groupware

import (
	"testing"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/stretchr/testify/require"
)

func TestSanitizeEmail(t *testing.T) {
	email := jmap.Email{
		Subject: "test",
		BodyValues: map[string]jmap.EmailBodyValue{
			"koze92I1": {
				Value: `<a onblur="alert(secret)" href="http://www.google.com">Google</a>`,
			},
		},
		HtmlBody: []jmap.EmailBodyPart{
			{
				PartId: "koze92I1",
				Type:   "text/html",
				Size:   65,
			},
		},
	}

	g := &Groupware{sanitize: true}

	safe := g.sanitizeEmail(email)

	require := require.New(t)
	require.Equal(`<a href="http://www.google.com" rel="nofollow">Google</a>`, safe.BodyValues["koze92I1"].Value)
	require.Equal(57, safe.HtmlBody[0].Size)
}
