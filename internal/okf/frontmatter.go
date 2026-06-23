package okf

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type frontmatterFields struct {
	Type        string   `yaml:"type"`
	Title       string   `yaml:"title"`
	Description string   `yaml:"description,omitempty"`
	Resource    string   `yaml:"resource,omitempty"`
	Tags        []string `yaml:"tags,omitempty"`
	Timestamp   string   `yaml:"timestamp,omitempty"`
}

func frontmatter(fields frontmatterFields) string {
	encoded, err := yaml.Marshal(fields)
	if err != nil {
		panic(fmt.Sprintf("marshal frontmatter: %v", err))
	}

	var b strings.Builder
	b.WriteString("---\n")
	b.Write(encoded)
	b.WriteString("---\n\n")
	return b.String()
}
