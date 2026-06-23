package catalog

import "testing"

func TestAppendColumnAndForeignKeyOnlyMutateKnownTables(t *testing.T) {
	tables := []Table{{Schema: "public", Name: "orders"}}
	byTable := tableMap(tables)

	appendColumn(byTable, "public", "orders", Column{Name: "id"})
	appendColumn(byTable, "public", "missing", Column{Name: "ignored"})
	appendForeignKey(byTable, "public", "orders", ForeignKey{Name: "orders_customer_id_fk"})
	appendForeignKey(byTable, "public", "missing", ForeignKey{Name: "ignored"})

	table := byTable[tableKey("public", "orders")]
	if len(table.Columns) != 1 || table.Columns[0].Name != "id" {
		t.Fatalf("columns = %#v", table.Columns)
	}
	if len(table.ForeignKeys) != 1 || table.ForeignKeys[0].Name != "orders_customer_id_fk" {
		t.Fatalf("foreign keys = %#v", table.ForeignKeys)
	}
}

func TestBuildBundlePreservesSchemaDiscoveryOrder(t *testing.T) {
	bundle := buildBundle(Source{Database: "app"}, []Table{
		{Schema: "sales", Name: "orders"},
		{Schema: "public", Name: "users"},
		{Schema: "sales", Name: "line_items"},
	})

	if len(bundle.Schemas) != 2 {
		t.Fatalf("schemas = %#v", bundle.Schemas)
	}
	if bundle.Schemas[0].Name != "sales" || bundle.Schemas[1].Name != "public" {
		t.Fatalf("schema order = %#v", bundle.Schemas)
	}
	if len(bundle.Schemas[0].Tables) != 2 {
		t.Fatalf("sales tables = %#v", bundle.Schemas[0].Tables)
	}
}

func TestFilterTablesAllowsUnqualifiedAndQualifiedNames(t *testing.T) {
	tables := []Table{
		{Schema: "public", Name: "users"},
		{Schema: "sales", Name: "users"},
		{Schema: "sales", Name: "orders"},
	}

	filtered, err := filterTables(tables, []string{"public.users", "orders"})
	if err != nil {
		t.Fatal(err)
	}

	if len(filtered) != 2 {
		t.Fatalf("filtered tables = %#v", filtered)
	}
	if filtered[0].Schema != "public" || filtered[0].Name != "users" {
		t.Fatalf("first filtered table = %#v", filtered[0])
	}
	if filtered[1].Schema != "sales" || filtered[1].Name != "orders" {
		t.Fatalf("second filtered table = %#v", filtered[1])
	}
}

func TestFilterTablesReportsMissingFilters(t *testing.T) {
	_, err := filterTables([]Table{{Schema: "public", Name: "users"}}, []string{"missing"})
	if err == nil {
		t.Fatal("expected missing table filter error")
	}
}
