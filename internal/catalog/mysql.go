package catalog

import (
	"context"
	"database/sql"
	"fmt"
)

func loadMySQLTables(ctx context.Context, db *sql.DB, source Source) ([]Table, error) {
	tableTypes := "('BASE TABLE')"
	if source.IncludeViews {
		tableTypes = "('BASE TABLE', 'VIEW')"
	}

	rows, err := db.QueryContext(ctx, fmt.Sprintf(`
SELECT TABLE_SCHEMA, TABLE_NAME, TABLE_TYPE, COALESCE(TABLE_COMMENT, '')
FROM information_schema.TABLES
WHERE TABLE_SCHEMA = ? AND TABLE_TYPE IN %s
ORDER BY TABLE_SCHEMA, TABLE_NAME`, tableTypes), source.Database)
	if err != nil {
		return nil, fmt.Errorf("inspect MySQL tables: %w", err)
	}
	return scanTables(rows)
}

func loadMySQLColumns(ctx context.Context, db *sql.DB, source Source, byTable map[string]*Table) error {
	rows, err := db.QueryContext(ctx, `
SELECT TABLE_SCHEMA, TABLE_NAME, COLUMN_NAME, ORDINAL_POSITION, DATA_TYPE, COLUMN_TYPE,
       IS_NULLABLE, COLUMN_DEFAULT, COALESCE(COLUMN_COMMENT, ''), COALESCE(EXTRA, ''),
       CHARACTER_MAXIMUM_LENGTH, NUMERIC_PRECISION, NUMERIC_SCALE
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = ?
ORDER BY TABLE_SCHEMA, TABLE_NAME, ORDINAL_POSITION`, source.Database)
	if err != nil {
		return fmt.Errorf("inspect MySQL columns: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var schema, tableName string
		var col Column
		var nullable string
		var defaultValue sql.NullString
		var maxLength, precision, scale sql.NullInt64
		if err := rows.Scan(&schema, &tableName, &col.Name, &col.Ordinal, &col.DataType, &col.ColumnType,
			&nullable, &defaultValue, &col.Comment, &col.Extra, &maxLength, &precision, &scale); err != nil {
			return err
		}
		col.Nullable = nullable == "YES"
		col.Default = nullString(defaultValue)
		col.MaxLength = nullInt(maxLength)
		col.NumericPrecision = nullInt(precision)
		col.NumericScale = nullInt(scale)
		appendColumn(byTable, schema, tableName, col)
	}
	return rows.Err()
}

func loadMySQLForeignKeys(ctx context.Context, db *sql.DB, source Source, byTable map[string]*Table) error {
	rows, err := db.QueryContext(ctx, `
SELECT CONSTRAINT_NAME, TABLE_SCHEMA, TABLE_NAME, COLUMN_NAME,
       REFERENCED_TABLE_SCHEMA, REFERENCED_TABLE_NAME, REFERENCED_COLUMN_NAME
FROM information_schema.KEY_COLUMN_USAGE
WHERE TABLE_SCHEMA = ? AND REFERENCED_TABLE_NAME IS NOT NULL
ORDER BY TABLE_SCHEMA, TABLE_NAME, CONSTRAINT_NAME, ORDINAL_POSITION`, source.Database)
	if err != nil {
		return fmt.Errorf("inspect MySQL foreign keys: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var schema, tableName string
		var fk ForeignKey
		if err := rows.Scan(&fk.Name, &schema, &tableName, &fk.Column, &fk.RefSchema, &fk.RefTable, &fk.RefColumn); err != nil {
			return err
		}
		appendForeignKey(byTable, schema, tableName, fk)
	}
	return rows.Err()
}
