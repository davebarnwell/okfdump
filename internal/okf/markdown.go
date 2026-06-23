package okf

import (
	"fmt"
	"strings"

	"github.com/davebarnwell/okfdump/internal/catalog"
	"github.com/davebarnwell/okfdump/internal/dbdriver"
)

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
