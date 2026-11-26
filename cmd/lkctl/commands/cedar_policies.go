package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/client"
)

// cedarScope is what the project-policies and warehouse-policies command
// groups differ in: the scope's name, whether a warehouse id argument comes
// first, and the façade methods. id is the warehouse id, or "" for a project.
type cedarScope struct {
	use     string
	noun    string // "project" or "warehouse"
	idArg   string // positional argument naming the scope, or ""
	example string
	list    func(ctx context.Context, c *client.Client, id string, opts []client.CallOption) (*managementv1.ScopePoliciesResponse, error)
	get     func(ctx context.Context, c *client.Client, id, name string, opts []client.CallOption) (*managementv1.ScopePolicyEntry, error)
	apply   func(ctx context.Context, c *client.Client, id string, req *managementv1.ApplyScopePoliciesRequest, opts []client.CallOption) (*managementv1.ApplyScopePoliciesResponse, error)
}

// callOptions maps the -p/--project flag (project scope only) and
// --break-glass (where registered) to façade call options. A given
// --break-glass, even an empty one, becomes WithBreakGlass, so an empty
// reason fails in the façade instead of being dropped.
func (s *cedarScope) callOptions(cmd *cobra.Command, project *string, breakGlass string) []client.CallOption {
	var opts []client.CallOption
	if project != nil {
		opts = projectFlagOptions(cmd, *project)
	}
	if f := cmd.Flags().Lookup("break-glass"); f != nil && f.Changed {
		opts = append(opts, client.WithBreakGlass(breakGlass))
	}
	return opts
}

// args splits the positional arguments into the scope id and the rest.
func (s *cedarScope) args(args []string) (string, []string) {
	if s.idArg == "" {
		return "", args
	}
	return args[0], args[1:]
}

func (s *cedarScope) usage(verb, rest string) string {
	u := verb
	if s.idArg != "" {
		u += " " + s.idArg
	}
	if rest != "" {
		u += " " + rest
	}
	return u
}

func (s *cedarScope) nArgs(rest int) cobra.PositionalArgs {
	if s.idArg != "" {
		rest++
	}
	return cobra.ExactArgs(rest)
}

var projectCedarScope = &cedarScope{
	use:     "project-policies",
	noun:    "project",
	example: "lkctl cedar project-policies",
	list: func(ctx context.Context, c *client.Client, _ string, opts []client.CallOption) (*managementv1.ScopePoliciesResponse, error) {
		return c.Cedar.ListProjectPolicies(ctx, opts...)
	},
	get: func(ctx context.Context, c *client.Client, _, name string, opts []client.CallOption) (*managementv1.ScopePolicyEntry, error) {
		return c.Cedar.GetProjectPolicy(ctx, name, opts...)
	},
	apply: func(ctx context.Context, c *client.Client, _ string, req *managementv1.ApplyScopePoliciesRequest, opts []client.CallOption) (*managementv1.ApplyScopePoliciesResponse, error) {
		return c.Cedar.ApplyProjectPolicies(ctx, req, opts...)
	},
}

var warehouseCedarScope = &cedarScope{
	use:     "warehouse-policies",
	noun:    "warehouse",
	idArg:   "WAREHOUSEID",
	example: "lkctl cedar warehouse-policies apply 0198618c-5be8-7a82-a0b9-1076c9dd12f0",
	list: func(ctx context.Context, c *client.Client, id string, opts []client.CallOption) (*managementv1.ScopePoliciesResponse, error) {
		return c.Cedar.ListWarehousePolicies(ctx, id, opts...)
	},
	get: func(ctx context.Context, c *client.Client, id, name string, _ []client.CallOption) (*managementv1.ScopePolicyEntry, error) {
		return c.Cedar.GetWarehousePolicy(ctx, id, name)
	},
	apply: func(ctx context.Context, c *client.Client, id string, req *managementv1.ApplyScopePoliciesRequest, opts []client.CallOption) (*managementv1.ApplyScopePoliciesResponse, error) {
		return c.Cedar.ApplyWarehousePolicies(ctx, id, req, opts...)
	},
}

func newCedarScopeCmd(opts *clientOptions, s *cedarScope) *cobra.Command {
	short := lakekeeperPlus("Manage the Cedar policies of a " + s.noun)
	cmd := &cobra.Command{
		Use:   s.use,
		Short: short,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	// The warehouse names its project, so only the project scope has -p.
	var project *string
	if s.idArg == "" {
		project = new(string)
		addProjectFlag(cmd, project)
	}

	cmd.AddCommand(newCedarScopeListCmd(opts, s, project))
	cmd.AddCommand(newCedarScopeGetCmd(opts, s, project))
	cmd.AddCommand(newCedarScopeApplyCmd(opts, s, project))
	return cmd
}

func addBreakGlassFlag(cmd *cobra.Command, reason *string) {
	cmd.Flags().StringVar(reason, "break-glass", "",
		"Use emergency access (break-glass), recording this reason; sent as the x-break-glass header")
}

func newCedarScopeListCmd(opts *clientOptions, s *cedarScope, project *string) *cobra.Command {
	var output, breakGlass string

	cmd := &cobra.Command{
		Use:     s.usage("list", ""),
		Aliases: []string{"ls"},
		Short:   lakekeeperPlus("List the Cedar policies stored for a " + s.noun),
		Long: lakekeeperPlus("List the Cedar policies stored for a "+s.noun) + "\n\n" +
			"Text output ends with the scope version and whether you can change the policies.\n" +
			"With --break-glass, \"Can write\" says whether emergency access could change them.",
		Args: s.nArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, _ := s.args(args)
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}
			resp, err := s.list(ctx, c, id, s.callOptions(cmd, project, breakGlass))
			if err != nil {
				return err
			}

			switch output {
			case "json":
				return printJSON(cmd.OutOrStdout(), resp)
			case "text", "wide":
				return printScopePolicies(cmd.OutOrStdout(), output, resp)
			default:
				return fmt.Errorf("unknown output format: %s", output)
			}
		},
	}

	addBreakGlassFlag(cmd, &breakGlass)
	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format. One of: json|text|wide")
	return cmd
}

func newCedarScopeGetCmd(opts *clientOptions, s *cedarScope, project *string) *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   s.usage("get", "NAME"),
		Short: lakekeeperPlus("Get a Cedar policy stored for a " + s.noun + ", by name"),
		Long: lakekeeperPlus("Get a Cedar policy stored for a "+s.noun+", by name") + "\n\n" +
			"JSON output includes the policy itself, as Cedar's JSON policy format.",
		Args: s.nArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, rest := s.args(args)
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}
			policy, err := s.get(ctx, c, id, rest[0], s.callOptions(cmd, project, ""))
			if err != nil {
				return err
			}

			switch output {
			case "json":
				return printJSON(cmd.OutOrStdout(), policy)
			case "text", "wide":
				return printScopePolicyEntries(cmd.OutOrStdout(), output, *policy)
			default:
				return fmt.Errorf("unknown output format: %s", output)
			}
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format. One of: json|text|wide")
	return cmd
}

func newCedarScopeApplyCmd(opts *clientOptions, s *cedarScope, project *string) *cobra.Command {
	var file, output, breakGlass string

	cmd := &cobra.Command{
		Use:   s.usage("apply", "-f FILE"),
		Short: lakekeeperPlus("Create, update and delete the Cedar policies of a " + s.noun),
		Long: lakekeeperPlus("Create, update and delete the Cedar policies of a "+s.noun) + "\n\n" +
			"Reads an apply request (JSON) from a file, or from stdin with -f -, and applies it\n" +
			"in one transaction. The request has the fields of the Management API's apply\n" +
			"request: writes (policies in Cedar's JSON policy format), deletes, replace,\n" +
			"dry-run and if-scope-version.",
		Example: fmt.Sprintf(`  # From a file
  %[1]s apply -f policies.json

  # From stdin, previewing the changes
  jq '.["dry-run"] = true' policies.json | %[1]s apply -f -`, s.example),
		Args: s.nArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, _ := s.args(args)
			req, err := readApplyScopePoliciesRequest(cmd, file)
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}
			resp, err := s.apply(ctx, c, id, req, s.callOptions(cmd, project, breakGlass))
			if err != nil {
				return err
			}

			switch output {
			case "json":
				return printJSON(cmd.OutOrStdout(), resp)
			case "text":
				return printScopePolicyChanges(cmd.OutOrStdout(), resp)
			default:
				return fmt.Errorf("unknown output format: %s", output)
			}
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "Apply request file (JSON), or '-' for stdin")
	if err := cmd.MarkFlagRequired("file"); err != nil {
		panic(err) // unreachable: the flag was just registered.
	}
	addBreakGlassFlag(cmd, &breakGlass)
	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format. One of: json|text")
	return cmd
}

func readApplyScopePoliciesRequest(cmd *cobra.Command, file string) (*managementv1.ApplyScopePoliciesRequest, error) {
	var reader io.Reader
	if file == "-" {
		reader = cmd.InOrStdin()
	} else {
		f, err := os.Open(file)
		if err != nil {
			return nil, fmt.Errorf("open apply request: %w", err)
		}
		defer f.Close()
		reader = f
	}
	var req managementv1.ApplyScopePoliciesRequest
	if err := json.NewDecoder(reader).Decode(&req); err != nil {
		return nil, fmt.Errorf("decode apply request: %w", err)
	}
	return &req, nil
}

func newCedarBreakGlassStatusCmd(opts *clientOptions) *cobra.Command {
	var project, output string

	cmd := &cobra.Command{
		Use:   "break-glass-status",
		Short: lakekeeperPlus("Report whether break-glass is available to you for a project"),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}
			status, err := c.Cedar.BreakGlassStatus(ctx, projectFlagOptions(cmd, project)...)
			if err != nil {
				return err
			}

			switch output {
			case "json":
				return printJSON(cmd.OutOrStdout(), status)
			case "text":
				_, err := fmt.Fprintf(cmd.OutOrStdout(), "Break-glass available: %t\n", status.GetBreakGlassAvailable())
				return err
			default:
				return fmt.Errorf("unknown output format: %s", output)
			}
		},
	}

	addProjectFlag(cmd, &project)
	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format. One of: json|text")
	return cmd
}

// printScopePolicies prints the policies as a table (or "No Cedar policies"),
// then the scope versions and whether the caller can change them.
func printScopePolicies(w io.Writer, output string, resp *managementv1.ScopePoliciesResponse) error {
	if len(resp.GetPolicies()) == 0 {
		fmt.Fprintln(w, "No Cedar policies")
	} else if err := printScopePolicyEntries(w, output, resp.GetPolicies()...); err != nil {
		return err
	}
	fmt.Fprintf(w, "\nScope version: %d\n", resp.GetScopeVersion())
	if v, ok := resp.GetProjectScopeVersionOk(); ok && v != nil {
		fmt.Fprintf(w, "Project scope version: %d\n", *v)
	}
	_, err := fmt.Fprintf(w, "Can write: %t\n", resp.GetCanWrite())
	return err
}

func printScopePolicyEntries(w io.Writer, output string, entries ...managementv1.ScopePolicyEntry) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if output == "wide" {
		fmt.Fprintln(tw, "ID\tNAME\tCREATED AT\tUPDATED AT\tDESCRIPTION")
	} else {
		fmt.Fprintln(tw, "ID\tNAME\tCREATED AT\tUPDATED AT")
	}
	for i := range entries {
		e := &entries[i]
		updatedAt, _ := e.GetUpdatedAtOk()
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s", e.GetId(), e.GetName(),
			e.GetCreatedAt().Format(time.RFC3339), formatTimePtr(updatedAt))
		if output == "wide" {
			description, _ := e.GetDescriptionOk()
			fmt.Fprintf(tw, "\t%s", formatStringPtr(description))
		}
		fmt.Fprintln(tw)
	}
	return tw.Flush()
}

// printScopePolicyChanges prints the changes as a table (or "No changes"),
// then the scope version after the apply.
func printScopePolicyChanges(w io.Writer, resp *managementv1.ApplyScopePoliciesResponse) error {
	if len(resp.GetChanged()) == 0 {
		fmt.Fprintln(w, "No changes")
	} else {
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "OP\tNAME\tID")
		for _, ch := range resp.GetChanged() {
			fmt.Fprintf(tw, "%s\t%s\t%s\n", ch.GetOp(), ch.GetName(), ch.GetId())
		}
		if err := tw.Flush(); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(w, "\nScope version: %d\n", resp.GetScopeVersion())
	return err
}
