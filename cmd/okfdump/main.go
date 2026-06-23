package main

import (
	"context"
	"fmt"
	"os"

	"github.com/davebarnwell/okfdump/internal/app"
	"github.com/spf13/cobra"
)

func main() {
	cmd := newRootCommand()
	if err := cmd.ExecuteContext(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "okfdump: %v\n", err)
		os.Exit(1)
	}
}

func newRootCommand() *cobra.Command {
	cfg := app.DefaultConfig()
	cmd := &cobra.Command{
		Use:           "okfdump",
		Short:         "Generate an Open Knowledge Format bundle from a database",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.Run(cmd.Context(), cfg)
		},
	}

	flags := cmd.Flags()
	flags.Var(&cfg.Driver, "driver", "database driver: mysql or postgres")
	flags.StringVar(&cfg.DSN, "dsn", cfg.DSN, "driver-specific database connection string")
	flags.StringVar(&cfg.Host, "host", cfg.Host, "database host")
	flags.IntVar(&cfg.Port, "port", cfg.Port, "database port; defaults to the driver default")
	flags.StringVar(&cfg.User, "user", cfg.User, "database username")
	flags.StringVar(&cfg.Password, "password", cfg.Password, "database password")
	flags.StringVar(&cfg.PasswordEnv, "password-env", cfg.PasswordEnv, "environment variable containing the database password")
	flags.StringVar(&cfg.Database, "database", cfg.Database, "database/schema name to inspect")
	flags.StringVar(&cfg.SSLMode, "sslmode", cfg.SSLMode, "Postgres sslmode when building a DSN")
	flags.StringVarP(&cfg.Out, "out", "o", cfg.Out, "output OKF bundle directory")
	flags.BoolVar(&cfg.IncludeViews, "include-views", cfg.IncludeViews, "include database views")
	flags.DurationVar(&cfg.Timeout, "timeout", cfg.Timeout, "connection and inspection timeout")

	flags.StringVar(&cfg.SSH.Host, "ssh-host", cfg.SSH.Host, "SSH bastion host for tunneling")
	flags.IntVar(&cfg.SSH.Port, "ssh-port", cfg.SSH.Port, "SSH port")
	flags.StringVar(&cfg.SSH.User, "ssh-user", cfg.SSH.User, "SSH username")
	flags.StringVar(&cfg.SSH.Password, "ssh-password", cfg.SSH.Password, "SSH password")
	flags.StringVar(&cfg.SSH.PasswordEnv, "ssh-password-env", cfg.SSH.PasswordEnv, "environment variable containing the SSH password")
	flags.StringVar(&cfg.SSH.KeyPath, "ssh-key", cfg.SSH.KeyPath, "SSH private key path")
	flags.StringVar(&cfg.SSH.KeyPassphrase, "ssh-key-passphrase", cfg.SSH.KeyPassphrase, "SSH private key passphrase")
	flags.StringVar(&cfg.SSH.KnownHostsPath, "ssh-known-hosts", cfg.SSH.KnownHostsPath, "SSH known_hosts file")
	flags.BoolVar(&cfg.SSH.InsecureIgnoreHostKey, "ssh-insecure-ignore-host-key", cfg.SSH.InsecureIgnoreHostKey, "disable SSH host key verification")

	return cmd
}
