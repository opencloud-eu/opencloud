package opensearchtest

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"testing"

	opensearchgo "github.com/opensearch-project/opensearch-go/v4"
	opensearchgoAPI "github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"github.com/stretchr/testify/require"

	"github.com/opencloud-eu/opencloud/services/search/pkg/config"
)

type TestClient struct {
	c       *opensearchgoAPI.Client
	Require *testRequireClient
}

func NewDefaultTestClient(t testing.TB, cfg config.EngineOpenSearchClient) *TestClient {
	client, err := opensearchgoAPI.NewClient(opensearchgoAPI.Config{
		Client: opensearchgo.Config{
			Addresses: cfg.Addresses,
			Username:  cfg.Username,
			Password:  cfg.Password,
		},
	})
	require.NoError(t, err, "failed to create OpenSearch client")

	return NewTestClient(t, client)
}

func NewTestClient(t testing.TB, client *opensearchgoAPI.Client) *TestClient {
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
		exist, err := tc.IndicesExists(ctx, []string{index})
		if err != nil {
			return fmt.Errorf("failed to check if index %s exists: %w", index, err)
		}

		if !exist {
			continue
		}

		indicesToDelete = append(indicesToDelete, index)
	}

	if len(indicesToDelete) == 0 {
		return nil
	}

	return tc.IndicesDelete(ctx, indicesToDelete)
}

func (tc *TestClient) IndicesExists(ctx context.Context, indices []string) (bool, error) {
	if err := tc.IndicesRefresh(ctx, indices, []int{404}); err != nil {
		return false, err
	}

	resp, err := tc.c.Indices.Exists(ctx, opensearchgoAPI.IndicesExistsReq{
		Indices: indices,
	})
	switch {
	case resp != nil && resp.StatusCode == 404:
		return false, nil
	case err != nil:
		return false, fmt.Errorf("failed to check if indices exist: %w", err)
	case resp != nil && resp.IsError():
		return false, fmt.Errorf("failed to check if indices exist: %s", resp.String())
	default:
		return true, nil
	}
}

func (tc *TestClient) IndicesRefresh(ctx context.Context, indices []string, allow []int) error {
	resp, err := tc.c.Indices.Refresh(ctx, &opensearchgoAPI.IndicesRefreshReq{
		Index: indices,
	})

	isAllowed := resp != nil
	isAllowed = isAllowed && resp.Inspect().Response != nil
	isAllowed = isAllowed && slices.Contains(allow, resp.Inspect().Response.StatusCode)

	if err != nil && !isAllowed {
		return fmt.Errorf("failed to refresh indices %v: %w", indices, err)
	}

	return nil
}

func (tc *TestClient) IndicesDelete(ctx context.Context, indices []string) error {
	if err := tc.IndicesRefresh(ctx, indices, []int{}); err != nil {
		return err
	}

	resp, err := tc.c.Indices.Delete(ctx, opensearchgoAPI.IndicesDeleteReq{
		Indices: indices,
	})
	switch {
	case err != nil:
		return fmt.Errorf("failed to delete indices: %w", err)
	case !resp.Acknowledged:
		return errors.New("indices deletion not acknowledged")
	default:
		return nil
	}
}

func (tc *TestClient) IndicesCreate(ctx context.Context, index string, body io.Reader) error {
	resp, err := tc.c.Indices.Create(ctx, opensearchgoAPI.IndicesCreateReq{
		Index: index,
		Body:  body,
	})

	switch {
	case err != nil:
		return fmt.Errorf("failed to create index %s: %w", index, err)
	case !resp.Acknowledged:
		return fmt.Errorf("index creation not acknowledged for %s", index)
	default:
		return nil
	}
}

// IndicesCount returns the number of documents in the given indices.
func (tc *TestClient) IndicesCount(ctx context.Context, indices []string, body io.Reader) (int, error) {
	resp, err := tc.c.Indices.Count(ctx, &opensearchgoAPI.IndicesCountReq{
		Indices: indices,
		Body:    body,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to count documents in %v: %w", indices, err)
	}

	return resp.Count, nil
}

type testRequireClient struct {
	tc *TestClient
	t  testing.TB
}

func (trc *testRequireClient) IndicesReset(indices []string) {
	require.NoError(trc.t, trc.tc.IndicesReset(trc.t.Context(), indices))
}

func (trc *testRequireClient) IndicesRefresh(indices []string, ignore []int) {
	require.NoError(trc.t, trc.tc.IndicesRefresh(trc.t.Context(), indices, ignore))
}

func (trc *testRequireClient) IndicesCreate(index string, body io.Reader) {
	require.NoError(trc.t, trc.tc.IndicesCreate(trc.t.Context(), index, body))
}

func (trc *testRequireClient) IndicesDelete(indices []string) {
	require.NoError(trc.t, trc.tc.IndicesDelete(trc.t.Context(), indices))
}

func (trc *testRequireClient) IndicesCount(indices []string, body io.Reader, want int) {
	got, err := trc.tc.IndicesCount(trc.t.Context(), indices, body)
	require.NoError(trc.t, err)
	require.Equal(trc.t, want, got)
}
