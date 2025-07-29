package ostest

import (
	"context"
	"errors"
	"fmt"
	"slices"
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
	require.NoError(t, err, "failed to create OpenSearch client")

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
		// If no indices to delete, return nil
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
		Indices: indices,
	})

	if err != nil && !(resp != nil && slices.Contains(allow, resp.Inspect().Response.StatusCode)) {
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
	case resp.Acknowledged != true:
		return errors.New("indices deletion not acknowledged")
	default:
		return nil
	}
}

func (tc *TestClient) IndicesCount(ctx context.Context, indices []string, body string) (int, error) {
	if err := tc.IndicesRefresh(ctx, indices, []int{404}); err != nil {
		return 0, err
	}

	resp, err := tc.c.Indices.Count(ctx, &opensearchgoAPI.IndicesCountReq{
		Indices: indices,
		Body:    strings.NewReader(body),
	})

	switch {
	case err != nil:
		return 0, fmt.Errorf("failed to count documents in indices: %w", err)
	default:
		return resp.Count, nil
	}
}

func (tc *TestClient) DocumentCreate(ctx context.Context, index string, id, body string) error {
	if err := tc.IndicesRefresh(ctx, []string{index}, []int{404}); err != nil {
		return err
	}

	_, err := tc.c.Document.Create(ctx, opensearchgoAPI.DocumentCreateReq{
		Index:      index,
		DocumentID: id,
		Body:       strings.NewReader(body),
	})
	switch {
	case err != nil:
		return fmt.Errorf("failed to create document in index %s: %w", index, err)
	default:
		return nil
	}
}

type testRequireClient struct {
	tc *TestClient
	t  *testing.T
}

func (trc *testRequireClient) IndicesReset(indices []string) {
	require.NoError(trc.t, trc.tc.IndicesReset(trc.t.Context(), indices))
}

func (trc *testRequireClient) IndicesRefresh(indices []string, ignore []int) {
	require.NoError(trc.t, trc.tc.IndicesRefresh(trc.t.Context(), indices, ignore))
}

func (trc *testRequireClient) IndicesDelete(indices []string) {
	require.NoError(trc.t, trc.tc.IndicesDelete(trc.t.Context(), indices))
}

func (trc *testRequireClient) IndicesCount(indices []string, body string, expected int) {
	count, err := trc.tc.IndicesCount(trc.t.Context(), indices, body)

	switch {
	case expected <= 0:
		require.True(trc.t, count <= 0, "expected indices to have no documents, but got a count of %d", count)
	default:
		require.Equal(trc.t, expected, count, "expected indices to have %d documents, but got %d", expected, count)
		require.NoError(trc.t, err, "expected indices to have documents, but got an error")
	}
}

func (trc *testRequireClient) DocumentCreate(index string, id, body string) {
	require.NoError(trc.t, trc.tc.DocumentCreate(trc.t.Context(), index, id, body))
}
