package commands

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/lakekeeper/go-lakekeeper/pkg/version"
)

const cliName = "lkctl"

// newVersionCmd returns the `lkctl version` command. It prints client and
// (optionally) server version information.
func newVersionCmd(opts *clientOptions) *cobra.Command {
	var (
		short      bool
		clientOnly bool
		output     string
	)

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Example: `  # Print full client and server version
  lkctl version

  # Print only the client version (no server connection)
  lkctl version --client

  # Print client and server version as JSON
  lkctl version -o json`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			cv := version.GetVersion()

			switch output {
			case "json":
				v := map[string]any{}
				if short {
					v["client"] = map[string]string{cliName: cv.Version}
				} else {
					v["client"] = cv
				}
				if !clientOnly {
					sv, err := getServerVersion(ctx, opts)
					if err != nil {
						return err
					}
					if short {
						v["server"] = map[string]string{"lakekeeper": sv}
					} else {
						v["server"] = sv
					}
				}
				return printJSON(cmd.OutOrStdout(), v)
			case "text", "short", "":
				fmt.Fprint(cmd.OutOrStdout(), printClientVersion(&cv, short || output == "short"))
				if !clientOnly {
					sv, err := getServerVersion(ctx, opts)
					if err != nil {
						return err
					}
					fmt.Fprint(cmd.OutOrStdout(), printServerVersion(sv))
				}
				return nil
			default:
				return fmt.Errorf("unknown output format: %s", output)
			}
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format. One of: json|text|short")
	cmd.Flags().BoolVar(&short, "short", false, "Print just the version number")
	cmd.Flags().BoolVar(&clientOnly, "client", false, "Only print the client version (no server required)")
	return cmd
}

func getServerVersion(ctx context.Context, opts *clientOptions) (string, error) {
	c, err := newClient(ctx, opts)
	if err != nil {
		return "", err
	}
	info, _, err := c.ServerAPI.GetServerInfo(ctx).Execute()
	if err != nil {
		return "", wrapAPIError("server info", err)
	}
	return info.Version, nil
}

func printClientVersion(v *version.Version, short bool) string {
	if short {
		return fmt.Sprintf("%s: %s\n", cliName, v.Version)
	}

	out := fmt.Sprintf("%s: %s\n", cliName, v)
	out += fmt.Sprintf("  BuildDate: %s\n", v.BuildDate)
	out += fmt.Sprintf("  GitCommit: %s\n", v.GitCommit)
	out += fmt.Sprintf("  GitTreeState: %s\n", v.GitTreeState)
	if v.GitTag != "" {
		out += fmt.Sprintf("  GitTag: %s\n", v.GitTag)
	}
	out += fmt.Sprintf("  GoVersion: %s\n", v.GoVersion)
	out += fmt.Sprintf("  Compiler: %s\n", v.Compiler)
	out += fmt.Sprintf("  Platform: %s\n", v.Platform)
	return out
}

func printServerVersion(v string) string {
	return fmt.Sprintf("lakekeeper: %s\n", v)
}
