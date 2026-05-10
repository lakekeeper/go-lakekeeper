package commands

import (
	"errors"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
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
			resp, _, err := c.ProjectAPI.ListProjects(ctx).Execute()
			if err != nil {
				return wrapAPIError("list projects", err)
			}

			switch output {
			case "json":
				return printJSON(cmd.OutOrStdout(), resp.Projects)
			case "text":
				if len(resp.Projects) == 0 {
					fmt.Fprintln(cmd.OutOrStdout(), "No projects available")
					return nil
				}
				return printProjects(cmd.OutOrStdout(), resp.Projects...)
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
			resp, _, err := c.ProjectAPI.GetProject(ctx).XProjectId(args[0]).Execute()
			if err != nil {
				return wrapAPIError("get project", err)
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

			req := managementv1.NewCreateProjectRequest(args[0])
			created, _, err := c.ProjectAPI.CreateProject(ctx).CreateProjectRequest(*req).Execute()
			if err != nil {
				return wrapAPIError("create project", err)
			}

			switch output {
			case "text":
				fmt.Fprintf(cmd.OutOrStdout(), "Project %s created with id %s\n", args[0], created.ProjectId)
				return nil
			case "json":
				project, _, err := c.ProjectAPI.GetProject(ctx).XProjectId(created.ProjectId).Execute()
				if err != nil {
					return wrapAPIError("get project", err)
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

			req := managementv1.NewRenameProjectRequest(args[1])
			if _, err := c.ProjectAPI.RenameProject(ctx).XProjectId(args[0]).RenameProjectRequest(*req).Execute(); err != nil {
				return wrapAPIError("rename project", err)
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
			if _, err := c.ProjectAPI.DeleteProject(ctx).XProjectId(args[0]).Execute(); err != nil {
				return wrapAPIError("delete project", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Project %s deleted\n", args[0])
			return nil
		},
	}
	return cmd
}

func newProjectAccessCmd(opts *clientOptions) *cobra.Command {
	var (
		access accessOpts
		output string
	)

	cmd := &cobra.Command{
		Use:   "access [PROJECT-ID]",
		Short: "Get project access",
		Long:  "Get project access. By default, the current user's access on the default project is returned.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if access.user != "" && access.role != "" {
				return errors.New("--user and --role are mutually exclusive")
			}
			project := uuid.Nil.String()
			if len(args) == 1 {
				project = args[0]
			}

			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}

			req := c.PermissionsOpenfgaAPI.GetProjectAccessById(ctx, project)
			if access.user != "" {
				req = req.PrincipalUser(access.user)
			}
			if access.role != "" {
				req = req.PrincipalRole(access.role)
			}
			resp, _, err := req.Execute()
			if err != nil {
				return wrapAPIError("get project access", err)
			}

			switch output {
			case "json":
				return printJSON(cmd.OutOrStdout(), resp)
			case "text":
				return printAllowedActions(cmd.OutOrStdout(), resp.AllowedActions)
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
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			project := uuid.Nil.String()
			if len(args) == 1 {
				project = args[0]
			}
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}

			req := c.PermissionsOpenfgaAPI.GetProjectAssignmentsById(ctx, project)
			if len(assignments.relations) > 0 {
				rels := make([]managementv1.ProjectRelation, 0, len(assignments.relations))
				for _, r := range assignments.relations {
					rels = append(rels, managementv1.ProjectRelation(r))
				}
				req = req.Relations(rels)
			}
			resp, _, err := req.Execute()
			if err != nil {
				return wrapAPIError("get project assignments", err)
			}

			switch output {
			case "json":
				return printJSON(cmd.OutOrStdout(), resp)
			case "text":
				return printAssignments(cmd.OutOrStdout(), resp.Assignments...)
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
		Use:     "grant [PROJECT-ID]",
		Aliases: []string{"assign"},
		Short:   "Add project assignments",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			project := uuid.Nil.String()
			if len(args) == 1 {
				project = args[0]
			}

			set, err := buildAssignmentSet[managementv1.ProjectAssignment](assignments, users, roles)
			if err != nil {
				return err
			}
			req := managementv1.NewUpdateProjectAssignmentsRequest()
			req.Writes = set

			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}
			if _, err := c.PermissionsOpenfgaAPI.UpdateProjectAssignmentsById(ctx, project).UpdateProjectAssignmentsRequest(*req).Execute(); err != nil {
				return wrapAPIError("update project assignments", err)
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
		Use:     "revoke [PROJECT-ID]",
		Aliases: []string{"unassign"},
		Short:   "Remove project assignments",
		Example: `  # Revoke project_admin from a user on the default project
  lkctl project revoke --users 0198618c-5be8-7a82-a0b9-1076c9dd12f0 --assignments project_admin

  # Revoke select from a role on a specific project
  lkctl project revoke 0198618c-5be8-7a82-a0b9-1076c9dd12f0 --roles 11111111-2222-3333-4444-555555555555 --assignments select`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			project := uuid.Nil.String()
			if len(args) == 1 {
				project = args[0]
			}

			set, err := buildAssignmentSet[managementv1.ProjectAssignment](assignments, users, roles)
			if err != nil {
				return err
			}
			req := managementv1.NewUpdateProjectAssignmentsRequest()
			req.Deletes = set

			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}
			if _, err := c.PermissionsOpenfgaAPI.UpdateProjectAssignmentsById(ctx, project).UpdateProjectAssignmentsRequest(*req).Execute(); err != nil {
				return wrapAPIError("update project assignments", err)
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
		fmt.Fprintf(tw, "%s\t%s\n", p.ProjectId, p.ProjectName)
	}
	return tw.Flush()
}
