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
		return inspectMySQL(ctx, db, source)
	case dbdriver.Postgres:
		return inspectPostgres(ctx, db, source)
	default:
		return Bundle{}, fmt.Errorf("unsupported driver %q", source.Driver)
	}
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
