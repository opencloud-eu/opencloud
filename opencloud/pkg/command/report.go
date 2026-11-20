package command

import (
	"cmp"
	"context"
	"crypto/tls"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"slices"
	"strconv"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/opencloud-eu/opencloud/opencloud/pkg/register"
	"github.com/opencloud-eu/opencloud/pkg/config"
	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/config/parser"
	mregistry "github.com/opencloud-eu/opencloud/pkg/registry"
	gatewayparser "github.com/opencloud-eu/opencloud/services/gateway/pkg/config/parser"
	graphparser "github.com/opencloud-eu/opencloud/services/graph/pkg/config/parser"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

const (
	_reportOutputFlagName   = "output"
	_reportDaysFlagName     = "days"
	_reportSignInFlagName   = "with-sign-in"
	_reportInsecureFlagName = "insecure"
	_reportTimeoutFlagName  = "timeout"

	// _reportDefaultDays is the default sign-in window in days.
	_reportDefaultDays = 31

	// _reportNoRole is the bucket for users without any assigned role.
	_reportNoRole = "(no role assigned)"

	// _reportDefaultTimeout caps the whole report run by default. The graph API
	// resolves the role assignments per user, so the run grows with the number
	// of users: a minute covers roughly ten thousand of them, and a larger
	// instance can raise it with --timeout.
	_reportDefaultTimeout = 5 * time.Minute

	_reportOutputTable = "table"
	_reportOutputJSON  = "json"
	_reportOutputCSV   = "csv"
	_reportOutputYAML  = "yaml"
)

// roleCount is the number of users sharing one assigned role.
type roleCount struct {
	Role string `json:"role" yaml:"role"`
	// Users counts every user holding the role, whether or not a sign-in is
	// known for them.
	Users int `json:"users" yaml:"users"`
	// SignedIn counts the subset that signed in within the reported window. It
	// is nil when the instance does not record sign-ins at all, to tell "not
	// available" apart from "nobody signed in".
	SignedIn *int `json:"signedIn" yaml:"signedIn"`
}

// userReport is the result of the report command.
type userReport struct {
	GeneratedAt time.Time `json:"generatedAt" yaml:"generatedAt"`
	Days        int       `json:"days" yaml:"days"`
	ActiveSince time.Time `json:"activeSince" yaml:"activeSince"`
	TotalUsers  int       `json:"totalUsers" yaml:"totalUsers"`
	// ActiveUsers is the number of users that signed in within the reported
	// window. Like roleCount.SignedIn it is nil when the instance does not
	// record sign-ins at all.
	ActiveUsers *int `json:"activeUsers" yaml:"activeUsers"`
	// UsersWithSignInData is the number of users that have a recorded sign-in
	// at all, regardless of the reported window. It is 0 when the deployment
	// does not store sign-in timestamps.
	UsersWithSignInData int         `json:"usersWithSignInData" yaml:"usersWithSignInData"`
	Roles               []roleCount `json:"roles" yaml:"roles"`
	// SignInReported tells the renderers that the sign-in window was reported.
	// Without it the sign-in fields are meaningless and are left out entirely.
	SignInReported bool `json:"-" yaml:"-"`
}

// plainRoleCount is a roleCount without its sign-in column.
type plainRoleCount struct {
	Role  string `json:"role" yaml:"role"`
	Users int    `json:"users" yaml:"users"`
}

// plainUserReport is a userReport without any sign-in data, used when the
// sign-in window is skipped. The sign-in fields are dropped rather than
// reported as null, because nothing was looked at in the first place.
type plainUserReport struct {
	GeneratedAt time.Time        `json:"generatedAt" yaml:"generatedAt"`
	TotalUsers  int              `json:"totalUsers" yaml:"totalUsers"`
	Roles       []plainRoleCount `json:"roles" yaml:"roles"`
}

// plain reduces the report to its user counts per role.
func (r userReport) plain() plainUserReport {
	roles := make([]plainRoleCount, 0, len(r.Roles))
	for _, role := range r.Roles {
		roles = append(roles, plainRoleCount{Role: role.Role, Users: role.Users})
	}
	return plainUserReport{
		GeneratedAt: r.GeneratedAt,
		TotalUsers:  r.TotalUsers,
		Roles:       roles,
	}
}

// reportOptions holds the settings a report run takes from the command line.
type reportOptions struct {
	// days is the width of the sign-in window, only used with withSignIn.
	days int
	// withSignIn adds the sign-in window to the report. Off by default: the
	// data only exists on a writable LDAP backend.
	withSignIn bool
	// insecure skips the TLS certificate check against the graph service, for
	// deployments serving the generated self-signed certificate.
	insecure bool
	// timeout caps the whole run, the context and the HTTP client alike.
	timeout time.Duration
}

// ReportCommand is the entrypoint for the report command.
func ReportCommand(cfg *config.Config) *cobra.Command {
	reportCmd := &cobra.Command{
		Use:   "report",
		Short: "report the users of this instance grouped by their assigned role",
		Long: "Report the users of OpenCloud grouped by their assigned role.\n\n" +
			"By default every user is counted once and the report is a plain user\n" +
			"count per role.\n\n" +
			"Pass --with-sign-in to additionally report, per role, how many of its\n" +
			"users signed in successfully within the last --days days. Users whose\n" +
			"sign-in is unknown still count towards their role. OpenCloud stores the\n" +
			"last sign-in on the user entry in LDAP, so this needs an LDAP identity\n" +
			"backend that OpenCloud may write to and is refused otherwise.\n" +
			"The data is read from the libregraph API of a running OpenCloud instance,\n" +
			"so the instance needs to be up while the report is generated.",
		GroupID: CommandGroupServer,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Parse base config
			if err := parser.ParseConfig(cfg, true); err != nil {
				return configlog.ReturnError(err)
			}

			// Parse graph config, the report talks to the graph service
			cfg.Graph.Commons = cfg.Commons
			if err := graphparser.ParseConfig(cfg.Graph); err != nil {
				return configlog.ReturnError(err)
			}

			// Parse gateway config, it carries the address the report falls back
			// to when the service registry holds no entry for the gateway
			cfg.Gateway.Commons = cfg.Commons
			return configlog.ReturnError(gatewayparser.ParseConfig(cfg.Gateway))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			days, err := cmd.Flags().GetInt(_reportDaysFlagName)
			if err != nil {
				return err
			}
			withSignIn, err := cmd.Flags().GetBool(_reportSignInFlagName)
			if err != nil {
				return err
			}
			if !withSignIn && cmd.Flags().Changed(_reportDaysFlagName) {
				return fmt.Errorf("--%s only applies together with --%s",
					_reportDaysFlagName, _reportSignInFlagName)
			}
			if withSignIn {
				if reason := signInUnavailable(cfg); reason != "" {
					return fmt.Errorf("--%s is not available on this instance: %s",
						_reportSignInFlagName, reason)
				}
				if days < 1 {
					return fmt.Errorf("--%s must be at least 1", _reportDaysFlagName)
				}
			}

			format, err := cmd.Flags().GetString(_reportOutputFlagName)
			if err != nil {
				return err
			}
			if !slices.Contains([]string{_reportOutputTable, _reportOutputJSON, _reportOutputCSV, _reportOutputYAML}, format) {
				return fmt.Errorf("unknown output format %q, supported formats are: %s, %s, %s, %s",
					format, _reportOutputTable, _reportOutputJSON, _reportOutputCSV, _reportOutputYAML)
			}

			insecure, err := cmd.Flags().GetBool(_reportInsecureFlagName)
			if err != nil {
				return err
			}

			timeout, err := cmd.Flags().GetDuration(_reportTimeoutFlagName)
			if err != nil {
				return err
			}
			if timeout <= 0 {
				return fmt.Errorf("--%s must be positive", _reportTimeoutFlagName)
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()

			report, err := userRoleReport(ctx, cfg, reportOptions{
				days:       days,
				withSignIn: withSignIn,
				insecure:   insecure,
				timeout:    timeout,
			})
			if err != nil {
				return err
			}

			return renderUserReport(os.Stdout, report, format)
		},
	}

	reportCmd.Flags().IntP(_reportDaysFlagName, "d", _reportDefaultDays,
		"window in days for counting users who signed in successfully")
	reportCmd.Flags().StringP(_reportOutputFlagName, "o", _reportOutputTable,
		fmt.Sprintf("output format, one of: %s, %s, %s, %s",
			_reportOutputTable, _reportOutputJSON, _reportOutputCSV, _reportOutputYAML))
	reportCmd.Flags().Bool(_reportSignInFlagName, false,
		"additionally report how many users per role signed in within the last --days days. Needs a writable LDAP identity backend")
	reportCmd.Flags().DurationP(_reportTimeoutFlagName, "t", _reportDefaultTimeout,
		"time budget for the whole report run, e.g. 10m. Raise it on instances with very many users")
	reportCmd.Flags().Bool(_reportInsecureFlagName, false,
		"do not verify the TLS certificate of the graph service, e.g. when it serves a generated self-signed certificate")

	return reportCmd
}

func init() {
	register.AddCommand(ReportCommand)
}

// signInUnavailable reports why this instance cannot report sign-ins, or an
// empty string when it can. OpenCloud stores the last successful sign-in as an
// attribute on the user entry, so it needs an LDAP identity backend that the
// graph service may write to. Without that no timestamp is ever recorded and
// the sign-in counts would be meaningless rather than merely empty.
func signInUnavailable(cfg *config.Config) string {
	if backend := cfg.Graph.Identity.Backend; backend != "ldap" {
		return fmt.Sprintf("the %q identity backend does not record sign-ins", backend)
	}
	if !cfg.Graph.Identity.LDAP.WriteEnabled {
		return "the LDAP identity backend is read-only, see OC_LDAP_SERVER_WRITE_ENABLED, so no sign-in is ever recorded"
	}
	return ""
}

// userRoleReport collects all users from the libregraph API and groups them by
// their assigned role. By default the result is a plain user count per role.
// With withSignIn it additionally reports, per role, how many of its users
// signed in within the last `days` days.
func userRoleReport(ctx context.Context, cfg *config.Config, opts reportOptions) (userReport, error) {
	client, err := newLibregraphServiceAccountClient(ctx, cfg, opts)
	if err != nil {
		return userReport{}, err
	}

	roleNames, err := appRoleNames(ctx, client)
	if err != nil {
		return userReport{}, err
	}

	users, err := listUsersWithRoleAssignments(ctx, client)
	if err != nil {
		return userReport{}, err
	}

	now := time.Now().UTC()
	// a nil window tells countUsersByRole to not look at sign-ins at all
	var activeSince *time.Time
	if opts.withSignIn {
		since := now.AddDate(0, 0, -opts.days)
		activeSince = &since
	}
	roles, activeUsers, withSignInData := countUsersByRole(users, roleNames, activeSince)

	report := userReport{
		GeneratedAt:    now,
		TotalUsers:     len(users),
		Roles:          roles,
		SignInReported: opts.withSignIn,
	}
	if activeSince != nil {
		report.Days = opts.days
		report.ActiveSince = *activeSince
		report.ActiveUsers = activeUsers
		report.UsersWithSignInData = withSignInData
	}
	return report, nil
}

// countUsersByRole groups every user by the display names of their assigned
// roles, counting for each role both its total number of users and the subset
// that signed in at or after activeSince. Users whose sign-in is unknown are
// still counted towards their role. It returns the counts ordered by size, the
// number of users that signed in within the window and the number of users
// that have a recorded sign-in at all. Users with multiple roles are counted
// once per role.
//
// When the instance holds no sign-in data at all the sign-in counts are
// returned as nil rather than 0, so that callers can report them as not
// available instead of claiming nobody signed in. A nil activeSince skips the
// sign-in accounting altogether and only groups the users by role.
func countUsersByRole(users []libregraph.User, roleNames map[string]string, activeSince *time.Time) ([]roleCount, *int, int) {
	totals := make(map[string]int, len(roleNames)+1)
	signedIn := make(map[string]int, len(roleNames)+1)
	activeUsers := 0
	withSignInData := 0
	for _, u := range users {
		active := false
		if activeSince != nil {
			signInActivity := u.GetSignInActivity()
			lastSignIn, hasSignIn := signInActivity.GetLastSuccessfulSignInDateTimeOk()
			if hasSignIn {
				withSignInData++
			}
			active = hasSignIn && !lastSignIn.Before(*activeSince)
			if active {
				activeUsers++
			}
		}

		for _, name := range assignedRoleNames(u, roleNames) {
			totals[name]++
			if active {
				signedIn[name]++
			}
		}
	}

	// without a single recorded sign-in we cannot tell an inactive user from
	// one whose sign-in was never stored, and without a window we never looked
	signInKnown := activeSince != nil && (withSignInData > 0 || len(users) == 0)

	roles := make([]roleCount, 0, len(totals))
	for name, total := range totals {
		roles = append(roles, roleCount{
			Role:     name,
			Users:    total,
			SignedIn: optionalCount(signedIn[name], signInKnown),
		})
	}
	// biggest role first, alphabetically for equal counts
	slices.SortFunc(roles, func(a, b roleCount) int {
		if c := cmp.Compare(b.Users, a.Users); c != 0 {
			return c
		}
		return cmp.Compare(a.Role, b.Role)
	})

	return roles, optionalCount(activeUsers, signInKnown), withSignInData
}

// optionalCount returns a pointer to count, or nil when the count is not known.
func optionalCount(count int, known bool) *int {
	if !known {
		return nil
	}
	return &count
}

// assignedRoleNames resolves the roles of a user to their display names. Users
// without any assignment get a bucket of their own, so that they still show up
// in the report.
func assignedRoleNames(user libregraph.User, roleNames map[string]string) []string {
	assignments := user.GetAppRoleAssignments()
	if len(assignments) == 0 {
		return []string{_reportNoRole}
	}

	names := make([]string, 0, len(assignments))
	for _, a := range assignments {
		name, ok := roleNames[a.GetAppRoleId()]
		if !ok {
			// an assignment pointing to a role the application no longer
			// exposes, report the raw id so it doesn't vanish silently
			name = a.GetAppRoleId()
		}
		names = append(names, name)
	}
	return names
}

// newLibregraphServiceAccountClient builds a libregraph client authenticated as
// the configured service account. The graph endpoint is looked up in the
// service registry, so this only works against a running instance.
func newLibregraphServiceAccountClient(ctx context.Context, cfg *config.Config, opts reportOptions) (*libregraph.APIClient, error) {
	// Initialize registry to make service lookup work
	reg := mregistry.GetRegistry()

	gatewayClient, err := selectGateway(cfg)
	if err != nil {
		return nil, err
	}

	token, err := utils.GetServiceUserToken(
		ctx,
		gatewayClient,
		cfg.Graph.ServiceAccount.ServiceAccountID,
		cfg.Graph.ServiceAccount.ServiceAccountSecret,
	)
	if err != nil {
		return nil, fmt.Errorf("could not authenticate the service account: %w", err)
	}

	serviceName := cfg.Graph.HTTP.Namespace + "." + cfg.Graph.Service.Name
	var baseURL string
	// A lookup error means the registry holds no entry, which is not fatal:
	// the fallback below covers it.
	if services, err := reg.GetService(serviceName); err == nil {
		for _, s := range services {
			for _, n := range s.Nodes {
				baseURL = fmt.Sprintf("%s://%s%s", nodeProtocol(n.Metadata), n.Address, cfg.Graph.HTTP.Root)
				break
			}
			if baseURL != "" {
				break
			}
		}
	}
	if baseURL == "" {
		baseURL = configuredGraphURL(cfg)
	}
	if baseURL == "" {
		return nil, fmt.Errorf("could not find the '%s' service, is OpenCloud running?", serviceName)
	}

	lgconf := libregraph.NewConfiguration()
	lgconf.Servers = libregraph.ServerConfigurations{{URL: baseURL}}
	lgconf.DefaultHeader = map[string]string{revactx.TokenHeader: token}
	lgconf.HTTPClient = newGraphHTTPClient(opts)

	return libregraph.NewAPIClient(lgconf), nil
}

// newGraphHTTPClient builds the HTTP client for the graph API. It requires TLS
// 1.2 or newer for an encrypted connection, and only skips the certificate
// check when --insecure asked for it. The default transport is cloned rather
// than replaced, so that proxy support, HTTP/2 and connection pooling stay as
// they are.
func newGraphHTTPClient(opts reportOptions) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
		// #nosec G402 -- off unless the admin passes --insecure
		InsecureSkipVerify: opts.insecure, //nolint:gosec // opt-in via --insecure
	}
	return &http.Client{
		Timeout:   opts.timeout,
		Transport: transport,
	}
}

// selectGateway returns a CS3 gateway client. It first tries the configured
// reva address, which is a service registry name by default, and falls back to
// dialling the configured gateway address directly over dns:///. That covers a
// deployment running the services with MICRO_REGISTRY=memory, where a separate
// process finds an empty registry but the gateway still listens next to it.
func selectGateway(cfg *config.Config) (gateway.GatewayAPIClient, error) {
	addresses := []string{cfg.Graph.Reva.Address}
	if fallback := configuredGatewayAddress(cfg); fallback != "" {
		addresses = append(addresses, fallback)
	}

	var lastErr error
	for _, address := range addresses {
		selector, err := pool.GatewaySelector(address)
		if err != nil {
			lastErr = err
			continue
		}
		client, err := selector.Next()
		if err != nil {
			lastErr = err
			continue
		}
		return client, nil
	}
	return nil, lastErr
}

// configuredGatewayAddress renders the configured gateway bind address as a
// dns:/// target, so that it is dialled directly instead of being looked up in
// the service registry. A wildcard bind address is rewritten to loopback,
// because the report runs next to the gateway it talks to.
func configuredGatewayAddress(cfg *config.Config) string {
	if cfg.Gateway == nil || cfg.Gateway.GRPC.Addr == "" {
		return ""
	}
	host, port, err := net.SplitHostPort(cfg.Gateway.GRPC.Addr)
	if err != nil {
		return ""
	}
	switch host {
	case "", "0.0.0.0", "::", "[::]":
		host = "127.0.0.1"
	}
	return "dns:///" + net.JoinHostPort(host, port)
}

// configuredGraphURL builds the graph base URL from the local configuration.
// It is the fallback when the service registry holds no entry, which is the
// case on a deployment that runs the services with MICRO_REGISTRY=memory and
// discovers them through Kubernetes instead. The report has to run next to the
// services anyway, so the configured address of the local graph service is the
// right answer rather than a guess.
func configuredGraphURL(cfg *config.Config) string {
	if cfg.Graph.HTTP.Addr == "" {
		return ""
	}
	scheme := "http"
	if cfg.Commons != nil && cfg.Commons.HTTPServiceTLS.Enabled {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s%s", scheme, cfg.Graph.HTTP.Addr, cfg.Graph.HTTP.Root)
}

// nodeProtocol resolves the URL scheme to talk to a registry node with. The
// registry reports "http" for every HTTP service and signals TLS separately
// through the use_tls metadata, so that flag decides, the same way the proxy
// router resolves it.
func nodeProtocol(metadata map[string]string) string {
	if metadata["use_tls"] == "true" {
		return "https"
	}
	if protocol := metadata["protocol"]; protocol != "" {
		return protocol
	}
	return "http"
}

// appRoleNames maps the app role ids of the OpenCloud application to their
// display names, e.g. "Admin" or "Space Admin".
func appRoleNames(ctx context.Context, client *libregraph.APIClient) (map[string]string, error) {
	applications, resp, err := client.ApplicationsApi.ListApplications(ctx).Execute()
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, fmt.Errorf("could not list applications: %w", err)
	}

	names := map[string]string{}
	for _, application := range applications.GetValue() {
		for _, role := range application.GetAppRoles() {
			name := role.GetDisplayName()
			if name == "" {
				name = role.GetId()
			}
			names[role.GetId()] = name
		}
	}
	return names, nil
}

// listUsersWithRoleAssignments returns all users including their app role assignments.
func listUsersWithRoleAssignments(ctx context.Context, client *libregraph.APIClient) ([]libregraph.User, error) {
	users, resp, err := client.UsersApi.ListUsers(ctx).Expand([]string{"appRoleAssignments"}).Execute()
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, fmt.Errorf("could not list users: %w", err)
	}
	return users.GetValue(), nil
}

// renderUserReport writes the report in the requested output format.
func renderUserReport(w io.Writer, report userReport, format string) error {
	switch format {
	case _reportOutputJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		if !report.SignInReported {
			return enc.Encode(report.plain())
		}
		return enc.Encode(report)
	case _reportOutputYAML:
		enc := yaml.NewEncoder(w)
		defer enc.Close()
		if !report.SignInReported {
			return enc.Encode(report.plain())
		}
		return enc.Encode(report)
	case _reportOutputCSV:
		if !report.SignInReported {
			return renderPlainCSV(w, report)
		}
		cw := csv.NewWriter(w)
		if err := cw.Write([]string{
			"days", "activeSince", "totalUsers", "activeUsers", "usersWithSignInData",
			"role", "users", "signedIn",
		}); err != nil {
			return err
		}
		// the report context is repeated per row so that a single flat table
		// carries the totals as well, even when there is no role at all
		prefix := []string{
			strconv.Itoa(report.Days),
			report.ActiveSince.Format(time.RFC3339),
			strconv.Itoa(report.TotalUsers),
			csvCount(report.ActiveUsers),
			strconv.Itoa(report.UsersWithSignInData),
		}
		if len(report.Roles) == 0 {
			// a placeholder row keeps the totals readable for a consumer that
			// only looks at the rows; its sign-in cell follows availability
			noSignedIn := csvCount(optionalCount(0, report.ActiveUsers != nil))
			if err := cw.Write(slices.Concat(prefix, []string{"", "0", noSignedIn})); err != nil {
				return err
			}
		}
		for _, r := range report.Roles {
			row := slices.Concat(prefix, []string{r.Role, strconv.Itoa(r.Users), csvCount(r.SignedIn)})
			if err := cw.Write(row); err != nil {
				return err
			}
		}
		cw.Flush()
		return cw.Error()
	default:
		if !report.SignInReported {
			return renderPlainTable(w, report)
		}
		fmt.Fprintf(w, "Sign-in window: last %s (since %s)\n\n",
			pluralDays(report.Days), report.ActiveSince.Format(time.RFC3339))

		signedInLabel := "Signed in within the last " + pluralDays(report.Days)
		totalLabel := "Users in total:"
		activeLabel := signedInLabel + ":"
		knownLabel := "Users with a recorded sign-in:"
		labelWidth := max(len(totalLabel), len(activeLabel), len(knownLabel))
		fmt.Fprintf(w, "%-*s %d\n", labelWidth, totalLabel, report.TotalUsers)
		fmt.Fprintf(w, "%-*s %s\n", labelWidth, activeLabel, tableCount(report.ActiveUsers))
		fmt.Fprintf(w, "%-*s %d\n\n", labelWidth, knownLabel, report.UsersWithSignInData)

		table := tablewriter.NewTable(w, tablewriter.WithConfig(tablewriter.Config{
			Header: tw.CellConfig{
				Formatting: tw.CellFormatting{AutoFormat: tw.Off},
			},
			Row: tw.CellConfig{
				Alignment: tw.CellAlignment{
					PerColumn: []tw.Align{tw.AlignLeft, tw.AlignRight, tw.AlignRight},
				},
			},
		}))
		table.Header([]string{"Role", "Users", signedInLabel})
		for _, r := range report.Roles {
			row := []string{r.Role, strconv.Itoa(r.Users), tableCount(r.SignedIn)}
			if err := table.Append(row); err != nil {
				return err
			}
		}
		if err := table.Render(); err != nil {
			return err
		}

		if report.TotalUsers > 0 && report.UsersWithSignInData == 0 {
			fmt.Fprint(w, "\nWarning: not a single user has a recorded sign-in, so the sign-in\n"+
				"counts are reported as N/A rather than 0. The identity backend accepts\n"+
				"the write, so either nobody has signed in since sign-in recording was\n"+
				"switched on, or the directory does not hold the OpenCloud LDAP schema.\n")
		}
		return nil
	}
}

// renderPlainCSV writes the user counts per role as csv, without any of the
// sign-in columns.
func renderPlainCSV(w io.Writer, report userReport) error {
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{"totalUsers", "role", "users"}); err != nil {
		return err
	}
	total := strconv.Itoa(report.TotalUsers)
	if len(report.Roles) == 0 {
		// a placeholder row keeps the total readable for a consumer that only
		// looks at the rows
		if err := cw.Write([]string{total, "", "0"}); err != nil {
			return err
		}
	}
	for _, r := range report.Roles {
		if err := cw.Write([]string{total, r.Role, strconv.Itoa(r.Users)}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// renderPlainTable writes the user counts per role as a table, without the
// sign-in column, the sign-in totals and the warning about missing sign-ins.
func renderPlainTable(w io.Writer, report userReport) error {
	fmt.Fprintf(w, "Users in total: %d\n\n", report.TotalUsers)

	table := tablewriter.NewTable(w, tablewriter.WithConfig(tablewriter.Config{
		Header: tw.CellConfig{
			Formatting: tw.CellFormatting{AutoFormat: tw.Off},
		},
		Row: tw.CellConfig{
			Alignment: tw.CellAlignment{
				PerColumn: []tw.Align{tw.AlignLeft, tw.AlignRight},
			},
		},
	}))
	table.Header([]string{"Role", "Users"})
	for _, r := range report.Roles {
		if err := table.Append([]string{r.Role, strconv.Itoa(r.Users)}); err != nil {
			return err
		}
	}
	return table.Render()
}

// tableCount renders a sign-in count for human readable output, spelling out
// that it is not available rather than showing a misleading 0.
func tableCount(count *int) string {
	if count == nil {
		return "N/A"
	}
	return strconv.Itoa(*count)
}

// csvCount renders a sign-in count for csv output, leaving the cell empty when
// the count is not available.
func csvCount(count *int) string {
	if count == nil {
		return ""
	}
	return strconv.Itoa(*count)
}

// pluralDays renders a day count for human readable output.
func pluralDays(days int) string {
	if days == 1 {
		return "1 day"
	}
	return strconv.Itoa(days) + " days"
}
