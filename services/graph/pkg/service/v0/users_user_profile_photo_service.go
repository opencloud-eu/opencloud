package svc

import (
	"context"
	"crypto/tls"
	"fmt"
	stdhttp "net/http"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/storage/metadata"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config"
	revaMetadata "github.com/opencloud-eu/reva/v2/pkg/storage/utils/metadata"
)

// NewConfiguredUserProfilePhotoService creates the shared photo storage and download service.
func NewConfiguredUserProfilePhotoService(ctx context.Context, cfg *config.Config) (UsersUserProfilePhotoProvider, error) {
	photoStorage, err := revaMetadata.NewCS3Storage(
		cfg.Metadata.GatewayAddress,
		cfg.Metadata.StorageAddress,
		cfg.Metadata.SystemUserID,
		cfg.Metadata.SystemUserIDP,
		cfg.Metadata.SystemUserAPIKey,
	)
	if err != nil {
		return nil, fmt.Errorf("could not initialize reva metadata storage: %w", err)
	}

	photoStorage, err = metadata.NewLazyStorage(photoStorage)
	if err != nil {
		return nil, fmt.Errorf("could not initialize lazy metadata storage: %w", err)
	}

	if err := photoStorage.Init(ctx, "f2bdd61a-da7c-49fc-8203-0558109d1b4f"); err != nil {
		return nil, fmt.Errorf("could not initialize metadata storage: %w", err)
	}

	oidcIssuer := cfg.OIDCIssuer
	if oidcIssuer == "" && cfg.Commons != nil {
		oidcIssuer = cfg.Commons.OpenCloudURL
	}
	userProfilePhotoService, err := NewUsersUserProfilePhotoService(photoStorage, WithProfilePictureSync(
		newProfilePictureHTTPClient(cfg.ProfilePictureInsecure),
		cfg.ProfilePictureURLAllowlist,
		oidcIssuer,
	))
	if err != nil {
		return nil, fmt.Errorf("could not initialize user profile photo service: %w", err)
	}
	return userProfilePhotoService, nil
}

// newProfilePictureHTTPClient keeps download TLS settings independent of the event bus.
func newProfilePictureHTTPClient(insecure bool) *stdhttp.Client {
	transport := stdhttp.DefaultTransport.(*stdhttp.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: insecure, //nolint:gosec // Configurable for development and testing.
	}
	return &stdhttp.Client{
		Timeout:   10 * time.Second,
		Transport: transport,
		// Redirect targets have not passed the profile picture URL allowlist.
		CheckRedirect: func(_ *stdhttp.Request, _ []*stdhttp.Request) error {
			return stdhttp.ErrUseLastResponse
		},
	}
}
