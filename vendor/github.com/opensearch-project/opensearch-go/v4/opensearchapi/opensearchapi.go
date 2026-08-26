// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/opensearch-project/opensearch-go/v4"
	"github.com/opensearch-project/opensearch-go/v4/errmask"
	"github.com/opensearch-project/opensearch-go/v4/internal/envvars"
	"github.com/opensearch-project/opensearch-go/v4/opensearchtransport"
)

// resolveErrorMask returns the effective [errmask.ErrorMask] for a Client
// by merging the cfg-supplied mask with the OPENSEARCH_GO_ERROR_MASK env var.
//
// In v4 the default is [errmask.All]: every partial-failure category is
// masked, preserving pre-bitfield behavior (no partial-failure errors
// returned). The resolver substitutes that default whenever
// [Config.Errors] is nil; a non-nil pointer is honored verbatim, so
// callers can opt in to "report everything" wholesale with
// `Errors: errmask.New()` or selectively with composite masks.
//
// Parsing is liberal (forward-compatible): tokens this binary doesn't
// recognize are skipped and reported via the debug logger, mirroring
// the policy used for OPENSEARCH_GO_ROUTING_CONFIG and
// OPENSEARCH_GO_DISCOVERY_CONFIG. An older client therefore tolerates
// new wrapper bits added by a newer release.
func resolveErrorMask(cfg Config) errmask.ErrorMask {
	var base errmask.ErrorMask
	if cfg.Errors == nil {
		base = errmask.All // v4 default: mask every partial-failure category (no behavior change)
	} else {
		base = *cfg.Errors
	}
	if v, ok := os.LookupEnv(envvars.ErrorMask); ok && v != "" {
		mask, unknown := errmask.Parse(v, base)
		if len(unknown) > 0 {
			if dl := opensearchtransport.LoadDebugLogger(); dl != nil {
				_ = dl.Logf("%s: ignored unknown tokens %q\n", envvars.ErrorMask, unknown)
			}
		}
		return mask
	}
	return base
}

// Config represents the client configuration
type Config struct {
	Client opensearch.Config

	// Errors masks specific categories of partial-failure errors so they
	// are NOT returned as typed Go errors. A set bit suppresses that
	// category; [errmask.Empty] reports every category.
	//
	//   nil               use this version's default: errmask.All
	//                     (mask everything; preserves pre-bitfield behavior)
	//   errmask.New()     caller wants every category reported
	//   errmask.New(errmask.All)
	//                     caller wants every category masked
	//
	// [errmask.None] and [errmask.Unknown] are doc-friendly aliases for
	// [errmask.Empty]; all three equal 0. Because the named values are
	// constants (not addressable), use [errmask.New] to build the pointer.
	// The pointer state is what disambiguates "use the default" from "caller
	// chose Empty".
	//
	// The OPENSEARCH_GO_ERROR_MASK environment variable can override
	// this value with comma-separated +/- tokens (see [errmask.Parse]).
	Errors *errmask.ErrorMask
}

// Client represents the opensearchapi Client summarizing all API calls
type Client struct {
	Client            *opensearch.Client
	errors            errmask.ErrorMask
	Cat               catClient
	Cluster           clusterClient
	Dangling          danglingClient
	Document          documentClient
	Indices           indicesClient
	Nodes             nodesClient
	Script            scriptClient
	ComponentTemplate componentTemplateClient
	IndexTemplate     indexTemplateClient
	// Deprecated: uses legacy API (/_template), correct API is /_index_template, use IndexTemplate instead
	Template    templateClient
	DataStream  dataStreamClient
	PointInTime pointInTimeClient
	Ingest      ingestClient
	Tasks       tasksClient
	Scroll      scrollClient
	Snapshot    snapshotClient
}

// clientInit inits the Client with all sub clients
func clientInit(rootClient *opensearch.Client, mask errmask.ErrorMask) *Client {
	client := &Client{
		Client: rootClient,
		errors: mask,
	}
	client.Cat = catClient{apiClient: client}
	client.Indices = indicesClient{
		apiClient: client,
		Alias:     aliasClient{apiClient: client},
		Mapping:   mappingClient{apiClient: client},
		Settings:  settingsClient{apiClient: client},
	}
	client.Nodes = nodesClient{apiClient: client}
	client.Cluster = clusterClient{apiClient: client}
	client.Dangling = danglingClient{apiClient: client}
	client.Script = scriptClient{apiClient: client}
	client.Document = documentClient{apiClient: client}
	client.ComponentTemplate = componentTemplateClient{apiClient: client}
	client.IndexTemplate = indexTemplateClient{apiClient: client}
	client.Template = templateClient{apiClient: client}
	client.DataStream = dataStreamClient{apiClient: client}
	client.PointInTime = pointInTimeClient{apiClient: client}
	client.Ingest = ingestClient{apiClient: client}
	client.Tasks = tasksClient{apiClient: client}
	client.Scroll = scrollClient{apiClient: client}
	client.Snapshot = snapshotClient{
		apiClient:  client,
		Repository: repositoryClient{apiClient: client},
	}

	return client
}

// NewClient returns a opensearchapi client
func NewClient(config Config) (*Client, error) {
	rootClient, err := opensearch.NewClient(config.Client)
	if err != nil {
		return nil, err
	}

	return clientInit(rootClient, resolveErrorMask(config)), nil
}

// NewDefaultClient returns a opensearchapi client using defaults
func NewDefaultClient() (*Client, error) {
	defaultAddress := opensearch.DefaultScheme + "://" + net.JoinHostPort(opensearch.DefaultHost, strconv.Itoa(opensearch.DefaultPort))
	rootClient, err := opensearch.NewClient(opensearch.Config{
		Addresses: []string{defaultAddress},
	})
	if err != nil {
		return nil, err
	}

	return clientInit(rootClient, resolveErrorMask(Config{})), nil
}

// Close releases the client's background resources by closing the underlying
// opensearch.Client. Safe on a zero value and idempotent.
func (c *Client) Close() error {
	if c.Client != nil {
		return c.Client.Close()
	}
	return nil
}

// NewFromClient creates an opensearchapi client from an existing opensearch.Client.
// In v4 this preserves the legacy "mask everything" default; use NewClient
// with Config to enable partial-failure errors.
func NewFromClient(client *opensearch.Client) *Client {
	return clientInit(client, resolveErrorMask(Config{}))
}

// NewFromClientWithErrors creates an opensearchapi client from an existing
// opensearch.Client with every partial-failure category reported as an error.
func NewFromClientWithErrors(client *opensearch.Client) *Client {
	return clientInit(client, resolveErrorMask(Config{Errors: errmask.New()}))
}

// do calls [opensearch.Do] and checks the response for OpenSearch API errors.
// The generic *T parameter enforces that dataPointer is a pointer at compile time.
//
// [opensearch.Do] routes through the buffered [opensearchtransport.Client.Perform],
// so resp.Body here is already an [io.NopCloser] over a [bytes.Reader] -- the
// connection has been drained and returned to the pool. The helper only needs
// to translate IsError into a typed error.
func do[T any](ctx context.Context, c *Client, method string, req opensearch.Request, dataPointer *T) (*opensearch.Response, error) {
	resp, err := opensearch.Do(ctx, c.Client, method, req, dataPointer)
	if err != nil {
		return nil, err
	}

	if resp.IsError() {
		if dataPointer == nil && resp.Body == nil {
			return resp, fmt.Errorf("status: %q", resp.Status())
		}
		return resp, opensearch.ParseError(resp)
	}

	return resp, nil
}

// noBody is a typed-nil sentinel passed to do() when the caller does not
// expect a response body. The nil pointer triggers the dataPointer == nil
// guard in [opensearch.Do], skipping json.Unmarshal.
var noBody *opensearch.NoBody //nolint:gochecknoglobals // typed-nil sentinel shared by callers that expect no response body

// formatDuration converts duration to a string in the format
// accepted by Opensearch.
func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return strconv.FormatInt(int64(d), 10) + "nanos"
	}

	return strconv.FormatInt(int64(d)/int64(time.Millisecond), 10) + "ms"
}

// ToPointer converts any value to a pointer, mainly used for request parameters
//
// Deprecated: ToPointer will be removed in v5. The helper is intentionally not
// part of the public API going forward; consumers within this module use the
// unexported `ptr` defined per-package. Once the module's go directive moves
// to 1.26, callers can drop any wrapper in favor of the native new(value)
// form (e.g. new(false)).
func ToPointer[V any](value V) *V {
	return ptr(value)
}

// ptr returns a pointer to a copy of value. Used for the *T query/body
// parameter pattern. Unexported by design.
//
// Once the module's go directive moves to 1.26, this helper can be deleted
// and call sites can switch to the native new(value) form: new(false).
func ptr[V any](value V) *V {
	return &value
}

// ResponseShards is a sub type of api repsonses containing information about shards
type ResponseShards struct {
	Total      int                     `json:"total"`
	Successful int                     `json:"successful"`
	Failed     int                     `json:"failed"`
	Failures   []ResponseShardsFailure `json:"failures"`
	Skipped    int                     `json:"skipped"`
}

// ResponseShardsFailure is a sub type of ResponseShards containing information about a failed shard
type ResponseShardsFailure struct {
	Shard   int    `json:"shard"`
	Index   any    `json:"index"`
	Node    string `json:"node"`
	Status  string `json:"status"`
	Primary bool   `json:"primary"`
	Reason  struct {
		Type   string `json:"type"`
		Reason string `json:"reason"`
	} `json:"reason"`
}

// DocumentError represents an error for an individual item in a multi-document response.
// Used by MGet, MTermvectors, MSearch, and similar APIs where each item can fail independently.
type DocumentError struct {
	Type      string          `json:"type"`
	Reason    string          `json:"reason"`
	Index     string          `json:"index,omitempty"`
	IndexUUID string          `json:"index_uuid,omitempty"`
	RootCause []DocumentError `json:"root_cause,omitempty"`
	CausedBy  *DocumentError  `json:"caused_by,omitempty"`
}

// BulkByScrollFailure represents a failure item in _delete_by_query, _update_by_query,
// and _reindex responses. Can represent either a bulk write failure or a search scroll failure.
type BulkByScrollFailure struct {
	Index  string         `json:"index"`
	ID     string         `json:"id,omitempty"`    // Bulk failures only
	Shard  *int           `json:"shard,omitempty"` // Search failures only
	Node   string         `json:"node,omitempty"`  // Search failures only
	Status int            `json:"status"`
	Cause  *DocumentError `json:"cause,omitempty"`  // Bulk failures: the exception that caused the failure
	Reason *DocumentError `json:"reason,omitempty"` // Search failures: the exception that caused the failure
}

// FailuresCause contains information about failure cause
type FailuresCause struct {
	Type   string `json:"type"`
	Reason string `json:"reason"`
	NodeID string `json:"node_id"`
	Cause  *struct {
		Type   string `json:"type"`
		Reason string `json:"reason"`
		Cause  *struct {
			Type   string `json:"type"`
			Reason string `json:"reason"`
			Cause  *struct {
				Type   string  `json:"type"`
				Reason *string `json:"reason"`
			} `json:"caused_by,omitempty"`
		} `json:"caused_by,omitempty"`
	} `json:"caused_by,omitempty"`
}

// FailuresShard contains information about shard failures
type FailuresShard struct {
	Node   *string       `json:"node,omitempty"` // Node ID where the failure occurred (present since OpenSearch 1.0.0)
	Shard  int           `json:"shard"`
	Index  string        `json:"index"`
	Status string        `json:"status"`
	Reason FailuresCause `json:"reason"`
}
