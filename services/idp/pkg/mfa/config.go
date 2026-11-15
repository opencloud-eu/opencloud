package mfa

import (
	"fmt"
	"os"

	"github.com/opencloud-eu/reva/v2/pkg/store"
	microstore "go-micro.dev/v4/store"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/idp/pkg/config"
)

// NewFromConfig creates the Service for the given idp configuration. The TOTP secrets
// are encrypted with a key derived from the encryption secret of the idp.
func NewFromConfig(cfg *config.Config, logger log.Logger) (*Service, error) {
	if cfg.IDP.EncryptionSecretFile == "" {
		return nil, fmt.Errorf("mfa: an encryption secret file is required")
	}
	masterSecret, err := os.ReadFile(cfg.IDP.EncryptionSecretFile)
	if err != nil {
		return nil, fmt.Errorf("mfa: could not read the encryption secret: %w", err)
	}

	st := store.Create(
		store.Store(cfg.MFA.Store.Store),
		microstore.Nodes(cfg.MFA.Store.Nodes...),
		microstore.Database(cfg.MFA.Store.Database),
		store.Authentication(cfg.MFA.Store.AuthUsername, cfg.MFA.Store.AuthPassword),
		store.TLSEnabled(cfg.MFA.Store.EnableTLS),
		store.TLSInsecure(cfg.MFA.Store.TLSInsecure),
		store.TLSRootCA(cfg.MFA.Store.TLSRootCACertificate),
	)

	return New(st, masterSecret, cfg.MFA.Issuer, logger)
}
