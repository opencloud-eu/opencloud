package opensearch

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	opensearchgoAPI "github.com/opensearch-project/opensearch-go/v4/opensearchapi"

	"github.com/opencloud-eu/opencloud/pkg/conversions"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

const (
	// migratePageSize is the scroll page size for reading the source index.
	migratePageSize = 1000
	// migrateLogInterval is how often reindex logs progress, in documents.
	migrateLogInterval = 50000
	// migrateScrollKeepAlive keeps the source snapshot alive across scroll pages.
	migrateScrollKeepAlive = 5 * time.Minute
)

// TargetIndex is the concrete index for the current schema revision. The
// revision is the index name suffix, so instances of different revisions use
// different indices and never share one.
func TargetIndex(base string) string {
	return fmt.Sprintf("%s-v%d", base, search.SchemaRevision)
}

// MigrateIndex fills the current-revision index (<base>-v<rev>) from the newest
// older-revision index, in-process. Because the revision is in the index name,
// instances of different revisions never share an index and no alias flip is
// needed: run it as a pre-rollout step so new instances find their index ready.
// Reindexing uses create-only bulk ops, so a document already present is skipped,
// never overwritten: the run is idempotent and safe against a target an instance
// is already writing to. A no-op when there is no older index to migrate from
// (fresh install, or pruned). Returns the number of documents newly created.
func MigrateIndex(ctx context.Context, base string, client *opensearchgoAPI.Client, logger log.Logger) (int, error) {
	target := TargetIndex(base)

	source, oldVersion, err := previousIndex(ctx, client, base)
	if err != nil {
		return 0, err
	}
	if source == "" {
		return 0, nil // nothing older to migrate from
	}

	exists, err := indexExists(ctx, client, target)
	if err != nil {
		return 0, err
	}
	if !exists {
		mappingB, err := IndexManagerLatest.MarshalJSON()
		if err != nil {
			return 0, err
		}
		resp, err := client.Indices.Create(ctx, opensearchgoAPI.IndicesCreateReq{
			Index: target,
			Body:  strings.NewReader(string(mappingB)),
		})
		switch {
		case err != nil:
			return 0, fmt.Errorf("create %s: %w", target, err)
		case !resp.Acknowledged:
			return 0, fmt.Errorf("create %s not acknowledged", target)
		}
	}
	logger.Info().Str("source", source).Str("target", target).Msg("starting search index migration")

	created, skipped, err := reindex(ctx, client, source, target, oldVersion, logger)
	if err != nil {
		return 0, err
	}
	logger.Info().Str("target", target).Int("created", created).Int("skipped", skipped).Msg("search index migration complete")
	return created, nil
}

// PruneOldIndices deletes every index older than the current schema revision:
// the versioned <base>-v<k> with k < SchemaRevision plus the legacy unversioned
// <base>. Run it after a rollout has fully drained the old instances. It refuses
// when the current-revision index does not exist yet, so a premature prune can
// not wipe the only data.
func PruneOldIndices(ctx context.Context, client *opensearchgoAPI.Client, base string, logger log.Logger) (int, error) {
	current, err := indexExists(ctx, client, TargetIndex(base))
	if err != nil {
		return 0, err
	}
	if !current {
		return 0, fmt.Errorf("current index %s does not exist; migrate before pruning", TargetIndex(base))
	}

	older, err := olderVersionedIndices(ctx, client, base)
	if err != nil {
		return 0, err
	}
	old := make([]string, 0, len(older))
	for _, name := range older {
		old = append(old, name)
	}
	if legacy, err := indexExists(ctx, client, base); err != nil {
		return 0, err
	} else if legacy {
		old = append(old, base)
	}
	if len(old) == 0 {
		return 0, nil
	}

	delResp, err := client.Indices.Delete(ctx, opensearchgoAPI.IndicesDeleteReq{Indices: old})
	if err != nil {
		return 0, fmt.Errorf("delete old indices %v: %w", old, err)
	}
	if !delResp.Acknowledged {
		return 0, fmt.Errorf("prune not acknowledged")
	}
	logger.Info().Strs("indices", old).Msg("pruned old search indices")
	return len(old), nil
}

// previousIndex returns the newest index to migrate from and its schema version:
// the highest-revision <base>-v<k> below the current SchemaRevision, or, if none
// exists yet, the legacy unversioned <base> index (version 0) left by a
// pre-versioning release. name is "" when there is nothing older to migrate from.
func previousIndex(ctx context.Context, client *opensearchgoAPI.Client, base string) (name string, version int, err error) {
	older, err := olderVersionedIndices(ctx, client, base)
	if err != nil {
		return "", 0, err
	}
	best := -1
	for k := range older {
		if k > best {
			best = k
		}
	}
	if best >= 0 {
		return older[best], best, nil
	}

	// no versioned index yet: fall back to the legacy unversioned index (version 0)
	legacy, err := indexExists(ctx, client, base)
	if err != nil {
		return "", 0, err
	}
	if legacy {
		return base, 0, nil
	}
	return "", 0, nil
}

// olderVersionedIndices returns the existing <base>-v<k> indices with k below the
// current SchemaRevision, keyed by version.
func olderVersionedIndices(ctx context.Context, client *opensearchgoAPI.Client, base string) (map[int]string, error) {
	resp, err := client.Indices.Get(ctx, opensearchgoAPI.IndicesGetReq{Indices: []string{base + "-v*"}})
	if err != nil {
		return nil, fmt.Errorf("list %s-v* indices: %w", base, err)
	}
	prefix := base + "-v"
	out := map[int]string{}
	for name := range resp.Indices {
		suffix, ok := strings.CutPrefix(name, prefix)
		if !ok {
			continue
		}
		if k, err := strconv.Atoi(suffix); err == nil && k < search.SchemaRevision {
			out[k] = name
		}
	}
	return out, nil
}

// indexExists reports whether the concrete index exists. A transient error is
// returned so callers do not treat "unknown" as "absent".
func indexExists(ctx context.Context, client *opensearchgoAPI.Client, index string) (bool, error) {
	resp, err := client.Indices.Exists(ctx, opensearchgoAPI.IndicesExistsReq{Indices: []string{index}})
	switch {
	case resp != nil && resp.StatusCode == 404:
		return false, nil
	case err != nil:
		return false, fmt.Errorf("check index %s: %w", index, err)
	}
	return true, nil
}

// reindex copies source into target with create-only bulk ops (same
// PrepareForIndex transform as the live write path). It uses a scroll: a
// consistent snapshot of the source even while old instances keep writing to it.
// Returns the number of documents created and skipped (already present).
func reindex(ctx context.Context, client *opensearchgoAPI.Client, source, target string, oldVersion int, logger log.Logger) (int, int, error) {
	first, err := client.Search(ctx, &opensearchgoAPI.SearchReq{
		Indices: []string{source},
		Params:  opensearchgoAPI.SearchParams{Scroll: migrateScrollKeepAlive},
		// track_total_hits so Total.Value is exact, not capped at 10000: the
		// completeness check below relies on it
		Body: strings.NewReader(fmt.Sprintf(`{"size":%d,"track_total_hits":true,"query":{"match_all":{}}}`, migratePageSize)),
	})
	if err != nil {
		return 0, 0, err
	}
	scrollID := ""
	if first.ScrollID != nil {
		scrollID = *first.ScrollID
	}
	defer func() {
		if scrollID != "" {
			_, _ = client.Scroll.Delete(ctx, opensearchgoAPI.ScrollDeleteReq{ScrollIDs: []string{scrollID}})
		}
	}()

	total := first.Hits.Total.Value
	var created, skipped int
	nextLog := migrateLogInterval
	hits := first.Hits.Hits
	for len(hits) > 0 {
		c, s, err := bulkCreate(ctx, client, target, hits, oldVersion)
		if err != nil {
			return created, skipped, err
		}
		created += c
		skipped += s
		if created+skipped >= nextLog {
			logger.Info().Msgf("migrating search index: %d of %d documents", created+skipped, total)
			nextLog += migrateLogInterval
		}
		next, err := client.Scroll.Get(ctx, opensearchgoAPI.ScrollGetReq{
			ScrollID: scrollID,
			Params:   opensearchgoAPI.ScrollGetParams{Scroll: migrateScrollKeepAlive},
		})
		if err != nil {
			return created, skipped, err
		}
		if next.ScrollID != nil {
			scrollID = *next.ScrollID
		}
		hits = next.Hits.Hits
	}
	if created+skipped != total {
		return created, skipped, fmt.Errorf("processed %d of %d source documents", created+skipped, total)
	}
	return created, skipped, nil
}

// bulkCreate inserts hits into target with create semantics: a document already
// present is a 409 conflict, counted as skipped and never overwritten.
func bulkCreate(ctx context.Context, client *opensearchgoAPI.Client, target string, hits []opensearchgoAPI.SearchHit, oldVersion int) (int, int, error) {
	var body strings.Builder
	for _, hit := range hits {
		r, err := legacyHitToResource(hit.Source, oldVersion)
		if err != nil {
			return 0, 0, fmt.Errorf("convert document %s: %w", hit.ID, err)
		}
		doc, err := mapping.PrepareForIndex(r, r.SearchFieldOverrides())
		if err != nil {
			return 0, 0, fmt.Errorf("prepare document %s: %w", hit.ID, err)
		}
		action, err := json.Marshal(map[string]any{"create": map[string]any{"_index": target, "_id": hit.ID}})
		if err != nil {
			return 0, 0, err
		}
		docJSON, err := json.Marshal(doc)
		if err != nil {
			return 0, 0, err
		}
		body.Write(action)
		body.WriteByte('\n')
		body.Write(docJSON)
		body.WriteByte('\n')
	}

	resp, err := client.Bulk(ctx, opensearchgoAPI.BulkReq{Body: strings.NewReader(body.String())})
	if err != nil {
		return 0, 0, fmt.Errorf("bulk create: %w", err)
	}
	var created, skipped int
	for _, item := range resp.Items {
		res, ok := item["create"]
		if !ok {
			continue
		}
		switch {
		case res.Status == 409:
			skipped++ // already present, keep the existing (newer) document
		case res.Error != nil || res.Status >= 300:
			reason := ""
			if res.Error != nil {
				reason = res.Error.Type + ": " + res.Error.Reason
			}
			return created, skipped, fmt.Errorf("create document %s failed: %s", res.ID, reason)
		default:
			created++
		}
	}
	return created, skipped, nil
}

// legacyHitToResource is the per-document fixup applied while reindexing.
// oldVersion is the schema version of the SOURCE index (0 for the legacy
// unversioned index), so fixups are gated on where the document comes from: the
// empty-Mtime strip only runs for documents from the pre-date schema. The switch
// is the hook for future revision-dependent migrations.
func legacyHitToResource(source json.RawMessage, oldVersion int) (search.Resource, error) {
	var m map[string]any
	if err := json.Unmarshal(source, &m); err != nil {
		return search.Resource{}, err
	}
	switch {
	case oldVersion <= 1:
		// pre-date schema: an empty Mtime is not a valid date, strip it
		if v, ok := m["Mtime"]; ok && v == "" {
			delete(m, "Mtime")
		}
	}
	return conversions.To[search.Resource](m)
}
