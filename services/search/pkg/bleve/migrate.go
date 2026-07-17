package bleve

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/blevesearch/bleve/v2"

	"github.com/opencloud-eu/opencloud/pkg/log"
	searchmapping "github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

const revisionKey = "_schema_revision"

// OpenOrMigrate opens the index and, when autoMigrate is set and the stored
// schema is incompatible, resets it first so a re-crawl can repopulate it at the
// current schema. The reset triggers on a schema revision bump, not on a mapping
// diff, so a breaking change without a matching bump still refuses to start.
func OpenOrMigrate(root string, autoMigrate bool, logger log.Logger) (bleve.Index, searchmapping.Classification, int, error) {
	idx, classification, openErr := NewIndex(root)
	if openErr == nil || !autoMigrate || !errors.Is(openErr, searchmapping.ErrManualActionRequired) {
		return idx, classification, 0, openErr
	}

	// incompatible schema: MigrateIndex drops a bumped-revision index so the
	// reopen below rebuilds it empty; a breaking change without a bump is a no-op
	// there and is surfaced by that reopen instead
	dropped, err := MigrateIndex(root, logger)
	if err != nil {
		return nil, classification, 0, err
	}
	idx, classification, err = NewIndex(root)
	return idx, classification, dropped, err
}

// MigrateIndex resets the index when its stored schema revision is older than
// search.SchemaRevision: it drops the outdated index so a full re-crawl
// ('opencloud search index --all-spaces --force-rescan') repopulates it at the
// current schema. No-op when the index is missing or already current. It takes
// the exclusive bolt lock, so the search service must be stopped. Returns the
// number of documents that were dropped.
func MigrateIndex(root string, logger log.Logger) (int, error) {
	dest := filepath.Join(root, "bleve")

	idx, err := bleve.OpenUsing(dest, openRuntimeConfig)
	if errors.Is(err, bleve.ErrorIndexPathDoesNotExist) {
		return 0, nil // fresh install, nothing to migrate
	}
	if err != nil {
		return 0, fmt.Errorf("open index (is the search service still running?): %w", err)
	}
	stored, rerr := readRevision(idx)
	count, cerr := idx.DocCount()
	_ = idx.Close()
	switch {
	case rerr != nil:
		return 0, rerr
	case cerr != nil:
		return 0, cerr
	case stored >= search.SchemaRevision:
		return 0, nil // already current
	}

	if err := os.RemoveAll(dest); err != nil {
		return 0, err
	}
	logger.Warn().Msgf("the bleve index at %s was reset for schema revision %d (%d documents dropped); run 'opencloud search index --all-spaces --force-rescan' to repopulate", root, search.SchemaRevision, count)
	return int(count), nil
}

// writeRevision stamps the current revision so a fresh index is not seen as
// outdated on the next start.
func writeRevision(index bleve.Index) error {
	return index.SetInternal([]byte(revisionKey), []byte(strconv.Itoa(search.SchemaRevision)))
}

// readRevision returns the stored schema revision, or 0 for an index that
// predates the marker.
func readRevision(index bleve.Index) (int, error) {
	raw, err := index.GetInternal([]byte(revisionKey))
	if err != nil {
		return 0, err
	}
	if len(raw) == 0 {
		return 0, nil
	}
	return strconv.Atoi(string(raw))
}
