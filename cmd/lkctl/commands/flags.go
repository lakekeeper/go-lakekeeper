package commands

import (
	"github.com/spf13/cobra"
)

// listOpts captures the standard pagination flags shared by `list` subcommands.
type listOpts struct {
	limit int64
	token string
	name  string
}

// accessOpts captures the principal-filter flags shared by `access` subcommands.
type accessOpts struct {
	user string
	role string
}

// assignmentsOpts captures the relations filter flag shared by `assignments`
// subcommands.
type assignmentsOpts struct {
	relations []string
}

// addProjectFlag registers the persistent -p/--project flag of a resource
// command. It has no default: when omitted, no project is sent and the server
// applies its default project. Pass the value to a façade as
// client.WithProject(project); an empty value sends nothing.
func addProjectFlag(cmd *cobra.Command, project *string) {
	cmd.PersistentFlags().StringVarP(project, "project", "p", "",
		"Project ID; when omitted, the server's default project is used")
}

func addListFlags(cmd *cobra.Command, opts *listOpts) {
	cmd.Flags().Int64Var(&opts.limit, "limit", 100, "Upper bound on the number of results returned to the client")
	cmd.Flags().StringVar(&opts.token, "token", "", "Pagination token")
	cmd.Flags().StringVar(&opts.name, "name", "", "Filter by name")
}

func addAccessFlags(cmd *cobra.Command, opts *accessOpts) {
	cmd.Flags().StringVar(&opts.user, "user", "", "Filter by user")
	cmd.Flags().StringVar(&opts.role, "role", "", "Filter by role")
}

func addAssignmentsFlags(cmd *cobra.Command, opts *assignmentsOpts) {
	cmd.Flags().StringSliceVar(&opts.relations, "relations", nil, "Filter by relations (repeat for multiple, also accepts comma-separated values)")
}
