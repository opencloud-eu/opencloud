package command

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"os"
	"os/signal"
	"strings"

	"github.com/go-ldap/ldif"
	"github.com/libregraph/idm/pkg/ldappassword"
	"github.com/libregraph/idm/pkg/ldbbolt"
	"github.com/libregraph/idm/server"
	"github.com/urfave/cli/v2"

	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	pkgcrypto "github.com/opencloud-eu/opencloud/pkg/crypto"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/runner"
	"github.com/opencloud-eu/opencloud/services/idm"
	"github.com/opencloud-eu/opencloud/services/idm/pkg/config"
	"github.com/opencloud-eu/opencloud/services/idm/pkg/config/parser"
	"github.com/opencloud-eu/opencloud/services/idm/pkg/logging"
	"github.com/opencloud-eu/opencloud/services/idm/pkg/server/debug"
)

// Server is the entrypoint for the server command.
func Server(cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:     "server",
		Usage:    fmt.Sprintf("start the %s service without runtime (unsupervised mode)", cfg.Service.Name),
		Category: "server",
		Before: func(c *cli.Context) error {
			return configlog.ReturnFatal(parser.ParseConfig(cfg))
		},
		Action: func(c *cli.Context) error {
			var cancel context.CancelFunc
			if cfg.Context == nil {
				cfg.Context, cancel = signal.NotifyContext(context.Background(), runner.StopSignals...)
				defer cancel()
			}
			ctx := cfg.Context

			logger := logging.Configure(cfg.Service.Name, cfg.Log)

			gr := runner.NewGroup()
			{
				servercfg := server.Config{
					Logger:          log.LogrusWrap(logger.Logger),
					LDAPHandler:     "boltdb",
					LDAPSListenAddr: cfg.IDM.LDAPSAddr,
					TLSCertFile:     cfg.IDM.Cert,
					TLSKeyFile:      cfg.IDM.Key,
					LDAPBaseDN:      "o=libregraph-idm",
					LDAPAdminDN:     "uid=libregraph,ou=sysusers,o=libregraph-idm",

					BoltDBFile: cfg.IDM.DatabasePath,
				}

				if cfg.IDM.LDAPSAddr != "" {
					// Generate a self-signing cert if no certificate is present
					if err := pkgcrypto.GenCert(cfg.IDM.Cert, cfg.IDM.Key, logger); err != nil {
						logger.Fatal().Err(err).Msgf("Could not generate test-certificate")
					}
				}
				if _, err := os.Stat(servercfg.BoltDBFile); errors.Is(err, os.ErrNotExist) {
					logger.Debug().Msg("Bootstrapping IDM database")
					if err = bootstrap(logger, cfg, servercfg); err != nil {
						logger.Error().Err(err).Msg("failed to bootstrap idm database")
					}
				}

				svc, err := server.NewServer(&servercfg)
				if err != nil {
					return err
				}

				// we need an additional context for the idm server in order to
				// cancel it anytime
				svcCtx, svcCancel := context.WithCancel(ctx)
				defer svcCancel()

				gr.Add(runner.New("idm_svc", func() error {
					return svc.Serve(svcCtx)
				}, func() {
					svcCancel()
				}))
			}

			{
				debugServer, err := debug.Server(
					debug.Logger(logger),
					debug.Context(ctx),
					debug.Config(cfg),
				)
				if err != nil {
					logger.Info().Err(err).Str("server", "debug").Msg("Failed to initialize server")
					return err
				}

				gr.Add(runner.NewGolangHttpServerRunner("idm_debug", debugServer))
			}

			grResults := gr.Run(ctx)

			// return the first non-nil error found in the results
			for _, grResult := range grResults {
				if grResult.RunnerError != nil {
					return grResult.RunnerError
				}
			}
			return nil
		},
	}
}

func bootstrap(logger log.Logger, cfg *config.Config, srvcfg server.Config) error {
	// Hash password if the config does not supply a hash already
	var err error

	type svcUser struct {
		Name     string
		Password string
		ID       string
		Issuer   string
	}

	serviceUsers := []svcUser{
		{
			Name:     "libregraph",
			Password: cfg.ServiceUserPasswords.Idm,
		},
		{
			Name:     "idp",
			Password: cfg.ServiceUserPasswords.Idp,
		},
		{
			Name:     "reva",
			Password: cfg.ServiceUserPasswords.Reva,
		},
	}

	if cfg.AdminUserID != "" {
		serviceUsers = append(serviceUsers, svcUser{
			Name:     "admin",
			Password: cfg.ServiceUserPasswords.OCAdmin,
			ID:       cfg.AdminUserID,
			Issuer:   cfg.DemoUsersIssuerUrl,
		})
	}

	bdb := &ldbbolt.LdbBolt{}

	if err := bdb.Configure(srvcfg.Logger, srvcfg.LDAPBaseDN, srvcfg.BoltDBFile, nil); err != nil {
		return err
	}
	defer bdb.Close()

	if err := bdb.Initialize(); err != nil {
		return err
	}

	// Prepare the initial Data from template. To be able to set the
	// supplied service user passwords
	tmpl, err := template.New("baseldif").Parse(idm.BaseLDIF)
	if err != nil {
		return err
	}

	for i := range serviceUsers {
		if strings.HasPrefix(serviceUsers[i].Password, "$argon2id$") {
			// password is alread hashed
			serviceUsers[i].Password = "{ARGON2}" + serviceUsers[i].Password
		} else {
			if serviceUsers[i].Password, err = ldappassword.Hash(serviceUsers[i].Password, "{ARGON2}"); err != nil {
				return err
			}
		}
		// We need to treat the hash as binary in the LDIF template to avoid
		// go-ldap/ldif to do any fancy escaping
		serviceUsers[i].Password = base64.StdEncoding.EncodeToString([]byte(serviceUsers[i].Password))
	}
	var tmplWriter strings.Builder
	err = tmpl.Execute(&tmplWriter, serviceUsers)
	if err != nil {
		return err
	}

	bootstrapData := tmplWriter.String()
	if cfg.CreateDemoUsers {
		demoUsersTmpl, err := template.New("demousers").Parse(idm.DemoUsersLDIF)
		if err != nil {
			return err
		}
		var demoUsersWriter strings.Builder
		err = demoUsersTmpl.Execute(&demoUsersWriter, cfg.DemoUsersIssuerUrl)
		if err != nil {
			return err
		}
		bootstrapData = bootstrapData + "\n" + demoUsersWriter.String()
	}

	s := strings.NewReader(bootstrapData)
	lf := &ldif.LDIF{}
	err = ldif.Unmarshal(s, lf)
	if err != nil {
		return err
	}

	for _, entry := range lf.AllEntries() {
		logger.Debug().Str("dn", entry.DN).Msg("Adding entry")
		if err := bdb.EntryPut(entry); err != nil {
			return fmt.Errorf("error adding Entry '%s': %w", entry.DN, err)
		}
	}

	return nil
}
