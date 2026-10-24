---
title: "5. Unified Search Index Mapping"
---

* Status: accepted
* Deciders: @aduffeck, @butonic, @dschmidt, @fschade
* Date: 2026-04-23, accepted and updated to the implemented state 2026-08-31

Reference: implemented by https://github.com/opencloud-eu/opencloud/pull/3345 (reflection-based mapping, search siblings, shared query lowering) and https://github.com/opencloud-eu/opencloud/pull/3197 (schema versioning and startup checks). https://github.com/opencloud-eu/opencloud/pull/2659 was the original proof-of-concept.

## Context and Problem Statement

This section describes the state at decision time (April 2026); the implementation has since resolved the problems listed here.

The search service supports two backends, bleve (embedded) and
OpenSearch (external). Each backend currently carries its own,
independently maintained description of the index layout:

- The bleve backend hand-builds a document mapping that explicitly
  declares only Name, Tags, Favorites and Content. Everything else,
  including the entire facet block (audio, image, photo, location),
  is left to bleve's dynamic mapping.
- The OpenSearch backend ships a static JSON template that covers a
  similar but not identical subset, plus a few OpenSearch-specific
  primitives (path_hierarchy analyzer, wildcard MimeType). It does
  not list the facet sub-fields either; they are produced by
  OpenSearch's dynamic templating at first write.
- The graph DriveItem assembly path keeps its own private copy of a
  reflection-based walker to turn CS3 ArbitraryMetadata back into
  typed libregraph facets, parallel to the search service's
  reflection helpers but maintained separately.
- The bleve KQL compiler keeps a hand-maintained set of field names
  whose query values need to be pre-lowercased, with a comment that
  literally says "Keep in sync with index.go".

The current implementation has three concrete problems:

1. **The two backends do not behave the same.** Both rely on their
   own implicit defaults for fields that are not explicitly
   declared. The inferred shapes differ: bleve produces keyword-
   analyzed text, OpenSearch produces a `text + keyword` multi-field
   with auto-detected dates. Nobody has written down which behavior
   is the intended one. Two concrete instances surfaced while building
   #2659:
   - **mtime** is stored as an RFC3339 string. OpenSearch's dynamic
     mapping auto-detects it as `date`; bleve leaves it `keyword`. So
     `mtime:>...` is a chronological range on OpenSearch but a
     lexicographic string compare on bleve.
   - **name/tags**: bleve indexes a single lowercase token (exact or
     wildcard match only); OpenSearch word-tokenizes, so a bare
     `name:report` matches "My Report.txt" on OpenSearch but not on
     bleve.
2. **Drift risk.** The OpenSearch JSON template is a subset of what
   actually gets indexed. Even where it overlaps with the bleve
   mapping it diverges on analyzer choices. Because the facet
   fields were not reachable from user queries at the time (no dot
   syntax in the KQL compilers, no facet exposure on the hit and
   REPORT paths), the divergence has been invisible, but it would
   surface the moment the first working cross-backend facet query
   landed.
3. **Per-facet cost.** Adding a new facet (motionPhoto, etc.)
   requires coordinated edits across the proto message, both backend
   mappings, the bleve hit converters, the OpenSearch convert
   closures, the search service's metadata persistence, the graph
   DriveItem assembly, and the KQL compiler's lowercasing set. Most
   of those edits are boilerplate following a copy-paste pattern.
   Adding a genuinely new index capability (geopoint, wildcard,
   ...) means wiring it in at every one of those sites, and there
   is no single place to hook a type-specific adapter.

### A note on backwards compatibility

That the facet fields were unreachable at decision time has a
useful corollary for this ADR: **changing the indexed shape of the
facet fields cannot break any existing client of the search
service**, because no client could successfully read them. The behavior changes
discussed below are therefore additive in a literal sense; nothing
that works today stops working as a result.

## Decision Drivers

* **Predictable OpenCloud API behavior independent of backend.**
  Consumers of the search service should be able to rely on the
  documented behavior of the API, not on which backend happens to
  be configured. Today the same query can give different results
  depending on whether bleve or OpenSearch is wired in (bleve's
  dynamic default is `keyword`, exact match; OpenSearch's dynamic
  default is `text + keyword`, also matches sub-tokens of a
  string). That is backend-implementation leakage, and trying to
  keep the two implicit defaults synchronized has not worked.
* Single source of truth for the indexed schema, so the two backends
  cannot drift silently again.
* Reduce the per-facet cost so future facets (motionPhoto and
  whatever comes next) can be added with minimal boilerplate.
* Establish a single place to hook index-type-specific behavior, so
  a new capability needs to be implemented at most once per backend
  and then becomes available for any field uniformly.
* A one-time reindex is an acceptable upgrade path. Both bleve and
  OpenSearch store their mapping alongside the data; existing
  indexes keep serving queries against their stored shape without
  any automatic reshaping. Benefiting from the new behavior is done
  by creating a fresh index and re-ingesting, which is the normal
  reindex flow, rather than by inventing migration tooling.

## Considered Options

### Option 1: Do nothing, keep relying on implicit backend defaults

Accept that bleve and OpenSearch each fall back to their own
dynamic-mapping defaults for whatever is not explicitly declared,
and treat the observable search behavior of OpenCloud as "whatever
the configured backend happens to do". Adding a facet stays a
copy-paste coordination across half a dozen sites; the existing
divergence between bleve (keyword) and OpenSearch (`text + keyword`
multi-field plus auto-date detection) stays silently in place
until a working query actually reaches the diverging field and
returns different answers on the two backends.

Low upfront work, but it makes the OpenCloud API behavior a
function of the backend rather than a contract, and it keeps the
per-facet boilerplate cost for every new field.

### Option 2: Generate one backend's mapping from the other

Treat one backend as canonical (likely bleve, because Go types) and
derive the other. Partial answer; it still does not help the reader
path or the graph walker, and still leaves per-facet boilerplate in
non-mapping code.

### Option 3: A struct-driven mapping (chosen)

Let the Go struct that represents an indexed document, together
with a small overrides map, be the single source of truth. A
reflection-based helper walks the struct via json tags and emits
each backend's index mapping. The same definition drives the
write-time path, the hit-decoding path, and the query compiler's
case-folding rules. Any future field follows one declaration in
one place and falls through the whole pipeline consistently.

## Decision Outcome

Adopt Option 3. The Go struct that represents an indexed document,
together with a small overrides map, becomes the single source of
truth for the search index. The bleve and OpenSearch index
mappings, the write-time conversion, the hit-decoding path, and
the query compiler's case-folding rules are all derived from that
same definition. Drift between backends is prevented by
construction, because there is no second place to edit.

The overrides surface stays small. Each entry declares one of a
handful of things per field: a semantic type for fields whose
intent cannot be inferred from the Go type (for example a path-
analyzed field, a fulltext field, a geopoint field), or search-
behavior flags (case-insensitivity, word breaking, inclusion in
the catch-all field). Any field that needs something beyond the
inferred defaults gets one line in the overrides map and that one
line flows through every derived piece. Overrides are validated at
startup so a typo fails loudly instead of silently disabling a
setting.

A practical consequence of having one place to hook things: when a
new capability is needed (a geopoint representation, a sibling
field for a different aggregation behavior, a different analyzer
for a class of fields, ...) it can be implemented once per backend
in the central pipeline. After that, turning the capability on for
a specific field is a single override entry, and both backends
adopt it the same way. This ADR does not decide which capabilities
to add, only that they will land in this uniform shape rather than
through coordinated per-site edits.

### Facet values are indexed as case-preserving keywords

All facet sub-fields, meaning any leaf inside `audio`, `photo`, `image`, `location` and the facets that followed (`video`, `motionPhoto`, `livePhoto`), keep a case-preserving keyword as their stored base field on both backends. The raw value the extractor saw, or the CS3 ArbitraryMetadata string, is what lands in the index, and it is what returning, sorting and aggregations read.

This is the single intended semantic for facets across bleve and
OpenSearch, and it is driven by what aggregations need.
Aggregation buckets ("group all files by `audio.artist`", "list
distinct `photo.cameraMake`") return bucket keys drawn from the
indexed terms. If the indexing analyzer lowercases (OpenSearch's
default `text + keyword` multi-field against the text leg, or a
`lowercaseKeyword`-style analyzer), the buckets come back lower-
cased: a distinct-artists query would answer `motörhead` and
`queen` instead of the original display casings, and two tag
writers using `Motörhead` versus `MOTÖRHEAD` would collapse into a
single bucket labelled `motörhead`. For a metadata display use
case (thumbnails, facet filters in the UI, distinct lists) that
behavior is not what we want.

Searching is layered on top as exactly the strict superset the proposal reserved for later, and it shipped with the implementation: every keyword field additionally gets search-only sibling fields derived from the same definition, a `_lowercase` keyword sibling (doc values disabled; serves wildcards and `=` whole-value matches) and a `_words` text sibling (`words` analyzer: dots to spaces, unicode tokenization, lowercasing, no stemming; serves token and phrase matches). Case-insensitive, word-broken search is the default for every keyword field including facets; fields opt out per override where that is wrong: opaque ids (`ID`, `RootID`, `ParentID`, `Favorites`, `livePhoto.contentId`), the POSIX `Path`, the normalized `MimeType`, and `Content`, which is a fulltext field of its own. Aggregation buckets keep their display casing because they read the base field, never the siblings.

The query side derives from the same source: the shared lowering pass resolves field names case-insensitively, folds values and routes each match to the right sibling (wildcards to `_lowercase`, tokens and phrases to `_words`, `=` as a whole-value term on `_lowercase`), and both backend compilers consume that one decision. The engine parity suite pins the resulting behavior against bleve and OpenSearch, so a divergence fails CI instead of surfacing in production. The case-sensitivity alignment started in #2633 is completed by deriving both sides from the same source.

### Schema versioning and upgrades

Index names carry a schema version derived from the single `search.SchemaVersion` constant (`opencloud-resource-v4`, `bleve-v4`). On startup the service classifies the stored mapping against the code: additive changes (new fields, unchanged analyzers) are reconciled in place without a version bump, breaking changes make the service refuse to start and name the reindex steps. The upgrade path is a plain reindex (`opencloud search index --all-spaces`) into the new versioned index; older indexes stay untouched and can be deleted afterwards (services/search/MIGRATION.md). Golden mapping tests on both backends pin the rendered mappings and reuse the same classifier to tell a contributor whether a change needs only a golden regeneration or a version bump.

### Known trade-off

The write-time pipeline produces the document as a generic map via
a json round-trip. The OpenSearch write path already does the
equivalent today via the same json-based conversion helper, so
that path is unchanged. The bleve write path, which previously
handed the struct directly to bleve's reflective indexer, now goes
through the same map-producing step and pays roughly the same
cost. On hot paths (initial indexing of a large space) this is
measurable but not significant; if it ever matters, a direct
reflection walker can replace the json round-trip without changing
any call site.

### Follow-ups out of scope for this ADR

- **WebDAV REPORT facet exposure.** The current webdav search
  endpoint renders none of the facet fields back to the client.
  This is a missing feature, not a regression of the proposal;
  its natural resolution is to let the graph-search endpoint
  (proposed in #3211) take over once graph search lands.
- **Graph search hit conversion.** Graph search (#3211) translates
  proto hits back into libregraph DriveItems with the same
  facet-copy helper the search service uses internally.
- **reva's PROPFIND facet listing** uses its own hand-maintained
  per-facet key lists. reva deliberately does not depend on the
  libregraph Go types, so unifying those key sets is a reva-side
  decision tracked separately.
- **Write-path performance.** The json round-trip in the bleve
  write path is an optional optimisation target with no call-site
  impact when it lands.
