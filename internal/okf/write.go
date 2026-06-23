package okf

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/davebarnwell/okfdump/internal/catalog"
	"github.com/davebarnwell/okfdump/internal/dbdriver"
)

var unsafePathChars = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

func WriteBundle(root string, bundle catalog.Bundle) error {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}

	writer := bundleWriter{root: root, bundle: bundle}
	if err := writer.writeRootFiles(); err != nil {
		return err
	}
	if err := writer.writeDatabaseFiles(); err != nil {
		return err
	}
	if err := writer.writeSchemaFiles(); err != nil {
		return err
	}
	return writer.writeTableFiles()
}

type bundleWriter struct {
	root   string
	bundle catalog.Bundle
}

func (w bundleWriter) writeRootFiles() error {
	body := strings.Builder{}
	body.WriteString("---\n")
	body.WriteString("okf_version: \"0.1\"\n")
	body.WriteString("---\n\n")
	body.WriteString(fmt.Sprintf("# %s\n\n", title(w.bundle.Database)))
	body.WriteString("## Concepts\n\n")
	body.WriteString(fmt.Sprintf("* [Database](databases/%s.md) - Database `%s`.\n", segment(w.bundle.Database), w.bundle.Database))
	body.WriteString("* [Schemas](schemas/) - Database schemas discovered from metadata.\n")
	body.WriteString("* [Tables](tables/) - Tables, columns, and relationships discovered from metadata.\n")
	if err := w.write("index.md", body.String()); err != nil {
		return err
	}

	logBody := fmt.Sprintf("# Bundle Update Log\n\n## %s\n\n* **Creation**: Generated OKF bundle for `%s` from %s metadata.\n",
		w.bundle.GeneratedAt.Format("2006-01-02"), w.bundle.Database, w.bundle.Driver)
	return w.write("log.md", logBody)
}

func (w bundleWriter) writeDatabaseFiles() error {
	dir := "databases"
	if err := os.MkdirAll(filepath.Join(w.root, dir), 0o755); err != nil {
		return err
	}

	index := fmt.Sprintf("# Databases\n\n* [%s](%s.md) - %s database `%s`.\n",
		title(w.bundle.Database), segment(w.bundle.Database), w.bundle.Driver.DisplayName(), w.bundle.Database)
	if err := w.write(filepath.Join(dir, "index.md"), index); err != nil {
		return err
	}

	var schemaLinks []string
	for _, schema := range w.bundle.Schemas {
		schemaLinks = append(schemaLinks, fmt.Sprintf("[%s](/schemas/%s.md)", schema.Name, segment(schema.Name)))
	}

	body := strings.Builder{}
	body.WriteString(frontmatter(map[string]any{
		"type":        fmt.Sprintf("%s Database", w.bundle.Driver.DisplayName()),
		"title":       w.bundle.Database,
		"description": fmt.Sprintf("%s database `%s` with %d schema(s).", w.bundle.Driver.DisplayName(), w.bundle.Database, len(w.bundle.Schemas)),
		"resource":    resourceURI(w.bundle, "", ""),
		"tags":        []string{w.bundle.Driver.String(), "database"},
		"timestamp":   w.bundle.GeneratedAt.Format("2006-01-02T15:04:05Z"),
	}))
	body.WriteString("# Schemas\n\n")
	if len(schemaLinks) == 0 {
		body.WriteString("No schemas were discovered.\n")
	} else {
		for _, link := range schemaLinks {
			body.WriteString("* " + link + "\n")
		}
	}

	return w.write(filepath.Join(dir, segment(w.bundle.Database)+".md"), body.String())
}

func (w bundleWriter) writeSchemaFiles() error {
	dir := "schemas"
	if err := os.MkdirAll(filepath.Join(w.root, dir), 0o755); err != nil {
		return err
	}

	index := strings.Builder{}
	index.WriteString("# Schemas\n\n")
	for _, schema := range w.bundle.Schemas {
		index.WriteString(fmt.Sprintf("* [%s](%s.md) - %d table(s) and view(s).\n", schema.Name, segment(schema.Name), len(schema.Tables)))
	}
	if err := w.write(filepath.Join(dir, "index.md"), index.String()); err != nil {
		return err
	}

	for _, schema := range w.bundle.Schemas {
		body := strings.Builder{}
		body.WriteString(frontmatter(map[string]any{
			"type":        fmt.Sprintf("%s Schema", w.bundle.Driver.DisplayName()),
			"title":       schema.Name,
			"description": fmt.Sprintf("Schema `%s` in database `%s` with %d table(s) and view(s).", schema.Name, w.bundle.Database, len(schema.Tables)),
			"resource":    resourceURI(w.bundle, schema.Name, ""),
			"tags":        []string{w.bundle.Driver.String(), "schema", schema.Name},
			"timestamp":   w.bundle.GeneratedAt.Format("2006-01-02T15:04:05Z"),
		}))
		body.WriteString("# Tables\n\n")
		if len(schema.Tables) == 0 {
			body.WriteString("No tables were discovered.\n")
		}
		for _, table := range schema.Tables {
			body.WriteString(fmt.Sprintf("* [%s](/%s) - %s.\n", table.Name, tablePath(table), tableDescription(table)))
		}
		body.WriteString(fmt.Sprintf("\nPart of the [%s database](/databases/%s.md).\n", w.bundle.Database, segment(w.bundle.Database)))

		if err := w.write(filepath.Join(dir, segment(schema.Name)+".md"), body.String()); err != nil {
			return err
		}
	}
	return nil
}

func (w bundleWriter) writeTableFiles() error {
	base := "tables"
	if err := os.MkdirAll(filepath.Join(w.root, base), 0o755); err != nil {
		return err
	}

	index := strings.Builder{}
	index.WriteString("# Tables\n\n")
	for _, schema := range w.bundle.Schemas {
		index.WriteString(fmt.Sprintf("* [%s](%s/) - %d table(s) and view(s).\n", schema.Name, segment(schema.Name), len(schema.Tables)))
	}
	if err := w.write(filepath.Join(base, "index.md"), index.String()); err != nil {
		return err
	}

	for _, schema := range w.bundle.Schemas {
		schemaDir := filepath.Join(base, segment(schema.Name))
		if err := os.MkdirAll(filepath.Join(w.root, schemaDir), 0o755); err != nil {
			return err
		}

		schemaIndex := strings.Builder{}
		schemaIndex.WriteString(fmt.Sprintf("# %s Tables\n\n", schema.Name))
		for _, table := range schema.Tables {
			schemaIndex.WriteString(fmt.Sprintf("* [%s](%s.md) - %s.\n", table.Name, segment(table.Name), tableDescription(table)))
		}
		if err := w.write(filepath.Join(schemaDir, "index.md"), schemaIndex.String()); err != nil {
			return err
		}

		for _, table := range schema.Tables {
			if err := w.writeTable(schemaDir, table); err != nil {
				return err
			}
		}
	}
	return nil
}

func (w bundleWriter) writeTable(schemaDir string, table catalog.Table) error {
	body := strings.Builder{}
	body.WriteString(frontmatter(map[string]any{
		"type":        fmt.Sprintf("%s %s", w.bundle.Driver.DisplayName(), title(table.Kind)),
		"title":       fmt.Sprintf("%s.%s", table.Schema, table.Name),
		"description": tableDescription(table),
		"resource":    resourceURI(w.bundle, table.Schema, table.Name),
		"tags":        tableTags(w.bundle.Driver, table),
		"timestamp":   w.bundle.GeneratedAt.Format("2006-01-02T15:04:05Z"),
	}))

	body.WriteString("# Schema\n\n")
	body.WriteString("| Column | Type | Nullable | Default | Description |\n")
	body.WriteString("|---|---|---:|---|---|\n")
	for _, column := range table.Columns {
		body.WriteString(fmt.Sprintf("| `%s` | `%s` | %s | %s | %s |\n",
			escapePipes(column.Name),
			escapePipes(columnType(column)),
			boolWord(column.Nullable),
			markdownCodeOrDash(column.Default),
			escapePipes(columnDescription(column)),
		))
	}

	body.WriteString("\n# Relationships\n\n")
	if len(table.ForeignKeys) == 0 {
		body.WriteString("No foreign-key relationships were discovered from database metadata.\n")
	} else {
		for _, fk := range table.ForeignKeys {
			body.WriteString(fmt.Sprintf("* `%s.%s` joins to [%s.%s](/tables/%s/%s.md) on `%s.%s`.\n",
				table.Name,
				fk.Column,
				fk.RefSchema,
				fk.RefTable,
				segment(fk.RefSchema),
				segment(fk.RefTable),
				fk.RefTable,
				fk.RefColumn,
			))
		}
	}

	body.WriteString("\n# Examples\n\n")
	body.WriteString("```sql\n")
	body.WriteString(fmt.Sprintf("SELECT *\nFROM %s\nLIMIT 100;\n", qualifiedTable(w.bundle.Driver, table.Schema, table.Name)))
	body.WriteString("```\n\n")
	body.WriteString(fmt.Sprintf("Part of the [%s schema](/schemas/%s.md).\n", table.Schema, segment(table.Schema)))

	return w.write(filepath.Join(schemaDir, segment(table.Name)+".md"), body.String())
}

func (w bundleWriter) write(relPath, content string) error {
	path := filepath.Join(w.root, relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func frontmatter(fields map[string]any) string {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.SliceStable(keys, func(i, j int) bool {
		order := map[string]int{"type": 0, "title": 1, "description": 2, "resource": 3, "tags": 4, "timestamp": 5}
		oi, iok := order[keys[i]]
		oj, jok := order[keys[j]]
		if iok && jok {
			return oi < oj
		}
		if iok {
			return true
		}
		if jok {
			return false
		}
		return keys[i] < keys[j]
	})

	var b strings.Builder
	b.WriteString("---\n")
	for _, key := range keys {
		switch value := fields[key].(type) {
		case []string:
			b.WriteString(key + ": [")
			for i, item := range value {
				if i > 0 {
					b.WriteString(", ")
				}
				b.WriteString(yamlString(item))
			}
			b.WriteString("]\n")
		case string:
			b.WriteString(key + ": " + yamlString(value) + "\n")
		default:
			b.WriteString(fmt.Sprintf("%s: %v\n", key, value))
		}
	}
	b.WriteString("---\n\n")
	return b.String()
}

func yamlString(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func segment(value string) string {
	cleaned := unsafePathChars.ReplaceAllString(strings.TrimSpace(value), "_")
	cleaned = strings.Trim(cleaned, "._-")
	if cleaned == "" {
		cleaned = "unnamed"
	}
	if cleaned == value {
		return cleaned
	}
	sum := sha1.Sum([]byte(value))
	return cleaned + "-" + hex.EncodeToString(sum[:4])
}

func tablePath(table catalog.Table) string {
	return fmt.Sprintf("tables/%s/%s.md", segment(table.Schema), segment(table.Name))
}

func tableDescription(table catalog.Table) string {
	if strings.TrimSpace(table.Comment) != "" {
		return strings.TrimSpace(table.Comment)
	}
	return fmt.Sprintf("%s `%s.%s` with %d column(s)", strings.ToLower(title(table.Kind)), table.Schema, table.Name, len(table.Columns))
}

func tableTags(driver dbdriver.Driver, table catalog.Table) []string {
	tags := []string{driver.String(), "table", table.Schema}
	if strings.Contains(strings.ToLower(table.Kind), "view") {
		tags[1] = "view"
	}
	return tags
}

func resourceURI(bundle catalog.Bundle, schema string, table string) string {
	hostPort := bundle.Host
	if bundle.Port != 0 {
		hostPort = fmt.Sprintf("%s:%d", bundle.Host, bundle.Port)
	}
	if table != "" {
		return fmt.Sprintf("%s://%s/%s/%s/%s", bundle.Driver.String(), hostPort, bundle.Database, schema, table)
	}
	if schema != "" {
		return fmt.Sprintf("%s://%s/%s/%s", bundle.Driver.String(), hostPort, bundle.Database, schema)
	}
	return fmt.Sprintf("%s://%s/%s", bundle.Driver.String(), hostPort, bundle.Database)
}

func columnType(column catalog.Column) string {
	if column.ColumnType != "" {
		return column.ColumnType
	}
	return column.DataType
}

func columnDescription(column catalog.Column) string {
	parts := []string{}
	if column.Comment != "" {
		parts = append(parts, column.Comment)
	}
	if column.Extra != "" {
		parts = append(parts, column.Extra)
	}
	if column.MaxLength != "" {
		parts = append(parts, "max length "+column.MaxLength)
	}
	if column.NumericPrecision != "" {
		detail := "precision " + column.NumericPrecision
		if column.NumericScale != "" {
			detail += ", scale " + column.NumericScale
		}
		parts = append(parts, detail)
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, "; ")
}

func qualifiedTable(driver dbdriver.Driver, schema, table string) string {
	switch driver {
	case dbdriver.MySQL:
		return fmt.Sprintf("`%s`.`%s`", schema, table)
	default:
		return fmt.Sprintf(`"%s"."%s"`, schema, table)
	}
}

func markdownCodeOrDash(value string) string {
	if value == "" {
		return "-"
	}
	return "`" + escapePipes(value) + "`"
}

func boolWord(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func title(value string) string {
	value = strings.ReplaceAll(value, "_", " ")
	words := strings.Fields(strings.ToLower(value))
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	return strings.Join(words, " ")
}

func escapePipes(value string) string {
	return strings.ReplaceAll(value, "|", `\|`)
}
