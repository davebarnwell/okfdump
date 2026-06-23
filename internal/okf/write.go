package okf

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/davebarnwell/okfdump/internal/catalog"
)

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
	body.WriteString(frontmatter(frontmatterFields{
		Type:        fmt.Sprintf("%s Database", w.bundle.Driver.DisplayName()),
		Title:       w.bundle.Database,
		Description: fmt.Sprintf("%s database `%s` with %d schema(s).", w.bundle.Driver.DisplayName(), w.bundle.Database, len(w.bundle.Schemas)),
		Resource:    resourceURI(w.bundle, "", ""),
		Tags:        []string{w.bundle.Driver.String(), "database"},
		Timestamp:   w.bundle.GeneratedAt.Format("2006-01-02T15:04:05Z"),
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
		body.WriteString(frontmatter(frontmatterFields{
			Type:        fmt.Sprintf("%s Schema", w.bundle.Driver.DisplayName()),
			Title:       schema.Name,
			Description: fmt.Sprintf("Schema `%s` in database `%s` with %d table(s) and view(s).", schema.Name, w.bundle.Database, len(schema.Tables)),
			Resource:    resourceURI(w.bundle, schema.Name, ""),
			Tags:        []string{w.bundle.Driver.String(), "schema", schema.Name},
			Timestamp:   w.bundle.GeneratedAt.Format("2006-01-02T15:04:05Z"),
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
	body.WriteString(frontmatter(frontmatterFields{
		Type:        fmt.Sprintf("%s %s", w.bundle.Driver.DisplayName(), title(table.Kind)),
		Title:       fmt.Sprintf("%s.%s", table.Schema, table.Name),
		Description: tableDescription(table),
		Resource:    resourceURI(w.bundle, table.Schema, table.Name),
		Tags:        tableTags(w.bundle.Driver, table),
		Timestamp:   w.bundle.GeneratedAt.Format("2006-01-02T15:04:05Z"),
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
