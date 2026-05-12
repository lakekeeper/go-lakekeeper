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
