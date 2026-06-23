package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/davebarnwell/okfdump/internal/catalog"
	"github.com/davebarnwell/okfdump/internal/dbconn"
	"github.com/davebarnwell/okfdump/internal/dbdriver"
	"github.com/davebarnwell/okfdump/internal/okf"
	"github.com/davebarnwell/okfdump/internal/sshforward"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type Config struct {
	Driver       dbdriver.Driver
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

func DefaultConfig() Config {
	return Config{
		Driver:       dbdriver.MySQL,
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

func Run(parent context.Context, cfg Config) error {
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

	dsn, err := dbconn.BuildDSN(dbConnConfig(cfg), dbHost, dbPort)
	if err != nil {
		return err
	}

	db, err := sql.Open(cfg.Driver.SQLDriverName(), dsn)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}

	metadata := catalog.Source{
		Driver:       cfg.Driver,
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

func validateConfig(cfg *Config) error {
	driver, err := dbdriver.Parse(cfg.Driver.String())
	if err != nil {
		return err
	}
	cfg.Driver = driver
	if cfg.Port == 0 {
		cfg.Port = cfg.Driver.DefaultPort()
	}

	if cfg.PasswordEnv != "" {
		cfg.Password = os.Getenv(cfg.PasswordEnv)
	}
	if cfg.SSH.PasswordEnv != "" {
		cfg.SSH.Password = os.Getenv(cfg.SSH.PasswordEnv)
	}
	if cfg.Database == "" && cfg.DSN != "" {
		cfg.Database = dbconn.InferDatabase(cfg.Driver, cfg.DSN)
	}
	if cfg.SSH.Host != "" && cfg.SSH.User == "" {
		return errors.New("--ssh-user is required when --ssh-host is set")
	}
	if cfg.Database == "" {
		return errors.New("--database is required when it cannot be inferred from --dsn")
	}
	return nil
}

func dbConnConfig(cfg Config) dbconn.Config {
	return dbconn.Config{
		Driver:        cfg.Driver,
		DSN:           cfg.DSN,
		Host:          cfg.Host,
		Port:          cfg.Port,
		User:          cfg.User,
		Password:      cfg.Password,
		Database:      cfg.Database,
		SSLMode:       cfg.SSLMode,
		TunnelEnabled: cfg.SSH.Host != "",
	}
}
