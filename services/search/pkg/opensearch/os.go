package opensearch

import (
	"context"
	"fmt"
	"time"

	opensearchgoAPI "github.com/opensearch-project/opensearch-go/v4/opensearchapi"
)

func clusterHealth(ctx context.Context, client *opensearchgoAPI.Client, indices []string) (*opensearchgoAPI.ClusterHealthResp, bool, error) {
	resp, err := client.Cluster.Health(ctx, &opensearchgoAPI.ClusterHealthReq{
		Indices: indices,
		Params: opensearchgoAPI.ClusterHealthParams{
			Local:   opensearchgoAPI.ToPointer(true),
			Timeout: 5 * time.Second,
		},
	})
	switch {
	case err != nil:
		return nil, false, fmt.Errorf("%w, failed to get cluster health: %w", ErrUnhealthyCluster, err)
	case resp.TimedOut:
		return resp, false, fmt.Errorf("%w, cluster health request timed out", ErrUnhealthyCluster)

	case resp.Status != "green" && resp.Status != "yellow":
		return resp, false, fmt.Errorf("%w, cluster health is not green or yellow: %s", ErrUnhealthyCluster, resp.Status)
	default:
		return resp, true, nil
	}
}
