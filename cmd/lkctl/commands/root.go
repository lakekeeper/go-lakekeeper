// Package commands hosts the lkctl CLI built atop the OpenAPI-generated
// managementv1 client.
package commands

import (
	"os"

	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/lakekeeper/go-lakekeeper/pkg/common"
	"github.com/lakekeeper/go-lakekeeper/pkg/core"
	"github.com/lakekeeper/go-lakekeeper/pkg/version"
)

func init() {
	log.SetOutput(os.Stderr)
	log.SetLevel(log.InfoLevel)
	if err := godotenv.Load(); err != nil {
		log.Debug("no .env file found: ", err)
	}
}

// NewCommand returns the lkctl root command with all subcommands registered.
func NewCommand() *cobra.Command {
	var opts clientOptions

	cmd := &cobra.Command{
		Use:               cliName,
		Short:             "Command-line client for the Lakekeeper Iceberg catalog.",
		Version:           version.GetVersion().Version,
		SilenceErrors:     true,
		SilenceUsage:      true,
		DisableAutoGenTag: true,
		PersistentPreRun: func(*cobra.Command, []string) {
			log.SetFormatter(&log.TextFormatter{DisableColors: true, FullTimestamp: true})
			if opts.debug {
				log.SetLevel(log.DebugLevel)
			}
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.PersistentFlags().StringVar(&opts.baseURL, "base-url",
		common.GetEnvOr(common.EnvBaseURL, common.DefaultBaseURL),
		"Lakekeeper base URL; or set "+common.EnvBaseURL)
	cmd.PersistentFlags().StringVar(&opts.authMode, "auth-mode",
		common.GetEnvOr(common.EnvAuthMode, common.DefaultAuthMode),
		"Authentication mode: oauth2 | token | k8s; or set "+common.EnvAuthMode)
	cmd.PersistentFlags().StringVar(&opts.tokenURL, "token-url",
		common.GetEnvOr(common.EnvTokenURL, ""),
		"OAuth2 token endpoint; or set "+common.EnvTokenURL)
	cmd.PersistentFlags().StringVar(&opts.clientID, "client-id",
		common.GetEnvOr(common.EnvClientID, ""),
		"OAuth2 client_id; or set "+common.EnvClientID)
	cmd.PersistentFlags().StringVar(&opts.clientSecret, "client-secret",
		common.GetEnvOr(common.EnvClientSecret, ""),
		"OAuth2 client_secret; or set "+common.EnvClientSecret)
	cmd.PersistentFlags().StringSliceVar(&opts.scope, "scopes",
		common.GetEnvSlice(common.EnvScope, " ", common.DefaultScope),
		"OAuth2 scopes; or set "+common.EnvScope)
	cmd.PersistentFlags().StringVar(&opts.accessToken, "access-token",
		common.GetEnvOr(common.EnvAccessToken, ""),
		"Static bearer token (auth-mode=token); or set "+common.EnvAccessToken)
	cmd.PersistentFlags().StringVar(&opts.k8sTokenPath, "k8s-token-path",
		common.GetEnvOr(common.EnvK8sTokenPath, core.DefaultK8sServiceAccountTokenPath),
		"Path to the Kubernetes service-account token (auth-mode=k8s); or set "+common.EnvK8sTokenPath)
	cmd.PersistentFlags().BoolVar(&opts.bootstrap, "bootstrap",
		common.GetBoolEnv(common.EnvBootstrap),
		"Bootstrap the server with the current user; or set "+common.EnvBootstrap)
	cmd.PersistentFlags().BoolVar(&opts.debug, "debug", false, "Enable debug logging")

	cmd.AddCommand(newCatalogCmd())
	cmd.AddCommand(newProjectCmd(&opts))
	cmd.AddCommand(newRoleCmd(&opts))
	cmd.AddCommand(newServerCmd(&opts))
	cmd.AddCommand(newUserCmd(&opts))
	cmd.AddCommand(newVersionCmd(&opts))
	cmd.AddCommand(newWarehouseCmd(&opts))
	cmd.AddCommand(newWhoamiCmd(&opts))

	return cmd
}
