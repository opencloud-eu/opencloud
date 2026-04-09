package groupware

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSort(t *testing.T) {
	req := Request{
		r:    &http.Request{},
		cotx: t.Context(),
	}
	require := require.New(t)
	{
		res, err := req.parseSort("name", []string{"name", "time"})
		require.Nil(err)
		require.Len(res, 1)
		require.Equal("name", res[0].Attribute)
		require.True(res[0].Ascending)
	}
	{
		res, err := req.parseSort("name:asc", []string{"name"})
		require.Nil(err)
		require.Len(res, 1)
		require.Equal("name", res[0].Attribute)
		require.True(res[0].Ascending)
	}
	{
		res, err := req.parseSort("name:desc", []string{"name"})
		require.Nil(err)
		require.Len(res, 1)
		require.Equal("name", res[0].Attribute)
		require.False(res[0].Ascending)
	}
	{
		res, err := req.parseSort("name:", []string{"name"})
		require.Nil(err)
		require.Len(res, 1)
		require.Equal("name", res[0].Attribute)
		require.True(res[0].Ascending)
	}
	{
		_, err := req.parseSort("name:xyz", []string{"name"})
		require.NotNil(err)
		require.Equal(ErrorCodeInvalidSortSpecification, err.Code)
	}
	{
		_, err := req.parseSort("age", []string{"name"})
		require.NotNil(err)
		require.Equal(ErrorCodeInvalidSortProperty, err.Code)
	}
	{
		res, err := req.parseSort("name:asc,updated:desc", []string{"name", "updated"})
		require.Nil(err)
		require.Len(res, 2)
		require.Equal("name", res[0].Attribute)
		require.True(res[0].Ascending)
		require.Equal("updated", res[1].Attribute)
		require.False(res[1].Ascending)
	}
}

func TestParseMap(t *testing.T) {
	require := require.New(t)
	for _, tt := range []struct {
		uri string
		ok  bool
		out map[string]string
	}{
		{"/foo", false, map[string]string{}},
		{"/foo?name=camina", false, map[string]string{}},
		{"/foo?map=camina", false, map[string]string{}},
		{"/foo?map.name=camina", true, map[string]string{"name": "camina"}},
		{"/foo?map.gn=camina&map.sn=drummer", true, map[string]string{"gn": "camina", "sn": "drummer"}},
		{"/foo?map.name=camina&map.name=chrissie", true, map[string]string{"name": "chrissie"}},
	} {
		t.Run(fmt.Sprintf("uri:%s", tt.uri), func(t *testing.T) {
			var req Request
			{
				u, err := url.Parse(tt.uri)
				require.NoError(err)
				req = Request{r: &http.Request{URL: u}, cotx: t.Context()}
			}
			res, ok, err := req.parseMapParam("map")
			require.Nil(err)
			if tt.ok {
				require.True(ok)
				require.Equal(res, tt.out)
			} else {
				require.False(ok)
				require.Empty(res)
			}
		})
	}
}
