package clihelper

import (
	"fmt"

	"github.com/opencloud-eu/opencloud/pkg/version"

	"github.com/spf13/cobra"
)

// DefaultApp is a wrapper for DefaultApp that adds Cobra specific settings
func DefaultApp(app *cobra.Command) *cobra.Command {
	// TODO: when migration is done this has to become DefaultApp
	// version info
	app.Version = fmt.Sprintf("%s (%s <%s>) (%s)", version.String, "OpenCloud GmbH", "support@opencloud.eu", version.Compiled())

	// cobra would print the error on top of what main() already prints
	app.SilenceErrors = true

	// keep the usage block for flag parse errors, drop it once RunE runs.
	// Traversing runs the hook even below a subcommand that brings its own,
	// e.g. every service below ServiceCommand.
	cobra.EnableTraverseRunHooks = true
	app.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true
		return nil
	}

	return app
}
