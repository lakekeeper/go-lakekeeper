package commands

import (
	"errors"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/spf13/cobra"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

func newServerCmd(opts *clientOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "server",
		Aliases: []string{"srv"},
		Short:   "Manage server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newServerInfoCmd(opts))
	cmd.AddCommand(newServerBootstrapCmd(opts))
	cmd.AddCommand(newServerAccessCmd(opts))
	cmd.AddCommand(newServerAssignmentsCmd(opts))
	cmd.AddCommand(newServerGrantCmd(opts))
	cmd.AddCommand(newServerRevokeCmd(opts))

	return cmd
}

func newServerInfoCmd(opts *clientOptions) *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "info",
		Short: "Print server information",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}
			info, _, err := c.ServerAPI.GetServerInfo(ctx).Execute()
			if err != nil {
				return wrapAPIError("get server info", err)
			}

			switch output {
			case "json":
				return printJSON(cmd.OutOrStdout(), info)
			case "text":
				return printServerInfo(cmd.OutOrStdout(), info)
			default:
				return fmt.Errorf("unknown output format: %s", output)
			}
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format. One of: json|text")
	return cmd
}

func newServerBootstrapCmd(opts *clientOptions) *cobra.Command {
	var (
		asOperator       bool
		acceptTermsOfUse bool
		output           string
	)

	cmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "Bootstrap the server with the current user",
		Example: `  # Bootstrap and get the server admin role
  lkctl server bootstrap --accept-terms-of-use

  # Bootstrap as an operator
  lkctl server bootstrap --accept-terms-of-use --as-operator`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}

			req := managementv1.NewBootstrapRequest(acceptTermsOfUse)
			req.IsOperator = &asOperator

			if _, err := c.ServerAPI.Bootstrap(ctx).BootstrapRequest(*req).Execute(); err != nil {
				return wrapAPIError("bootstrap", err)
			}

			switch output {
			case "json":
				info, _, err := c.ServerAPI.GetServerInfo(ctx).Execute()
				if err != nil {
					return wrapAPIError("get server info", err)
				}
				return printJSON(cmd.OutOrStdout(), info)
			case "text":
				fmt.Fprintln(cmd.OutOrStdout(), "Server bootstrapped successfully")
				return nil
			default:
				return fmt.Errorf("unknown output format: %s", output)
			}
		},
	}

	cmd.Flags().BoolVar(&asOperator, "as-operator", false, "Bootstrap the server as an operator")
	cmd.Flags().BoolVar(&acceptTermsOfUse, "accept-terms-of-use", false, "Accept the terms of use")
	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format. One of: json|text")
	return cmd
}

func newServerAccessCmd(opts *clientOptions) *cobra.Command {
	var (
		access accessOpts
		output string
	)

	cmd := &cobra.Command{
		Use:   "access",
		Short: "Get server access",
		Long:  "Get server-level access. By default, the current user's access is returned.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if access.user != "" && access.role != "" {
				return errors.New("--user and --role are mutually exclusive")
			}
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}

			req := c.PermissionsOpenfgaAPI.GetServerAccess(ctx)
			if access.user != "" {
				req = req.PrincipalUser(access.user)
			}
			if access.role != "" {
				req = req.PrincipalRole(access.role)
			}
			resp, _, err := req.Execute()
			if err != nil {
				return wrapAPIError("get server access", err)
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

func newServerAssignmentsCmd(opts *clientOptions) *cobra.Command {
	var (
		assignments assignmentsOpts
		output      string
	)

	cmd := &cobra.Command{
		Use:   "assignments",
		Short: "Get server assignments",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}

			req := c.PermissionsOpenfgaAPI.GetServerAssignments(ctx)
			if len(assignments.relations) > 0 {
				rels := make([]managementv1.ServerRelation, 0, len(assignments.relations))
				for _, r := range assignments.relations {
					rels = append(rels, managementv1.ServerRelation(r))
				}
				req = req.Relations(rels)
			}
			resp, _, err := req.Execute()
			if err != nil {
				return wrapAPIError("get server assignments", err)
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

func newServerGrantCmd(opts *clientOptions) *cobra.Command {
	var (
		users       []string
		roles       []string
		assignments []string
	)

	cmd := &cobra.Command{
		Use:     "grant",
		Aliases: []string{"assign"},
		Short:   "Add server assignments",
		Example: `  # Grant admin to a user
  lkctl server grant --users 0198618c-5be8-7a82-a0b9-1076c9dd12f0 --assignments admin

  # Grant operator to a role
  lkctl server grant --roles 0198618c-5be8-7a82-a0b9-1076c9dd12f0 --assignments operator`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			set, err := buildAssignmentSet[managementv1.ServerAssignment](assignments, users, roles)
			if err != nil {
				return err
			}
			req := managementv1.NewUpdateServerAssignmentsRequest()
			req.Writes = set

			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}
			if _, err := c.PermissionsOpenfgaAPI.UpdateServerAssignments(ctx).UpdateServerAssignmentsRequest(*req).Execute(); err != nil {
				return wrapAPIError("update server assignments", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Server permissions updated")
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

func newServerRevokeCmd(opts *clientOptions) *cobra.Command {
	var (
		users       []string
		roles       []string
		assignments []string
	)

	cmd := &cobra.Command{
		Use:     "revoke",
		Aliases: []string{"unassign"},
		Short:   "Remove server assignments",
		Example: `  # Revoke admin from a user
  lkctl server revoke --users 0198618c-5be8-7a82-a0b9-1076c9dd12f0 --assignments admin

  # Revoke operator from a role
  lkctl server revoke --roles 0198618c-5be8-7a82-a0b9-1076c9dd12f0 --assignments operator`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			set, err := buildAssignmentSet[managementv1.ServerAssignment](assignments, users, roles)
			if err != nil {
				return err
			}
			req := managementv1.NewUpdateServerAssignmentsRequest()
			req.Deletes = set

			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}
			if _, err := c.PermissionsOpenfgaAPI.UpdateServerAssignments(ctx).UpdateServerAssignmentsRequest(*req).Execute(); err != nil {
				return wrapAPIError("update server assignments", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Server permissions updated")
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

func printServerInfo(w io.Writer, info *managementv1.ServerInfo) error {
	fmt.Fprintf(w, "ID: %s\n", info.ServerId)
	fmt.Fprintf(w, "Version: %s\n", info.Version)
	fmt.Fprintf(w, "Lakekeeper Version: %s\n", info.GetLakekeeperVersion())
	fmt.Fprintf(w, "Default Project ID: %s\n", formatStringPtr(info.DefaultProjectId))
	fmt.Fprintf(w, "Bootstrapped: %t\n", info.Bootstrapped)
	fmt.Fprintf(w, "Authorization Backend: %s\n", info.AuthzBackend)
	fmt.Fprintf(w, "AWS System Identities Enabled: %t\n", info.AwsSystemIdentitiesEnabled)
	fmt.Fprintf(w, "Azure System Identities Enabled: %t\n", info.AzureSystemIdentitiesEnabled)
	fmt.Fprintf(w, "GCP System Identities Enabled: %t\n", info.GcpSystemIdentitiesEnabled)
	fmt.Fprintln(w, "Queues:")
	for _, q := range info.Queues {
		fmt.Fprintf(w, "  %s\n", q)
	}
	return nil
}

func printAllowedActions[T ~string](w io.Writer, actions []T) error {
	if len(actions) == 0 {
		fmt.Fprintln(w, "No access")
		return nil
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ALLOWED ACTIONS")
	for _, a := range actions {
		fmt.Fprintln(tw, string(a))
	}
	return tw.Flush()
}
