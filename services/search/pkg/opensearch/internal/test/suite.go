package opensearchtest

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/opensearch"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/opencloud-eu/opencloud/services/search/pkg/config"
	"github.com/opencloud-eu/opencloud/services/search/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/services/search/pkg/config/parser"
)

const (
	openSearchImage = "opensearchproject/opensearch:2"
)

func SetupTests(ctx context.Context) (*config.Config, func(), error) {
	cfg, err := setupDefaultConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to setup default configuration: %w", err)
	}

	cleanupOpenSearch, err := setupOpenSearchTestContainer(ctx, cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to setup OpenSearch test container: %w", err)
	}

	return cfg, cleanupOpenSearch, nil
}

func setupDefaultConfig() (*config.Config, error) {
	cfg := defaults.DefaultConfig()
	defaults.EnsureDefaults(cfg)

	{ // parser.Validate requires these fields to be set even if they are not used in these tests.
		cfg.TokenManager.JWTSecret = "any-jwt"
		cfg.ServiceAccount.ServiceAccountID = "any-service-account-id"
		cfg.ServiceAccount.ServiceAccountSecret = "any-service-account-secret"
	}

	if err := parser.ParseConfig(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse default configuration: %w", err)
	}

	return cfg, nil
}

func setupOpenSearchTestContainer(ctx context.Context, cfg *config.Config) (func(), error) {
	usesExternalOpensearch := slices.Contains([]bool{
		os.Getenv("CI") == "woodpecker",
		os.Getenv("CI_SYSTEM_NAME") == "woodpecker",
		os.Getenv("USE_TESTCONTAINERS") == "false",
	}, true)
	if usesExternalOpensearch {
		return func() {}, nil
	}

	containerName := fmt.Sprintf("opencloud/test/%s", openSearchImage)
	containerName = strings.Replace(containerName, "/", "__", -1)
	containerName = strings.Replace(containerName, ":", "_", -1)
	container, err := opensearch.Run(
		ctx,
		openSearchImage,
		opensearch.WithUsername(cfg.Engine.OpenSearch.Client.Username),
		opensearch.WithPassword(cfg.Engine.OpenSearch.Client.Password),
		testcontainers.WithName(containerName),
		testcontainers.WithReuseByName(containerName),
		testcontainers.WithWaitStrategy(
			wait.ForLog("ML configuration initialized successfully").
				WithStartupTimeout(5*time.Second),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start OpenSearch container: %w", err)
	}

	address, err := container.Address(ctx)
	if err != nil {
		_ = container.Terminate(ctx) // attempt to clean up the container
		return nil, fmt.Errorf("failed to get OpenSearch container address: %w", err)
	}

	// Ensure the address is set in the default configuration.
	cfg.Engine.OpenSearch.Client.Addresses = []string{address}

	return func() {
		err := container.Terminate(ctx)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "failed to terminate OpenSearch container: %v\n", err)
		}
	}, nil
}
