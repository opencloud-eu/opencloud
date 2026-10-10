package mapping

import (
	"github.com/opencloud-eu/opencloud/pkg/log"
)

// SchemaReconciler is the engine-specific half of the startup schema check that
// Reconcile drives, keeping the verdict-to-action policy in one place.
type SchemaReconciler interface {
	Classify() (Classification, error)
	// ApplyAdditive applies an additive change. persisted is true once the
	// schema is on disk, even if a later step (e.g. a bleve reopen) then fails.
	ApplyAdditive() (persisted bool, err error)
}

// Reconcile applies the shared verdict policy: equal is silent, breaking refuses
// with ManualActionRequiredError, additive is applied and warned about.
func Reconcile(index string, r SchemaReconciler, logger log.Logger) (Classification, error) {
	classification, err := r.Classify()
	if err != nil {
		return classification, err
	}

	switch classification.Verdict {
	case VerdictBreaking:
		return classification, ManualActionRequiredError(index, classification.Reasons)
	case VerdictAdditive:
		persisted, err := r.ApplyAdditive()
		if persisted {
			logger.Warn().Strs("fields", classification.NewFields).Str("index", index).Msg("extended the search index mapping with new fields; documents indexed before the upgrade do not contain them and queries on these fields will miss those documents until they are re-indexed; to re-index everything run: opencloud search index --all-spaces --force-rescan")
		}
		if err != nil {
			return classification, err
		}
	}

	return classification, nil
}

// LogNewIndexCreated logs that a fresh, empty index was created and how to
// backfill it. The create path does not run through Reconcile.
func LogNewIndexCreated(logger log.Logger, index string) {
	logger.Info().Str("index", index).Msg("created a new empty search index; if this OpenCloud instance already held files, they are not in it yet, index them by running: opencloud search index --all-spaces --force-rescan")
}
