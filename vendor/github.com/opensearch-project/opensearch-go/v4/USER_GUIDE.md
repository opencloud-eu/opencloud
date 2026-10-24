- [User Guide](#user-guide)
  - [Example](#example)
  - [Amazon OpenSearch Service](#amazon-opensearch-service)
    - [AWS SDK v1](#aws-sdk-v1)
    - [AWS SDK v2](#aws-sdk-v2)
  - [Custom Transport](#custom-transport)
  - [Guides by Topic](#guides-by-topic)

# User Guide

## Example

In the example below, we create a client, an index with non-default settings, insert a document to the index, search for the document, delete the document and finally delete the index.

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/opensearch-project/opensearch-go/v4"
	"github.com/opensearch-project/opensearch-go/v4/errmask"
	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"github.com/opensearch-project/opensearch-go/v4/opensearchtransport"
	"github.com/opensearch-project/opensearch-go/v4/opensearchutil"
)

const IndexName = "go-test-index1"

func main() {
	if err := example(); err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
}

func example() error {
	// Initialize the client with SSL/TLS enabled.
	router, err := opensearchtransport.NewDefaultRouter()
	if err != nil {
		return err
	}

	// Surface partial failures (bulk item errors, shard failures) as Go
	// errors. See guides/error_handling.md for the per-category bitmask.
	errMask := errmask.Empty
	discoverOnStart := true
	client, err := opensearchapi.NewClient(
		opensearchapi.Config{
			Client: opensearch.Config{
				InsecureSkipVerify: true, // For testing only. Use certificate for validation.
				Addresses:          []string{"https://localhost:9200"},
				Username:           "admin", // For testing only. Don't store credentials in code.
				Password:           "myStrongPassword123!",

				// Optional: Enable node discovery
				DiscoverNodesOnStart:  &discoverOnStart,
				DiscoverNodesInterval: 5 * time.Minute,

				// Optional: Enable intelligent request routing
				Router: router,
			},

			// Optional: Surface partial failures (bulk item errors, shard failures)
			// as Go errors. See guides/error_handling.md for details.
			Errors: &errMask,
		},
	)
	if err != nil {
		return err
	}

	ctx := context.Background()
	// Print OpenSearch version information on console.
	infoResp, err := client.Info(ctx, nil)
	if err != nil {
		return err
	}
	fmt.Printf("Cluster INFO:\n  Cluster Name: %s\n  Cluster UUID: %s\n  Version Number: %s\n", infoResp.ClusterName, infoResp.ClusterUUID, infoResp.Version.Number)

	// Define index mapping.
	// Note: these particular settings (eg, shards/replicas)
	// will have no effect in AWS OpenSearch Serverless
	mapping := strings.NewReader(`{
	    "settings": {
	        "index": {
	            "number_of_shards": 4
	        }
	    }
	}`)

	// Create an index with non-default settings.
	createIndexResponse, err := client.Indices.Create(
		ctx,
		opensearchapi.IndicesCreateReq{
			Index: IndexName,
			Body:  mapping,
		},
	)

	var opensearchError *opensearch.StructError

	// Load err into opensearch.StructError to access the fields and tolerate if the index already exists
	if err != nil {
		if errors.As(err, &opensearchError) {
			if opensearchError.Err.Type != "resource_already_exists_exception" {
				return err
			}
		} else {
			return err
		}
	}
	fmt.Printf("Created Index: %s\n  Shards Acknowledged: %t\n", createIndexResponse.Index, createIndexResponse.ShardsAcknowledged)

	// When using a structure, the conversion process to io.Reader can be omitted using utility functions.
	document := struct {
		Title    string `json:"title"`
		Director string `json:"director"`
		Year     string `json:"year"`
	}{
		Title:    "Moneyball",
		Director: "Bennett Miller",
		Year:     "2011",
	}

	docId := "1"
	insertResp, err := client.Index(
		ctx,
		opensearchapi.IndexReq{
			Index:      IndexName,
			DocumentID: docId,
			Body:       opensearchutil.NewJSONReader(&document),
			Params: opensearchapi.IndexParams{
				Refresh: "true",
			},
		},
	)
	if err != nil {
		return err
	}
	fmt.Printf("Created document in %s\n  ID: %s\n", insertResp.Index, insertResp.ID)

	// Search for the document.
	content := strings.NewReader(`{
		"size": 5,
	    "query": {
	        "multi_match": {
	            "query": "miller",
	            "fields": ["title^2", "director"]
	        }
	    }
	}`)

	searchResp, err := client.Search(
		ctx,
		&opensearchapi.SearchReq{
			Body: content,
		},
	)
	if err != nil {
		return err
	}
	fmt.Printf("Search hits: %v\n", searchResp.Hits.Total.Value)

	if searchResp.Hits.Total.Value > 0 {
		indices := make([]string, 0)
		for _, hit := range searchResp.Hits.Hits {
			add := true
			for _, index := range indices {
				if index == hit.Index {
					add = false
				}
			}
			if add {
				indices = append(indices, hit.Index)
			}
		}
		fmt.Printf("Search indices: %s\n", strings.Join(indices, ","))
	}

	// Delete the document.
	deleteReq := opensearchapi.DocumentDeleteReq{
		Index:      IndexName,
		DocumentID: docId,
	}

	deleteResponse, err := client.Document.Delete(ctx, deleteReq)
	if err != nil {
		return err
	}
	fmt.Printf("Deleted document: %t\n", deleteResponse.Result == "deleted")

	// Delete previously created index.
	deleteIndex := opensearchapi.IndicesDeleteReq{Indices: []string{IndexName}}

	deleteIndexResp, err := client.Indices.Delete(ctx, deleteIndex)
	if err != nil {
		return err
	}
	fmt.Printf("Deleted index: %t\n", deleteIndexResp.Acknowledged)

	// Try to delete the index again which fails as it does not exist
	_, err = client.Indices.Delete(ctx, deleteIndex)

	// Load err into opensearch.StructError to access the fields and tolerate if the index is missing
	if err != nil {
		if errors.As(err, &opensearchError) {
			if opensearchError.Err.Type != "index_not_found_exception" {
				return err
			}
		} else {
			return err
		}
	}
	return nil
}

```

## Amazon OpenSearch Service

Before starting, we strongly recommend reading the full AWS documentation regarding using IAM credentials to sign requests to OpenSearch APIs. See [Identity and Access Management in Amazon OpenSearch Service.](https://docs.aws.amazon.com/opensearch-service/latest/developerguide/ac.html)

> Even if you configure a completely open resource-based access policy, all requests to the OpenSearch Service configuration API must be signed. If your policies specify IAM users or roles, requests to the OpenSearch APIs also must be signed using AWS Signature Version 4.
>
> See [Managed Domains signing-service requests.](https://docs.aws.amazon.com/opensearch-service/latest/developerguide/ac.html#managedomains-signing-service-requests)

Depending on the version of AWS SDK used, import the request signer from `signer/aws` (recommended) or `signer/awsv2`. Both signers use AWS SDK v2 and provide AWS Signature Version 4 (SigV4).

**BREAKING CHANGE**: As of this version, the main `signer/aws` package has been migrated from AWS SDK v1 to AWS SDK v2 due to AWS SDK v1 reaching end-of-support on July 31, 2025.

To read more about SigV4 see [Signature Version 4 signing process](https://docs.aws.amazon.com/general/latest/gr/signature-version-4.html)

Here are some Go samples that show how to sign each OpenSearch request and automatically search for AWS credentials from the ~/.aws folder or environment variables:

### AWS SDK v2 (Recommended)

**Migration Note**: If you were previously using `signer/aws` with AWS SDK v1, you need to update your imports and configuration as shown below.

**Credential Caching**: The signer automatically enables credential caching for improved performance, especially when using STS credentials (assume role, web identity, etc.). This reduces API calls to AWS STS and improves signing performance.

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	requestsigner "github.com/opensearch-project/opensearch-go/v4/signer/aws"

	"github.com/opensearch-project/opensearch-go/v4"
	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
)

const IndexName = "go-test-index1"

func main() {
	if err := example(); err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
}

const endpoint = "" // e.g. https://opensearch-domain.region.com

func example() error {
	ctx := context.Background()

	// Load AWS configuration
	awsCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}

	// Create an AWS request Signer
	signer, err := requestsigner.NewSignerWithService(awsCfg, requestsigner.OpenSearchService)
	// Use requestsigner.OpenSearchServerless for Amazon OpenSearch Serverless
	if err != nil {
		return err
	}

	// Create an opensearch client and use the request-signer
	client, err := opensearchapi.NewClient(
		opensearchapi.Config{
			Client: opensearch.Config{
				Addresses: []string{endpoint},
				Signer:    signer,
			},
		},
	)
	if err != nil {
		return err
	}

    ping, err := client.Ping(ctx, nil)
    if err != nil {
        return err
    }

	fmt.Println(ping)

	return nil
}
```

### Alternative: Using signer/awsv2

Use the AWS SDK v2 for Go to authenticate with Amazon OpenSearch service.

```go
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"

	"github.com/opensearch-project/opensearch-go/v4"
	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	requestsigner "github.com/opensearch-project/opensearch-go/v4/signer/awsv2"
)

const endpoint = "" // e.g. https://opensearch-domain.region.com or Amazon OpenSearch Serverless endpoint

func main() {
	if err := example(); err != nil {
		fmt.Println(fmt.Sprintf("Error: %s", err))
		os.Exit(1)
	}
}
func example() error {
	ctx := context.Background()

	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("<AWS_REGION>"),
		config.WithCredentialsProvider(
			getCredentialProvider("<AWS_ACCESS_KEY>", "<AWS_SECRET_ACCESS_KEY>", "<AWS_SESSION_TOKEN>"),
		),
	)
	if err != nil {
		return err
	}

	// Create an AWS request Signer and load AWS configuration using default config folder or env vars.
	signer, err := requestsigner.NewSignerWithService(awsCfg, "es") // Use "aoss" for Amazon OpenSearch Serverless
	if err != nil {
		return err
	}

	// Create an opensearch client and use the request-signer.
	client, err := opensearchapi.NewClient(
		opensearchapi.Config{
			Client: opensearch.Config{
				Addresses: []string{endpoint},
				Signer:    signer,
			},
		},
	)
	if err != nil {
		return err
	}

	indexName := "go-test-index"

	// Define index mapping.
	mapping := strings.NewReader(`{
	 "settings": {
	   "index": {
	        "number_of_shards": 4
	        }
	      }
	 }`)

	// Create an index with non-default settings.
	createResp, err := client.Indices.Create(
		ctx,
		opensearchapi.IndicesCreateReq{
			Index: indexName,
			Body:  mapping,
		},
	)
	if err != nil {
		return err
	}

	fmt.Printf("created index: %s\n", createResp.Index)

	delResp, err := client.Indices.Delete(ctx, opensearchapi.IndicesDeleteReq{Indices: []string{indexName}})
	if err != nil {
		return err
	}

	fmt.Printf("deleted index: %#v\n", delResp.Acknowledged)
	return nil
}

func getCredentialProvider(accessKey, secretAccessKey, token string) aws.CredentialsProviderFunc {
	return func(ctx context.Context) (aws.Credentials, error) {
		c := &aws.Credentials{
			AccessKeyID:     accessKey,
			SecretAccessKey: secretAccessKey,
			SessionToken:    token,
		}
		return *c, nil
	}
}
```

## Custom Transport

For common TLS options, prefer the built-in config fields (`InsecureSkipVerify`, `CACert`) over constructing a custom `http.Transport`. These options clone the default transport internally, preserving connection pooling, HTTP/2, and timeout defaults.

If you need a custom transport (e.g. for mutual TLS or a corporate proxy), **always clone `http.DefaultTransport` rather than constructing a bare `&http.Transport{}`**. A bare transport defaults to `MaxIdleConnsPerHost: 2`, disables HTTP/2 multiplexing, and has no dialer timeouts -- causing excessive TLS handshakes and connection churn under concurrency.

```go
// Good: clone DefaultTransport, then customize.
tp := http.DefaultTransport.(*http.Transport).Clone()
tp.TLSClientConfig.Certificates = []tls.Certificate{cert}

client, err := opensearch.NewClient(opensearch.Config{
    Addresses: []string{"https://localhost:9200"},
    Transport: tp,
})
```

```go
// Bad: bare transport loses all DefaultTransport defaults.
client, err := opensearch.NewClient(opensearch.Config{
    Addresses: []string{"https://localhost:9200"},
    Transport: &http.Transport{
        TLSClientConfig: &tls.Config{
            Certificates: []tls.Certificate{cert},
        },
    },
    // MaxIdleConnsPerHost = 2, no HTTP/2, no dialer timeouts!
})
```

Key settings to verify on any custom transport:

| Setting                | `DefaultTransport` | Bare `&http.Transport{}` |
| ---------------------- | ------------------ | ------------------------ |
| `MaxIdleConns`         | 100                | 0 (unlimited)            |
| `MaxIdleConnsPerHost`  | 2                  | 2                        |
| `ForceAttemptHTTP2`    | true               | false                    |
| `IdleConnTimeout`      | 90s                | 0 (no timeout)           |
| `TLSHandshakeTimeout`  | 10s                | 0 (no timeout)           |
| `DialContext` timeouts | 30s                | none                     |

`Clone()` copies all of these. If you must build from scratch (e.g. for a non-`*http.Transport` round tripper), set at minimum `ForceAttemptHTTP2: true` and `MaxIdleConnsPerHost` >= your expected concurrency.

## Operation Classifier

The `opensearchtransport.OperationClassifier` maps HTTP method+path pairs to structured `OperationID` values. This enables transparent metrics, tracing, or access-control middleware at the `http.RoundTripper` layer without per-operation wrapper code.

```go
import "github.com/opensearch-project/opensearch-go/v4/opensearchtransport"

// Build once, reuse across requests. Safe for concurrent use.
classifier := opensearchtransport.NewOperationClassifier()

// In an http.RoundTripper:
func (t *MetricsTransport) RoundTrip(req *http.Request) (*http.Response, error) {
    op := t.classifier.Classify(req.Method, req.URL.Path)

    start := time.Now()
    resp, err := t.next.RoundTrip(req)
    duration := time.Since(start)

    status := 0
    if resp != nil {
        status = resp.StatusCode
    }
    t.histogram.WithLabelValues(op.String(), strconv.Itoa(status)).Observe(
        float64(duration.Milliseconds()),
    )
    return resp, err
}
```

`OperationID` is a bit-packed `int64` with masking helpers:

```go
op := classifier.Classify("POST", "/my-index/_search")
op.String()     // "search"
op.IsWrite()    // false
op.IsRead()     // true
op.Category()   // CatSearch
```

Returns `OpOther` for unrecognized patterns.

## Debugging

Set the `OPENSEARCH_GO_DEBUG` environment variable to enable debug logging for connection management, node discovery, and request routing. Debug output is written to stderr.

```bash
OPENSEARCH_GO_DEBUG=true go run myapp.go
```

For programmatic control, set `EnableDebugLogger: true` in the client configuration:

```go
client, err := opensearchapi.NewClient(
    opensearchapi.Config{
        Client: opensearch.Config{
            Addresses:         []string{"http://localhost:9200"},
            EnableDebugLogger: true,
        },
    },
)
```

In tests, use the `testutil.IsDebugEnabled(t)` helper which also reads `OPENSEARCH_GO_DEBUG`:

```bash
OPENSEARCH_GO_DEBUG=true go test ./...
```

## Policy Overrides

Disable specific routing policies at startup via environment variables for debugging, A/B testing, or emergency overrides:

```bash
# Disable all connection scoring (fall back to plain role-based)
OPENSEARCH_GO_POLICY_ROUTER=false myapp

# Disable a specific policy instance by path
OPENSEARCH_GO_POLICY_ROLE=chain[0].mux[0].role[0]=false myapp

# Regex path matching
OPENSEARCH_GO_POLICY_ROLE=.*mux.*role.*=false myapp
```

Set `OPENSEARCH_GO_DEBUG=true` to see policy paths and override actions. See [Request Routing](guides/routing.md#policy-override-variables) for full documentation.

## Environment Variables

All `OPENSEARCH_GO_*` environment variables are evaluated once at client initialization. A complete reference is in [Request Routing: Configuration Reference](guides/routing.md#14-configuration-reference). Quick summary:

| Variable                            | Default       | Description                                                                 |
| ----------------------------------- | ------------- | --------------------------------------------------------------------------- |
| `OPENSEARCH_GO_REQUEST_TIMEOUT`     | 0 (none)      | Per-attempt HTTP request timeout (duration or secs)                         |
| `OPENSEARCH_GO_DEBUG`               | `false`       | Debug logging to stderr                                                     |
| `OPENSEARCH_GO_ROUTER`              | `false`       | Auto-construct DefaultRouter when no Router is set                          |
| `OPENSEARCH_GO_ROUTING_CONFIG`      | (all enabled) | Shard-exact routing (`-shard_exact`)                                        |
| `OPENSEARCH_GO_SHARD_REQUESTS`      | `true`        | Adaptive `max_concurrent_shard_requests` bounds                             |
| `OPENSEARCH_GO_SHARD_COST`          | (defaults)    | Shard cost multipliers for connection scoring                               |
| `OPENSEARCH_GO_DISCOVERY_CONFIG`    | (all enabled) | Skip specific discovery calls                                               |
| `OPENSEARCH_GO_FALLBACK`            | `true`        | Seed URL fallback when all pools exhausted                                  |
| `OPENSEARCH_GO_NODE_STATS_INTERVAL` | auto (5s-30s) | Stats polling interval                                                      |
| `OPENSEARCH_GO_POLICY_*`            | (all enabled) | Disable specific routing policies (10 variables)                            |
| `OPENSEARCH_GO_ACTIVE_LIST_CAP`     | auto          | Max active connections per pool                                             |
| `OPENSEARCH_GO_STANDBY_*`           | (see guide)   | Standby rotation and promotion tuning (3 variables)                         |
| `OPENSEARCH_GO_OVERLOADED_*`        | (see guide)   | JVM heap and breaker thresholds (2 variables)                               |
| `OPENSEARCH_GO_ERROR_MASK`          | (version-dep) | Mask partial-failure categories (`+`/`-` tokens; overrides `Config.Errors`) |

## Guides by Topic

- [**Security**](guides/security.md) - TLS, credentials, input validation, index patterns, error disclosure
- [Index Lifecycle](guides/index_lifecycle.md)
- [Document Lifecycle](guides/document_lifecycle.md)
- [Search](guides/search.md)
- [Bulk](guides/bulk.md)
- [Advanced Index Actions](guides/advanced_index_actions.md)
- [Index Templates](guides/index_template.md)
- [Data Streams](guides/data_streams.md)
- [Making Raw JSON REST Requests](guides/json.md)
- [Request Routing](guides/routing.md)
- [Cluster Health Checking](guides/cluster_health_checking.md)
- [Node Discovery and Role Management](guides/node_discovery_and_roles.md)
- [Response Body Buffering](guides/response_buffering.md)
- [Retry and Backoff](guides/retry_backoff.md)
- [Error Handling](guides/error_handling.md)
