package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
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

	cmd.PersistentFlags().StringVarP(&project, "project", "p", uuid.Nil.String(), "Select a project")

	cmd.AddCommand(newWarehouseListCmd(opts, &project))
	cmd.AddCommand(newWarehouseGetCmd(opts))
	cmd.AddCommand(newWarehouseCreateCmd(opts, &project))
	cmd.AddCommand(newWarehouseDeleteCmd(opts))

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

			req := c.WarehouseAPI.ListWarehouses(ctx).ProjectId(*project)
			if len(status) > 0 {
				vals := make([]managementv1.WarehouseStatus, 0, len(status))
				for _, s := range status {
					vals = append(vals, managementv1.WarehouseStatus(s))
				}
				req = req.WarehouseStatus(vals)
			}
			resp, _, err := req.Execute()
			if err != nil {
				return fmt.Errorf("list warehouses: %w", err)
			}

			switch output {
			case "json":
				return printJSON(cmd.OutOrStdout(), resp)
			case "text":
				return printWarehouses(cmd.OutOrStdout(), output, resp.Warehouses...)
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
			wh, _, err := c.WarehouseAPI.GetWarehouse(ctx, args[0]).Execute()
			if err != nil {
				return fmt.Errorf("get warehouse: %w", err)
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
			if req.WarehouseName != args[0] {
				return errors.New("warehouse name in config does not match the name argument")
			}

			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}
			if !req.ProjectId.IsSet() {
				req.SetProjectId(*project)
			} else if pid := req.ProjectId.Get(); pid != nil && *pid != *project {
				return errors.New("project id in config does not match --project")
			}
			wh, _, err := c.WarehouseAPI.CreateWarehouse(ctx).CreateWarehouseRequest(req).Execute()
			if err != nil {
				return fmt.Errorf("create warehouse: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Warehouse %s created with id %s\n", wh.Name, wh.WarehouseId)
			return nil
		},
	}

	cmd.Flags().StringVarP(&config, "file", "f", "", "Warehouse config file (JSON), or '-' for stdin")
	if err := cmd.MarkFlagRequired("file"); err != nil {
		panic(err) // unreachable: the flag was just registered.
	}
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
			if _, err := c.WarehouseAPI.DeleteWarehouse(ctx, args[0]).Execute(); err != nil {
				return fmt.Errorf("delete warehouse: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Warehouse %s deleted\n", args[0])
			return nil
		},
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
				wh.WarehouseId, wh.Name, storageFamily(wh.StorageProfile), wh.Status, wh.ProjectId)
		}
		return tw.Flush()
	default:
		return fmt.Errorf("unknown output format: %s", output)
	}
}

// storageFamily returns "s3", "gcs", or "adls" depending on which variant of
// the StorageProfile union is populated. Falls back to "unknown".
func storageFamily(sp managementv1.StorageProfile) string {
	switch {
	case sp.StorageProfileS3 != nil:
		return "s3"
	case sp.StorageProfileGcs != nil:
		return "gcs"
	case sp.StorageProfileAdls != nil:
		return "adls"
	default:
		return "unknown"
	}
}
