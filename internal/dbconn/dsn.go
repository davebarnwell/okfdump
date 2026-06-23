package dbconn

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/davebarnwell/okfdump/internal/dbdriver"
	"github.com/go-sql-driver/mysql"
)

type Config struct {
	Driver        dbdriver.Driver
	DSN           string
	Host          string
	Port          int
	User          string
	Password      string
	Database      string
	SSLMode       string
	TunnelEnabled bool
}

func BuildDSN(cfg Config, host string, port int) (string, error) {
	if cfg.DSN != "" && !cfg.TunnelEnabled {
		return cfg.DSN, nil
	}

	switch cfg.Driver {
	case dbdriver.MySQL:
		return buildMySQLDSN(cfg, host, port), nil
	case dbdriver.Postgres:
		return buildPostgresDSN(cfg, host, port), nil
	default:
		return "", fmt.Errorf("unsupported --driver %q", cfg.Driver)
	}
}

func InferDatabase(driver dbdriver.Driver, dsn string) string {
	switch driver {
	case dbdriver.MySQL:
		cfg, err := mysql.ParseDSN(dsn)
		if err == nil {
			return cfg.DBName
		}
	case dbdriver.Postgres:
		u, err := url.Parse(dsn)
		if err == nil && u.Scheme != "" {
			return strings.TrimPrefix(u.Path, "/")
		}
		return inferPostgresKeywordDatabase(dsn)
	}
	return ""
}

func buildMySQLDSN(cfg Config, host string, port int) string {
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

func buildPostgresDSN(cfg Config, host string, port int) string {
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

func inferPostgresKeywordDatabase(dsn string) string {
	for _, field := range strings.Fields(dsn) {
		key, value, ok := strings.Cut(field, "=")
		if ok && (key == "dbname" || key == "database") {
			return strings.Trim(value, "'")
		}
	}
	return ""
}
