package groupware

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSort(t *testing.T) {
	req := Request{
		r:   &http.Request{},
		ctx: context.Background(),
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
