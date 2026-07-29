package mapping

import (
	"github.com/opencloud-eu/opencloud/pkg/log"
)

// SchemaReconciler is the engine-specific half of the startup schema check.
// Classify reads the stored and code schema and returns the verdict (with any
// engine-specific extras, e.g. analyzer/settings drift, already folded in);
// ApplyAdditive applies an additive change to the live index. Reconcile drives
// them so the verdict-to-action mapping lives in one place for every backend.
type SchemaReconciler interface {
	Classify() (Classification, error)
	// ApplyAdditive applies an additive change to the live index and reports
	// whether the schema was persisted. It returns persisted=true even when a
	// later step then fails (e.g. a bleve reopen), so Reconcile can still warn:
	// the change is on disk, a subsequent start classifies equal and stays
	// silent, so this is the only chance to surface the new fields.
	ApplyAdditive() (persisted bool, err error)
}

// Reconcile runs the shared schema-verdict flow: an equal schema starts
// silently, a breaking one refuses with ManualActionRequiredError, an additive
// one is applied and warned about. index names the index in the messages.
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
