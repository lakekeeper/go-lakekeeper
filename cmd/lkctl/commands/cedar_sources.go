package commands

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	managementv1 "github.com/lakekeeper/go-lakekeeper/pkg/apis/management/v1"
)

// cedarActionType is the entity type of Cedar actions. Every resolve
// result carries all of them, so text output leaves them out.
const cedarActionType = "Lakekeeper::Action"

func newCedarPolicySourcesCmd(opts *clientOptions) *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "policy-sources",
		Short: lakekeeperPlus("List the sources of the server's Cedar policies"),
		Long: lakekeeperPlus("List the sources of the server's Cedar policies") + "\n\n" +
			"Lists the files and Kubernetes ConfigMaps the server loads its Cedar policies from,\n" +
			"with each source's id, for `lkctl cedar server-policies list --source`.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}
			resp, err := c.Cedar.PolicySources(ctx)
			if err != nil {
				return err
			}

			switch output {
			case "json":
				return printJSON(cmd.OutOrStdout(), resp)
			case "text":
				return printCedarPolicySources(cmd.OutOrStdout(), resp.GetPolicySources())
			default:
				return fmt.Errorf("unknown output format: %s", output)
			}
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format. One of: json|text")
	return cmd
}

func newCedarEntitySourcesCmd(opts *clientOptions) *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "entity-sources",
		Short: lakekeeperPlus("List the sources of the server's Cedar entities"),
		Long: lakekeeperPlus("List the sources of the server's Cedar entities") + "\n\n" +
			"Lists the files and Kubernetes ConfigMaps the server loads additional Cedar entities from.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}
			resp, err := c.Cedar.EntitySources(ctx)
			if err != nil {
				return err
			}

			switch output {
			case "json":
				return printJSON(cmd.OutOrStdout(), resp)
			case "text":
				return printCedarEntitySources(cmd.OutOrStdout(), resp.GetEntitySources())
			default:
				return fmt.Errorf("unknown output format: %s", output)
			}
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format. One of: json|text")
	return cmd
}

func newCedarServerPoliciesCmd(opts *clientOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server-policies",
		Short: lakekeeperPlus("Inspect the server's Cedar policies"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newCedarServerPoliciesListCmd(opts))
	return cmd
}

func newCedarServerPoliciesListCmd(opts *clientOptions) *cobra.Command {
	var (
		output, effect, pageToken string
		sources                   []string
		pageSize                  int32
	)

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   lakekeeperPlus("List the server's active Cedar policies"),
		Long: lakekeeperPlus("List the server's active Cedar policies") + "\n\n" +
			"Without --source the list includes the predefined policies. It never includes the\n" +
			"policies stored for projects and warehouses; see project-policies and warehouse-policies.\n" +
			"Text output shows each policy's scope; -o json prints the policies in Cedar's JSON\n" +
			"policy format, conditions included. When more policies follow, text output ends\n" +
			"with the token for --page-token.",
		Example: "  lkctl cedar server-policies list --source $(lkctl cedar policy-sources -o json | jq -r '.\"policy-sources\"[0].\"source-id\"')",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			query := managementv1.ListCedarPoliciesQuery{SourceFilter: sources}
			switch strings.ToLower(effect) {
			case "":
			case "permit":
				query.SetEffectFilter(managementv1.CEDAREFFECTFILTER_PERMIT)
			case "forbid":
				query.SetEffectFilter(managementv1.CEDAREFFECTFILTER_FORBID)
			default:
				return fmt.Errorf("invalid --effect %q: want permit or forbid", effect)
			}
			if cmd.Flags().Changed("page-size") {
				query.SetPageSize(pageSize)
			}
			if pageToken != "" {
				query.SetPageToken(pageToken)
			}

			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}
			resp, err := c.Cedar.ListServerPolicies(ctx, &query)
			if err != nil {
				return err
			}

			switch output {
			case "json":
				return printJSON(cmd.OutOrStdout(), resp)
			case "text":
				return printCedarPolicies(cmd.OutOrStdout(), resp)
			default:
				return fmt.Errorf("unknown output format: %s", output)
			}
		},
	}

	cmd.Flags().StringArrayVar(&sources, "source", nil,
		"Only policies from this source id (from policy-sources); repeat for several")
	cmd.Flags().StringVar(&effect, "effect", "", "Only policies with this effect: permit or forbid")
	cmd.Flags().Int32Var(&pageSize, "page-size", 0, "Maximum number of policies to return (server default if unset)")
	cmd.Flags().StringVar(&pageToken, "page-token", "", "Page token from a previous list, to get the next page")
	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format. One of: json|text")
	return cmd
}

func newCedarResolveEntitiesCmd(opts *clientOptions) *cobra.Command {
	var output, user, role string

	cmd := &cobra.Command{
		Use:   "resolve-entities server | project [PROJECTID] | warehouse WAREHOUSEID",
		Short: lakekeeperPlus("Show the Cedar entities the server derives for a resource"),
		Long: lakekeeperPlus("Show the Cedar entities the server derives for a resource") + "\n\n" +
			"Resolves the Cedar entities an authorization decision on the resource would use,\n" +
			"for you or, with --user or --role, for another principal.\n" +
			"Without a PROJECTID the server chooses the project: this operation takes no\n" +
			"-p/--project, because the API gives it no project header.\n" +
			"Text output leaves out the action entities, which every result carries;\n" +
			"-o wide shows them, -o json prints the entities with their attributes.",
		Example: "  lkctl cedar resolve-entities server\n" +
			"  lkctl cedar resolve-entities warehouse 0198618c-5be8-7a82-a0b9-1076c9dd12f0 --user oidc~<subject>",
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			resource, err := cedarResolveResource(args)
			if err != nil {
				return err
			}
			req := managementv1.NewResolveCedarEntitiesRequest(resource)
			switch {
			case user != "":
				req.SetPrincipal(managementv1.UserOrRole{User: &user})
			case role != "":
				req.SetPrincipal(managementv1.UserOrRole{Role: &role})
			}

			ctx := cmd.Context()
			c, err := newClient(ctx, opts)
			if err != nil {
				return err
			}
			resp, err := c.Cedar.ResolveEntities(ctx, req)
			if err != nil {
				return err
			}

			switch output {
			case "json":
				return printJSON(cmd.OutOrStdout(), resp)
			case "text", "wide":
				return printCedarEntities(cmd.OutOrStdout(), output == "wide", resp)
			default:
				return fmt.Errorf("unknown output format: %s", output)
			}
		},
	}

	cmd.Flags().StringVar(&user, "user", "", "Resolve for this user id instead of yourself")
	cmd.Flags().StringVar(&role, "role", "", "Resolve for this role id instead of yourself")
	cmd.MarkFlagsMutuallyExclusive("user", "role")
	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format. One of: json|text|wide")
	return cmd
}

// cedarResolveResource maps the resolve-entities arguments to the request's
// resource.
func cedarResolveResource(args []string) (managementv1.CedarResolveResource, error) {
	if len(args) == 0 {
		return managementv1.CedarResolveResource{},
			errors.New("requires a resource: server, project [PROJECTID] or warehouse WAREHOUSEID")
	}
	kind, ids := args[0], args[1:]
	if len(ids) > 1 {
		return managementv1.CedarResolveResource{}, fmt.Errorf("%s takes at most one id", kind)
	}
	switch kind {
	case "server":
		if len(ids) > 0 {
			return managementv1.CedarResolveResource{}, errors.New("server takes no id")
		}
		server := managementv1.CEDARRESOLVERESOURCESERVER_SERVER
		return managementv1.CedarResolveResourceServerAsCedarResolveResource(&server), nil
	case "project":
		project := managementv1.NewCedarResolveResourceProjectProject()
		if len(ids) == 1 {
			project.SetProjectId(ids[0])
		}
		return managementv1.CedarResolveResourceProjectAsCedarResolveResource(
			managementv1.NewCedarResolveResourceProject(*project)), nil
	case "warehouse":
		if len(ids) == 0 {
			return managementv1.CedarResolveResource{}, errors.New("warehouse requires a WAREHOUSEID")
		}
		return managementv1.CedarResolveResourceWarehouseAsCedarResolveResource(
			managementv1.NewCedarResolveResourceWarehouse(*managementv1.NewCedarResolveResourceWarehouseWarehouse(ids[0]))), nil
	default:
		return managementv1.CedarResolveResource{},
			fmt.Errorf("unknown resource %q: want server, project or warehouse", kind)
	}
}

// formatCedarSource renders a source's config as "file <path>" or
// "configmap <name>".
func formatCedarSource(cfg managementv1.CedarSourceConfig) string {
	switch {
	case cfg.CedarSourceConfigFile != nil:
		return "file " + cfg.CedarSourceConfigFile.GetFilePath()
	case cfg.CedarSourceConfigKubernetesConfigMap != nil:
		return "configmap " + cfg.CedarSourceConfigKubernetesConfigMap.GetConfigmapName()
	default:
		return ""
	}
}

func printCedarPolicySources(w io.Writer, sources []managementv1.CedarPolicySourceInfo) error {
	if len(sources) == 0 {
		_, err := fmt.Fprintln(w, "No Cedar policy sources")
		return err
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "SOURCE ID\tTIER\tPOLICIES\tSOURCE")
	for _, s := range sources {
		fmt.Fprintf(tw, "%s\t%s\t%d\t%s\n", s.GetSourceId(), s.GetTier(), s.GetPolicyCount(), formatCedarSource(s.Config))
	}
	return tw.Flush()
}

func printCedarEntitySources(w io.Writer, sources []managementv1.CedarEntitySourceInfo) error {
	if len(sources) == 0 {
		_, err := fmt.Fprintln(w, "No Cedar entity sources")
		return err
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "SOURCE ID\tENTITIES\tSOURCE")
	for _, s := range sources {
		fmt.Fprintf(tw, "%s\t%d\t%s\n", s.GetSourceId(), s.GetEntityCount(), formatCedarSource(s.Config))
	}
	return tw.Flush()
}

func printCedarPolicies(w io.Writer, resp *managementv1.GetCedarPoliciesResponse) error {
	if len(resp.GetPolicies()) == 0 {
		fmt.Fprintln(w, "No Cedar policies")
	} else {
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "EFFECT\tPRINCIPAL\tACTION\tRESOURCE")
		for i := range resp.GetPolicies() {
			p := &resp.Policies[i]
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", p.GetEffect(),
				formatCedarScope(p.Principal), formatCedarScope(p.Action), formatCedarScope(p.Resource))
		}
		if err := tw.Flush(); err != nil {
			return err
		}
	}
	if token := resp.GetNextPageToken(); token != "" {
		fmt.Fprintf(w, "\nNext page token: %s\n", token)
	}
	return nil
}

// formatCedarScope renders a policy scope constraint from Cedar's JSON
// policy format in Cedar syntax: All, == E, in E, in [E, ...], is T,
// is T in E, with a slot in place of an entity in templates.
func formatCedarScope(s managementv1.CedarPolicyScopeConstraint) string {
	op := s.GetOp()
	switch {
	case op == "All":
		return "All"
	case op == "is":
		out := "is " + s.GetEntityType()
		if target := formatCedarInTarget(s.In); target != "" {
			out += " in " + target
		}
		return out
	case s.Entity != nil:
		return op + " " + formatCedarUID(*s.Entity)
	case s.Entities != nil:
		uids := make([]string, len(s.Entities))
		for i, e := range s.Entities {
			uids[i] = formatCedarUID(e)
		}
		return op + " [" + strings.Join(uids, ", ") + "]"
	case s.Slot != nil:
		return op + " " + *s.Slot
	default:
		return op
	}
}

// formatCedarInTarget renders the "in" part of an "is ... in ..." scope:
// {"entity": {...}} or {"slot": "?principal"}.
func formatCedarInTarget(in map[string]any) string {
	if slot, ok := in["slot"].(string); ok {
		return slot
	}
	e, ok := in["entity"].(map[string]any)
	if !ok {
		return ""
	}
	typ, _ := e["type"].(string)
	id, _ := e["id"].(string)
	return formatCedarUID(managementv1.CedarEntityUid{Type: typ, Id: id})
}

func formatCedarUID(uid managementv1.CedarEntityUid) string {
	return fmt.Sprintf("%s::%q", uid.GetType(), uid.GetId())
}

func printCedarEntities(w io.Writer, wide bool, resp *managementv1.ResolveCedarEntitiesResponse) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "TYPE\tID\tPARENTS")
	hidden := 0
	for _, e := range resp.GetEntities() {
		if !wide && e.Uid.GetType() == cedarActionType {
			hidden++
			continue
		}
		parents := make([]string, len(e.GetParents()))
		for i, p := range e.GetParents() {
			parents[i] = formatCedarUID(p)
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\n", e.Uid.GetType(), e.Uid.GetId(), strings.Join(parents, ", "))
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	fmt.Fprintf(w, "\nEntities: %d", resp.GetEntityCount())
	if hidden > 0 {
		fmt.Fprintf(w, " (%d %s entities not shown; -o wide shows them)", hidden, cedarActionType)
	}
	fmt.Fprintln(w)
	if below, ok := resp.GetPrincipalHasPrivilegeBelowOk(); ok && below != nil {
		fmt.Fprintf(w, "Principal has privilege below: %t\n", *below)
	}
	return nil
}
