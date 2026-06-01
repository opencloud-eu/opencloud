// Package render generates the Authelia configuration consumed by the embedded Authelia provider.
//
// It writes two files (mirroring how the idp (lico) service generates its config in
// services/idp/pkg/service/v0/service.go, and how OpenCloud keeps secrets separate from config):
//
//   - authelia.yaml: regenerated from the current OpenCloud configuration on every start, so it
//     always reflects the live values (OC_URL, SMTP settings, the idp bind password, ...). It carries
//     a "do not edit" header because manual changes are overwritten.
//   - authelia.secrets.yaml: the random secrets (session, storage encryption, OIDC HMAC and signing
//     key, password-reset JWT). Generated once and then left untouched so existing sessions and OIDC
//     tokens survive restarts and config changes.
//
// Authelia deep-merges multiple config files, so the two are passed together. An optional
// authelia.override.yaml (admin-managed, never generated) is merged last and takes precedence.
//
// There is intentionally no dependency on 'opencloud init': the only value the service needs from
// init is the LDAP bind password of the libregraph-idm admin user it binds as, read from its config.
package render

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/opencloud-eu/opencloud/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/pkg/generators"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/auth-authelia/pkg/config"
)

const (
	// secretLength is the length of the random secrets generated for Authelia (session, storage
	// encryption, OIDC HMAC and the password-reset JWT). Authelia recommends long secrets.
	secretLength = 64
	// oidcKeyBits is the size of the RSA key used by Authelia to sign OIDC tokens.
	oidcKeyBits = 2048
	// port is the loopback port the embedded Authelia HTTP server binds to. It must match the
	// backend referenced by the proxy route (services/proxy .../defaultconfig.go).
	port = 9091
	// basePath is the URL base path Authelia is served under. Serving Authelia on a subpath lets a
	// single proxy route forward to it without colliding with the web frontend or the lico idp.
	basePath = "authelia"
	// ldapBindDN is the DN of the LDAP service user Authelia binds as (service-user binding). It must
	// be the libregraph-idm admin DN, because that directory only permits its configured admin DN to
	// modify entries (e.g. a user's userPassword during a password change); a read-only account such
	// as 'uid=idp' is rejected with "Insufficient Access Rights". This is the same admin account the
	// graph service uses for user management. The matching password (the 'idm' service-user password)
	// is provided via cfg.LDAP.BindPassword.
	ldapBindDN = "uid=libregraph,ou=sysusers,o=libregraph-idm"
	// ldapAddress is the loopback LDAPS address of the embedded libregraph-idm server.
	ldapAddress = "ldaps://127.0.0.1:9235"

	// secretsFilename is the generate-once secrets file, kept next to the main config file.
	secretsFilename = "authelia.secrets.yaml"
	// overrideFilename is an optional admin-managed file merged last (highest precedence). It is
	// never generated or modified by OpenCloud.
	overrideFilename = "authelia.override.yaml"
)

// secretsData holds the values rendered into authelia.secrets.yaml.
type secretsData struct {
	SessionSecret          string
	StorageEncryptionKey   string
	ResetPasswordJWTSecret string
	OIDCHMACSecret         string
	OIDCKeyPEMIndented     string
}

// configData holds the (non-secret) values rendered into authelia.yaml on every start.
type configData struct {
	LogLevel        string
	Address         string
	Domain          string
	AutheliaURL     string
	LDAPAddress     string
	LDAPBindDN      string
	LDAPPassword    string
	StoragePath     string
	OIDCClientsYAML string
	NotifierYAML    string
}

// Config writes the Authelia configuration files and returns the ordered list of paths to pass to
// the embedded provider. The secrets file is generated once and reused; the main config file is
// regenerated from the current OpenCloud configuration on every call so it never goes stale. An
// optional authelia.override.yaml is appended last so admin overrides take precedence.
//
// The LDAP bind password is read from cfg.LDAP.BindPassword; Authelia binds as the libregraph-idm
// admin user (uid=libregraph) so it can both look users up and modify passwords. No dedicated
// Authelia LDAP account is needed.
func Config(logger log.Logger, cfg *config.Config) ([]string, error) {
	targetPath := cfg.ConfigPath
	if targetPath == "" {
		return nil, fmt.Errorf("authelia config path is empty")
	}
	if cfg.LDAP.BindPassword == "" {
		return nil, fmt.Errorf("the 'authelia' LDAP bind password is not configured; run 'opencloud init' " +
			"so the password gets generated and seeded, or set AUTH_AUTHELIA_LDAP_BIND_PASSWORD")
	}

	dir := filepath.Dir(targetPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("could not create authelia config directory: %w", err)
	}

	storagePath := filepath.Join(defaults.BaseDataPath(), "authelia", "authelia.sqlite3")
	if err := os.MkdirAll(filepath.Dir(storagePath), 0700); err != nil {
		return nil, fmt.Errorf("could not create authelia storage directory: %w", err)
	}

	// 1. Secrets: generate once, then leave untouched.
	secretsPath := filepath.Join(dir, secretsFilename)
	if err := ensureSecrets(logger, secretsPath); err != nil {
		return nil, err
	}

	// 2. Main config: always regenerated from the current OpenCloud configuration.
	ocURL := openCloudURL(cfg)
	domain := "localhost"
	if u, perr := url.Parse(ocURL); perr == nil && u.Hostname() != "" {
		domain = u.Hostname()
	}

	data := configData{
		LogLevel:        cfg.LogLevel,
		Address:         fmt.Sprintf("tcp://127.0.0.1:%d/%s", port, basePath),
		Domain:          domain,
		AutheliaURL:     ocURL + "/" + basePath,
		LDAPAddress:     ldapAddress,
		LDAPBindDN:      ldapBindDN,
		LDAPPassword:    cfg.LDAP.BindPassword,
		StoragePath:     storagePath,
		OIDCClientsYAML: renderOIDCClients(ocURL),
		NotifierYAML:    renderNotifier(cfg, domain),
	}
	if err := renderToFile(configTemplate, data, targetPath); err != nil {
		return nil, fmt.Errorf("could not write authelia config: %w", err)
	}
	logger.Info().Str("config", targetPath).Str("oidc_issuer", data.AutheliaURL).
		Msg("regenerated authelia config from the current OpenCloud configuration")

	// Authelia requires the session cookie domain to contain a dot or be an IP address; it rejects
	// a bare hostname such as 'localhost' and refuses to start. The default OC_URL is
	// https://localhost:9200, so warn explicitly that Authelia needs a real domain.
	if !isValidCookieDomain(domain) {
		logger.Warn().Str("domain", domain).Str("config", targetPath).
			Msg("session cookie domain is not valid for Authelia (needs a dot or must be an IP); " +
				"Authelia will not start until 'session.cookies' uses a fully-qualified domain. Set " +
				"OC_URL to your public URL.")
	}

	paths := []string{secretsPath, targetPath}

	// 3. Optional admin override, merged last so it takes precedence.
	overridePath := filepath.Join(dir, overrideFilename)
	if _, err := os.Stat(overridePath); err == nil {
		logger.Info().Str("override", overridePath).Msg("merging authelia override config")
		paths = append(paths, overridePath)
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("could not stat authelia override config %q: %w", overridePath, err)
	}

	return paths, nil
}

// ensureSecrets generates the secrets file if it does not exist yet. An existing file is left
// untouched so the secrets persist across restarts and config regenerations.
func ensureSecrets(logger log.Logger, path string) error {
	if _, err := os.Stat(path); err == nil {
		logger.Info().Str("secrets", path).Msg("using existing authelia secrets")
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("could not stat authelia secrets %q: %w", path, err)
	}

	sessionSecret, err := generators.GenerateRandomPassword(secretLength)
	if err != nil {
		return fmt.Errorf("could not generate authelia session secret: %w", err)
	}
	storageKey, err := generators.GenerateRandomPassword(secretLength)
	if err != nil {
		return fmt.Errorf("could not generate authelia storage encryption key: %w", err)
	}
	resetJWTSecret, err := generators.GenerateRandomPassword(secretLength)
	if err != nil {
		return fmt.Errorf("could not generate authelia password reset jwt secret: %w", err)
	}
	oidcHMAC, err := generators.GenerateRandomPassword(secretLength)
	if err != nil {
		return fmt.Errorf("could not generate authelia oidc hmac secret: %w", err)
	}
	oidcKeyPEM, err := generateRSAPrivateKeyPEM(oidcKeyBits)
	if err != nil {
		return fmt.Errorf("could not generate authelia oidc signing key: %w", err)
	}

	data := secretsData{
		SessionSecret:          sessionSecret,
		StorageEncryptionKey:   storageKey,
		ResetPasswordJWTSecret: resetJWTSecret,
		OIDCHMACSecret:         oidcHMAC,
		OIDCKeyPEMIndented:     indentLines(oidcKeyPEM, 10),
	}
	if err := renderToFile(secretsTemplate, data, path); err != nil {
		return fmt.Errorf("could not write authelia secrets: %w", err)
	}
	logger.Info().Str("secrets", path).Msg("generated authelia secrets")
	return nil
}

// renderToFile parses and executes the given template into path with mode 0600.
func renderToFile(tmplText string, data any, path string) error {
	tmpl, err := template.New("authelia").Parse(tmplText)
	if err != nil {
		return fmt.Errorf("could not parse template: %w", err)
	}
	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("could not render template: %w", err)
	}
	return os.WriteFile(path, []byte(buf.String()), 0600)
}

// openCloudURL resolves the public OpenCloud URL used to derive the OIDC issuer, session domain and
// the OIDC client redirect URIs. It prefers the shared commons value (env OC_URL), falling back to
// OC_URL directly and finally the localhost default.
func openCloudURL(cfg *config.Config) string {
	ocURL := ""
	if cfg.Commons != nil {
		ocURL = cfg.Commons.OpenCloudURL
	}
	if ocURL == "" {
		ocURL = os.Getenv("OC_URL")
	}
	if ocURL == "" {
		ocURL = "https://localhost:9200"
	}
	return strings.TrimRight(ocURL, "/")
}

// isValidCookieDomain reports whether domain is acceptable as an Authelia session cookie domain.
// Authelia requires either an IP address or a hostname containing at least one dot; a bare label
// such as 'localhost' is rejected.
func isValidCookieDomain(domain string) bool {
	if net.ParseIP(domain) != nil {
		return true
	}
	return strings.Contains(domain, ".")
}

// generateRSAPrivateKeyPEM generates a new RSA private key and returns it PEM (PKCS#1) encoded.
func generateRSAPrivateKeyPEM(bits int) (string, error) {
	key, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return "", err
	}
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}
	return string(pem.EncodeToMemory(block)), nil
}

// indentLines prefixes every non-empty line with n spaces. Used to embed multi-line PEM data into a
// YAML block scalar.
func indentLines(s string, n int) string {
	prefix := strings.Repeat(" ", n)
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i, line := range lines {
		if line == "" {
			continue
		}
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}

// oidcClient mirrors an OpenCloud OIDC client (see services/idp default clients). All clients are
// public/PKCE clients (no secret), matching the lico configuration.
//
// Note: Authelia (v4.39) has no per-client post-logout redirect URI list; RP-initiated logout
// validates the redirect against the client's registered redirect_uris. The native clients below
// therefore only need their redirect_uris registered.
type oidcClient struct {
	ID           string
	Name         string
	ConsentMode  string // "implicit" for trusted clients, "auto" otherwise
	RedirectURIs []string
}

// defaultOIDCClients returns the default OpenCloud OIDC clients, mirroring the lico/idp defaults.
// {{OC_URL}} placeholders are substituted with the configured OpenCloud URL.
func defaultOIDCClients() []oidcClient {
	return []oidcClient{
		{
			ID:          "web",
			Name:        "OpenCloud Web App",
			ConsentMode: "implicit",
			RedirectURIs: []string{
				"{{OC_URL}}/",
				"{{OC_URL}}/oidc-callback.html",
				"{{OC_URL}}/oidc-silent-redirect.html",
			},
		},
		{
			ID:          "OpenCloudDesktop",
			Name:        "OpenCloud Desktop Client",
			ConsentMode: "auto",
			RedirectURIs: []string{
				"http://127.0.0.1",
				"http://localhost",
			},
		},
		{
			ID:          "OpenCloudAndroid",
			Name:        "OpenCloud Android App",
			ConsentMode: "auto",
			RedirectURIs: []string{
				"oc://android.opencloud.eu",
			},
		},
		{
			ID:          "OpenCloudIOS",
			Name:        "OpenCloud iOS App",
			ConsentMode: "auto",
			RedirectURIs: []string{
				"oc://ios.opencloud.eu",
			},
		},
	}
}

// renderOIDCClients renders the OIDC clients list as YAML, indented to sit under
// 'identity_providers.oidc.clients:'. List item dashes are at 6 spaces, fields at 8 spaces.
func renderOIDCClients(ocURL string) string {
	var b strings.Builder
	for _, c := range defaultOIDCClients() {
		b.WriteString("      - client_id: '" + c.ID + "'\n")
		b.WriteString("        client_name: '" + c.Name + "'\n")
		b.WriteString("        public: true\n")
		b.WriteString("        authorization_policy: 'one_factor'\n")
		b.WriteString("        consent_mode: '" + c.ConsentMode + "'\n")
		b.WriteString("        require_pkce: true\n")
		b.WriteString("        pkce_challenge_method: 'S256'\n")
		b.WriteString("        token_endpoint_auth_method: 'none'\n")
		b.WriteString("        redirect_uris:\n")
		for _, uri := range c.RedirectURIs {
			b.WriteString("          - '" + strings.ReplaceAll(uri, "{{OC_URL}}", ocURL) + "'\n")
		}
		// 'offline_access' is required for the 'refresh_token' grant below; without it Authelia
		// rejects the grant (refresh tokens are how the web/desktop/mobile clients renew sessions).
		b.WriteString("        scopes:\n")
		b.WriteString("          - 'openid'\n")
		b.WriteString("          - 'profile'\n")
		b.WriteString("          - 'email'\n")
		b.WriteString("          - 'groups'\n")
		b.WriteString("          - 'offline_access'\n")
		b.WriteString("        grant_types:\n")
		b.WriteString("          - 'authorization_code'\n")
		b.WriteString("          - 'refresh_token'\n")
		b.WriteString("        response_types:\n")
		b.WriteString("          - 'code'\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// renderNotifier renders the 'notifier' YAML block. When an SMTP host is configured it renders an
// SMTP notifier; otherwise it falls back to a filesystem notifier that writes mails to a file under
// the data dir - so a fresh deployment works without any mail server. The notifier startup check is
// disabled either way so an unreachable or flaky mail server never blocks startup.
func renderNotifier(cfg *config.Config, domain string) string {
	var b strings.Builder
	b.WriteString("notifier:\n")
	b.WriteString("  disable_startup_check: true\n")

	if cfg.SMTP.Host == "" {
		filename := filepath.Join(defaults.BaseDataPath(), "authelia", "notification.txt")
		b.WriteString("  # No SMTP host configured: notifications (password reset, 2FA registration) are\n")
		b.WriteString("  # written to this file. Configure SMTP (NOTIFICATIONS_SMTP_* / AUTH_AUTHELIA_SMTP_*)\n")
		b.WriteString("  # to send them by email instead.\n")
		b.WriteString("  filesystem:\n")
		b.WriteString("    filename: " + yamlSingleQuote(filename) + "\n")
		return strings.TrimRight(b.String(), "\n")
	}

	sender := cfg.SMTP.Sender
	if sender == "" {
		sender = fmt.Sprintf("OpenCloud <no-reply@%s>", domain)
	}
	b.WriteString("  smtp:\n")
	b.WriteString("    address: " + yamlSingleQuote(smtpAddress(cfg.SMTP)) + "\n")
	b.WriteString("    sender: " + yamlSingleQuote(sender) + "\n")
	if cfg.SMTP.Username != "" {
		b.WriteString("    username: " + yamlSingleQuote(cfg.SMTP.Username) + "\n")
	}
	if cfg.SMTP.Password != "" {
		b.WriteString("    password: " + yamlSingleQuote(cfg.SMTP.Password) + "\n")
	}
	if strings.EqualFold(cfg.SMTP.Encryption, "none") {
		// No transport encryption: Authelia refuses to send credentials in the clear unless told to.
		b.WriteString("    disable_require_tls: true\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// smtpAddress builds an Authelia SMTP address (scheme://host[:port]) from the OpenCloud SMTP
// settings. The scheme encodes the transport encryption: 'submissions' for implicit TLS (ssltls),
// 'submission' for STARTTLS (starttls), and 'smtp' otherwise.
func smtpAddress(s config.SMTP) string {
	scheme := "smtp"
	switch strings.ToLower(s.Encryption) {
	case "ssltls":
		scheme = "submissions"
	case "starttls":
		scheme = "submission"
	}
	addr := scheme + "://" + s.Host
	if s.Port > 0 {
		addr += ":" + strconv.Itoa(s.Port)
	}
	return addr
}

// yamlSingleQuote wraps a value in a single-quoted YAML scalar, escaping embedded single quotes by
// doubling them. Used for operator-supplied values (SMTP credentials, paths) that may contain
// characters which are unsafe unquoted.
func yamlSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// secretsTemplate renders authelia.secrets.yaml. It contains only the persisted secrets; everything
// else lives in the regenerated main config. Authelia deep-merges the two at load time.
const secretsTemplate = `# GENERATED FILE - DO NOT EDIT.
# Authelia secrets, generated once by the auth-authelia service and then left untouched so that
# existing sessions and issued OIDC tokens remain valid across restarts and config changes.
# Deleting this file makes the service generate new secrets on the next start, which invalidates all
# sessions and OIDC tokens. Keep this file private (mode 0600).

identity_validation:
  reset_password:
    jwt_secret: '{{ .ResetPasswordJWTSecret }}'

session:
  secret: '{{ .SessionSecret }}'

storage:
  encryption_key: '{{ .StorageEncryptionKey }}'

identity_providers:
  oidc:
    hmac_secret: '{{ .OIDCHMACSecret }}'
    jwks:
      - key_id: 'default'
        algorithm: 'RS256'
        use: 'sig'
        key: |
{{ .OIDCKeyPEMIndented }}
`

// configTemplate renders authelia.yaml. It targets the Authelia v4.39 configuration schema and holds
// only values derived from the OpenCloud configuration (no secrets) so it can be safely regenerated
// on every start. The embedded Authelia provider validates the merged configuration on startup.
const configTemplate = `# GENERATED FILE - DO NOT EDIT.
# The auth-authelia service regenerates this file from the current OpenCloud configuration every time
# it starts, so manual changes here are lost. Change these values through the OpenCloud configuration
# instead (OC_URL, AUTH_AUTHELIA_* / NOTIFICATIONS_SMTP_* environment variables, ...).
#
# Secrets are kept separately in 'authelia.secrets.yaml' (generated once).
# To set or override options that OpenCloud does not manage, create 'authelia.override.yaml' in this
# directory: it is merged last (takes precedence) and is never modified by OpenCloud.

server:
  address: '{{ .Address }}'

log:
  level: '{{ .LogLevel }}'

theme: 'auto'

totp:
  issuer: '{{ .Domain }}'

authentication_backend:
  ldap:
    implementation: 'custom'
    address: '{{ .LDAPAddress }}'
    tls:
      # The embedded libregraph-idm LDAPS server uses a self-signed certificate on localhost.
      skip_verify: true
    base_dn: 'o=libregraph-idm'
    additional_users_dn: 'ou=users'
    additional_groups_dn: 'ou=groups'
    users_filter: '(&(|({username_attribute}={input})({mail_attribute}={input}))(objectClass=inetOrgPerson))'
    groups_filter: '(&(member={dn})(objectClass=groupOfNames))'
    user: '{{ .LDAPBindDN }}'
    password: '{{ .LDAPPassword }}'
    attributes:
      username: 'uid'
      display_name: 'displayName'
      mail: 'mail'
      group_name: 'cn'

access_control:
  default_policy: 'one_factor'

session:
  cookies:
    - domain: '{{ .Domain }}'
      authelia_url: '{{ .AutheliaURL }}'

storage:
  local:
    path: '{{ .StoragePath }}'

# NTP is only used to detect clock skew for TOTP. The startup connectivity check is disabled so the
# embedded service does not require outbound NTP to boot.
ntp:
  disable_startup_check: true

{{ .NotifierYAML }}

identity_providers:
  oidc:
    clients:
{{ .OIDCClientsYAML }}
`
