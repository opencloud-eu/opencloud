- [Upgrading OpenSearch Go Client](#upgrading-opensearch-go-client)
  - [Upgrading to >= 5.0.0](#upgrading-to->=-5.0.0)
    - [Partial failure errors (Config.Errors)](#partial-failure-errors-configerrors)
    - [Default Router Injection in v5preview](#default-router-injection-in-v5preview)
    - [DiscoverNodes() blocking semantics](#discovernodes-blocking-semantics)
    - [opensearchtransport.Route interface gained OpID()](#opensearchtransportroute-interface-gained-opid)
    - [Response.Body becomes a method](#responsebody-becomes-a-method)
  - [Upgrading to >= 4.7.0](#upgrading-to->=-4.7.0)
    - [opensearch.Request interface signature change](#opensearchrequest-interface-signature-change)
    - [Path segment values are percent-encoded](#path-segment-values-are-percent-encoded)
    - [v5preview/opensearchapi/ package - v5 preview API surface](#v5previewopensearchapi-package---v5-preview-api-surface)
  - [Upgrading to >= 4.0.0](#upgrading-to->=-4.0.0)
    - [Import path](#import-path)
    - [Error types](#error-types)
    - [StringError for unknown JSON responses](#stringerror-for-unknown-json-responses)
    - [AWS signer](#aws-signer)
    - [Typed failure arrays in by-query and reindex responses](#typed-failure-arrays-in-by-query-and-reindex-responses)
    - [Inline `_shards` structs replaced with ResponseShards](#inline-_shards-structs-replaced-with-responseshards)
    - [`_type` field tags now include omitempty](#_type-field-tags-now-include-omitempty)
  - [Upgrading to >= 3.0.0](#upgrading-to->=-3.0.0)
    - [Client creation](#client-creation)
    - [Requests](#requests)
    - [Responses](#responses)
    - [Error handling](#error-handling)
    - [API reorganization](#api-reorganization)
  - [Upgrading to >= 2.3.0](#upgrading-to->=-2.3.0)
    - [Snapshot delete](#snapshot-delete)

# Upgrading OpenSearch Go Client

## Upgrading to >= 5.0.0

### Partial Failure Errors (Config.Errors)

Version 5.0.0 introduces typed partial-failure errors and a per-category bitmask that controls which categories surface as Go errors. OpenSearch returns HTTP 200 for many operations that partially succeed (bulk item failures, shard failures on search, replica failures on writes). The new model turns those partial failures into typed errors that callers can dispatch on; idiomatic partial error handling is shown below.

**Default behavior change:**

| Surface | `Config.Errors == nil` means | Effect                                            |
| ------- | ---------------------------- | ------------------------------------------------- |
| v4      | `errmask.All`                | mask everything (preserves pre-bitfield behavior) |
| v5+     | `errmask.Empty`              | report every partial-failure category             |

A v4 caller upgrading to v5 who never set `Config.Errors` will start seeing partial failures as `error`. To preserve v4-style silence on v5, set `Errors: errmask.New(errmask.All)` explicitly. To opt v4 in to v5-style surfacing today, set `Errors: errmask.New()`.

**New surface (v5):**

- `Config.Errors *errmask.ErrorMask` field with the matrix above.
- `OPENSEARCH_GO_ERROR_MASK` environment-variable override (comma-separated `+`/`-` tokens of lowercase snake_case wrapper names like `bulk_items`, `search_shards`, `write_shards`, `multi_search_items`).
- Typed errors: `*PartialBulkError`, `*PartialSearchError`, `*ShardFailureError`, `*MultiSearchItemError`, `*MSearchErrors`, `*MSearchTemplateErrors`.
- `opensearchapi.Errors(err) []error` to flatten single- and multi-wrapper error shapes into a uniform slice.
- Helper functions: `IsPartialFailure(err)`, `ToleratePartialFailures(err)`, `RequireSuccessRate(err, threshold)`.
- Operation constants: `OperationIndex`, `OperationCreate`, `OperationUpdate`, `OperationDelete`.

The recommended call-site pattern is a `for`/`switch` over `opensearchapi.Errors(err)`, not `errors.As` against a specific type. Per-Resp helper methods (`BulkItemFailures()`, `SearchShardFailures()`, `WriteShardFailures()`, `MultiSearchItemFailures()`, `PartialFailures(mask)`) exist on the response types as engine machinery for the dispatch and remain available for focused inspection of a known category, but new code should use the type switch -- see [`guides/error_handling.md`](guides/error_handling.md#why-a-type-switch-not-errorsas-has-or-per-resp-helpers) for why.

**Where to read more:**

- [`v5preview/opensearchapi/README.md`](v5preview/opensearchapi/README.md) - full v5preview usage guide for these errors, including the type-switch pattern and the rationale for preferring it over `errors.As`/`Has`.
- [`guides/error_handling.md`](guides/error_handling.md) - cross-version best-practices guide with v4 and v5preview examples side-by-side.
- [`v5preview/opensearchapi/MIGRATING.md`](v5preview/opensearchapi/MIGRATING.md) - v4 -> v5preview surface delta.

**Error types in v4 `opensearchapi/`** (the upgrade source):

| Error Type               | Returned By                                                                                                                         | Key Fields                                                                |
| ------------------------ | ----------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------- |
| `*PartialBulkError`      | `Bulk`                                                                                                                              | `FailedItems []BulkRespItem`, `SucceededCount int`                        |
| `*PartialSearchError`    | `Search`, `Scroll.Get`, `SearchTemplate` (single-bit); also via `*MSearchErrors` and `*MSearchTemplateErrors` for shard aggregation | `FailedShards int`, `TotalShards int`, `Failures []ResponseShardsFailure` |
| `*ShardFailureError`     | `Index`, `Document.Create`, `Document.Delete`, `Update`                                                                             | `Operation string`, `FailedShards int`, `TotalShards int`                 |
| `*MultiSearchItemError`  | `MSearch`, `MSearchTemplate` (per-sub-response error inspection)                                                                    | `Items []MultiSearchItemFailure`, `SucceededCount int`                    |
| `*MSearchErrors`         | `MSearch` when 2+ wrappers fire                                                                                                     | `Unwrap() []error` (multi-error contract)                                 |
| `*MSearchTemplateErrors` | `MSearchTemplate` when 2+ wrappers fire                                                                                             | `Unwrap() []error`                                                        |

The v5preview surface ports the same model with internal field types regenerated from the [OpenSearch API specification](https://github.com/opensearch-project/opensearch-api-specification) ([see MIGRATING.md](v5preview/opensearchapi/MIGRATING.md#partial-failure-type-renames) for the table).

### Default Router Injection in v5preview

`v5preview/opensearchapi.NewClient` (and `NewDefaultClient`) now inject [`opensearchtransport.NewDefaultRouter`](https://pkg.go.dev/github.com/opensearch-project/opensearch-go/v4/opensearchtransport#NewDefaultRouter) when the caller leaves `config.Client.Router` nil. The `OPENSEARCH_GO_ROUTER` environment variable acts as an opt-out:

| `OPENSEARCH_GO_ROUTER` | v4                                                  | v5preview                                                   |
| ---------------------- | --------------------------------------------------- | ----------------------------------------------------------- |
| unset                  | no Router, no auto-discovery                        | **default Router injected**, no auto-discovery              |
| `true` / `1`           | default Router (transport layer), auto-discovery on | **default Router injected, auto-discovery on**              |
| `false` / `0`          | no Router, no auto-discovery                        | **injection skipped (Router stays nil)**, no auto-discovery |
| unparseable            | no Router, no auto-discovery                        | default Router injected, no auto-discovery                  |

Truthy and falsy semantics are preserved end-to-end: a v4 caller running with `OPENSEARCH_GO_ROUTER=true` keeps auto-discovery when migrating to v5preview, and `=false` opts out of both Router injection and auto-discovery. A caller-supplied `DiscoverNodesOnStart` value always wins over the env-var-driven side-effect.

v4's `opensearchapi.NewClient` is unchanged: it doesn't auto-inject a Router, so existing v4 code keeps its current behavior.

For full usage and rationale see [`v5preview/opensearchapi/README.md` Default Router Injection](v5preview/opensearchapi/README.md#default-router-injection).

### `DiscoverNodes()` blocking semantics

`opensearch.Client.DiscoverNodes()` and `opensearchtransport.Client.DiscoverNodes()` now **block** when an in-flight discovery cycle is active and return that cycle's error verbatim, instead of the previous immediate `nil` no-op.

The previous "fire-and-forget" behavior masked discovery failures: a caller that handed control back after a failing discovery saw `err == nil` and continued against a stale node list. The new behavior surfaces the failure synchronously so callers can react (retry, alert, fall back to seed nodes).

Client construction itself never blocks on discovery: when `Config.DiscoverNodesOnStart` is `&true`, `opensearch.NewClient` launches a detached goroutine that calls `DiscoverNodes()`, then returns. Callers who want fully manual control set `DiscoverNodesOnStart: &false` and `DiscoverNodesInterval: 0` -- no on-start goroutine, no polling loop -- and call `client.DiscoverNodes(ctx)` themselves before issuing requests:

```go
discoverOnStart := false
client, err := opensearch.NewClient(opensearch.Config{
    Addresses:             []string{"https://localhost:9200"},
    DiscoverNodesOnStart:  &discoverOnStart, // skip auto-discovery
    DiscoverNodesInterval: 0,                // disable polling loop
})
if err != nil {
    log.Fatal(err)
}

ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
if err := client.DiscoverNodes(ctx); err != nil {
    log.Printf("initial discovery failed: %s", err)
}

// proceed to use the client; topology data is ready.
```

The signature already changed earlier in this version to take `context.Context` (see CHANGELOG); this is a behavioral change on top of the signature change.

### `opensearchtransport.Route` interface gained `OpID()`

The exported `Route` interface in `opensearchtransport` gained a new method:

```go
type Route interface {
    Policy() Policy
    Attrs() routeAttr
    PoolName() string
    OpID() OperationID  // new in v5
}
```

External code that implements `Route` (custom routing policies) must add an `OpID() OperationID` method returning the [`OperationID`](https://pkg.go.dev/github.com/opensearch-project/opensearch-go/v4/opensearchtransport#OperationID) for the route -- typically the `Op*` constant matching the route's HTTP method+path. Built-in routes built via `NewRouteMux` are populated automatically; only hand-written `Route` implementations are affected.

### `Response.Body` becomes a method

`Response.Body` changes from a public `io.ReadCloser` field to a `Body()` method. The new `RawBody() []byte` method (available since v4) provides access to the buffered response bytes without consuming the body reader.

Before 5.0.0:

```go
body, err := io.ReadAll(resp.Body)
```

After 5.0.0:

```go
body, err := io.ReadAll(resp.Body())

// Or, if you only need the raw bytes and the response was already read:
raw := resp.RawBody()
```

## Upgrading to >= 4.7.0

### `opensearch.Request` interface signature change

`GetRequest()` now receives the HTTP method from the caller:

```go
// Before
GetRequest() (*http.Request, error)

// After
GetRequest(method string) (*http.Request, error)
```

This change is invisible to almost all callers: the typed `Req` structs that the client consumes (e.g. `opensearchapi.SearchReq`, the v5-preview `opensearchapi.IndexReq`) already implement the new signature. Only code that defines a custom type satisfying `opensearch.Request` is affected.

If you maintain such a type, add a `method string` parameter and forward it to your request builder. The `opensearch.BuildRequest` helper that earlier v4 releases exposed for this purpose was **removed in 4.7.0**; construct the request with `net/http` directly instead. Give the path a leading slash (e.g. `/_plugins/my_plugin/status`) -- the transport prepends the base URL by string concatenation, so a path without a leading slash produces a malformed URL.

```go
// Before (<= 4.6.0): method stored on the struct, built via the removed
// opensearch.BuildRequest helper (which set Content-Type for a non-nil body).
func (r customReq) GetRequest() (*http.Request, error) {
    return opensearch.BuildRequest(r.method, r.path, r.body, nil, nil)
}

// After (>= 4.7.0): method comes from the caller, built with net/http.
func (r customReq) GetRequest(method string) (*http.Request, error) {
    req, err := http.NewRequest(method, r.path, r.body)
    if err != nil {
        return nil, err
    }
    // BuildRequest set this automatically for a non-nil body; http.NewRequest
    // does not, so set it here or OpenSearch may reject a JSON body with 400/415.
    if r.body != nil {
        req.Header.Set("Content-Type", "application/json")
    }
    return req, nil
}
```

`opensearch.BuildRequest` also accepted `params map[string]string` and `headers http.Header` arguments. To preserve those, set them on the `*http.Request` after construction: encode params onto `req.URL.RawQuery` (via `url.Values`) and add headers to `req.Header`.

### Path segment values are percent-encoded

Every typed `Req.GetRequest()` method now constructs URL paths through the generated `internal/path/*Path` builders, which unconditionally percent-encode user-supplied segment values (via `url.PathEscape`, plus an explicit `/` -> `%2F` substitution). This closes the [#650] path-injection class of bugs: an `Index` value containing `../../_cluster/health` can no longer escape its segment to alter routing.

Prior to this release, `buildPath` wrote segment values raw, which left the wire format ambiguous and the encoding contract undefined: callers who passed unencoded metacharacters got a malformed URL, and callers who passed pre-encoded values got the encoded bytes through to the server. Both interpretations existed simultaneously. The client now defines a single contract — pass raw, unencoded values; the client encodes — and applies it everywhere.

The practical consequence for callers who were already passing pre-encoded values is that those values are now double-encoded:

```go
// Before: the URL contained "my%2Findex" verbatim (relying on undefined behavior)
// After:  the URL contains "my%252Findex" (the percent itself is encoded)
client.Indices.Get(ctx, opensearchapi.IndicesGetReq{Index: []string{"my%2Findex"}})
```

If your code intentionally passes percent-encoded values, decode them with `url.PathUnescape` before populating the Req struct.

[#650]: https://github.com/opensearch-project/opensearch-go/issues/650

### `v5preview/opensearchapi/` package — v5 preview API surface

This release introduces a new `v5preview/opensearchapi/` package alongside the existing top-level `opensearchapi/` package. The new package is the **preview of the v5 API in the v4 branch** and is generated from the [OpenSearch API specification](https://github.com/opensearch-project/opensearch-api-specification) by `cmd/osgen`. It deliberately reuses the package name `opensearchapi` so that callers who migrate during the v4 branch only need to change the import path at v5 release time -- every reference in code (e.g. `opensearchapi.IndexReq`, `opensearchapi.NewClient`) stays the same.

**Migration Considerations:**

- Migrating to `v5preview/opensearchapi/` in v4 gives you the v5 surface ahead of v5 release. The trade-off at v5 release time is a single edit per consuming file: change the import path from `/v4/v5preview/opensearchapi` to `/v5/opensearchapi`. Package qualifiers do not change.
- Staying on the top-level `opensearchapi/` package is fine through the rest of v4. At v5, the hand-written `opensearchapi/` is removed; the only forward path is the code-generated API surface (closely matches the existing hand-written ergonomics).

**Import path:**

```go
import "github.com/opensearch-project/opensearch-go/v4/v5preview/opensearchapi"

client, err := opensearchapi.NewClient(opensearchapi.Config{...})
```

**Surface differences worth knowing about:**

- Optional `Params` are `*Params` pointer fields (nil-safe; pass `&opensearchapi.IndexParams{...}` to set).
- Optional boolean query parameters are `*bool` so a deliberate `false` can be sent over the wire.
- Multi-index `Req` types use `Index []string` (the spec spelling); v4's hand-written `Indices` is renamed.
- Plugin APIs (k-NN, ML, Security, ISM, etc.) live in `v5preview/opensearchapi/plugins/`.

For the full v4 -> v5preview surface delta and the optional forward-compatible `replace` directive, see [`v5preview/opensearchapi/MIGRATING.md`](v5preview/opensearchapi/MIGRATING.md). For everyday usage (errors, routing, response handling) see [`v5preview/opensearchapi/README.md`](v5preview/opensearchapi/README.md).

## Upgrading to >= 4.0.0

Version 4.0.0 updated the module import path, moved error types from opensearchapi to opensearch, renamed them, added new error types, and migrated the `signer/aws` package from AWS SDK v1 to AWS SDK v2.

### Import Path

Update all import paths from `v3` to `v4`:

```go
// Before (v3)
import (
    "github.com/opensearch-project/opensearch-go/v3"
    "github.com/opensearch-project/opensearch-go/v3/opensearchapi"
)

// After (v4)
import (
    "github.com/opensearch-project/opensearch-go/v4"
    "github.com/opensearch-project/opensearch-go/v4/opensearchapi"
)
```

Update your `go.mod`:

```bash
go get github.com/opensearch-project/opensearch-go/v4@latest
```

### Error Types

Before 4.0.0: Error types:

- `opensearchapi.Error`
- `opensearchapi.StringError`

With 4.0.0: Error types

- `opensearch.Error` -- base error with string `Err` field
- `opensearch.StringError` -- raw string error body
- `opensearch.ReasonError` -- error with `Reason` and `Status` fields
- `opensearch.MessageError` -- error with `Message` field
- `opensearch.StructError` -- structured JSON error with `Type`, `Reason`, `RootCause` (was `opensearchapi.Error`)

Update `errors.As` targets to use `opensearch.*` instead of `opensearchapi.*`:

```go
// Before (v3)
var opensearchError *opensearchapi.Error
if errors.As(err, &opensearchError) {
    fmt.Println(opensearchError.Err.Type)
}

// After (v4)
var opensearchError *opensearch.StructError
if errors.As(err, &opensearchError) {
    fmt.Println(opensearchError.Err.Type)
}
```

### StringError for Unknown JSON Responses

Version 4.0.0 returns `*opensearch.StringError` error type instead of `*fmt.wrapError` when response received from the server is an unknown JSON. For example, consider delete document API which returns an unknown JSON body when document is not found.

Before 4.0.0:

```go
docDelResp, err = client.Document.Delete(ctx, opensearchapi.DocumentDeleteReq{Index: "movies", DocumentID: "3"})
if err != nil {
	fmt.Println(err)

	if !errors.Is(err, opensearch.ErrJSONUnmarshalBody) && docDelResp != nil {
		resp := docDelResp.Inspect().Response
		// get http status
		fmt.Println(resp.StatusCode)
		body := strings.TrimPrefix(err.Error(), "opensearch error response could not be parsed as error: ")
		errResp := opensearchapi.DocumentDeleteResp{}
		json.Unmarshal([]byte(body), &errResp)
		// extract result field from the body
		fmt.Println(errResp.Result)
	}
}
```

After 4.0.0:

```go
docDelResp, err = client.Document.Delete(ctx, opensearchapi.DocumentDeleteReq{Index: "movies", DocumentID: "3"})
if err != nil {
	// parse into *opensearch.StringError
	var myStringErr *opensearch.StringError
	if errors.As(err, &myStringErr) {
		// get http status
		fmt.Println(myStringErr.Status)
		errResp := opensearchapi.DocumentDeleteResp{}
		json.Unmarshal([]byte(myStringErr.Err), &errResp)
		// extract result field from the body
		fmt.Println(errResp.Result)
	}
}
```

### AWS Signer

The `signer/aws` package now uses AWS SDK v2 instead of AWS SDK v1. AWS SDK v1 reached end-of-support on July 31, 2025.

Before 4.0.0 (AWS SDK v1):

```go
import (
    "github.com/aws/aws-sdk-go/aws/session"
    signer "github.com/opensearch-project/opensearch-go/v3/signer/aws"
)

awsSigner, err := signer.NewSigner(session.Options{
    Config: aws.Config{Region: aws.String("us-east-1")},
})
```

With 4.0.0 (AWS SDK v2):

```go
import (
    "context"
    "github.com/aws/aws-sdk-go-v2/config"
    signer "github.com/opensearch-project/opensearch-go/v4/signer/aws"
)

cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-east-1"))
if err != nil {
    return err
}
awsSigner, err := signer.NewSigner(cfg)
```

The `signer/awsv2` package (which already used AWS SDK v2) remains available at `github.com/opensearch-project/opensearch-go/v4/signer/awsv2` with the same API.

### Typed Failure Arrays in By-Query and Reindex Responses

The `Failures` field in `DocumentDeleteByQueryResp`, `UpdateByQueryResp`, and `ReindexResp` changed from `[]json.RawMessage` to `[]BulkByScrollFailure`.

**Who is affected:** Only callers that directly access the `.Failures` slice. Code that ignores failures or only checks `len(resp.Failures) > 0` compiles without changes.

Before:

```go
for _, raw := range resp.Failures {
    var f MyFailureType
    if err := json.Unmarshal(raw, &f); err != nil { ... }
    // use f.Index, f.Status, etc.
}
```

After:

```go
for _, f := range resp.Failures {
    // Fields are available directly — no unmarshaling needed.
    fmt.Println(f.Index, f.Status, f.Cause)
}
```

### Inline `_shards` Structs Replaced with ResponseShards

The `Shards` field in `IndexResp`, `DocumentCreateResp`, `DocumentDeleteResp`, `UpdateResp`, `IndicesRefreshResp`, and `IndicesCountResp` changed from an anonymous inline struct to the named `ResponseShards` type.

**Who is affected:** Code accessing `resp.Shards.Total`, `resp.Shards.Successful`, or `resp.Shards.Failed` compiles unchanged. The only theoretical break is code using reflection or type assertions on the `Shards` field itself. The new type additionally exposes `Failures []ResponseShardsFailure` and `Skipped int` which were previously unavailable.

### `_type` Field Tags Now Include `omitempty`

All deprecated `_type` fields across response structs now use `json:"_type,omitempty"`. Mapping types were deprecated in Elasticsearch 6.0 (2017), reduced to the single `_doc` type in Elasticsearch 7.0 (2019), and completely removed from the OpenSearch server in 2.0.0 (May 2022). OpenSearch forked from Elasticsearch 7.10.2, so the deprecation was inherited from day one. This change means the empty string is no longer emitted when marshaling response structs back to JSON. Deserialization behavior is unchanged.

## Upgrading to >= 3.0.0

Version 3.0.0 is a major refactor of the client.

### Client Creation

You now create the client from the opensearchapi package instead of opensearch. This was done to make the different APIs independent from each other. Plugin APIs like Security get their own folder and therefore their own sub-lib.

Before 3.0.0:

```go
// default client
client, err := opensearch.NewDefaultClient()

// with config
client, err := opensearch.NewClient(
    opensearch.Config{
	    InsecureSkipVerify: true,
		Addresses: []string{"https://localhost:9200"},
		Username:  "admin",
		Password:  "admin",
	},
)
```

With 3.0.0:

```go
// default client
client, err := opensearchapi.NewDefaultClient()

// with config
client, err := opensearchapi.NewClient(
    opensearchapi.Config{
		Client: opensearch.Config{
			InsecureSkipVerify: true, // For testing only. Use certificate for validation.
			Addresses:          []string{"https://localhost:9200"},
			Username:           "admin", // For testing only. Don't store credentials in code.
			Password:           "admin",
		},
	},
)
```

### Requests

Prior version 3.0.0 there were two options on how to perform requests. You could either use the request struct of the wished function and execute it with the client .Do() function or use the client function and add wanted args with so called With<arg>() functions. With the new version you now use functions attached to the client and give a context and the wanted request body as argument.

Before 3.0.0:

```go
// using the client function and adding args by using the With<arg>() functions
createIndex, err := client.Indices.Create(
    "some-index",
    client.Indices.Create.WithContext(ctx),
    client.Indices.Create.WithBody(strings.NewReader(`{"settings":{"index":{"number_of_shards":4}}}`)),
)

// using the request struct
createIndex := opensearchapi.IndicesCreateRequest{
    Index: "some-index",
    Body:  strings.NewReader(`{"settings":{"index":{"number_of_shards":4}}}`),
}
createIndexResponse, err := createIndex.Do(ctx, client)
```

With 3.0.0:

```go
createIndexResponse, err := client.Indices.Create(
    ctx,
    opensearchapi.IndicesCreateReq{
        Index: "some-index",
        Body:  strings.NewReader(`{"settings":{"index":{"number_of_shards":4}}}`),
    },
)
```

### Responses

With the version 3.0.0 the lib no longer returns the opensearch.Response which is just a wrap up http.Response. Instead it will check the response for errors and try to parse the body into existing structs. Please note that some responses are so complex that we parse them as [json.RawMessage](https://pkg.go.dev/encoding/json#RawMessage) so you can parse them to your expected struct. If you need the opensearch.Response, then you can call .Inspect().

Before 3.0.0:

```go
// Create the request
createIndex := opensearchapi.IndicesCreateRequest{
    Index: "some-index",
    Body:  strings.NewReader(`{"settings":{"index":{"number_of_shards":4}}}`),
}
// Execute the requests
resp, err := createIndex.Do(ctx, client)
if err != nil {
	return err
}
// Close the body
defer resp.Body.Close()

// Check if the status code is >299
if resp.IsError() {
	return fmt.Errorf("Opensearch Returned an error: %#v", resp)
}

// Create a struct that represents the create index response
createResp := struct {
	Acknowledged       bool   `json:"acknowledged"`
	ShardsAcknowledged bool   `json:"shards_acknowledged"`
	Index              string `json:"index"`
}{}

// Try to parse the response into the created struct
if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
	return fmt.Errorf("unexpected response body: %d, %#v, %s", resp.StatusCode, resp.Body, err)
}
// Print the created index name
fmt.Println(createResp.Index)
```

With 3.0.0:

```go
// Create and execute the requests
createResp, err := client.Indices.Create(
    ctx,
    opensearchapi.IndicesCreateReq{
        Index: "some-index",
        Body:  strings.NewReader(`{"settings":{"index":{"number_of_shards":4}}}`),
    },
)
if err != nil {
	return err
}
// Print the created index name
fmt.Println(createResp.Index)

// To get the opensearch.Response/http.Response
rawResp := createResp.Inspect().Response
```

### Error Handling

With opensearch-go >= 3.0.0 opensearchapi responses are now checked for errors. Checking for errors twice is no longer needed.

Prior versions only returned an error if the request failed to execute. For example if the client can't reach the server or the TLS handshake failed. With opensearch-go >= 3.0.0 each opensearchapi requests will return an error if the response http status code is > 299. The error can be parsed into the new `opensearchapi.Error` type by using `errors.As` to match for exceptions and get a more detailed view.

Before 3.0.0:

```go
// Create the request
createIndex := opensearchapi.IndicesCreateRequest{
    Index: "some-index",
    Body:  strings.NewReader(`{"settings":{"index":{"number_of_shards":4}}}`),
}

// Execute the requests
resp, err := createIndex.Do(ctx, client)
if err != nil {
	return err
}
// Close the body
defer resp.Body.Close()

// Check if the status code is >299
if createIndexResp.IsError() {
    fmt.Errorf("Opensearch returned an error. Status: %d", createIndexResp.StatusCode)
}
```

With 3.0.0:

```go
var opensearchError opensearchapi.Error
// Create and execute the requests
createResp, err := client.Indices.Create(
    ctx,
    opensearchapi.IndicesCreateReq{
        Index: "some-index",
        Body:  strings.NewReader(`{"settings":{"index":{"number_of_shards":4}}}`),
    },
)
// Load err into opensearchapi.Error to access the fields and tolerate if the index already exists
if err != nil {
	if errors.As(err, &opensearchError) {
		if opensearchError.Err.Type != "resource_already_exists_exception" {
			return err
		}
	} else {
		return err
	}
}
```

### API Reorganization

Version 3.0.0 reorganized APIs into logical sub-clients. The following tables cover every method that moved, was renamed, or was removed.

**Naming conventions changed:**

- Request types: `*Request` to `*Req` (e.g., `SearchRequest` to `SearchReq`)
- Query parameters: separate `*Params` sub-struct (e.g., `SearchParams`)
- Functional options (`With*`) removed entirely
- `req.Do(ctx, client)` pattern removed; use `client.Method(ctx, req)` instead

#### Document Operations -- Moved to `client.Document`

| Before 3.0.0                          | With 3.0.0                                          |
| ------------------------------------- | --------------------------------------------------- |
| `client.Create(...)`                  | `client.Document.Create(ctx, req)`                  |
| `client.Delete(...)`                  | `client.Document.Delete(ctx, req)`                  |
| `client.DeleteByQuery(...)`           | `client.Document.DeleteByQuery(ctx, req)`           |
| `client.DeleteByQueryRethrottle(...)` | `client.Document.DeleteByQueryRethrottle(ctx, req)` |
| `client.Exists(...)`                  | `client.Document.Exists(ctx, req)`                  |
| `client.ExistsSource(...)`            | `client.Document.ExistsSource(ctx, req)`            |
| `client.Explain(...)`                 | `client.Document.Explain(ctx, req)`                 |
| `client.Get(...)`                     | `client.Document.Get(ctx, req)`                     |
| `client.GetSource(...)`               | `client.Document.Source(ctx, req)`                  |

#### Scroll Operations -- Moved to `client.Scroll`

| Before 3.0.0              | With 3.0.0                       |
| ------------------------- | -------------------------------- |
| `client.ClearScroll(...)` | `client.Scroll.Delete(ctx, req)` |
| `client.Scroll(...)`      | `client.Scroll.Get(ctx, req)`    |

#### Script Operations -- Moved to `client.Script`

| Before 3.0.0                         | With 3.0.0                                |
| ------------------------------------ | ----------------------------------------- |
| `client.DeleteScript(...)`           | `client.Script.Delete(ctx, req)`          |
| `client.GetScript(...)`              | `client.Script.Get(ctx, req)`             |
| `client.GetScriptContext(...)`       | `client.Script.Context(ctx, req)`         |
| `client.GetScriptLanguages(...)`     | `client.Script.Language(ctx, req)`        |
| `client.PutScript(...)`              | `client.Script.Put(ctx, req)`             |
| `client.ScriptsPainlessExecute(...)` | `client.Script.PainlessExecute(ctx, req)` |

#### Index Alias, Mapping, Settings -- Moved to Nested Sub-clients

| Before 3.0.0                          | With 3.0.0                               |
| ------------------------------------- | ---------------------------------------- |
| `client.Indices.DeleteAlias(...)`     | `client.Indices.Alias.Delete(ctx, req)`  |
| `client.Indices.ExistsAlias(...)`     | `client.Indices.Alias.Exists(ctx, req)`  |
| `client.Indices.GetAlias(...)`        | `client.Indices.Alias.Get(ctx, req)`     |
| `client.Indices.PutAlias(...)`        | `client.Indices.Alias.Put(ctx, req)`     |
| `client.Indices.UpdateAliases(...)`   | `client.Aliases(ctx, req)`               |
| `client.Indices.GetMapping(...)`      | `client.Indices.Mapping.Get(ctx, req)`   |
| `client.Indices.PutMapping(...)`      | `client.Indices.Mapping.Put(ctx, req)`   |
| `client.Indices.GetFieldMapping(...)` | `client.Indices.Mapping.Field(ctx, req)` |
| `client.Indices.GetSettings(...)`     | `client.Indices.Settings.Get(ctx, req)`  |
| `client.Indices.PutSettings(...)`     | `client.Indices.Settings.Put(ctx, req)`  |

#### Templates -- Moved to Top-level Sub-clients

| Before 3.0.0                                  | With 3.0.0                                     |
| --------------------------------------------- | ---------------------------------------------- |
| `client.Indices.DeleteIndexTemplate(...)`     | `client.IndexTemplate.Delete(ctx, req)`        |
| `client.Indices.ExistsIndexTemplate(...)`     | `client.IndexTemplate.Exists(ctx, req)`        |
| `client.Indices.GetIndexTemplate(...)`        | `client.IndexTemplate.Get(ctx, req)`           |
| `client.Indices.PutIndexTemplate(...)`        | `client.IndexTemplate.Create(ctx, req)`        |
| `client.Indices.SimulateIndexTemplate(...)`   | `client.IndexTemplate.SimulateIndex(ctx, req)` |
| `client.Indices.SimulateTemplate(...)`        | `client.IndexTemplate.Simulate(ctx, req)`      |
| `client.Indices.DeleteTemplate(...)`          | `client.Template.Delete(ctx, req)`             |
| `client.Indices.ExistsTemplate(...)`          | `client.Template.Exists(ctx, req)`             |
| `client.Indices.GetTemplate(...)`             | `client.Template.Get(ctx, req)`                |
| `client.Indices.PutTemplate(...)`             | `client.Template.Create(ctx, req)`             |
| `client.Cluster.DeleteComponentTemplate(...)` | `client.ComponentTemplate.Delete(ctx, req)`    |
| `client.Cluster.ExistsComponentTemplate(...)` | `client.ComponentTemplate.Exists(ctx, req)`    |
| `client.Cluster.GetComponentTemplate(...)`    | `client.ComponentTemplate.Get(ctx, req)`       |
| `client.Cluster.PutComponentTemplate(...)`    | `client.ComponentTemplate.Create(ctx, req)`    |

#### Data Streams -- Moved to `client.DataStream`

| Before 3.0.0                             | With 3.0.0                           |
| ---------------------------------------- | ------------------------------------ |
| `client.Indices.CreateDataStream(...)`   | `client.DataStream.Create(ctx, req)` |
| `client.Indices.DeleteDataStream(...)`   | `client.DataStream.Delete(ctx, req)` |
| `client.Indices.GetDataStream(...)`      | `client.DataStream.Get(ctx, req)`    |
| `client.Indices.GetDataStreamStats(...)` | `client.DataStream.Stats(ctx, req)`  |

#### Snapshot Repository -- Moved to `client.Snapshot.Repository`

| Before 3.0.0                             | With 3.0.0                                     |
| ---------------------------------------- | ---------------------------------------------- |
| `client.Snapshot.CreateRepository(...)`  | `client.Snapshot.Repository.Create(ctx, req)`  |
| `client.Snapshot.DeleteRepository(...)`  | `client.Snapshot.Repository.Delete(ctx, req)`  |
| `client.Snapshot.GetRepository(...)`     | `client.Snapshot.Repository.Get(ctx, req)`     |
| `client.Snapshot.CleanupRepository(...)` | `client.Snapshot.Repository.Cleanup(ctx, req)` |
| `client.Snapshot.VerifyRepository(...)`  | `client.Snapshot.Repository.Verify(ctx, req)`  |

#### Ingest -- Renamed Methods

| Before 3.0.0                        | With 3.0.0                       |
| ----------------------------------- | -------------------------------- |
| `client.Ingest.PutPipeline(...)`    | `client.Ingest.Create(ctx, req)` |
| `client.Ingest.DeletePipeline(...)` | `client.Ingest.Delete(ctx, req)` |
| `client.Ingest.GetPipeline(...)`    | `client.Ingest.Get(ctx, req)`    |
| `client.Ingest.ProcessorGrok(...)`  | `client.Ingest.Grok(ctx, req)`   |

#### Dangling Indices -- Moved to `client.Dangling`

| Before 3.0.0                                     | With 3.0.0                         |
| ------------------------------------------------ | ---------------------------------- |
| `client.DanglingIndicesDeleteDanglingIndex(...)` | `client.Dangling.Delete(ctx, req)` |
| `client.DanglingIndicesImportDanglingIndex(...)` | `client.Dangling.Import(ctx, req)` |
| `client.DanglingIndicesListDanglingIndices(...)` | `client.Dangling.Get(ctx, req)`    |

#### Other Renames

| Before 3.0.0                             | With 3.0.0                              |
| ---------------------------------------- | --------------------------------------- |
| `client.Count(...)`                      | `client.Indices.Count(ctx, req)`        |
| `client.FieldCaps(...)`                  | `client.Indices.FieldCaps(ctx, req)`    |
| `client.Mget(...)`                       | `client.MGet(ctx, req)`                 |
| `client.MSearch(...)`                    | `client.MSearch(ctx, req)`              |
| `client.MSearchTemplate(...)`            | `client.MSearchTemplate(ctx, req)`      |
| `client.Mtermvectors(...)`               | `client.MTermvectors(ctx, req)`         |
| `client.Indices.AddBlock(...)`           | `client.Indices.Block(ctx, req)`        |
| `client.Indices.ResolveIndex(...)`       | `client.Indices.Resolve(ctx, req)`      |
| `client.Cat.Fielddata(...)`              | `client.Cat.FieldData(ctx, req)`        |
| `client.Cat.Nodeattrs(...)`              | `client.Cat.NodeAttrs(ctx, req)`        |
| `client.Nodes.ReloadSecureSettings(...)` | `client.Nodes.ReloadSecurity(ctx, req)` |

#### Removed APIs (no v3+ equivalent)

- `client.TermsEnum(...)`
- `client.Cat.Help(...)`
- `client.Indices.DiskUsage(...)`
- `client.Indices.FieldUsageStats(...)`
- `client.Indices.GetUpgrade(...)`
- `client.Indices.Upgrade(...)`

## Upgrading to >= 2.3.0

### Snapshot Delete

`SnapshotDeleteRequest` and `SnapshotDelete` changed the argument `Snapshot` type from `string` to `[]string`.

Before 2.3.0:

```go
// If you have a string containing your snapshot
stringSnapshotsToDelete := "snapshot-1,snapshot-2"
reqSnapshots := &opensearchapi.SnapshotDeleteRequest{
  Repository: repo,
	Snapshot: stringSnapshotsToDelete,
}

// If you have a slice of strings containing your snapshot
sliceSnapshotToDelete := []string{"snapshot-1","snapshot-2"}
reqSnapshots := &opensearchapi.SnapshotDeleteRequest{
  Repository: repo,
  Snapshot: strings.Join(sliceSnapshotsToDelete, ","),
}
```

With 2.3.0:

```go
// If you have a string containing your snapshots
stringSnapshotsToDelete := strings.Split("snapshot-1,snapshot-2", ",")
reqSnapshots := &opensearchapi.SnapshotDeleteRequest{
  Repository: repo,
  Snapshot:   stringSnapshotsToDelete,
}

// If you have a slice of strings containing your snapshots
sliceSnapshotToDelete := []string{"snapshot-1", "snapshot-2"}
reqSnapshots := &opensearchapi.SnapshotDeleteRequest{
  Repository: repo,
  Snapshot: sliceSnapshotsToDelete,
}
```
