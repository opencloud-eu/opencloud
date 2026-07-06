package groupware

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGroupwareIndex(t *testing.T) {
	require := require.New(t)
	g, err := newGroupwareTest(t)
	require.NoError(err)
	require.NotEmpty(g.Users)

	client := &http.Client{}
	for _, u := range g.Users {
		req, err := http.NewRequest(http.MethodGet, g.BaseURL, nil)
		require.NoError(err)
		req.SetBasicAuth(u.Name, u.Password)
		resp, err := client.Do(req)
		require.NoError(err)
		require.Equal(200, resp.StatusCode)
		defer resp.Body.Close()
		index := IndexResponse{}
		err = json.NewDecoder(resp.Body).Decode(&index)
		require.NoError(err)
		require.Len(index.Accounts, 1)
		require.Equal(u.Email, index.Accounts[0].Name)
		require.Len(index.Accounts[0].Identities, 2) // email + alias
	}
}
