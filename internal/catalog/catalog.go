package catalog

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/davebarnwell/okfdump/internal/dbdriver"
)

type Source struct {
	Driver       dbdriver.Driver
	Host         string
	Port         int
	Database     string
	TableFilters []string
	IncludeViews bool
	GeneratedAt  time.Time
}

type Bundle struct {
	Source
	Schemas []Schema
}

type Schema struct {
	Name   string
	Tables []Table
}

type Table struct {
	Schema      string
	Name        string
	Kind        string
	Comment     string
	Columns     []Column
	ForeignKeys []ForeignKey
}

type Column struct {
	Name             string
	Ordinal          int
	DataType         string
	ColumnType       string
	Nullable         bool
	Default          string
	Comment          string
	Extra            string
	MaxLength        string
	NumericPrecision string
	NumericScale     string
}

type ForeignKey struct {
	Name      string
	Column    string
	RefSchema string
	RefTable  string
	RefColumn string
}

func Inspect(ctx context.Context, db *sql.DB, source Source) (Bundle, error) {
	switch source.Driver {
	case dbdriver.MySQL:
		return inspectRelational(ctx, db, source, loadMySQLTables, loadMySQLColumns, loadMySQLForeignKeys)
	case dbdriver.Postgres:
		return inspectRelational(ctx, db, source, loadPostgresTables, loadPostgresColumns, loadPostgresForeignKeys)
	default:
		return Bundle{}, fmt.Errorf("unsupported driver %q", source.Driver)
	}
}

type tableLoader func(context.Context, *sql.DB, Source) ([]Table, error)
type relationLoader func(context.Context, *sql.DB, Source, map[string]*Table) error

func inspectRelational(ctx context.Context, db *sql.DB, source Source, loadTables tableLoader, loadColumns relationLoader, loadForeignKeys relationLoader) (Bundle, error) {
	tables, err := loadTables(ctx, db, source)
	if err != nil {
		return Bundle{}, err
	}
	tables, err = filterTables(tables, source.TableFilters)
	if err != nil {
		return Bundle{}, err
	}

	byTable := tableMap(tables)
	if err := loadColumns(ctx, db, source, byTable); err != nil {
		return Bundle{}, err
	}
	if err := loadForeignKeys(ctx, db, source, byTable); err != nil {
		return Bundle{}, err
	}

	return buildBundle(source, tables), nil
}

func scanTables(rows *sql.Rows) ([]Table, error) {
	defer rows.Close()

	var tables []Table
	for rows.Next() {
		var table Table
		if err := rows.Scan(&table.Schema, &table.Name, &table.Kind, &table.Comment); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}
	return tables, rows.Err()
}

func appendColumn(byTable map[string]*Table, schema string, tableName string, column Column) {
	if table := byTable[tableKey(schema, tableName)]; table != nil {
		table.Columns = append(table.Columns, column)
	}
}

func appendForeignKey(byTable map[string]*Table, schema string, tableName string, fk ForeignKey) {
	if table := byTable[tableKey(schema, tableName)]; table != nil {
		table.ForeignKeys = append(table.ForeignKeys, fk)
	}
}

type tableSelector struct {
	Schema string
	Table  string
	Raw    string
}

func filterTables(tables []Table, filters []string) ([]Table, error) {
	selectors := tableSelectors(filters)
	if len(selectors) == 0 {
		return tables, nil
	}

	matched := make([]bool, len(selectors))
	var result []Table
	for _, table := range tables {
		for i, selector := range selectors {
			if selector.matches(table) {
				result = append(result, table)
				matched[i] = true
				break
			}
		}
	}

	var missing []string
	for i, selector := range selectors {
		if !matched[i] {
			missing = append(missing, selector.Raw)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("table filter(s) not found: %s", strings.Join(missing, ", "))
	}
	return result, nil
}

func tableSelectors(filters []string) []tableSelector {
	var selectors []tableSelector
	for _, filter := range filters {
		for _, part := range strings.Split(filter, ",") {
			raw := strings.TrimSpace(part)
			if raw == "" {
				continue
			}
			selector := tableSelector{Raw: raw}
			schema, table, ok := strings.Cut(raw, ".")
			if ok {
				selector.Schema = strings.ToLower(strings.TrimSpace(schema))
				selector.Table = strings.ToLower(strings.TrimSpace(table))
			} else {
				selector.Table = strings.ToLower(raw)
			}
			if selector.Table != "" {
				selectors = append(selectors, selector)
			}
		}
	}
	return selectors
}

func (s tableSelector) matches(table Table) bool {
	if s.Schema != "" && s.Schema != strings.ToLower(table.Schema) {
		return false
	}
	return s.Table == strings.ToLower(table.Name)
}

func buildBundle(source Source, tables []Table) Bundle {
	schemaMap := map[string][]Table{}
	var schemaOrder []string
	for _, table := range tables {
		if _, ok := schemaMap[table.Schema]; !ok {
			schemaOrder = append(schemaOrder, table.Schema)
		}
		schemaMap[table.Schema] = append(schemaMap[table.Schema], table)
	}

	bundle := Bundle{Source: source}
	for _, schemaName := range schemaOrder {
		bundle.Schemas = append(bundle.Schemas, Schema{
			Name:   schemaName,
			Tables: schemaMap[schemaName],
		})
	}
	return bundle
}

func tableMap(tables []Table) map[string]*Table {
	result := make(map[string]*Table, len(tables))
	for i := range tables {
		key := tableKey(tables[i].Schema, tables[i].Name)
		result[key] = &tables[i]
	}
	return result
}

func tableKey(schema, table string) string {
	return strings.ToLower(schema + "\x00" + table)
}

func nullString(v sql.NullString) string {
	if v.Valid {
		return v.String
	}
	return ""
}

func nullInt(v sql.NullInt64) string {
	if v.Valid {
		return fmt.Sprintf("%d", v.Int64)
	}
	return ""
}
