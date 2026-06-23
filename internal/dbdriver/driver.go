package dbdriver

import (
	"fmt"
	"strings"
)

type Driver string

const (
	MySQL    Driver = "mysql"
	Postgres Driver = "postgres"
)

func Parse(value string) (Driver, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(MySQL):
		return MySQL, nil
	case string(Postgres), "postgresql":
		return Postgres, nil
	default:
		return "", fmt.Errorf("unsupported --driver %q", value)
	}
}

func (d Driver) String() string {
	return string(d)
}

func (d *Driver) Set(value string) error {
	parsed, err := Parse(value)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

func (d Driver) Type() string {
	return "driver"
}

func (d Driver) DefaultPort() int {
	switch d {
	case MySQL:
		return 3306
	case Postgres:
		return 5432
	default:
		return 0
	}
}

func (d Driver) SQLDriverName() string {
	switch d {
	case MySQL:
		return "mysql"
	case Postgres:
		return "pgx"
	default:
		return string(d)
	}
}

func (d Driver) DisplayName() string {
	switch d {
	case MySQL:
		return "MySQL"
	case Postgres:
		return "Postgres"
	default:
		return title(string(d))
	}
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
