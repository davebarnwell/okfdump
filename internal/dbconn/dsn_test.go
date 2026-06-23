package dbconn

import (
	"strings"
	"testing"

	"github.com/davebarnwell/okfdump/internal/dbdriver"
)

func TestInferDatabase(t *testing.T) {
	tests := []struct {
		name   string
		driver dbdriver.Driver
		dsn    string
		want   string
	}{
		{
			name:   "mysql",
			driver: dbdriver.MySQL,
			dsn:    "user:pass@tcp(localhost:3306)/app_db?parseTime=true",
			want:   "app_db",
		},
		{
			name:   "postgres url",
			driver: dbdriver.Postgres,
			dsn:    "postgres://user:pass@localhost:5432/app_db?sslmode=disable",
			want:   "app_db",
		},
		{
			name:   "postgres keyword",
			driver: dbdriver.Postgres,
			dsn:    "host=localhost port=5432 dbname=app_db user=app",
			want:   "app_db",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := InferDatabase(tt.driver, tt.dsn); got != tt.want {
				t.Fatalf("InferDatabase() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildDSNReturnsOriginalDSNWithoutTunnel(t *testing.T) {
	dsn := "host=private dbname=app user=app"
	got, err := BuildDSN(Config{Driver: dbdriver.Postgres, DSN: dsn}, "127.0.0.1", 5000)
	if err != nil {
		t.Fatal(err)
	}
	if got != dsn {
		t.Fatalf("BuildDSN() = %q, want original DSN", got)
	}
}

func TestBuildDSNRewritesPostgresKeywordDSNForTunnel(t *testing.T) {
	got, err := BuildDSN(Config{
		Driver:        dbdriver.Postgres,
		DSN:           "host=private dbname=app user=app",
		TunnelEnabled: true,
	}, "127.0.0.1", 5000)
	if err != nil {
		t.Fatal(err)
	}
	want := "host=private dbname=app user=app host='127.0.0.1' port=5000"
	if got != want {
		t.Fatalf("BuildDSN() = %q, want %q", got, want)
	}
}

func TestBuildDSNConstructsMySQLDSN(t *testing.T) {
	got, err := BuildDSN(Config{
		Driver:   dbdriver.MySQL,
		User:     "app",
		Password: "secret",
		Database: "app_db",
	}, "db.local", 3306)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"app:secret", "tcp(db.local:3306)", "/app_db", "parseTime=true"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in MySQL DSN %q", want, got)
		}
	}
}
