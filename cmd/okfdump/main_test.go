package main

import (
	"strings"
	"testing"
)

func TestRootCommandRejectsPositionalArgs(t *testing.T) {
	cmd := newRootCommand()
	cmd.SetArgs([]string{"unexpected"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected positional argument error")
	}
	if !strings.Contains(err.Error(), "unknown command") && !strings.Contains(err.Error(), "accepts 0 arg") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRootCommandHelpIncludesFlags(t *testing.T) {
	cmd := newRootCommand()
	cmd.SetArgs([]string{"--help"})

	var out strings.Builder
	cmd.SetOut(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"--driver", "--ssh-host", "--table", "-o, --out"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("expected help to include %q:\n%s", want, out.String())
		}
	}
}
