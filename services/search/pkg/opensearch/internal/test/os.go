package ostest

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	opensearchgoAPI "github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"github.com/stretchr/testify/require"
)

type TestClient struct {
	c       *opensearchgoAPI.Client
	Require *testRequireClient
}

func NewDefaultTestClient(t *testing.T) *TestClient {
	client, err := opensearchgoAPI.NewDefaultClient()
	require.NoError(t, err, "Failed to create OpenSearch client")

	return NewTestClient(t, client)
}

func NewTestClient(t *testing.T, client *opensearchgoAPI.Client) *TestClient {
	tc := &TestClient{c: client}
	trc := &testRequireClient{tc: tc, t: t}
	tc.Require = trc

	return tc
}

func (tc *TestClient) Client() *opensearchgoAPI.Client {
	return tc.c
}

func (tc *TestClient) IndicesReset(ctx context.Context, indices []string) error {
	indicesToDelete := make([]string, 0, len(indices))
	for _, index := range indices {
		if err := tc.IndicesRefresh(ctx, indices); err != nil {
			continue
		}

		indicesToDelete = append(indicesToDelete, index)
	}

	if len(indicesToDelete) == 0 {
		// If no indices to delete, return nil
		return nil
	}

	return tc.IndicesDelete(ctx, indicesToDelete)
}

func (tc *TestClient) IndicesExists(ctx context.Context, indices []string) (bool, error) {
	if err := tc.IndicesRefresh(ctx, indices); err != nil {
		return false, err
	}

	resp, err := tc.c.Indices.Exists(ctx, opensearchgoAPI.IndicesExistsReq{
		Indices: indices,
	})
	switch {
	case err != nil:
		return false, err
	case resp.IsError():
		return false, fmt.Errorf("failed to check if indices exist: %s", resp.String())
	default:
		return true, nil
	}
}

func (tc *TestClient) IndicesRefresh(ctx context.Context, indices []string) error {
	_, err := tc.c.Indices.Refresh(ctx, &opensearchgoAPI.IndicesRefreshReq{
		Indices: indices,
	})

	return err
}

func (tc *TestClient) IndicesDelete(ctx context.Context, indices []string) error {
	if err := tc.IndicesRefresh(ctx, indices); err != nil {
		return err
	}

	resp, err := tc.c.Indices.Delete(ctx, opensearchgoAPI.IndicesDeleteReq{
		Indices: indices,
	})
	switch {
	case err != nil:
		return fmt.Errorf("failed to delete indices: %w", err)
	case resp.Acknowledged != true:
		return errors.New("indices deletion not acknowledged")
	default:
		return nil
	}
}

func (tc *TestClient) IndicesCount(ctx context.Context, indices []string) (int, error) {
	if err := tc.IndicesRefresh(ctx, indices); err != nil {
		return 0, err
	}

	resp, err := tc.c.Indices.Count(ctx, &opensearchgoAPI.IndicesCountReq{
		Indices: indices,
	})

	switch {
	case err != nil:
		return 0, fmt.Errorf("failed to count documents in indices: %w", err)
	default:
		return resp.Count, nil
	}
}

func (tc *TestClient) IndexCreate(ctx context.Context, index string, body string) error {
	resp, err := tc.c.Indices.Create(ctx, opensearchgoAPI.IndicesCreateReq{
		Index: index,
		Body:  strings.NewReader(body),
	})

	switch {
	case err != nil:
		return fmt.Errorf("failed to create index %s: %w", index, err)
	case !resp.Acknowledged:
		return fmt.Errorf("index creation not acknowledged for index %s", index)
	default:
		return nil
	}
}

type testRequireClient struct {
	tc *TestClient
	t  *testing.T
}

func (trc *testRequireClient) IndicesReset(indices []string) {
	require.NoError(trc.t, trc.tc.IndicesReset(trc.t.Context(), indices), "Failed to reset indices")
}

func (trc *testRequireClient) IndicesExists(indices []string, expected bool) {
	exist, err := trc.tc.IndicesExists(trc.t.Context(), indices)
	switch {
	case expected == true:
		require.NoError(trc.t, err, "Expected indices to exist, but got an error")
		require.True(trc.t, exist, "Expected indices to exist, but got an error response")
	default:
		require.Error(trc.t, err)
		require.False(trc.t, exist, "Expected indices to not exist, but got an error response")
	}
}

func (trc *testRequireClient) IndicesRefresh(indices []string) {
	require.NoError(trc.t, trc.tc.IndicesRefresh(trc.t.Context(), indices), "Failed to refresh indices")
}

func (trc *testRequireClient) IndicesDelete(indices []string) {
	require.NoError(trc.t, trc.tc.IndicesDelete(trc.t.Context(), indices), "Failed to delete indices")
}

func (trc *testRequireClient) IndicesCount(indices []string, expected int) {
	count, err := trc.tc.IndicesCount(trc.t.Context(), indices)

	switch {
	case expected <= 0:
		require.True(trc.t, count <= 0, "Expected indices to have no documents, but got a count of %d", count)
	default:
		require.Equal(trc.t, expected, count, "Expected indices to have %d documents, but got %d", expected, count)
		require.NoError(trc.t, err, "Expected indices to have documents, but got an error")
	}
}

func (trc *testRequireClient) IndexCreate(index string, body string) {
	require.NoError(trc.t, trc.tc.IndexCreate(trc.t.Context(), index, body), "Failed to create index %s", index)
}
