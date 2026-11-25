package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/spf13/cobra"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
	"github.com/lakekeeper/go-lakekeeper/pkg/client"
)

// predefinedScope is what the project-predefined-policies and
// warehouse-predefined-policies command groups differ in. The embedded
// cedarScope supplies the scope's name, id argument and option mapping; its
// policy functions are unused.
type predefinedScope struct {
	cedarScope
	list  func(ctx context.Context, c *client.Client, id string, o *client.PredefinedPoliciesOptions, opts []client.CallOption) (*managementv1.PredefinedPoliciesResponse, error)
	set   func(ctx context.Context, c *client.Client, id string, changes []managementv1.PredefinedPolicyChange, o *client.PredefinedPoliciesOptions, opts []client.CallOption) (*managementv1.PredefinedPoliciesResponse, error)
	reset func(ctx context.Context, c *client.Client, id string, opts []client.CallOption) error
}

var projectPredefinedScope = &predefinedScope{
	cedarScope: cedarScope{
		use:     "project-predefined-policies",
		noun:    "project",
		example: "lkctl cedar project-predefined-policies",
	},
	list: func(ctx context.Context, c *client.Client, _ string, o *client.PredefinedPoliciesOptions, opts []client.CallOption) (*managementv1.PredefinedPoliciesResponse, error) {
		return c.Cedar.ListProjectPredefinedPolicies(ctx, o, opts...)
	},
	set: func(ctx context.Context, c *client.Client, _ string, changes []managementv1.PredefinedPolicyChange, o *client.PredefinedPoliciesOptions, opts []client.CallOption) (*managementv1.PredefinedPoliciesResponse, error) {
		return c.Cedar.SetProjectPredefinedPolicies(ctx, changes, o, opts...)
	},
	reset: func(ctx context.Context, c *client.Client, _ string, opts []client.CallOption) error {
		return c.Cedar.ResetProjectPredefinedPolicies(ctx, opts...)
	},
}

var warehousePredefinedScope = &predefinedScope{
	cedarScope: cedarScope{
		use:     "warehouse-predefined-policies",
		noun:    "warehouse",
		idArg:   "WAREHOUSEID",
		example: "lkctl cedar warehouse-predefined-policies set 0198618c-5be8-7a82-a0b9-1076c9dd12f0",
	},
	list: func(ctx context.Context, c *client.Client, id string, o *client.PredefinedPoliciesOptions, opts []client.CallOption) (*managementv1.PredefinedPoliciesResponse, error) {
		return c.Cedar.ListWarehousePredefinedPolicies(ctx, id, o, opts...)
	},
	set: func(ctx context.Context, c *client.Client, id string, changes []managementv1.PredefinedPolicyChange, o *client.PredefinedPoliciesOptions, opts []client.CallOption) (*managementv1.PredefinedPoliciesResponse, error) {
		return c.Cedar.SetWarehousePredefinedPolicies(ctx, id, changes, o, opts...)
	},
	reset: func(ctx context.Context, c *client.Client, id string, _ []client.CallOption) error {
		return c.Cedar.ResetWarehousePredefinedPolicies(ctx, id)
	},
}

func newCedarPredefinedCmd(opts *clientOptions, s *predefinedScope) *cobra.Command {
	cmd := &cobra.Command{
		Use:   s.use,
		Short: lakekeeperPlus("Manage the predefined Cedar policies of a " + s.noun),
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

	cmd.AddCommand(newCedarPredefinedListCmd(opts, s, project))
	cmd.AddCommand(newCedarPredefinedSetCmd(opts, s, project))
	cmd.AddCommand(newCedarPredefinedResetCmd(opts, s, project))
	return cmd
}

func addIncludePolicyFlag(cmd *cobra.Command, include *bool) {
	cmd.Flags().BoolVar(include, "include-policy", false,
		"Include each policy's rule (Cedar's JSON policy format) in JSON output")
}

func newCedarPredefinedListCmd(opts *clientOptions, s *predefinedScope, project *string) *cobra.Command {
	var output, breakGlass string
	var includePolicy bool

	short := "List the predefined Cedar policies of a " + s.noun + " with their enablement"
	cmd := &cobra.Command{
		Use:     s.usage("list", ""),
		Aliases: []string{"ls"},
		Short:   lakekeeperPlus(short),
		Long: lakekeeperPlus(short) + "\n\n" +
			"Text output ends with whether the " + s.noun + " has decided anything itself (curated)\n" +
			"and whether the server evaluates predefined policies at all. With --break-glass,\n" +
			"CAN TOGGLE says whether emergency access could switch each policy.",
		Args: s.nArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, _ := s.args(args)
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}
			resp, err := s.list(ctx, c, id, &client.PredefinedPoliciesOptions{IncludePolicy: includePolicy},
				s.callOptions(cmd, project, breakGlass))
			if err != nil {
				return err
			}
			return printPredefinedPoliciesAs(cmd.OutOrStdout(), output, resp)
		},
	}

	addBreakGlassFlag(cmd, &breakGlass)
	addIncludePolicyFlag(cmd, &includePolicy)
	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format. One of: json|text|wide")
	return cmd
}

func newCedarPredefinedSetCmd(opts *clientOptions, s *predefinedScope, project *string) *cobra.Command {
	var output, breakGlass string
	var enable, disable []string
	var includePolicy bool

	short := "Enable or disable predefined Cedar policies of a " + s.noun
	cmd := &cobra.Command{
		Use:   s.usage("set", "[--enable ID]... [--disable ID]..."),
		Short: lakekeeperPlus(short),
		Long: lakekeeperPlus(short) + "\n\n" +
			"Sends the changes in one request and prints the " + s.noun + "'s listing as of the write.\n" +
			"Policies not named keep their state. The first change curates the " + s.noun + ".",
		Example: fmt.Sprintf(`  # Disable one policy, enable two
  %[1]s set --disable predefined-grants-table-select --enable predefined-a,predefined-b`, s.example),
		Args: s.nArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, _ := s.args(args)
			changes := predefinedPolicyChanges(enable, disable)
			if len(changes) == 0 {
				return errors.New("name at least one policy with --enable or --disable")
			}
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}
			resp, err := s.set(ctx, c, id, changes, &client.PredefinedPoliciesOptions{IncludePolicy: includePolicy},
				s.callOptions(cmd, project, breakGlass))
			if err != nil {
				return err
			}
			return printPredefinedPoliciesAs(cmd.OutOrStdout(), output, resp)
		},
	}

	cmd.Flags().StringSliceVar(&enable, "enable", nil, "Predefined policy ids to enable (repeatable, comma-separated)")
	cmd.Flags().StringSliceVar(&disable, "disable", nil, "Predefined policy ids to disable (repeatable, comma-separated)")
	addBreakGlassFlag(cmd, &breakGlass)
	addIncludePolicyFlag(cmd, &includePolicy)
	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format. One of: json|text|wide")
	return cmd
}

func newCedarPredefinedResetCmd(opts *clientOptions, s *predefinedScope, project *string) *cobra.Command {
	var follows string
	if s.idArg == "" {
		follows = "the provided defaults"
	} else {
		follows = "its project"
	}
	short := "Reset the predefined Cedar policies of a " + s.noun + " to " + follows

	cmd := &cobra.Command{
		Use:   s.usage("reset", ""),
		Short: lakekeeperPlus(short),
		Long: lakekeeperPlus(short) + "\n\n" +
			"Drops every decision the " + s.noun + " has recorded about predefined policies.",
		Args: s.nArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, _ := s.args(args)
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}
			if err := s.reset(ctx, c, id, s.callOptions(cmd, project, "")); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Predefined policies of %s reset\n", s.target(cmd, id, project))
			return err
		},
	}
	return cmd
}

// target names the scope a command acted on: the warehouse, the project
// given with -p, or "the project" (the server's default).
func (s *predefinedScope) target(cmd *cobra.Command, id string, project *string) string {
	switch {
	case s.idArg != "":
		return s.noun + " " + id
	case project != nil && cmd.Flags().Changed("project"):
		return "project " + *project
	default:
		return "the project"
	}
}

// predefinedPolicyChanges turns the --enable and --disable ids into changes,
// enables first, each in the order given.
func predefinedPolicyChanges(enable, disable []string) []managementv1.PredefinedPolicyChange {
	changes := make([]managementv1.PredefinedPolicyChange, 0, len(enable)+len(disable))
	for _, id := range enable {
		changes = append(changes, managementv1.PredefinedPolicyChange{Id: id, Enabled: true})
	}
	for _, id := range disable {
		changes = append(changes, managementv1.PredefinedPolicyChange{Id: id, Enabled: false})
	}
	return changes
}

func printPredefinedPoliciesAs(w io.Writer, output string, resp *managementv1.PredefinedPoliciesResponse) error {
	switch output {
	case "json":
		return printJSON(w, resp)
	case "text", "wide":
		return printPredefinedPolicies(w, output, resp)
	default:
		return fmt.Errorf("unknown output format: %s", output)
	}
}

// printPredefinedPolicies prints the policies as a table (or "No predefined
// policies"), then whether the scope is curated and whether the server
// evaluates predefined policies.
func printPredefinedPolicies(w io.Writer, output string, resp *managementv1.PredefinedPoliciesResponse) error {
	if len(resp.GetPolicies()) == 0 {
		fmt.Fprintln(w, "No predefined policies")
	} else {
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		if output == "wide" {
			fmt.Fprintln(tw, "ID\tENABLED\tDISABLED BY\tCAN TOGGLE\tUPDATED AT\tUPDATED BY\tDESCRIPTION")
		} else {
			fmt.Fprintln(tw, "ID\tENABLED\tDISABLED BY\tCAN TOGGLE")
		}
		for _, p := range resp.GetPolicies() {
			fmt.Fprintf(tw, "%s\t%t\t%s\t%t", p.GetId(), p.GetEnabled(), disabledBy(&p), p.GetCanToggle())
			if output == "wide" {
				updatedAt, _ := p.GetUpdatedAtOk()
				updatedBy, _ := p.GetUpdatedByOk()
				fmt.Fprintf(tw, "\t%s\t%s\t%s", formatTimePtr(updatedAt), formatStringPtr(updatedBy), p.GetDescription())
			}
			fmt.Fprintln(tw)
		}
		if err := tw.Flush(); err != nil {
			return err
		}
	}
	fmt.Fprintf(w, "\nCurated: %t\n", resp.GetCurated())
	_, err := fmt.Fprintf(w, "Predefined policies enabled: %t\n", resp.GetPredefinedPoliciesEnabled())
	return err
}

// disabledBy is why a policy is not in force: "(retired)" for a retired
// row, "-" when it is in force.
func disabledBy(p *managementv1.PredefinedPolicyStatus) string {
	if p.GetRetired() {
		return "(retired)"
	}
	if by, ok := p.GetDisabledByOk(); ok && by != nil {
		return string(*by)
	}
	return "-"
}
