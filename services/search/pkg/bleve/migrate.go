package bleve

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"

	"github.com/blevesearch/bleve/v2"

	"github.com/opencloud-eu/opencloud/pkg/log"
	searchmapping "github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

const (
	revisionKey        = "_schema_revision"
	migratingSuffix    = ".migrating"
	backupSuffix       = ".bak"
	migrateBatch       = 100
	migrateLogInterval = 50000 // how often buildReplacement logs progress, in documents
)

// OpenOrMigrate opens the index and, when autoMigrate is set and the stored
// schema is incompatible, rebuilds it in place first. The migration triggers on
// a schema revision bump, not on a mapping diff, so a breaking change without a
// matching bump still refuses to start. Returns the number of documents migrated
// (0 for none). Opening first means a fresh install and the common up-to-date
// case open the index only once; the migration path is taken only on a mismatch.
func OpenOrMigrate(root string, autoMigrate bool, logger log.Logger) (bleve.Index, searchmapping.Classification, int, error) {
	idx, classification, openErr := NewIndex(root)
	if openErr == nil || !autoMigrate || !errors.Is(openErr, searchmapping.ErrManualActionRequired) {
		// opened cleanly, or auto-migrate is off, or the failure is not a
		// schema mismatch we could migrate away
		return idx, classification, 0, openErr
	}

	// the stored schema is incompatible; rebuild only if the revision was bumped,
	// otherwise surface the original error (a schema change without a bump)
	migrated, err := MigrateIndex(root, logger)
	if err != nil {
		return nil, classification, 0, err
	}
	if migrated == 0 {
		return nil, classification, 0, openErr
	}
	idx, classification, err = NewIndex(root)
	return idx, classification, migrated, err
}

// MigrateIndex rebuilds the index from the documents it already holds when the
// stored schema revision is older than search.SchemaRevision, else a no-op. It
// takes the exclusive bolt lock, so the search service must be stopped.
func MigrateIndex(root string, logger log.Logger) (int, error) {
	dest := filepath.Join(root, "bleve")
	tmp := dest + migratingSuffix
	bak := dest + backupSuffix

	if err := recoverMigration(root); err != nil {
		return 0, err
	}

	old, err := bleve.OpenUsing(dest, openRuntimeConfig)
	if errors.Is(err, bleve.ErrorIndexPathDoesNotExist) {
		return 0, nil // nothing to migrate yet; NewIndex creates a fresh index
	}
	if err != nil {
		return 0, fmt.Errorf("open index (is the search service still running?): %w", err)
	}

	stored, err := readRevision(old)
	if err != nil {
		_ = old.Close()
		return 0, err
	}
	if stored >= search.SchemaRevision {
		_ = old.Close()
		return 0, nil
	}

	total, err := old.DocCount()
	if err != nil {
		_ = old.Close()
		return 0, err
	}
	logger.Info().Uint64("documents", total).Msg("starting search index migration")

	copied, err := buildReplacement(old, tmp, total, logger)
	if err != nil {
		_ = old.Close()
		_ = os.RemoveAll(tmp)
		return 0, err
	}

	// verify before the swap, so a failure leaves the original untouched
	if err := verifyMigrated(tmp, total); err != nil {
		_ = old.Close()
		_ = os.RemoveAll(tmp)
		return 0, err
	}

	if err := swap(old, dest, tmp, bak); err != nil {
		return 0, err
	}
	logger.Info().Int("documents", copied).Msg("search index migration complete")
	return copied, nil
}

// buildReplacement copies every document from src into a fresh index at tmp via
// the same Deserialize -> PrepareForIndex round-trip the Move/Delete/Restore
// paths use, and stamps the current revision.
func buildReplacement(src bleve.Index, tmp string, total uint64, logger log.Logger) (int, error) {
	if err := os.RemoveAll(tmp); err != nil {
		return 0, err
	}
	m, err := NewMapping()
	if err != nil {
		return 0, err
	}
	dst, err := bleve.New(tmp, m)
	if err != nil {
		return 0, err
	}

	batch, err := NewBatch(dst, migrateBatch)
	if err != nil {
		_ = dst.Close()
		return 0, err
	}

	var copied int
	nextLog := migrateLogInterval
	var after []string
	for {
		req := bleve.NewSearchRequest(bleve.NewMatchAllQuery())
		req.Size = migrateBatch
		req.Fields = []string{"*"}
		req.SortBy([]string{"_id"}) // total order, no skips or dupes across pages
		req.SearchAfter = after
		res, err := src.Search(req)
		if err != nil {
			_ = dst.Close()
			return 0, err
		}
		if len(res.Hits) == 0 {
			break
		}
		for _, hit := range res.Hits {
			r := legacyHitToResource(hit.Fields)
			if err := batch.Upsert(hit.ID, r); err != nil { // hit.ID: the bleve key is authoritative
				_ = dst.Close()
				return 0, err
			}
			copied++
		}
		if copied >= nextLog {
			logger.Info().Msgf("migrating search index: %d of %d documents", copied, total)
			nextLog += migrateLogInterval
		}
		after = []string{res.Hits[len(res.Hits)-1].ID}
	}
	if err := batch.Push(); err != nil {
		_ = dst.Close()
		return 0, err
	}
	if err := writeRevision(dst); err != nil {
		_ = dst.Close()
		return 0, err
	}
	return copied, dst.Close()
}

// legacyHitToResource is the fixup hook before re-indexing, symmetric to the
// opensearch side. bleve Deserialize is fail-soft, so nothing is stripped here.
func legacyHitToResource(fields map[string]any) search.Resource {
	return *searchmapping.Deserialize[search.Resource](fields)
}

// verifyMigrated checks the rebuilt index holds want documents before the swap
// replaces the original. The count catches a pagination drop or batch-push loss;
// the mapping needs no check, it is built deterministically from NewMapping().
func verifyMigrated(path string, want uint64) error {
	idx, err := bleve.OpenUsing(path, openRuntimeConfig)
	if err != nil {
		return err
	}
	defer func() { _ = idx.Close() }()

	got, err := idx.DocCount()
	if err != nil {
		return err
	}
	if got != want {
		return fmt.Errorf("migrated index holds %d documents, expected %d", got, want)
	}
	return nil
}

// swap replaces the live index with the rebuilt one. old still holds the lock
// and is closed here right before the rename.
func swap(old bleve.Index, dest, tmp, bak string) error {
	if err := old.Close(); err != nil {
		return err
	}
	if err := os.Rename(dest, bak); err != nil {
		return err
	}
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Rename(bak, dest) // best effort roll back
		return err
	}
	return os.RemoveAll(bak)
}

// recoverMigration cleans up a crashed MigrateIndex and must run before the
// index is opened, else NewIndex would recreate an empty index mid-swap. Rolls
// back, never forward: a truncated tmp is a valid index, completeness is not
// observable.
func recoverMigration(root string) error {
	dest := filepath.Join(root, "bleve")
	tmp := dest + migratingSuffix
	bak := dest + backupSuffix

	if _, err := os.Stat(dest); err == nil {
		return errors.Join(os.RemoveAll(tmp), os.RemoveAll(bak)) // index present: tmp/bak are leftovers
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	if _, err := os.Stat(bak); err == nil {
		// crashed mid-swap: roll back
		if err := os.RemoveAll(tmp); err != nil {
			return err
		}
		return os.Rename(bak, dest)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	return os.RemoveAll(tmp) // fresh install, or index removed by hand
}

// writeRevision stamps the current revision so a fresh or migrated index is not
// seen as outdated.
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
