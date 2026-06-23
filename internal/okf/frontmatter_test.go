package okf

import (
	"strings"
	"testing"
)

func TestFrontmatterUsesTypedYAMLFields(t *testing.T) {
	got := frontmatter(frontmatterFields{
		Type:      "MySQL Table",
		Title:     "app.users",
		Resource:  "mysql://db/app/users",
		Tags:      []string{"mysql", "table"},
		Timestamp: "2026-06-23T12:00:00Z",
	})

	for _, want := range []string{
		"---\n",
		"type: MySQL Table\n",
		"title: app.users\n",
		"resource: mysql://db/app/users\n",
		"tags:\n    - mysql\n    - table\n",
		"timestamp: \"2026-06-23T12:00:00Z\"\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in frontmatter:\n%s", want, got)
		}
	}
}
