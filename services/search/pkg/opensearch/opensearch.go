package opensearch

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"dario.cat/mergo"
	opensearchgoAPI "github.com/opensearch-project/opensearch-go/v4/opensearchapi"
)

var (
	ErrUnhealthyCluster = fmt.Errorf("cluster is not healthy")
)

func isEmpty(x any) bool {
	switch {
	case x == nil:
		return true
	case reflect.ValueOf(x).Kind() == reflect.Bool:
		return false
	case reflect.DeepEqual(x, reflect.Zero(reflect.TypeOf(x)).Interface()):
		return true
	case reflect.ValueOf(x).Kind() == reflect.Map && reflect.ValueOf(x).Len() == 0:
		return true
	default:
		return false
	}
}

func merge[T any](options ...T) T {
	mapOptions := make(map[string]any)

	for _, option := range options {
		data, err := convert[map[string]any](option)
		if err != nil {
			continue
		}

		_ = mergo.Merge(&mapOptions, data)
	}

	data, _ := convert[T](mapOptions)

	return data
}

func convert[T any](v any) (T, error) {
	var t T

	if v == nil {
		return t, nil
	}

	j, err := json.Marshal(v)
	if err != nil {
		return t, err
	}

	if err := json.Unmarshal(j, &t); err != nil {
		return t, err
	}

	return t, nil
}

func clusterHealth(ctx context.Context, client *opensearchgoAPI.Client, indices []string) (*opensearchgoAPI.ClusterHealthResp, bool, error) {
	resp, err := client.Cluster.Health(ctx, &opensearchgoAPI.ClusterHealthReq{
		Indices: indices,
		Params: opensearchgoAPI.ClusterHealthParams{
			Local:   opensearchgoAPI.ToPointer(true),
			Timeout: 5 * time.Second,
		},
	})
	if err != nil {
		return nil, false, fmt.Errorf("%w, failed to get cluster health: %w", ErrUnhealthyCluster, err)
	}

	if resp.TimedOut {
		return resp, false, fmt.Errorf("%w, cluster health request timed out", ErrUnhealthyCluster)
	}

	if resp.Status != "green" && resp.Status != "yellow" {
		return resp, false, fmt.Errorf("%w, cluster health is not green or yellow: %s", ErrUnhealthyCluster, resp.Status)
	}

	return resp, true, nil
}
