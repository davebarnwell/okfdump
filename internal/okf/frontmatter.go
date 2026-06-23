package okf

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

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
