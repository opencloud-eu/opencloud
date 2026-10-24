package command

import (
	"os"
	"slices"
	"strings"

	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config/parser"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/unifiedrole"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/spf13/cobra"
)

var (
	unifiedRolesNames = map[string]string{
		unifiedrole.UnifiedRoleViewerID:                                    "Viewer",
		unifiedrole.UnifiedRoleViewerWithVersionsID:                        "ViewerWithVersions",
		unifiedrole.UnifiedRoleViewerListGrantsID:                          "ViewerListGrants",
		unifiedrole.UnifiedRoleSpaceViewerID:                               "SpaceViewer",
		unifiedrole.UnifiedRoleSpaceViewerWithVersionsID:                   "SpaceViewerWithVersions",
		unifiedrole.UnifiedRoleEditorID:                                    "Editor",
		unifiedrole.UnifiedRoleEditorWithVersionsID:                        "EditorWithVersions",
		unifiedrole.UnifiedRoleEditorListGrantsID:                          "EditorListGrants",
		unifiedrole.UnifiedRoleEditorListGrantsWithVersionsID:              "EditorListGrantsWithVersions",
		unifiedrole.UnifiedRoleSpaceEditorID:                               "SpaceEditor",
		unifiedrole.UnifiedRoleSpaceEditorWithoutVersionsID:                "SpaceEditorWithoutVersions",
		unifiedrole.UnifiedRoleSpaceEditorWithoutTrashbinID:                "SpaceEditorWithoutTrashbin",
		unifiedrole.UnifiedRoleSpaceEditorWithoutVersionsWithoutTrashbinID: "SpaceEditorWithoutVersionsWithoutTrashbin",
		unifiedrole.UnifiedRoleFileEditorID:                                "FileEditor",
		unifiedrole.UnifiedRoleFileEditorWithVersionsID:                    "FileEditorWithVersions",
		unifiedrole.UnifiedRoleFileEditorListGrantsID:                      "FileEditorListGrants",
		unifiedrole.UnifiedRoleFileEditorListGrantsWithVersionsID:          "FileEditorListGrantsWithVersions",
		unifiedrole.UnifiedRoleEditorLiteID:                                "EditorLite",
		unifiedrole.UnifiedRoleManagerID:                                   "SpaceManager",
		unifiedrole.UnifiedRoleSecureViewerID:                              "SecureViewer",
		unifiedrole.UnifiedRoleDeniedID:                                    "Denied",
	}
)

// UnifiedRoles bundles available commands for unified roles
func UnifiedRoles(cfg *config.Config) []*cobra.Command {
	cmds := []*cobra.Command{
		listUnifiedRoles(cfg),
	}

	for _, cmd := range cmds {
		cmd.Use = strings.Join([]string{cmd.Use, "unified-roles"}, "-")
		cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
			return configlog.ReturnError(parser.ParseConfig(cfg))
		}
	}

	return cmds
}

// unifiedRolesStatus lists available unified roles, it contains an indicator to show if the role is enabled or not
func listUnifiedRoles(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "list available unified roles",
		RunE: func(cmd *cobra.Command, args []string) error {
			outputFormat, err := cmd.Flags().GetString("output-format")
			if err != nil {
				return err
			}

			r := tw.Rendition{
				Settings: tw.Settings{
					Separators: tw.Separators{
						BetweenRows: tw.On,
					},
				},
			}

			var opt tablewriter.Option
			switch outputFormat {
			case "md":
				opt = tablewriter.WithRenderer(renderer.NewMarkdown())
			case "colorized":
				opt = tablewriter.WithRenderer(renderer.NewColorized())
			default:
				opt = tablewriter.WithRenderer(renderer.NewBlueprint(r))
			}
			tbl := tablewriter.NewTable(os.Stdout, opt)

			headers := []string{"Name", "UID", "Enabled", "Description", "Condition", "Allowed resource actions"}
			tbl.Header(headers)

			for _, definition := range unifiedrole.GetRoles(unifiedrole.RoleFilterAll()) {
				const enabled = "enabled"
				const disabled = "disabled"

				rows := [][]string{
					{unifiedRolesNames[definition.GetId()], definition.GetId(), disabled, definition.GetDescription()},
				}
				if slices.Contains(cfg.UnifiedRoles.AvailableRoles, definition.GetId()) {
					rows[0][2] = enabled
				}

				for i, rolePermission := range definition.GetRolePermissions() {
					actions := strings.Join(rolePermission.GetAllowedResourceActions(), "\n")
					row := []string{rolePermission.GetCondition(), actions}
					switch i {
					case 0:
						rows[0] = append(rows[0], row...)
					default:
						rows[0][4] = rows[0][4] + "\n" + rolePermission.GetCondition()
					}
				}

				for _, row := range rows {
					// balance the row before adding it to the table,
					// this prevents the row from having empty columns.
					tbl.Append(append(row, make([]string, len(headers)-len(row))...))
				}
			}

			tbl.Render()
			return nil
		},
	}

	cmd.Flags().StringP("output-format", "o", "",
		"Adjust the basic table output. Available options: md, colorized")

	return cmd
}
