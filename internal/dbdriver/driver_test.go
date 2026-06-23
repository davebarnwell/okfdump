package dbdriver

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		input string
		want  Driver
	}{
		{input: "mysql", want: MySQL},
		{input: "postgres", want: Postgres},
		{input: "postgresql", want: Postgres},
		{input: " PostgreSQL ", want: Postgres},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := Parse(tt.input)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("Parse() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDriverMetadata(t *testing.T) {
	if MySQL.DefaultPort() != 3306 {
		t.Fatalf("MySQL default port = %d", MySQL.DefaultPort())
	}
	if Postgres.SQLDriverName() != "pgx" {
		t.Fatalf("Postgres SQL driver name = %q", Postgres.SQLDriverName())
	}
	if MySQL.DisplayName() != "MySQL" {
		t.Fatalf("MySQL display name = %q", MySQL.DisplayName())
	}
}
