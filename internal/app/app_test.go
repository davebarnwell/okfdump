package app

import (
	"github.com/davebarnwell/okfdump/internal/dbdriver"
	"testing"
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
