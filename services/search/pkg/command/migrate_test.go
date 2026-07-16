package command_test

import (
	"context"
	"path/filepath"
	"testing"

	bleveSearch "github.com/blevesearch/bleve/v2"
	"github.com/stretchr/testify/require"

	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/services/search/pkg/bleve"
	"github.com/opencloud-eu/opencloud/services/search/pkg/command"
	"github.com/opencloud-eu/opencloud/services/search/pkg/config/defaults"
	searchmapping "github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
)

func TestMigrateCommandBleve(t *testing.T) {
	root := t.TempDir()

	// an index left behind by an older release: a plain dynamic mapping, which
	// classifies as breaking against the code schema
	old, err := bleveSearch.New(filepath.Join(root, "bleve"), bleveSearch.NewIndexMapping())
	require.NoError(t, err)
	require.NoError(t, old.Index("doc1", map[string]any{"Name": "file.txt", "Mtime": "2026-01-01T00:00:00Z"}))
	require.NoError(t, old.Close())

	cfg := defaults.DefaultConfig()
	defaults.EnsureDefaults(cfg)
	cfg.Commons = &shared.Commons{Log: &shared.Log{}} // normally filled by the config pipeline
	cfg.Engine.Type = "bleve"
	cfg.Engine.Bleve.Datapath = root

	run := func() error {
		cmd := command.Migrate(cfg)
		cmd.SetContext(context.Background())
		return cmd.RunE(cmd, nil) // bypass PreRunE (parser requires service credentials)
	}

	// first run migrates, the index then classifies equal
	require.NoError(t, run())
	idx, classification, err := bleve.NewIndex(root)
	require.NoError(t, err)
	require.NoError(t, idx.Close())
	require.Equal(t, searchmapping.VerdictEqual, classification.Verdict)

	// second run is a no-op (idempotent), still succeeds
	require.NoError(t, run())
}
