package commands

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/client"
)

// newUserCmd returns `lkctl user`, the parent for user-management subcommands.
func newUserCmd(opts *clientOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Manage users",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newUserListCmd(opts))
	cmd.AddCommand(newUserGetCmd(opts))
	cmd.AddCommand(newUserCreateCmd(opts))
	cmd.AddCommand(newUserDeleteCmd(opts))
	cmd.AddCommand(newUserSearchCmd(opts))
	cmd.AddCommand(newUserRolesCmd(opts))
	cmd.AddCommand(newUserAccessCmd(opts))

	return cmd
}

func newUserListCmd(opts *clientOptions) *cobra.Command {
	var (
		listOpts listOpts
		output   string
	)

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List users",
		Aliases: []string{"ls"},
		Example: `  # List users
  lkctl user ls`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}

			resp, err := c.Users.List(ctx, &client.ListUserOptions{
				Name:      listOpts.name,
				PageSize:  listOpts.limit,
				PageToken: listOpts.token,
			})
			if err != nil {
				return err
			}

			switch output {
			case "json":
				return printJSON(cmd.OutOrStdout(), resp)
			case "text", "wide":
				return printUsers(cmd.OutOrStdout(), output, resp.GetNextPageToken(), resp.GetUsers()...)
			default:
				return fmt.Errorf("unknown output format: %s", output)
			}
		},
	}

	addListFlags(cmd, &listOpts)
	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format. One of: json|text|wide")
	return cmd
}

func newUserGetCmd(opts *clientOptions) *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "get USERID",
		Short: "Get a user by id",
		Example: `  # Get a user by id
  lkctl user get oidc~d223d88c-85b6-4859-b5c5-27f3825e47f6`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}

			user, err := c.Users.Get(ctx, args[0])
			if err != nil {
				return err
			}

			switch output {
			case "json":
				return printJSON(cmd.OutOrStdout(), user)
			case "text", "wide":
				return printUsers(cmd.OutOrStdout(), output, "", *user)
			default:
				return fmt.Errorf("unknown output format: %s", output)
			}
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format. One of: json|text|wide")
	return cmd
}

func newUserDeleteCmd(opts *clientOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete USERID",
		Short:   "Delete a user by id",
		Aliases: []string{"rm"},
		Example: `  # Delete a user
  lkctl user delete oidc~d223d88c-85b6-4859-b5c5-27f3825e47f6`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}

			if err := c.Users.Delete(ctx, args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "User %s deleted\n", args[0])
			return nil
		},
	}
	return cmd
}

func newUserCreateCmd(opts *clientOptions) *cobra.Command {
	var (
		output string
		email  string
		update bool
	)

	cmd := &cobra.Command{
		Use:     "create USERID NAME USERTYPE",
		Aliases: []string{"add"},
		Short:   "Create a new user",
		Example: `  # Create an OIDC human user
  lkctl user create oidc~d223d88c-85b6-4859-b5c5-27f3825e47f6 "Peter Cold" human

  # Create a Kubernetes application user
  lkctl user create kubernetes~d223d88c-85b6-4859-b5c5-27f3825e47f6 "Service Account" application

  # Add an email
  lkctl user create oidc~d223d88c-85b6-4859-b5c5-27f3825e47f6 "Peter Cold" human --email peter@example.com`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}

			req := managementv1.NewCreateUserRequest()
			req.SetId(args[0])
			req.SetName(args[1])
			req.SetUserType(managementv1.UserType(args[2]))
			if email != "" {
				req.SetEmail(email)
			}
			if update {
				req.SetUpdateIfExists(true)
			}

			user, err := c.Users.Create(ctx, req)
			if err != nil {
				return err
			}

			switch output {
			case "text":
				fmt.Fprintf(cmd.OutOrStdout(), "User %s registered with id %s\n", args[1], user.GetId())
				return nil
			case "json":
				return printJSON(cmd.OutOrStdout(), user)
			default:
				return fmt.Errorf("unknown output format: %s", output)
			}
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format. One of: json|text")
	cmd.Flags().StringVar(&email, "email", "", "Email of the user")
	cmd.Flags().BoolVar(&update, "update", false, "Update the user if it already exists")
	return cmd
}

func newUserSearchCmd(opts *clientOptions) *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "search TEXT",
		Short: "Fuzzy-search users",
		Long: `Fuzzy-search users. The server searches with
the first 64 characters of TEXT and returns only the closest matches, so
an exact match may be missing; use "user list --name" for an exhaustive
name filter.`,
		Example: `  # Find users whose name is close to "peter"
  lkctl user search peter`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}

			resp, err := c.Users.Search(ctx, managementv1.NewSearchUserRequest(args[0]))
			if err != nil {
				return err
			}

			switch output {
			case "json":
				return printJSON(cmd.OutOrStdout(), resp)
			case "text":
				return printSearchUsers(cmd.OutOrStdout(), resp.GetUsers()...)
			default:
				return fmt.Errorf("unknown output format: %s", output)
			}
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format. One of: json|text")
	return cmd
}

func newUserRolesCmd(opts *clientOptions) *cobra.Command {
	var (
		project    string
		limit      int64
		token      string
		transitive bool
		output     string
	)

	cmd := &cobra.Command{
		Use:   "roles USERID",
		Short: "List the roles a user is assigned to",
		Long: `List the roles a user is directly assigned to, in one project.

With --transitive, list the user's effective roles instead: the direct
assignments plus every role reachable through role membership. The server
supports --transitive only when assignments are catalog-managed; with an
assignment-managing authorizer such as OpenFGA it answers 501 Not
Implemented.`,
		Example: `  # Roles a user is directly assigned to in the default project
  lkctl user roles oidc~d223d88c-85b6-4859-b5c5-27f3825e47f6

  # Effective roles in a given project
  lkctl user roles oidc~d223d88c-85b6-4859-b5c5-27f3825e47f6 --transitive -p <PROJECT-ID>`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}

			list := c.Users.Roles
			if transitive {
				list = c.Users.TransitiveRoles
			}
			resp, err := list(ctx, args[0], &client.ListUserRolesOptions{
				PageSize:  limit,
				PageToken: token,
			}, projectFlagOptions(cmd, project)...)
			if err != nil {
				return err
			}

			switch output {
			case "json":
				return printJSON(cmd.OutOrStdout(), resp)
			case "text":
				return printRoleMemberships(cmd.OutOrStdout(), resp.GetNextPageToken(), resp.GetRoles()...)
			default:
				return fmt.Errorf("unknown output format: %s", output)
			}
		},
	}

	addProjectFlag(cmd, &project)
	cmd.Flags().Int64Var(&limit, "limit", 100, "Upper bound on the number of results returned to the client")
	cmd.Flags().StringVar(&token, "token", "", "Pagination token")
	cmd.Flags().BoolVar(&transitive, "transitive", false, "List effective roles, including those reached through role membership")
	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format. One of: json|text")
	return cmd
}

func newUserAccessCmd(opts *clientOptions) *cobra.Command {
	var (
		access accessOpts
		output string
	)

	cmd := &cobra.Command{
		Use:   "access USERID",
		Short: "Get the allowed actions on a user",
		Long:  "Get the allowed actions on a user. By default, the current user's access is returned.",
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

			resp, err := c.Users.Actions(ctx, args[0], &client.GetUserActionsOptions{
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
				actions, err := userActionNames(resp.GetAllowedActions())
				if err != nil {
					return err
				}
				return printAllowedActions(cmd.OutOrStdout(), actions)
			default:
				return fmt.Errorf("unknown output format: %s", output)
			}
		},
	}

	addAccessFlags(cmd, &access)
	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format. One of: json|text")
	return cmd
}
