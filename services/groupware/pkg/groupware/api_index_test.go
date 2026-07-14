package groupware

import (
	"testing"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/structs"
	"github.com/stretchr/testify/require"
)

func TestGroupwareIndex(t *testing.T) {
	require := require.New(t)
	g, err := newGroupwareTest(t)
	require.NoError(err)
	u := g.user()

	index := IndexResponse{}
	gget(g, "get-index", u, "/", &index)
	require.Len(index.Accounts, 1)
	require.Equal(u.Email, index.Accounts[0].Name)
	require.Len(index.Accounts[0].Identities, 2) // email + alias
	require.Len(structs.Filter(index.Accounts[0].Identities, func(i jmap.Identity) bool { return i.Email == u.Email }), 1)
	require.Len(structs.Filter(index.Accounts[0].Identities, func(i jmap.Identity) bool { return i.Email == u.Alias }), 1)
}
