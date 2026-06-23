package okf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/davebarnwell/okfdump/internal/catalog"
	"github.com/davebarnwell/okfdump/internal/dbdriver"
)

func TestWriteBundleCreatesConformantConceptFiles(t *testing.T) {
	root := t.TempDir()
	bundle := catalog.Bundle{
		Source: catalog.Source{
			Driver:      dbdriver.MySQL,
			Host:        "db.example.com",
			Port:        3306,
			Database:    "app",
			GeneratedAt: time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC),
		},
		Schemas: []catalog.Schema{
			{
				Name: "app",
				Tables: []catalog.Table{
					{
						Schema:  "app",
						Name:    "orders",
						Kind:    "BASE TABLE",
						Comment: "Customer orders",
						Columns: []catalog.Column{
							{Name: "id", Ordinal: 1, ColumnType: "bigint", Nullable: false, Comment: "Primary key"},
							{Name: "customer_id", Ordinal: 2, ColumnType: "bigint", Nullable: false},
						},
						ForeignKeys: []catalog.ForeignKey{
							{Name: "orders_customer_id_fk", Column: "customer_id", RefSchema: "app", RefTable: "customers", RefColumn: "id"},
						},
					},
				},
			},
		},
	}

	if err := WriteBundle(root, bundle); err != nil {
		t.Fatal(err)
	}

	tablePath := filepath.Join(root, "tables", "app", "orders.md")
	content, err := os.ReadFile(tablePath)
	if err != nil {
		t.Fatal(err)
	}

	text := string(content)
	for _, want := range []string{
		"---\n",
		"type: MySQL Base Table",
		"title: app.orders",
		"# Schema",
		"| `customer_id` | `bigint` | no | - | - |",
		"[app.customers](/tables/app/customers.md)",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected %q in generated table file:\n%s", want, text)
		}
	}
}
