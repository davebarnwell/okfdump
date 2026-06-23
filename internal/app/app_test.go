package app

import (
	"testing"

	"github.com/davebarnwell/okfdump/internal/dbdriver"
)

func TestValidateConfigAppliesDriverDefaultsAndInfersDatabase(t *testing.T) {
	cfg := Config{
		Driver: dbdriver.Postgres,
		DSN:    "postgres://user:pass@localhost:5432/app_db?sslmode=disable",
	}

	if err := validateConfig(&cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 5432 {
		t.Fatalf("port = %d, want 5432", cfg.Port)
	}
	if cfg.Database != "app_db" {
		t.Fatalf("database = %q, want app_db", cfg.Database)
	}
}

func TestValidateConfigRequiresSSHUserWhenTunneling(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Database = "app_db"
	cfg.SSH.Host = "bastion.example.com"

	err := validateConfig(&cfg)
	if err == nil {
		t.Fatal("expected SSH user validation error")
	}
}

func TestValidateConfigResolvesSSHPasswordEnv(t *testing.T) {
	t.Setenv("OKFDUMP_TEST_SSH_PASSWORD", "secret")
	cfg := DefaultConfig()
	cfg.Database = "app_db"
	cfg.SSH.PasswordEnv = "OKFDUMP_TEST_SSH_PASSWORD"

	if err := validateConfig(&cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.SSH.Password != "secret" {
		t.Fatalf("SSH password = %q, want env value", cfg.SSH.Password)
	}
}
