package command

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/search/pkg/bleve"
	"github.com/opencloud-eu/opencloud/services/search/pkg/config"
	"github.com/opencloud-eu/opencloud/services/search/pkg/config/parser"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch"
)

// Migrate rebuilds the index for the current schema revision from the documents
// it holds, no re-crawl. Stop the service first. Idempotent.
func Migrate(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "rebuild the search index for the current schema revision",
		Long:  "Rebuild the search index from the documents it already holds when the schema revision changed, instead of re-crawling storage. Stop the search service before running it. Idempotent: a no-op when the index is already current.",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return configlog.ReturnFatal(parser.ParseConfig(cfg))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := log.Configure(cfg.Service.Name, cfg.Commons, cfg.LogLevel)

			switch cfg.Engine.Type {
			case "bleve":
				n, err := bleve.MigrateIndex(cfg.Engine.Bleve.Datapath, logger)
				if err != nil {
					return err
				}
				reportMigration(logger, n)
			case "open-search":
				client, err := opensearch.NewClient(cfg.Engine.OpenSearch.Client)
				if err != nil {
					return err
				}
				n, err := opensearch.MigrateIndex(cmd.Context(), cfg.Engine.OpenSearch.ResourceIndex.Name, client, logger)
				if err != nil {
					return err
				}
				reportMigration(logger, n)
			default:
				return fmt.Errorf("unknown search engine: %s", cfg.Engine.Type)
			}
			return nil
		},
	}
}

// rescanRecommendation is appended after a rebuild/migration: the index holds
// the migrated documents, but a full crawl is the only guaranteed-clean state.
const rescanRecommendation = "re-indexing with 'opencloud search index --all-spaces --force-rescan' is still recommended for a guaranteed-clean index"

func reportMigration(logger log.Logger, migrated int) {
	if migrated == 0 {
		logger.Info().Msg("the search index is already at the current schema revision, nothing to migrate")
		return
	}
	logger.Info().Int("documents", migrated).Msg("the search index was migrated; " + rescanRecommendation)
}

// Prune deletes the OpenSearch indices left by older schema revisions. Run it
// after a rollout has fully drained the old instances. bleve keeps one index in
// place, so there is nothing to prune there.
func Prune(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "prune",
		Short: "delete search indices older than the current schema revision",
		Long:  "Delete search indices left by older schema revisions, after a rollout has fully drained the old instances. OpenSearch only; refuses if the current-revision index does not exist yet.",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return configlog.ReturnFatal(parser.ParseConfig(cfg))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := log.Configure(cfg.Service.Name, cfg.Commons, cfg.LogLevel)

			switch cfg.Engine.Type {
			case "bleve":
				logger.Info().Msg("bleve keeps a single index in place, nothing to prune")
			case "open-search":
				client, err := opensearch.NewClient(cfg.Engine.OpenSearch.Client)
				if err != nil {
					return err
				}
				n, err := opensearch.PruneOldIndices(cmd.Context(), client, cfg.Engine.OpenSearch.ResourceIndex.Name, logger)
				if err != nil {
					return err
				}
				if n == 0 {
					logger.Info().Msg("no old search indices to prune")
				} else {
					logger.Info().Int("indices", n).Msg("pruned old search indices")
				}
			default:
				return fmt.Errorf("unknown search engine: %s", cfg.Engine.Type)
			}
			return nil
		},
	}
}
