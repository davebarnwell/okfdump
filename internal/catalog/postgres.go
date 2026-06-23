package catalog

import (
	"context"
	"database/sql"
	"fmt"
)

func inspectPostgres(ctx context.Context, db *sql.DB, source Source) (Bundle, error) {
	relKinds := "('r', 'p')"
	if source.IncludeViews {
		relKinds = "('r', 'p', 'v', 'm')"
	}

	rows, err := db.QueryContext(ctx, fmt.Sprintf(`
SELECT n.nspname,
       c.relname,
       CASE c.relkind
         WHEN 'r' THEN 'BASE TABLE'
         WHEN 'p' THEN 'PARTITIONED TABLE'
         WHEN 'v' THEN 'VIEW'
         WHEN 'm' THEN 'MATERIALIZED VIEW'
         ELSE c.relkind::text
       END AS table_type,
       COALESCE(obj_description(c.oid, 'pg_class'), '') AS comment
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE c.relkind IN %s
  AND n.nspname NOT IN ('pg_catalog', 'information_schema')
  AND n.nspname NOT LIKE 'pg_toast%%'
ORDER BY n.nspname, c.relname`, relKinds))
	if err != nil {
		return Bundle{}, fmt.Errorf("inspect Postgres tables: %w", err)
	}
	defer rows.Close()

	var tables []Table
	for rows.Next() {
		var table Table
		if err := rows.Scan(&table.Schema, &table.Name, &table.Kind, &table.Comment); err != nil {
			return Bundle{}, err
		}
		tables = append(tables, table)
	}
	if err := rows.Err(); err != nil {
		return Bundle{}, err
	}

	byTable := tableMap(tables)
	if err := loadPostgresColumns(ctx, db, byTable); err != nil {
		return Bundle{}, err
	}
	if err := loadPostgresForeignKeys(ctx, db, byTable); err != nil {
		return Bundle{}, err
	}

	return buildBundle(source, tables), nil
}

func loadPostgresColumns(ctx context.Context, db *sql.DB, byTable map[string]*Table) error {
	rows, err := db.QueryContext(ctx, `
SELECT c.table_schema,
       c.table_name,
       c.column_name,
       c.ordinal_position,
       c.data_type,
       CASE
         WHEN c.udt_schema = 'pg_catalog' THEN c.udt_name
         ELSE c.udt_schema || '.' || c.udt_name
       END AS column_type,
       c.is_nullable,
       c.column_default,
       COALESCE(col_description((quote_ident(c.table_schema) || '.' || quote_ident(c.table_name))::regclass::oid, c.ordinal_position), '') AS comment,
       c.character_maximum_length,
       c.numeric_precision,
       c.numeric_scale
FROM information_schema.columns c
WHERE c.table_schema NOT IN ('pg_catalog', 'information_schema')
ORDER BY c.table_schema, c.table_name, c.ordinal_position`)
	if err != nil {
		return fmt.Errorf("inspect Postgres columns: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var schema, tableName string
		var col Column
		var nullable string
		var defaultValue sql.NullString
		var maxLength, precision, scale sql.NullInt64
		if err := rows.Scan(&schema, &tableName, &col.Name, &col.Ordinal, &col.DataType, &col.ColumnType,
			&nullable, &defaultValue, &col.Comment, &maxLength, &precision, &scale); err != nil {
			return err
		}
		col.Nullable = nullable == "YES"
		col.Default = nullString(defaultValue)
		col.MaxLength = nullInt(maxLength)
		col.NumericPrecision = nullInt(precision)
		col.NumericScale = nullInt(scale)
		if table := byTable[tableKey(schema, tableName)]; table != nil {
			table.Columns = append(table.Columns, col)
		}
	}
	return rows.Err()
}

func loadPostgresForeignKeys(ctx context.Context, db *sql.DB, byTable map[string]*Table) error {
	rows, err := db.QueryContext(ctx, `
SELECT con.conname,
       ns.nspname,
       rel.relname,
       att.attname,
       nsf.nspname,
       relf.relname,
       attf.attname
FROM pg_constraint con
JOIN pg_class rel ON rel.oid = con.conrelid
JOIN pg_namespace ns ON ns.oid = rel.relnamespace
JOIN pg_class relf ON relf.oid = con.confrelid
JOIN pg_namespace nsf ON nsf.oid = relf.relnamespace
JOIN unnest(con.conkey) WITH ORDINALITY AS cols(attnum, ord) ON true
JOIN unnest(con.confkey) WITH ORDINALITY AS fcols(attnum, ord) ON fcols.ord = cols.ord
JOIN pg_attribute att ON att.attrelid = rel.oid AND att.attnum = cols.attnum
JOIN pg_attribute attf ON attf.attrelid = relf.oid AND attf.attnum = fcols.attnum
WHERE con.contype = 'f'
  AND ns.nspname NOT IN ('pg_catalog', 'information_schema')
  AND ns.nspname NOT LIKE 'pg_toast%'
ORDER BY ns.nspname, rel.relname, con.conname, cols.ord`)
	if err != nil {
		return fmt.Errorf("inspect Postgres foreign keys: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var schema, tableName string
		var fk ForeignKey
		if err := rows.Scan(&fk.Name, &schema, &tableName, &fk.Column, &fk.RefSchema, &fk.RefTable, &fk.RefColumn); err != nil {
			return err
		}
		if table := byTable[tableKey(schema, tableName)]; table != nil {
			table.ForeignKeys = append(table.ForeignKeys, fk)
		}
	}
	return rows.Err()
}
