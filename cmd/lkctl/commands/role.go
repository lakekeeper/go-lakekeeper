package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/client"
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

	addProjectFlag(cmd, &project)

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

			var (
				roles     []managementv1.Role
				nextToken string
			)
			if listOpts.name != "" {
				roles, err = findRolesByName(ctx, c, listOpts.name, listOpts.limit, client.WithProject(*project))
				if err != nil {
					return err
				}
			} else {
				resp, err := c.Roles.List(ctx,
					&client.ListRolesOptions{PageSize: listOpts.limit, PageToken: listOpts.token},
					client.WithProject(*project))
				if err != nil {
					return err
				}
				roles, nextToken = resp.GetRoles(), resp.GetNextPageToken()
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
				if nextToken != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "\nNext page token: %s\n", nextToken)
				}
				return nil
			default:
				return fmt.Errorf("unknown output format: %s", output)
			}
		},
	}

	addListFlags(cmd, &listOpts)
	cmd.Flags().Lookup("name").Usage = "Show only the role named exactly this, searched across the whole project (not just the current page)"
	cmd.MarkFlagsMutuallyExclusive("name", "token")
	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format. One of: json|text|wide")
	return cmd
}

// findRolesByName returns the role named exactly name, or none. Lakekeeper
// keeps role names unique per project (ignoring case), so there is at most
// one. It runs the server's role search and keeps the exact match. The search
// returns only the 10 nearest roles and truncates the search string to 64
// characters, so it can miss an exact match (for example among 10 or more
// names that differ only in punctuation). On a miss it therefore pages
// through the whole listing, pageSize roles at a time, before reporting none.
func findRolesByName(ctx context.Context, c *client.Client, name string, pageSize int64, callOpts ...client.CallOption) ([]managementv1.Role, error) {
	found, err := c.Roles.Search(ctx, managementv1.NewSearchRoleRequest(name), callOpts...)
	if err != nil {
		return nil, err
	}
	if matches := exactRoleNameMatches(found.GetRoles(), name); len(matches) > 0 {
		return matches, nil
	}

	matches := []managementv1.Role{}
	opts := &client.ListRolesOptions{PageSize: pageSize}
	for {
		page, err := c.Roles.List(ctx, opts, callOpts...)
		if err != nil {
			return nil, err
		}
		matches = append(matches, exactRoleNameMatches(page.GetRoles(), name)...)
		opts.PageToken = page.GetNextPageToken()
		if opts.PageToken == "" {
			return matches, nil
		}
	}
}

func exactRoleNameMatches(roles []managementv1.Role, name string) []managementv1.Role {
	matches := make([]managementv1.Role, 0, 1)
	for i := range roles {
		if roles[i].GetName() == name {
			matches = append(matches, roles[i])
		}
	}
	return matches
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

			role, err := c.Roles.Get(ctx, args[0], client.WithProject(*project))
			if err != nil {
				return err
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
			role, err := c.Roles.Create(ctx, req, client.WithProject(*project))
			if err != nil {
				return err
			}

			switch output {
			case "text":
				fmt.Fprintf(cmd.OutOrStdout(), "Role %s created with id %s\n", args[0], role.GetId())
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
			role, err := c.Roles.Update(ctx, args[0], req, client.WithProject(*project))
			if err != nil {
				return err
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
			if err := c.Roles.Delete(ctx, args[0], client.WithProject(*project)); err != nil {
				return err
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

			resp, err := c.Permissions.Roles.Access(ctx, args[0], &client.GetAuthorizerRoleActionsOptions{
				PrincipalUser: access.user,
				PrincipalRole: access.role,
			})
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

			resp, err := c.Permissions.Roles.Assignments(ctx, args[0], &client.GetRoleAssignmentsByIDOptions{
				Relations: toEnums[managementv1.RoleRelation](assignments.relations),
			})
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
			set, err := buildAssignmentSet[managementv1.RoleAssignment](assignments, users, roles)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}
			if err := c.Permissions.Roles.Grant(ctx, args[0], set); err != nil {
				return err
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
			set, err := buildAssignmentSet[managementv1.RoleAssignment](assignments, users, roles)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}
			if err := c.Permissions.Roles.Revoke(ctx, args[0], set); err != nil {
				return err
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
			updatedAt, _ := r.GetUpdatedAtOk()
			description, _ := r.GetDescriptionOk()
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
				r.GetId(), r.GetName(), r.GetProjectId(),
				r.GetCreatedAt().Format(time.RFC3339),
				formatTimePtr(updatedAt),
				formatStringPtr(description))
		}
	} else {
		fmt.Fprintln(tw, "ID\tNAME\tPROJECT ID\tCREATED AT\tUPDATED AT")
		for i := range roles {
			r := &roles[i]
			updatedAt, _ := r.GetUpdatedAtOk()
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
				r.GetId(), r.GetName(), r.GetProjectId(),
				r.GetCreatedAt().Format(time.RFC3339),
				formatTimePtr(updatedAt))
		}
	}
	return tw.Flush()
}
