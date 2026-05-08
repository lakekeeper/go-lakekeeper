package commands

import (
	"errors"
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/permissions"
)

func newRoleCmd(opts *clientOptions) *cobra.Command {
	var project string

	cmd := &cobra.Command{
		Use:   "role",
		Short: "Manage roles",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.PersistentFlags().StringVarP(&project, "project", "p", uuid.Nil.String(), "Select a project")

	cmd.AddCommand(newRoleListCmd(opts, &project))
	cmd.AddCommand(newRoleGetCmd(opts, &project))
	cmd.AddCommand(newRoleCreateCmd(opts, &project))
	cmd.AddCommand(newRoleUpdateCmd(opts, &project))
	cmd.AddCommand(newRoleDeleteCmd(opts, &project))
	cmd.AddCommand(newRoleAccessCmd(opts))
	cmd.AddCommand(newRoleAssignmentsCmd(opts))
	cmd.AddCommand(newRoleGrantCmd(opts))
	cmd.AddCommand(newRoleRevokeCmd(opts))

	return cmd
}

func newRoleListCmd(opts *clientOptions, project *string) *cobra.Command {
	var (
		listOpts listOpts
		output   string
	)

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List available roles",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}

			req := c.RoleAPI.ListRoles(ctx).XProjectId(*project).PageSize(listOpts.limit)
			if listOpts.token != "" {
				req = req.PageToken(listOpts.token)
			}

			resp, _, err := req.Execute()
			if err != nil {
				return wrapAPIError("list roles", err)
			}

			roles := resp.Roles
			if listOpts.name != "" {
				filtered := make([]managementv1.Role, 0, len(roles))
				for i := range roles {
					if roles[i].Name == listOpts.name {
						filtered = append(filtered, roles[i])
					}
				}
				roles = filtered
			}

			switch output {
			case "json":
				return printJSON(cmd.OutOrStdout(), roles)
			case "text", "wide":
				if len(roles) == 0 {
					fmt.Fprintln(cmd.OutOrStdout(), "No roles available")
					return nil
				}
				if err := printRoles(cmd.OutOrStdout(), output, roles...); err != nil {
					return err
				}
				if resp.NextPageToken.IsSet() {
					fmt.Fprintf(cmd.OutOrStdout(), "\nNext page token: %s\n", *resp.NextPageToken.Get())
				}
				return nil
			default:
				return fmt.Errorf("unknown output format: %s", output)
			}
		},
	}

	addListFlags(cmd, &listOpts)
	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format. One of: json|text|wide")
	return cmd
}

func newRoleGetCmd(opts *clientOptions, project *string) *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "get ROLEID",
		Short: "Get a role by id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}

			role, _, err := c.RoleAPI.GetRole(ctx, args[0]).XProjectId(*project).Execute()
			if err != nil {
				return wrapAPIError("get role", err)
			}

			switch output {
			case "json":
				return printJSON(cmd.OutOrStdout(), role)
			case "text", "wide":
				return printRoles(cmd.OutOrStdout(), output, *role)
			default:
				return fmt.Errorf("unknown output format: %s", output)
			}
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format. One of: json|text|wide")
	return cmd
}

func newRoleCreateCmd(opts *clientOptions, project *string) *cobra.Command {
	var (
		output      string
		description string
	)

	cmd := &cobra.Command{
		Use:     "create ROLENAME",
		Aliases: []string{"add"},
		Short:   "Create a new role",
		Example: `  # Create a role
  lkctl role create "New Role"

  # Create a role with a description
  lkctl role create "New Role" --description "With a description"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}

			req := managementv1.NewCreateRoleRequest(args[0])
			if description != "" {
				req.SetDescription(description)
			}
			role, _, err := c.RoleAPI.CreateRole(ctx).XProjectId(*project).CreateRoleRequest(*req).Execute()
			if err != nil {
				return wrapAPIError("create role", err)
			}

			switch output {
			case "text":
				fmt.Fprintf(cmd.OutOrStdout(), "Role %s created with id %s\n", args[0], role.Id)
				return nil
			case "json":
				return printJSON(cmd.OutOrStdout(), role)
			default:
				return fmt.Errorf("unknown output format: %s", output)
			}
		},
	}

	cmd.Flags().StringVar(&description, "description", "", "Description of the role")
	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format. One of: json|text")
	return cmd
}

func newRoleUpdateCmd(opts *clientOptions, project *string) *cobra.Command {
	var (
		output      string
		description string
	)

	cmd := &cobra.Command{
		Use:   "update ROLEID ROLENAME",
		Short: "Update a role",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}

			req := managementv1.NewUpdateRoleRequest(args[1])
			if description != "" {
				req.SetDescription(description)
			}
			role, _, err := c.RoleAPI.UpdateRole(ctx, args[0]).XProjectId(*project).UpdateRoleRequest(*req).Execute()
			if err != nil {
				return wrapAPIError("update role", err)
			}

			switch output {
			case "text":
				fmt.Fprintf(cmd.OutOrStdout(), "Role %s updated\n", args[0])
				return nil
			case "json":
				return printJSON(cmd.OutOrStdout(), role)
			default:
				return fmt.Errorf("unknown output format: %s", output)
			}
		},
	}

	cmd.Flags().StringVar(&description, "description", "", "Updated description for the role; empty leaves it unchanged")
	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format. One of: json|text")
	return cmd
}

func newRoleDeleteCmd(opts *clientOptions, project *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete ROLEID",
		Aliases: []string{"rm"},
		Short:   "Delete a role by id",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}
			if _, err := c.RoleAPI.DeleteRole(ctx, args[0]).XProjectId(*project).Execute(); err != nil {
				return wrapAPIError("delete role", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Role %s deleted\n", args[0])
			return nil
		},
	}
	return cmd
}

func newRoleAccessCmd(opts *clientOptions) *cobra.Command {
	var (
		access accessOpts
		output string
	)

	cmd := &cobra.Command{
		Use:   "access ROLEID",
		Short: "Get role access",
		Long:  "Get role access. By default, the current user's access is returned.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if access.user != "" && access.role != "" {
				return errors.New("--user and --role are mutually exclusive")
			}
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}

			req := c.PermissionsOpenfgaAPI.GetAuthorizerRoleActions(ctx, args[0])
			if access.user != "" {
				req = req.PrincipalUser(access.user)
			}
			if access.role != "" {
				req = req.PrincipalRole(access.role)
			}
			resp, _, err := req.Execute()
			if err != nil {
				return wrapAPIError("get role access", err)
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

func newRoleAssignmentsCmd(opts *clientOptions) *cobra.Command {
	var (
		assignments assignmentsOpts
		output      string
	)

	cmd := &cobra.Command{
		Use:   "assignments ROLEID",
		Short: "Get role assignments",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}

			req := c.PermissionsOpenfgaAPI.GetRoleAssignmentsById(ctx, args[0])
			if len(assignments.relations) > 0 {
				rels := make([]managementv1.RoleRelation, 0, len(assignments.relations))
				for _, r := range assignments.relations {
					rels = append(rels, managementv1.RoleRelation(r))
				}
				req = req.Relations(rels)
			}

			resp, _, err := req.Execute()
			if err != nil {
				return wrapAPIError("get role assignments", err)
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

func newRoleGrantCmd(opts *clientOptions) *cobra.Command {
	var (
		users       []string
		roles       []string
		assignments []string
	)

	cmd := &cobra.Command{
		Use:     "grant ROLEID",
		Aliases: []string{"assign"},
		Short:   "Add role assignments",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(users) == 0 && len(roles) == 0 {
				return errors.New("at least one --users or --roles value is required")
			}

			req := managementv1.NewUpdateRoleAssignmentsRequest()
			for _, rel := range assignments {
				for _, u := range users {
					a, err := permissions.BuildAssignment[managementv1.RoleAssignment](rel, permissions.PrincipalUser, u)
					if err != nil {
						return err
					}
					req.Writes = append(req.Writes, a)
				}
				for _, r := range roles {
					a, err := permissions.BuildAssignment[managementv1.RoleAssignment](rel, permissions.PrincipalRole, r)
					if err != nil {
						return err
					}
					req.Writes = append(req.Writes, a)
				}
			}

			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}
			if _, err := c.PermissionsOpenfgaAPI.UpdateRoleAssignmentsById(ctx, args[0]).UpdateRoleAssignmentsRequest(*req).Execute(); err != nil {
				return wrapAPIError("update role assignments", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Role permissions updated")
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

func newRoleRevokeCmd(opts *clientOptions) *cobra.Command {
	var (
		users       []string
		roles       []string
		assignments []string
	)

	cmd := &cobra.Command{
		Use:     "revoke ROLEID",
		Aliases: []string{"unassign"},
		Short:   "Remove role assignments",
		Example: `  # Revoke ownership from a user
  lkctl role revoke 0198618c-5be8-7a82-a0b9-1076c9dd12f0 --users 11111111-2222-3333-4444-555555555555 --assignments ownership

  # Revoke assignee from a role
  lkctl role revoke 0198618c-5be8-7a82-a0b9-1076c9dd12f0 --roles 11111111-2222-3333-4444-555555555555 --assignments assignee`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(users) == 0 && len(roles) == 0 {
				return errors.New("at least one --users or --roles value is required")
			}

			req := managementv1.NewUpdateRoleAssignmentsRequest()
			for _, rel := range assignments {
				for _, u := range users {
					a, err := permissions.BuildAssignment[managementv1.RoleAssignment](rel, permissions.PrincipalUser, u)
					if err != nil {
						return err
					}
					req.Deletes = append(req.Deletes, a)
				}
				for _, r := range roles {
					a, err := permissions.BuildAssignment[managementv1.RoleAssignment](rel, permissions.PrincipalRole, r)
					if err != nil {
						return err
					}
					req.Deletes = append(req.Deletes, a)
				}
			}

			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}
			if _, err := c.PermissionsOpenfgaAPI.UpdateRoleAssignmentsById(ctx, args[0]).UpdateRoleAssignmentsRequest(*req).Execute(); err != nil {
				return wrapAPIError("update role assignments", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Role permissions updated")
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

func printRoles(w io.Writer, output string, roles ...managementv1.Role) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if output == "wide" {
		fmt.Fprintln(tw, "ID\tNAME\tPROJECT ID\tCREATED AT\tUPDATED AT\tDESCRIPTION")
		for i := range roles {
			r := &roles[i]
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
				r.Id, r.Name, r.ProjectId,
				r.CreatedAt.Format(time.RFC3339),
				formatNullableTime(r.UpdatedAt),
				formatNullableString(r.Description))
		}
	} else {
		fmt.Fprintln(tw, "ID\tNAME\tPROJECT ID\tCREATED AT\tUPDATED AT")
		for i := range roles {
			r := &roles[i]
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
				r.Id, r.Name, r.ProjectId,
				r.CreatedAt.Format(time.RFC3339),
				formatNullableTime(r.UpdatedAt))
		}
	}
	return tw.Flush()
}
