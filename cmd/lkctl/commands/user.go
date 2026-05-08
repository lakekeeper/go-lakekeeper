package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
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

			req := c.UserAPI.ListUser(ctx).PageSize(listOpts.limit)
			if listOpts.token != "" {
				req = req.PageToken(listOpts.token)
			}
			if listOpts.name != "" {
				req = req.Name(listOpts.name)
			}

			resp, _, err := req.Execute()
			if err != nil {
				return wrapAPIError("list users", err)
			}

			switch output {
			case "json":
				return printJSON(cmd.OutOrStdout(), resp)
			case "text", "wide":
				users := make([]*managementv1.User, len(resp.Users))
				for i := range resp.Users {
					users[i] = &resp.Users[i]
				}
				var nextToken *string
				if resp.NextPageToken.IsSet() {
					nextToken = resp.NextPageToken.Get()
				}
				return printUsers(cmd.OutOrStdout(), output, nextToken, users...)
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

			user, _, err := c.UserAPI.GetUser(ctx, args[0]).Execute()
			if err != nil {
				return wrapAPIError("get user", err)
			}

			switch output {
			case "json":
				return printJSON(cmd.OutOrStdout(), user)
			case "text", "wide":
				return printUsers(cmd.OutOrStdout(), output, nil, user)
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

			if _, err := c.UserAPI.DeleteUser(ctx, args[0]).Execute(); err != nil {
				return wrapAPIError("delete user", err)
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

			user, _, err := c.UserAPI.CreateUser(ctx).CreateUserRequest(*req).Execute()
			if err != nil {
				return wrapAPIError("create user", err)
			}

			switch output {
			case "text":
				fmt.Fprintf(cmd.OutOrStdout(), "User %s registered with id %s\n", args[1], user.Id)
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
