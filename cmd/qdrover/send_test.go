package main

import (
	"strings"
	"testing"
)

func TestResolveQuery_PrefersArgsOverStdin(t *testing.T) {
	text, err := resolveQuery([]string{"hallo", "welt"}, strings.NewReader("ignoriert"))
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if text != "hallo welt" {
		t.Fatalf("erwarte 'hallo welt', habe %q", text)
	}
}

func TestResolveQuery_FallsBackToStdin(t *testing.T) {
	text, err := resolveQuery(nil, strings.NewReader("  aus stdin  "))
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if text != "aus stdin" {
		t.Fatalf("erwarte 'aus stdin', habe %q", text)
	}
}

func TestResolveQuery_FailsWithoutArgsOrStdin(t *testing.T) {
	_, err := resolveQuery(nil, strings.NewReader("   "))
	if err == nil {
		t.Fatal("erwarte Fehler bei leerer Query")
	}
}

// Bei deaktiviertem HERDR_ENV schlägt Gateway.CurrentPane sofort fehl, noch
// bevor "herdr" als Subprozess ausgeführt wird - der Command muss diesen
// Fehler durchreichen statt zu hängen oder einen echten Subprozess zu starten.
func TestSendCmd_FailsFastWhenHerdrDisabled(t *testing.T) {
	t.Setenv("HERDR_ENV", "")

	err := runSend([]string{"hallo welt"}, strings.NewReader(""))
	if err == nil {
		t.Fatal("erwarte Fehler, wenn HERDR_ENV nicht gesetzt ist")
	}
	if !strings.Contains(err.Error(), "herdr ist deaktiviert") {
		t.Fatalf("erwarte Fehlermeldung über deaktiviertes herdr, habe %q", err.Error())
	}
}

func TestRun_UnknownSubcommand_ReturnsError(t *testing.T) {
	err := run([]string{"bogus"})
	if err == nil {
		t.Fatal("erwarte Fehler bei unbekanntem Subcommand")
	}
}
