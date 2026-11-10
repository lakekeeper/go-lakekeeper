package commands

import (
	"errors"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/spf13/cobra"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/client"
)

func newProjectCmd(opts *clientOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "project",
		Aliases: []string{"proj"},
		Short:   "Manage projects",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newProjectListCmd(opts))
	cmd.AddCommand(newProjectGetCmd(opts))
	cmd.AddCommand(newProjectCreateCmd(opts))
	cmd.AddCommand(newProjectRenameCmd(opts))
	cmd.AddCommand(newProjectDeleteCmd(opts))
	cmd.AddCommand(newProjectAccessCmd(opts))
	cmd.AddCommand(newProjectAssignmentsCmd(opts))
	cmd.AddCommand(newProjectGrantCmd(opts))
	cmd.AddCommand(newProjectRevokeCmd(opts))

	return cmd
}

func newProjectListCmd(opts *clientOptions) *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List projects available to the current user",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}
			resp, err := c.Projects.List(ctx)
			if err != nil {
				return err
			}

			switch output {
			case "json":
				return printJSON(cmd.OutOrStdout(), resp.GetProjects())
			case "text":
				if len(resp.GetProjects()) == 0 {
					fmt.Fprintln(cmd.OutOrStdout(), "No projects available")
					return nil
				}
				return printProjects(cmd.OutOrStdout(), resp.GetProjects()...)
			default:
				return fmt.Errorf("unknown output format: %s", output)
			}
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format. One of: json|text")
	return cmd
}

func newProjectGetCmd(opts *clientOptions) *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "get PROJECT-ID",
		Short: "Get a project by id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}
			resp, err := c.Projects.Get(ctx, client.WithProject(args[0]))
			if err != nil {
				return err
			}

			switch output {
			case "json":
				return printJSON(cmd.OutOrStdout(), resp)
			case "text":
				return printProjects(cmd.OutOrStdout(), *resp)
			default:
				return fmt.Errorf("unknown output format: %s", output)
			}
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format. One of: json|text")
	return cmd
}

func newProjectCreateCmd(opts *clientOptions) *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "create PROJECTNAME",
		Short: "Create a new project",
		Example: `  # Create a new project
  lkctl project create "New Project"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}

			created, err := c.Projects.Create(ctx, managementv1.NewCreateProjectRequest(args[0]))
			if err != nil {
				return err
			}

			switch output {
			case "text":
				fmt.Fprintf(cmd.OutOrStdout(), "Project %s created with id %s\n", args[0], created.GetProjectId())
				return nil
			case "json":
				project, err := c.Projects.Get(ctx, client.WithProject(created.GetProjectId()))
				if err != nil {
					return err
				}
				return printJSON(cmd.OutOrStdout(), project)
			default:
				return fmt.Errorf("unknown output format: %s", output)
			}
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format. One of: json|text")
	return cmd
}

func newProjectRenameCmd(opts *clientOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rename PROJECT-ID NEW-NAME",
		Short: "Rename a project",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}

			if err := c.Projects.Rename(ctx, managementv1.NewRenameProjectRequest(args[1]), client.WithProject(args[0])); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Project %s renamed to %s\n", args[0], args[1])
			return nil
		},
	}
	return cmd
}

func newProjectDeleteCmd(opts *clientOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete PROJECT-ID",
		Aliases: []string{"rm"},
		Short:   "Delete a project",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}
			if err := c.Projects.Delete(ctx, client.WithProject(args[0])); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Project %s deleted\n", args[0])
			return nil
		},
	}
	return cmd
}

// projectArgHelp documents the optional [PROJECT-ID] argument of project
// access and assignments.
const projectArgHelp = "PROJECT-ID is optional; when omitted, the server's default project is used."

// projectRequiredHelp documents the required PROJECT-ID of grant and
// revoke, which never fall back to the server's default project.
const projectRequiredHelp = "PROJECT-ID is required: changing permissions never falls back to the server's default project. " +
	"To target the default project, pass its id (the nil UUID 00000000-0000-0000-0000-000000000000 unless the operator configured another)."

// optionalProject maps the optional [PROJECT-ID] argument to façade call
// options: no argument sends no project, like an omitted -p. A given but
// empty argument becomes WithProject(""), which the façade rejects.
func optionalProject(args []string) []client.CallOption {
	if len(args) == 0 {
		return nil
	}
	return []client.CallOption{client.WithProject(args[0])}
}

func newProjectAccessCmd(opts *clientOptions) *cobra.Command {
	var (
		access accessOpts
		output string
	)

	cmd := &cobra.Command{
		Use:   "access [PROJECT-ID]",
		Short: "Get project access",
		Long:  "Get project access. By default, the current user's access is returned.\n\n" + projectArgHelp,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if access.user != "" && access.role != "" {
				return errors.New("--user and --role are mutually exclusive")
			}

			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}

			resp, err := c.Permissions.Projects.Access(ctx, &client.GetProjectAccessOptions{
				PrincipalUser: access.user,
				PrincipalRole: access.role,
			}, optionalProject(args)...)
			if err != nil {
				return err
			}

			switch output {
			case "json":
				return printJSON(cmd.OutOrStdout(), resp)
			case "text":
				return printAllowedActions(cmd.OutOrStdout(), resp.GetAllowedActions())
			default:
				return fmt.Errorf("unknown output format: %s", output)
			}
		},
	}

	addAccessFlags(cmd, &access)
	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format. One of: json|text")
	return cmd
}

func newProjectAssignmentsCmd(opts *clientOptions) *cobra.Command {
	var (
		assignments assignmentsOpts
		output      string
	)

	cmd := &cobra.Command{
		Use:   "assignments [PROJECT-ID]",
		Short: "Get project assignments",
		Long:  "Get project assignments.\n\n" + projectArgHelp,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}

			resp, err := c.Permissions.Projects.Assignments(ctx, &client.GetProjectAssignmentsOptions{
				Relations: toEnums[managementv1.ProjectRelation](assignments.relations),
			}, optionalProject(args)...)
			if err != nil {
				return err
			}

			switch output {
			case "json":
				return printJSON(cmd.OutOrStdout(), resp)
			case "text":
				return printAssignments(cmd.OutOrStdout(), resp.GetAssignments()...)
			default:
				return fmt.Errorf("unknown output format: %s", output)
			}
		},
	}

	addAssignmentsFlags(cmd, &assignments)
	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format. One of: json|text")
	return cmd
}

func newProjectGrantCmd(opts *clientOptions) *cobra.Command {
	var (
		users       []string
		roles       []string
		assignments []string
	)

	cmd := &cobra.Command{
		Use:     "grant PROJECT-ID",
		Aliases: []string{"assign"},
		Short:   "Add project assignments",
		Long:    "Add project assignments.\n\n" + projectRequiredHelp,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			set, err := buildAssignmentSet[managementv1.ProjectAssignment](assignments, users, roles)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}
			if err := c.Permissions.Projects.Grant(ctx, set, client.WithProject(args[0])); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Project permissions updated")
			return nil
		},
	}

	cmd.Flags().StringSliceVar(&users, "users", nil, "Grant access to users; repeat or comma-separate for multiple")
	cmd.Flags().StringSliceVar(&roles, "roles", nil, "Grant access to roles; repeat or comma-separate for multiple")
	cmd.Flags().StringSliceVar(&assignments, "assignments", nil, "Assignment relations to apply; repeat or comma-separate for multiple")
	if err := cmd.MarkFlagRequired("assignments"); err != nil {
		panic(err) // unreachable: the flag was just registered.
	}
	return cmd
}

func newProjectRevokeCmd(opts *clientOptions) *cobra.Command {
	var (
		users       []string
		roles       []string
		assignments []string
	)

	cmd := &cobra.Command{
		Use:     "revoke PROJECT-ID",
		Aliases: []string{"unassign"},
		Short:   "Remove project assignments",
		Long:    "Remove project assignments.\n\n" + projectRequiredHelp,
		Example: `  # Revoke project_admin from a user on the default project (nil UUID in the default configuration)
  lkctl project revoke 00000000-0000-0000-0000-000000000000 --users oidc~alice --assignments project_admin

  # Revoke select from a role on a specific project
  lkctl project revoke 0198618c-5be8-7a82-a0b9-1076c9dd12f0 --roles 11111111-2222-3333-4444-555555555555 --assignments select`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			set, err := buildAssignmentSet[managementv1.ProjectAssignment](assignments, users, roles)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}
			if err := c.Permissions.Projects.Revoke(ctx, set, client.WithProject(args[0])); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Project permissions updated")
			return nil
		},
	}

	cmd.Flags().StringSliceVar(&users, "users", nil, "Revoke access from users; repeat or comma-separate for multiple")
	cmd.Flags().StringSliceVar(&roles, "roles", nil, "Revoke access from roles; repeat or comma-separate for multiple")
	cmd.Flags().StringSliceVar(&assignments, "assignments", nil, "Assignment relations to remove; repeat or comma-separate for multiple")
	if err := cmd.MarkFlagRequired("assignments"); err != nil {
		panic(err) // unreachable: the flag was just registered.
	}
	return cmd
}

func printProjects(w io.Writer, projects ...managementv1.GetProjectResponse) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tNAME")
	for _, p := range projects {
		fmt.Fprintf(tw, "%s\t%s\n", p.GetProjectId(), p.GetProjectName())
	}
	return tw.Flush()
}
