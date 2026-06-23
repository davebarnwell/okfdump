package okf

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	"github.com/davebarnwell/okfdump/internal/catalog"
)

var unsafePathChars = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

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
