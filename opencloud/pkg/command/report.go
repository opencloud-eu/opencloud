package command

import (
	"cmp"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strconv"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/opencloud-eu/opencloud/opencloud/pkg/register"
	"github.com/opencloud-eu/opencloud/pkg/config"
	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/config/parser"
	mregistry "github.com/opencloud-eu/opencloud/pkg/registry"
	graphparser "github.com/opencloud-eu/opencloud/services/graph/pkg/config/parser"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

const (
	_reportOutputFlagName = "output"
	_reportDaysFlagName   = "days"

	// _reportDefaultDays is the default sign-in window in days.
	_reportDefaultDays = 31

	// _reportNoRole is the bucket for users without any assigned role.
	_reportNoRole = "(no role assigned)"

	// _reportRequestTimeout caps the whole report run.
	_reportRequestTimeout = 1 * time.Minute

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
}

// ReportCommand is the entrypoint for the report command.
func ReportCommand(cfg *config.Config) *cobra.Command {
	reportCmd := &cobra.Command{
		Use:   "report",
		Short: "report the users of this instance grouped by their assigned role",
		Long: "Report the users of OpenCloud grouped by their assigned role.\n\n" +
			"Every user is counted, and each role additionally shows how many of its\n" +
			"users signed in successfully within the last --days days. Users whose\n" +
			"sign-in is unknown still count towards their role.\n" +
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
			return configlog.ReturnError(graphparser.ParseConfig(cfg.Graph))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			days, err := cmd.Flags().GetInt(_reportDaysFlagName)
			if err != nil {
				return err
			}
			if days < 1 {
				return fmt.Errorf("--%s must be at least 1", _reportDaysFlagName)
			}

			format, err := cmd.Flags().GetString(_reportOutputFlagName)
			if err != nil {
				return err
			}
			if !slices.Contains([]string{_reportOutputTable, _reportOutputJSON, _reportOutputCSV, _reportOutputYAML}, format) {
				return fmt.Errorf("unknown output format %q, supported formats are: %s, %s, %s, %s",
					format, _reportOutputTable, _reportOutputJSON, _reportOutputCSV, _reportOutputYAML)
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), _reportRequestTimeout)
			defer cancel()

			report, err := userRoleReport(ctx, cfg, days)
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

	return reportCmd
}

func init() {
	register.AddCommand(ReportCommand)
}

// userRoleReport collects all users from the libregraph API and groups the ones
// that signed in within the last `days` days by their assigned role.
func userRoleReport(ctx context.Context, cfg *config.Config, days int) (userReport, error) {
	client, err := newLibregraphServiceAccountClient(ctx, cfg)
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
	activeSince := now.AddDate(0, 0, -days)
	roles, activeUsers, withSignInData := countUsersByRole(users, roleNames, activeSince)

	return userReport{
		GeneratedAt:         now,
		Days:                days,
		ActiveSince:         activeSince,
		TotalUsers:          len(users),
		ActiveUsers:         activeUsers,
		UsersWithSignInData: withSignInData,
		Roles:               roles,
	}, nil
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
// available instead of claiming nobody signed in.
func countUsersByRole(users []libregraph.User, roleNames map[string]string, activeSince time.Time) ([]roleCount, *int, int) {
	totals := make(map[string]int, len(roleNames)+1)
	signedIn := make(map[string]int, len(roleNames)+1)
	activeUsers := 0
	withSignInData := 0
	for _, u := range users {
		signInActivity := u.GetSignInActivity()
		lastSignIn, hasSignIn := signInActivity.GetLastSuccessfulSignInDateTimeOk()
		if hasSignIn {
			withSignInData++
		}
		active := hasSignIn && !lastSignIn.Before(activeSince)
		if active {
			activeUsers++
		}

		for _, name := range assignedRoleNames(u, roleNames) {
			totals[name]++
			if active {
				signedIn[name]++
			}
		}
	}

	// without a single recorded sign-in we cannot tell an inactive user from
	// one whose sign-in was never stored
	signInKnown := withSignInData > 0 || len(users) == 0

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
func newLibregraphServiceAccountClient(ctx context.Context, cfg *config.Config) (*libregraph.APIClient, error) {
	// Initialize registry to make service lookup work
	reg := mregistry.GetRegistry()

	gatewaySelector, err := pool.GatewaySelector(cfg.Graph.Reva.Address)
	if err != nil {
		return nil, err
	}
	gatewayClient, err := gatewaySelector.Next()
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
	services, err := reg.GetService(serviceName)
	if err != nil {
		return nil, fmt.Errorf("could not look up the '%s' service, is OpenCloud running? %w", serviceName, err)
	}

	var baseURL string
	for _, s := range services {
		for _, n := range s.Nodes {
			protocol := n.Metadata["protocol"]
			if protocol == "" {
				protocol = "http"
			}
			baseURL = fmt.Sprintf("%s://%s%s", protocol, n.Address, cfg.Graph.HTTP.Root)
			break
		}
		if baseURL != "" {
			break
		}
	}
	if baseURL == "" {
		return nil, fmt.Errorf("no running instance of the '%s' service found", serviceName)
	}

	lgconf := libregraph.NewConfiguration()
	lgconf.Servers = libregraph.ServerConfigurations{{URL: baseURL}}
	lgconf.DefaultHeader = map[string]string{revactx.TokenHeader: token}
	lgconf.HTTPClient = &http.Client{Timeout: _reportRequestTimeout}

	return libregraph.NewAPIClient(lgconf), nil
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
		return enc.Encode(report)
	case _reportOutputYAML:
		enc := yaml.NewEncoder(w)
		defer enc.Close()
		return enc.Encode(report)
	case _reportOutputCSV:
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
			if err := cw.Write(append(prefix, "", "0", noSignedIn)); err != nil {
				return err
			}
		}
		for _, r := range report.Roles {
			row := append(prefix, r.Role, strconv.Itoa(r.Users), csvCount(r.SignedIn))
			if err := cw.Write(row); err != nil {
				return err
			}
		}
		cw.Flush()
		return cw.Error()
	default:
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
				"counts are reported as N/A rather than 0. This instance does not store\n"+
				"sign-in timestamps. They are only recorded when the identity backend\n"+
				"is writable, see OC_LDAP_SERVER_WRITE_ENABLED, and when the directory\n"+
				"holds the OpenCloud LDAP schema.\n")
		}
		return nil
	}
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
