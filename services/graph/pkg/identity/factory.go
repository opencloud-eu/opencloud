package identity

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"strings"

	ldapv3 "github.com/go-ldap/ldap/v3"
	ocldap "github.com/opencloud-eu/opencloud/pkg/ldap"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/registry"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/metrics"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/utils/ldap"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"
)

const (
	cs3Backend  = "cs3"
	ldapBackend = "ldap"
)

var supportedBackends = []string{cs3Backend, ldapBackend}

func CreateIdentityBackends(name string, cfg *config.Config, logger *log.Logger, registrer prometheus.Registerer, traceProvider trace.TracerProvider) (Backend, EducationBackend, error) {
	switch name {
	case cs3Backend:
		gatewaySelector, err := pool.GatewaySelector(
			cfg.Reva.Address,
			append(
				cfg.Reva.GetRevaOptions(),
				pool.WithRegistry(registry.GetRegistry()),
				pool.WithTracerProvider(traceProvider),
			)...,
		)
		if err != nil {
			return nil, nil, err
		}

		if cs3, err := NewCS3Backend(cfg.Reva, gatewaySelector, logger); err != nil {
			return nil, nil, err
		} else {
			return cs3, nil, nil
		}
	case ldapBackend:
		var err error

		var tlsConf *tls.Config
		if cfg.Identity.LDAP.Insecure {
			// When insecure is set to true then we don't need a certificate.
			cfg.Identity.LDAP.CACert = ""
			tlsConf = &tls.Config{
				MinVersion: tls.VersionTLS12,

				//nolint:gosec // We need the ability to run with "insecure" (dev/testing)
				InsecureSkipVerify: cfg.Identity.LDAP.Insecure,
			}
		}

		if cfg.Identity.LDAP.CACert != "" {
			if err := ocldap.WaitForCA(*logger,
				cfg.Identity.LDAP.Insecure,
				cfg.Identity.LDAP.CACert); err != nil {
				logger.Fatal().Err(err).Msg("The configured LDAP CA cert does not exist")
			}
			if tlsConf == nil {
				tlsConf = &tls.Config{
					MinVersion: tls.VersionTLS12,
				}
			}
			certs := x509.NewCertPool()
			pemData, err := os.ReadFile(cfg.Identity.LDAP.CACert)
			if err != nil {
				logger.Error().Err(err).Msg("Error initializing LDAP Backend")
				return nil, nil, err
			}
			if !certs.AppendCertsFromPEM(pemData) {
				logger.Error().Msg("Error initializing LDAP Backend. Adding CA cert failed")
				return nil, nil, err
			}
			tlsConf.RootCAs = certs
		}

		ldapConfig := ldap.Config{
			URI:          cfg.Identity.LDAP.URI,
			BindDN:       cfg.Identity.LDAP.BindDN,
			BindPassword: cfg.Identity.LDAP.BindPassword,
			TLSConfig:    tlsConf,
		}

		logger = &log.Logger{Logger: logger.With().
			Str("ldap-uri", ldapConfig.URI).
			Logger(),
		}

		conn := ldap.NewLDAPWithReconnect(ldapConfig)
		conn.SetLogger(&logger.Logger)
		lb, err := NewLDAPBackend(conn, cfg.Identity.LDAP, logger, metrics.Namespace, metrics.Subsystem, registrer)
		if err != nil {
			logger.Error().Err(err).Msg("Error initializing LDAP Backend")
			return nil, nil, err
		}

		var identityBackend Backend = lb
		var eduBackend EducationBackend = lb

		if !cfg.Identity.Metrics.Disabled && registrer != nil {
			backendApiOperationDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
				Namespace: metrics.Namespace,
				Subsystem: metrics.Subsystem,
				Name:      "identity_backend_api_duration_seconds",
				Help:      "Duration of API operations performed by the Graph service identity backend in seconds.",
				Buckets:   prometheus.DefBuckets,
				ConstLabels: prometheus.Labels{
					MetricLabelType: name,
				},
			}, []string{MetricLabelOperation, metrics.LabelResult})

			if err := registrer.Register(backendApiOperationDuration); err != nil {
				logger.Warn().Err(err).Msg("failed to register backend API operation duration metric")
			}

			identityBackend = NewPrometheusBackend(identityBackend, backendApiOperationDuration)
			eduBackend = NewPrometheusEducationBackend(eduBackend, backendApiOperationDuration)
		}

		if !cfg.Identity.LDAP.EducationResourcesEnabled {
			// in this case, simply bury the previous eduBackend, no need to wrap or anything: if we had
			// a previous implementation in there that wrapped with metrics or such, we don't want to
			// have any cross-cutting concerns running here, just use this implementation that returns
			// errors on purpose and that's it:
			eduBackend = &ErrEducationBackend{}
		}

		disableMechanismType, err := ParseDisableMechanismType(cfg.Identity.LDAP.DisableUserMechanism)
		if err != nil {
			logger.Error().Err(err).Msg("Error initializing LDAP Backend")
			return nil, nil, err
		}

		if disableMechanismType == DisableMechanismGroup {
			logger.Info().Msg("LocalUserDisable is true, will create group if not exists")
			err := lb.CreateLDAPGroupByDN(cfg.Identity.LDAP.LdapDisabledUsersGroupDN)
			if err != nil {
				isAnError := false
				var lerr *ldapv3.Error
				if errors.As(err, &lerr) {
					if lerr.ResultCode != ldapv3.LDAPResultEntryAlreadyExists {
						isAnError = true
					}
				} else {
					isAnError = true
				}

				if isAnError {
					msg := "error adding group for disabling users"
					logger.Error().Err(err).Str("local_user_disable", cfg.Identity.LDAP.LdapDisabledUsersGroupDN).Msg(msg)
					return nil, nil, fmt.Errorf("%s: %w", msg, err)
				}
			}
		}

		return identityBackend, eduBackend, nil

	default:
		err := fmt.Errorf("unknown identity backend: %q, must be one of [%s]", name, strings.Join(supportedBackends, ", "))
		logger.Error().Err(err).Msgf("failed to create identity backend %q", name)
		return nil, nil, err
	}
}
