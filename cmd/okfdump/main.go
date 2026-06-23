package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/davebarnwell/okfdump/internal/catalog"
	"github.com/davebarnwell/okfdump/internal/okf"
	"github.com/davebarnwell/okfdump/internal/sshforward"
	"github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/spf13/cobra"
)

type config struct {
	Driver       string
	DSN          string
	Host         string
	Port         int
	User         string
	Password     string
	PasswordEnv  string
	Database     string
	SSLMode      string
	Out          string
	IncludeViews bool
	Timeout      time.Duration
	SSH          sshforward.Config
}

func main() {
	cmd := newRootCommand()
	if err := cmd.ExecuteContext(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "okfdump: %v\n", err)
		os.Exit(1)
	}
}

func newRootCommand() *cobra.Command {
	cfg := defaultConfig()
	cmd := &cobra.Command{
		Use:           "okfdump",
		Short:         "Generate an Open Knowledge Format bundle from a database",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg.Driver = strings.ToLower(strings.TrimSpace(cfg.Driver))
			return run(cmd.Context(), cfg)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&cfg.Driver, "driver", cfg.Driver, "database driver: mysql or postgres")
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

func defaultConfig() config {
	return config{
		Driver:       "mysql",
		Host:         "127.0.0.1",
		SSLMode:      "prefer",
		Out:          "./okf-bundle",
		IncludeViews: true,
		Timeout:      30 * time.Second,
		SSH: sshforward.Config{
			Port:           22,
			KnownHostsPath: "~/.ssh/known_hosts",
		},
	}
}

func run(parent context.Context, cfg config) error {
	if err := validateConfig(&cfg); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(parent, cfg.Timeout)
	defer cancel()

	dbHost, dbPort := cfg.Host, cfg.Port
	var tunnel *sshforward.Tunnel
	if cfg.SSH.Host != "" {
		cfg.SSH.TargetHost = cfg.Host
		cfg.SSH.TargetPort = cfg.Port
		var err error
		tunnel, err = sshforward.Start(ctx, cfg.SSH)
		if err != nil {
			return fmt.Errorf("start SSH tunnel: %w", err)
		}
		defer tunnel.Close()
		dbHost = tunnel.LocalHost()
		dbPort = tunnel.LocalPort()
	}

	dsn, err := buildDSN(cfg, dbHost, dbPort)
	if err != nil {
		return err
	}

	driverName := cfg.Driver
	if driverName == "postgres" || driverName == "postgresql" {
		driverName = "pgx"
	}

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}

	metadata := catalog.Source{
		Driver:       canonicalDriver(cfg.Driver),
		Host:         cfg.Host,
		Port:         cfg.Port,
		Database:     cfg.Database,
		IncludeViews: cfg.IncludeViews,
		GeneratedAt:  time.Now().UTC(),
	}

	bundle, err := catalog.Inspect(ctx, db, metadata)
	if err != nil {
		return err
	}

	if err := okf.WriteBundle(cfg.Out, bundle); err != nil {
		return err
	}

	fmt.Printf("Wrote OKF bundle for %s database %q to %s\n", bundle.Driver, bundle.Database, cfg.Out)
	return nil
}

func validateConfig(cfg *config) error {
	switch cfg.Driver {
	case "mysql":
		if cfg.Port == 0 {
			cfg.Port = 3306
		}
	case "postgres", "postgresql":
		if cfg.Port == 0 {
			cfg.Port = 5432
		}
	default:
		return fmt.Errorf("unsupported --driver %q", cfg.Driver)
	}

	if cfg.PasswordEnv != "" {
		cfg.Password = os.Getenv(cfg.PasswordEnv)
	}
	if cfg.SSH.PasswordEnv != "" {
		cfg.SSH.Password = os.Getenv(cfg.SSH.PasswordEnv)
	}
	if cfg.Database == "" && cfg.DSN != "" {
		cfg.Database = inferDatabase(cfg.Driver, cfg.DSN)
	}
	if cfg.SSH.Host != "" && cfg.SSH.User == "" {
		return errors.New("--ssh-user is required when --ssh-host is set")
	}
	if cfg.Database == "" {
		return errors.New("--database is required when it cannot be inferred from --dsn")
	}
	return nil
}

func buildDSN(cfg config, host string, port int) (string, error) {
	if cfg.DSN != "" && cfg.SSH.Host == "" {
		return cfg.DSN, nil
	}

	switch cfg.Driver {
	case "mysql":
		return buildMySQLDSN(cfg, host, port), nil
	case "postgres", "postgresql":
		return buildPostgresDSN(cfg, host, port), nil
	default:
		return "", fmt.Errorf("unsupported --driver %q", cfg.Driver)
	}
}

func buildMySQLDSN(cfg config, host string, port int) string {
	if cfg.DSN != "" {
		return rewriteMySQLDSNAddress(cfg.DSN, host, port)
	}

	mysqlCfg := mysql.NewConfig()
	mysqlCfg.User = cfg.User
	mysqlCfg.Passwd = cfg.Password
	mysqlCfg.Net = "tcp"
	mysqlCfg.Addr = net.JoinHostPort(host, strconv.Itoa(port))
	mysqlCfg.DBName = cfg.Database
	mysqlCfg.ParseTime = true
	return mysqlCfg.FormatDSN()
}

func buildPostgresDSN(cfg config, host string, port int) string {
	if cfg.DSN != "" {
		return rewritePostgresDSNAddress(cfg.DSN, host, port)
	}

	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.User, cfg.Password),
		Host:   net.JoinHostPort(host, strconv.Itoa(port)),
		Path:   "/" + cfg.Database,
	}
	q := u.Query()
	q.Set("sslmode", cfg.SSLMode)
	u.RawQuery = q.Encode()
	return u.String()
}

func rewriteMySQLDSNAddress(dsn string, host string, port int) string {
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return dsn
	}
	cfg.Net = "tcp"
	cfg.Addr = net.JoinHostPort(host, strconv.Itoa(port))
	return cfg.FormatDSN()
}

func rewritePostgresDSNAddress(dsn string, host string, port int) string {
	u, err := url.Parse(dsn)
	if err != nil || u.Scheme == "" {
		return strings.TrimSpace(dsn) + " host=" + quotePostgresKeywordValue(host) + " port=" + strconv.Itoa(port)
	}
	u.Host = net.JoinHostPort(host, strconv.Itoa(port))
	return u.String()
}

func quotePostgresKeywordValue(value string) string {
	escaped := strings.ReplaceAll(value, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `'`, `\'`)
	return "'" + escaped + "'"
}

func inferDatabase(driver string, dsn string) string {
	switch driver {
	case "mysql":
		cfg, err := mysql.ParseDSN(dsn)
		if err == nil {
			return cfg.DBName
		}
	case "postgres", "postgresql":
		u, err := url.Parse(dsn)
		if err == nil && u.Scheme != "" {
			return strings.TrimPrefix(u.Path, "/")
		}
		return inferPostgresKeywordDatabase(dsn)
	}
	return ""
}

func inferPostgresKeywordDatabase(dsn string) string {
	for _, field := range strings.Fields(dsn) {
		key, value, ok := strings.Cut(field, "=")
		if ok && (key == "dbname" || key == "database") {
			return strings.Trim(value, "'")
		}
	}
	return ""
}

func canonicalDriver(driver string) string {
	if driver == "postgresql" {
		return "postgres"
	}
	return driver
}
