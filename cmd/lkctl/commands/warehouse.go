package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/client"
	"github.com/lakekeeper/go-lakekeeper/pkg/storage/profile"
)

func newWarehouseCmd(opts *clientOptions) *cobra.Command {
	var project string

	cmd := &cobra.Command{
		Use:     "warehouse",
		Aliases: []string{"wh"},
		Short:   "Manage warehouses",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	addProjectFlag(cmd, &project)

	cmd.AddCommand(newWarehouseListCmd(opts, &project))
	cmd.AddCommand(newWarehouseGetCmd(opts))
	cmd.AddCommand(newWarehouseCreateCmd(opts, &project))
	cmd.AddCommand(newWarehouseRenameCmd(opts))
	cmd.AddCommand(newWarehouseActivateCmd(opts))
	cmd.AddCommand(newWarehouseDeactivateCmd(opts))
	cmd.AddCommand(newWarehouseSetProtectionCmd(opts))
	cmd.AddCommand(newWarehouseStatisticsCmd(opts))
	cmd.AddCommand(newWarehouseDeleteCmd(opts))
	cmd.AddCommand(newWarehouseAccessCmd(opts))
	cmd.AddCommand(newWarehouseAssignmentsCmd(opts))
	cmd.AddCommand(newWarehouseGrantCmd(opts))
	cmd.AddCommand(newWarehouseRevokeCmd(opts))

	return cmd
}

func newWarehouseListCmd(opts *clientOptions, project *string) *cobra.Command {
	var (
		status []string
		output string
	)

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List warehouses",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}

			resp, err := c.Warehouses.List(ctx,
				&client.ListWarehousesOptions{Status: toEnums[managementv1.WarehouseStatus](status)},
				projectOption(cmd, *project)...)
			if err != nil {
				return err
			}

			switch output {
			case "json":
				return printJSON(cmd.OutOrStdout(), resp)
			case "text":
				return printWarehouses(cmd.OutOrStdout(), output, resp.GetWarehouses()...)
			default:
				return fmt.Errorf("unknown output format: %s", output)
			}
		},
	}

	cmd.Flags().StringSliceVar(&status, "status", nil, "Filter by status; repeat or comma-separate. One of: active|inactive")
	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format. One of: json|text")
	return cmd
}

func newWarehouseGetCmd(opts *clientOptions) *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "get WAREHOUSEID",
		Short: "Get a warehouse by id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}
			wh, err := c.Warehouses.Get(ctx, args[0])
			if err != nil {
				return err
			}
			switch output {
			case "json":
				return printJSON(cmd.OutOrStdout(), wh)
			case "text":
				return printWarehouses(cmd.OutOrStdout(), output, *wh)
			default:
				return fmt.Errorf("unknown output format: %s", output)
			}
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format. One of: json|text")
	return cmd
}

func newWarehouseCreateCmd(opts *clientOptions, project *string) *cobra.Command {
	var config string

	cmd := &cobra.Command{
		Use:   "create WAREHOUSENAME -f JSONCONFIGFILE",
		Short: "Create a warehouse from a JSON config file",
		Example: `  # From a file
  lkctl warehouse create "New Warehouse" -f warehouse-config.json

  # From stdin
  cat warehouse-config.json | lkctl warehouse create "New Warehouse" -f -`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var reader io.Reader
			if config == "-" {
				reader = cmd.InOrStdin()
			} else {
				f, err := os.Open(config)
				if err != nil {
					return fmt.Errorf("open config: %w", err)
				}
				defer f.Close()
				reader = f
			}

			var req managementv1.CreateWarehouseRequest
			if err := json.NewDecoder(reader).Decode(&req); err != nil {
				return fmt.Errorf("decode config: %w", err)
			}
			if req.GetWarehouseName() != args[0] {
				return errors.New("warehouse name in config does not match the name argument")
			}
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}
			// A project-id in the config is sent as the x-project-id header,
			// like --project; the façade rejects a config project that
			// contradicts --project.
			wh, err := c.Warehouses.Create(ctx, &req, projectOption(cmd, *project)...)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Warehouse %s created with id %s\n", wh.GetName(), wh.GetWarehouseId())
			return nil
		},
	}

	cmd.Flags().StringVarP(&config, "file", "f", "", "Warehouse config file (JSON), or '-' for stdin")
	if err := cmd.MarkFlagRequired("file"); err != nil {
		panic(err) // unreachable: the flag was just registered.
	}
	return cmd
}

func newWarehouseRenameCmd(opts *clientOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rename WAREHOUSEID NEW-NAME",
		Short: "Rename a warehouse",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}

			if _, err := c.Warehouses.Rename(ctx, args[0], managementv1.NewRenameWarehouseRequest(args[1])); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Warehouse %s renamed to %s\n", args[0], args[1])
			return nil
		},
	}
	return cmd
}

func newWarehouseActivateCmd(opts *clientOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "activate WAREHOUSEID",
		Short: "Activate a warehouse",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}
			if err := c.Warehouses.Activate(ctx, args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Warehouse %s activated\n", args[0])
			return nil
		},
	}
	return cmd
}

func newWarehouseDeactivateCmd(opts *clientOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deactivate WAREHOUSEID",
		Short: "Deactivate a warehouse",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}
			if err := c.Warehouses.Deactivate(ctx, args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Warehouse %s deactivated\n", args[0])
			return nil
		},
	}
	return cmd
}

func newWarehouseSetProtectionCmd(opts *clientOptions) *cobra.Command {
	var (
		protected bool
		output    string
	)

	cmd := &cobra.Command{
		Use:   "set-protection WAREHOUSEID --protected=true|false",
		Short: "Set protection on a warehouse",
		Long:  "Set protection on a warehouse. A protected warehouse cannot be deleted unless force is used.",
		Example: `  # Protect a warehouse
  lkctl warehouse set-protection 0198618c-5be8-7a82-a0b9-1076c9dd12f0 --protected=true

  # Unprotect a warehouse
  lkctl warehouse set-protection 0198618c-5be8-7a82-a0b9-1076c9dd12f0 --protected=false`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}

			resp, err := c.Warehouses.SetProtection(ctx, args[0], managementv1.NewSetProtectionRequest(protected))
			if err != nil {
				return err
			}

			switch output {
			case "json":
				return printJSON(cmd.OutOrStdout(), resp)
			case "text":
				fmt.Fprintf(cmd.OutOrStdout(), "Warehouse %s protection set to %t\n", args[0], resp.GetProtected())
				return nil
			default:
				return fmt.Errorf("unknown output format: %s", output)
			}
		},
	}

	cmd.Flags().BoolVar(&protected, "protected", false, "Whether the warehouse is protected from deletion")
	if err := cmd.MarkFlagRequired("protected"); err != nil {
		panic(err) // unreachable: the flag was just registered.
	}
	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format. One of: json|text")
	return cmd
}

func newWarehouseStatisticsCmd(opts *clientOptions) *cobra.Command {
	var (
		pageSize  int64
		pageToken string
		output    string
	)

	cmd := &cobra.Command{
		Use:   "statistics WAREHOUSEID",
		Short: "Get warehouse statistics",
		Long:  "Get warehouse statistics. Returns ordered statistics entries with table and view counts at each sample timestamp.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}

			resp, err := c.Warehouses.Statistics(ctx, args[0], &client.GetWarehouseStatisticsOptions{
				PageSize:  pageSize,
				PageToken: pageToken,
			})
			if err != nil {
				return err
			}

			switch output {
			case "json":
				return printJSON(cmd.OutOrStdout(), resp)
			case "text":
				if err := printWarehouseStatistics(cmd.OutOrStdout(), resp.GetStats()...); err != nil {
					return err
				}
				if next := resp.GetNextPageToken(); next != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "\nNext page token: %s\n", next)
				}
				return nil
			default:
				return fmt.Errorf("unknown output format: %s", output)
			}
		},
	}

	cmd.Flags().Int64Var(&pageSize, "page-size", 100, "Upper bound on the number of results returned to the client")
	cmd.Flags().StringVar(&pageToken, "page-token", "", "Pagination token")
	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format. One of: json|text")
	return cmd
}

func newWarehouseDeleteCmd(opts *clientOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete WAREHOUSEID",
		Aliases: []string{"rm"},
		Short:   "Delete a warehouse by id",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}
			if err := c.Warehouses.Delete(ctx, args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Warehouse %s deleted\n", args[0])
			return nil
		},
	}
	return cmd
}

func newWarehouseAccessCmd(opts *clientOptions) *cobra.Command {
	var (
		access accessOpts
		output string
	)

	cmd := &cobra.Command{
		Use:   "access WAREHOUSEID",
		Short: "Get warehouse access",
		Long:  "Get warehouse access. By default, the current user's access on the warehouse is returned.",
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

			resp, err := c.Permissions.Warehouses.Access(ctx, args[0], &client.GetWarehouseAccessByIDOptions{
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

func newWarehouseAssignmentsCmd(opts *clientOptions) *cobra.Command {
	var (
		assignments assignmentsOpts
		output      string
	)

	cmd := &cobra.Command{
		Use:   "assignments WAREHOUSEID",
		Short: "Get warehouse assignments",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}

			resp, err := c.Permissions.Warehouses.Assignments(ctx, args[0], &client.GetWarehouseAssignmentsByIDOptions{
				Relations: toEnums[managementv1.WarehouseRelation](assignments.relations),
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

func newWarehouseGrantCmd(opts *clientOptions) *cobra.Command {
	var (
		users       []string
		roles       []string
		assignments []string
	)

	cmd := &cobra.Command{
		Use:     "grant WAREHOUSEID",
		Aliases: []string{"assign"},
		Short:   "Add warehouse assignments",
		Example: `  # Grant ownership to a user
  lkctl warehouse grant 0198618c-5be8-7a82-a0b9-1076c9dd12f0 --users 11111111-2222-3333-4444-555555555555 --assignments ownership

  # Grant describe to a role
  lkctl warehouse grant 0198618c-5be8-7a82-a0b9-1076c9dd12f0 --roles 11111111-2222-3333-4444-555555555555 --assignments describe`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			set, err := buildAssignmentSet[managementv1.WarehouseAssignment](assignments, users, roles)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}
			if err := c.Permissions.Warehouses.Grant(ctx, args[0], set); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Warehouse permissions updated")
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

func newWarehouseRevokeCmd(opts *clientOptions) *cobra.Command {
	var (
		users       []string
		roles       []string
		assignments []string
	)

	cmd := &cobra.Command{
		Use:     "revoke WAREHOUSEID",
		Aliases: []string{"unassign"},
		Short:   "Remove warehouse assignments",
		Example: `  # Revoke ownership from a user
  lkctl warehouse revoke 0198618c-5be8-7a82-a0b9-1076c9dd12f0 --users 11111111-2222-3333-4444-555555555555 --assignments ownership

  # Revoke describe from a role
  lkctl warehouse revoke 0198618c-5be8-7a82-a0b9-1076c9dd12f0 --roles 11111111-2222-3333-4444-555555555555 --assignments describe`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			set, err := buildAssignmentSet[managementv1.WarehouseAssignment](assignments, users, roles)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}
			if err := c.Permissions.Warehouses.Revoke(ctx, args[0], set); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Warehouse permissions updated")
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

func printWarehouses(w io.Writer, output string, warehouses ...managementv1.GetWarehouseResponse) error {
	if len(warehouses) == 0 {
		_, err := fmt.Fprintln(w, "No warehouses available")
		return err
	}

	switch output {
	case "text":
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "ID\tNAME\tSTORAGE\tSTATUS\tPROJECT ID")
		for i := range warehouses {
			wh := &warehouses[i]
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
				wh.GetWarehouseId(), wh.GetName(), storageFamily(wh.GetStorageProfile()), wh.GetStatus(), wh.GetProjectId())
		}
		return tw.Flush()
	default:
		return fmt.Errorf("unknown output format: %s", output)
	}
}

func printWarehouseStatistics(w io.Writer, stats ...managementv1.WarehouseStatistics) error {
	if len(stats) == 0 {
		_, err := fmt.Fprintln(w, "No statistics available")
		return err
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "TIMESTAMP\tTABLES\tVIEWS\tUPDATED AT")
	for _, s := range stats {
		fmt.Fprintf(tw, "%s\t%d\t%d\t%s\n",
			s.GetTimestamp().Format(time.RFC3339), s.GetNumberOfTables(), s.GetNumberOfViews(), s.GetUpdatedAt().Format(time.RFC3339))
	}
	return tw.Flush()
}

// storageFamily returns "s3", "gcs", or "adls" depending on which variant of
// the StorageProfile union is populated. Falls back to "unknown".
func storageFamily(sp managementv1.StorageProfile) string {
	if _, ok := profile.AsS3(sp); ok {
		return "s3"
	}
	if _, ok := profile.AsGCS(sp); ok {
		return "gcs"
	}
	if _, ok := profile.AsADLS(sp); ok {
		return "adls"
	}
	return "unknown"
}
