package okf

import (
	"strings"
	"testing"

	"github.com/davebarnwell/okfdump/internal/catalog"
	"github.com/davebarnwell/okfdump/internal/dbdriver"
)

func TestSegmentSanitizesUnsafePathParts(t *testing.T) {
	got := segment("weird schema/name")
	if !strings.HasPrefix(got, "weird_schema_name-") {
		t.Fatalf("segment() = %q, want sanitized prefix with hash suffix", got)
	}
	if got == segment("weird_schema_name") {
		t.Fatalf("unsafe segment collided with safe segment: %q", got)
	}
}

func TestResourceURIIncludesDriverHostPortAndScope(t *testing.T) {
	bundle := catalog.Bundle{
		Source: catalog.Source{
			Driver:   dbdriver.Postgres,
			Host:     "db.example.com",
			Port:     5432,
			Database: "app",
		},
	}

	got := resourceURI(bundle, "public", "users")
	want := "postgres://db.example.com:5432/app/public/users"
	if got != want {
		t.Fatalf("resourceURI() = %q, want %q", got, want)
	}
}

func TestMarkdownHelpers(t *testing.T) {
	table := catalog.Table{Schema: "public", Name: "active_users", Kind: "VIEW"}
	if got := tableTags(dbdriver.Postgres, table); got[1] != "view" {
		t.Fatalf("tableTags()[1] = %q, want view", got[1])
	}
	if got := qualifiedTable(dbdriver.MySQL, "app", "users"); got != "`app`.`users`" {
		t.Fatalf("qualifiedTable() = %q", got)
	}
}
