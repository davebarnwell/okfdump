package main

import (
	"strings"
	"testing"
)

func TestRootCommandRejectsPositionalArgs(t *testing.T) {
	cmd := newRootCommand()
	cmd.SetArgs([]string{"unexpected"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected positional argument error")
	}
	if !strings.Contains(err.Error(), "unknown command") && !strings.Contains(err.Error(), "accepts 0 arg") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRootCommandHelpIncludesFlags(t *testing.T) {
	cmd := newRootCommand()
	cmd.SetArgs([]string{"--help"})

	var out strings.Builder
	cmd.SetOut(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"--driver", "--ssh-host", "-o, --out"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("expected help to include %q:\n%s", want, out.String())
		}
	}
}

func TestInferDatabase(t *testing.T) {
	tests := []struct {
		name   string
		driver string
		dsn    string
		want   string
	}{
		{
			name:   "mysql",
			driver: "mysql",
			dsn:    "user:pass@tcp(localhost:3306)/app_db?parseTime=true",
			want:   "app_db",
		},
		{
			name:   "postgres url",
			driver: "postgres",
			dsn:    "postgres://user:pass@localhost:5432/app_db?sslmode=disable",
			want:   "app_db",
		},
		{
			name:   "postgres keyword",
			driver: "postgres",
			dsn:    "host=localhost port=5432 dbname=app_db user=app",
			want:   "app_db",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := inferDatabase(tt.driver, tt.dsn); got != tt.want {
				t.Fatalf("inferDatabase() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRewritePostgresKeywordDSNAddress(t *testing.T) {
	got := rewritePostgresDSNAddress("host=private dbname=app user=app", "127.0.0.1", 5000)
	want := "host=private dbname=app user=app host='127.0.0.1' port=5000"
	if got != want {
		t.Fatalf("rewritePostgresDSNAddress() = %q, want %q", got, want)
	}
}
