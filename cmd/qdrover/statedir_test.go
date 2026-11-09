package main

import (
	"strings"
	"testing"
)

func TestDefaultStateDir_EndsWithExpectedSuffix(t *testing.T) {
	dir, err := defaultStateDir()
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if !strings.HasSuffix(dir, "/.local/state/qdrover") {
		t.Fatalf("erwarte Suffix '/.local/state/qdrover', habe %q", dir)
	}
}
