package herdr

import (
	"context"
	"strings"
	"testing"
	"time"
)

type fakeRunner struct {
	responses map[string][]byte
	lastArgs  []string
	calls     [][]string // alle Aufrufe in Reihenfolge, für Reihenfolge-Assertions
}

func (r *fakeRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	r.lastArgs = args
	r.calls = append(r.calls, args)
	key := args[0] + ":" + args[1]
	return r.responses[key], nil
}

func TestGateway_CurrentPane_ParsesPaneID(t *testing.T) {
	t.Setenv("HERDR_ENV", "1")
	runner := &fakeRunner{responses: map[string][]byte{
		"pane:current": []byte(`{"result":{"pane":{"pane_id":"pane-42"}}}`),
	}}
	g := NewGatewayWithRunner(runner)

	pane, err := g.CurrentPane(context.Background())
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if pane != "pane-42" {
		t.Fatalf("erwarte pane-42, habe %q", pane)
	}
}

func TestGateway_AgentPrompt_PassesPaneIDAndText(t *testing.T) {
	t.Setenv("HERDR_ENV", "1")
	runner := &fakeRunner{responses: map[string][]byte{}}
	g := NewGatewayWithRunner(runner)

	if err := g.AgentPrompt(context.Background(), "pane-42", "mein prompt"); err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	want := []string{"agent", "prompt", "pane-42", "mein prompt"}
	if len(runner.lastArgs) != len(want) {
		t.Fatalf("erwarte Args %v, habe %v", want, runner.lastArgs)
	}
	for i := range want {
		if runner.lastArgs[i] != want[i] {
			t.Fatalf("erwarte Args %v, habe %v", want, runner.lastArgs)
		}
	}
}

func TestGateway_CurrentPane_FailsWhenHerdrDisabled(t *testing.T) {
	t.Setenv("HERDR_ENV", "")
	g := NewGatewayWithRunner(&fakeRunner{})

	_, err := g.CurrentPane(context.Background())
	if err == nil {
		t.Fatal("erwarte Fehler, wenn HERDR_ENV nicht gesetzt ist")
	}
}

func TestGateway_CurrentPane_FailsOnEmptyPaneID(t *testing.T) {
	t.Setenv("HERDR_ENV", "1")
	runner := &fakeRunner{responses: map[string][]byte{
		"pane:current": []byte(`{"result":{"pane":{"pane_id":""}}}`),
	}}
	g := NewGatewayWithRunner(runner)

	_, err := g.CurrentPane(context.Background())
	if err == nil {
		t.Fatal("erwarte Fehler bei leerer pane-id")
	}
}

func TestGateway_NeighborPane_FailsOnEmptyPaneID(t *testing.T) {
	t.Setenv("HERDR_ENV", "1")
	runner := &fakeRunner{responses: map[string][]byte{
		"pane:neighbor": []byte(`{"result":{"neighbor":{"neighbor_pane_id":""}}}`),
	}}
	g := NewGatewayWithRunner(runner)

	_, err := g.NeighborPane(context.Background(), "pane-self", "up")
	if err == nil {
		t.Fatal("erwarte Fehler bei leerer pane-id")
	}
}

func TestGateway_NeighborPane_ParsesNeighborPaneID(t *testing.T) {
	t.Setenv("HERDR_ENV", "1")
	runner := &fakeRunner{responses: map[string][]byte{
		"pane:neighbor": []byte(`{"result":{"neighbor":{"neighbor_pane_id":"pane-42"}}}`),
	}}
	g := NewGatewayWithRunner(runner)

	pane, err := g.NeighborPane(context.Background(), "pane-self", "up")
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if pane != "pane-42" {
		t.Fatalf("erwarte pane-42, habe %q", pane)
	}
}

// TestGateway_NeighborPane_ParsesRealHerdrPayload nutzt eine echte,
// unverkürzte Antwort einer laufenden herdr-Installation (verifiziert
// 2026-09-11) als Regressionsschutz gegen erneute Struktur-Annahmen.
func TestGateway_NeighborPane_ParsesRealHerdrPayload(t *testing.T) {
	t.Setenv("HERDR_ENV", "1")
	realPayload := `{"id":"cli:pane:neighbor","result":{"neighbor":{"direction":"up","layout":{"area":{"height":67,"width":171,"x":0,"y":0},"focused_pane_id":"w5:pH","panes":[{"focused":false,"pane_id":"w5:p1","rect":{"height":49,"width":171,"x":0,"y":0}},{"focused":true,"pane_id":"w5:pH","rect":{"height":18,"width":171,"x":0,"y":49}}],"splits":[{"direction":"down","id":"split_0_root","ratio":0.73134327,"rect":{"height":67,"width":171,"x":0,"y":0}}],"tab_id":"w5:t1","workspace_id":"w5","zoomed":false},"neighbor_pane_id":"w5:p1","pane_id":"w5:pH"},"type":"pane_neighbor"}}`
	runner := &fakeRunner{responses: map[string][]byte{
		"pane:neighbor": []byte(realPayload),
	}}
	g := NewGatewayWithRunner(runner)

	pane, err := g.NeighborPane(context.Background(), "w5:pH", "up")
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if pane != "w5:p1" {
		t.Fatalf("erwarte w5:p1, habe %q", pane)
	}
}

func TestGateway_ResolveAndPromptWithPrefix_NoPrefix_WithoutDirection_SendsToCurrentPane(t *testing.T) {
	t.Setenv("HERDR_ENV", "1")
	runner := &fakeRunner{responses: map[string][]byte{
		"pane:current": []byte(`{"result":{"pane":{"pane_id":"pane-self"}}}`),
	}}
	g := NewGatewayWithRunner(runner)

	if err := g.ResolveAndPromptWithPrefix(context.Background(), "", nil, "mein prompt"); err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	want := []string{"agent", "prompt", "pane-self", "mein prompt"}
	if len(runner.lastArgs) != len(want) {
		t.Fatalf("erwarte Args %v, habe %v", want, runner.lastArgs)
	}
	for i := range want {
		if runner.lastArgs[i] != want[i] {
			t.Fatalf("erwarte Args %v, habe %v", want, runner.lastArgs)
		}
	}
}

func TestGateway_ResolveAndPromptWithPrefix_NoPrefix_WithDirection_SendsToNeighborPane(t *testing.T) {
	t.Setenv("HERDR_ENV", "1")
	runner := &fakeRunner{responses: map[string][]byte{
		"pane:current":  []byte(`{"result":{"pane":{"pane_id":"pane-self"}}}`),
		"pane:neighbor": []byte(`{"result":{"neighbor":{"neighbor_pane_id":"pane-neighbor"}}}`),
	}}
	g := NewGatewayWithRunner(runner)

	if err := g.ResolveAndPromptWithPrefix(context.Background(), "up", nil, "mein prompt"); err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	want := []string{"agent", "prompt", "pane-neighbor", "mein prompt"}
	if len(runner.lastArgs) != len(want) {
		t.Fatalf("erwarte Args %v, habe %v", want, runner.lastArgs)
	}
	for i := range want {
		if runner.lastArgs[i] != want[i] {
			t.Fatalf("erwarte Args %v, habe %v", want, runner.lastArgs)
		}
	}
}

func TestGateway_CurrentPane_SurfacesRpcErrorMessage(t *testing.T) {
	t.Setenv("HERDR_ENV", "1")
	runner := &fakeRunner{responses: map[string][]byte{
		"pane:current": []byte(`{"id":"req_1","error":{"code":"not_found","message":"pane not found"}}`),
	}}
	g := NewGatewayWithRunner(runner)

	_, err := g.CurrentPane(context.Background())
	if err == nil || !strings.Contains(err.Error(), "pane not found") {
		t.Fatalf("erwarte Fehler mit 'pane not found', habe %v", err)
	}
}

func TestGateway_NeighborPane_SurfacesRpcErrorMessage(t *testing.T) {
	t.Setenv("HERDR_ENV", "1")
	runner := &fakeRunner{responses: map[string][]byte{
		"pane:neighbor": []byte(`{"id":"req_1","error":{"code":"not_found","message":"pane not found"}}`),
	}}
	g := NewGatewayWithRunner(runner)

	_, err := g.NeighborPane(context.Background(), "pane-self", "up")
	if err == nil || !strings.Contains(err.Error(), "pane not found") {
		t.Fatalf("erwarte Fehler mit 'pane not found', habe %v", err)
	}
}

func TestGateway_ResolveAndPromptWithPrefix_NoPrefix_FailsWhenHerdrDisabled(t *testing.T) {
	t.Setenv("HERDR_ENV", "")
	g := NewGatewayWithRunner(&fakeRunner{})

	if err := g.ResolveAndPromptWithPrefix(context.Background(), "", nil, "text"); err == nil {
		t.Fatal("erwarte Fehler, wenn HERDR_ENV nicht gesetzt ist")
	}
}

func TestGateway_ResolveAndPromptWithPrefix_SendsPrefixThenActualText_ToSamePane(t *testing.T) {
	t.Setenv("HERDR_ENV", "1")
	old := prefixSettleDelay
	prefixSettleDelay = time.Millisecond
	defer func() { prefixSettleDelay = old }()

	runner := &fakeRunner{responses: map[string][]byte{
		"pane:current": []byte(`{"result":{"pane":{"pane_id":"pane-self"}}}`),
	}}
	g := NewGatewayWithRunner(runner)

	if err := g.ResolveAndPromptWithPrefix(context.Background(), "", []string{"/plan"}, "mein prompt"); err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}

	// calls[0] = "pane current" (Auflösung), calls[1] = Präfix, calls[2] = eigentlicher Text.
	if len(runner.calls) != 3 {
		t.Fatalf("erwarte 3 Aufrufe (pane current, Präfix, text), habe %d: %v", len(runner.calls), runner.calls)
	}
	prefixCall := runner.calls[1]
	wantPrefix := []string{"agent", "prompt", "pane-self", "/plan"}
	for i := range wantPrefix {
		if prefixCall[i] != wantPrefix[i] {
			t.Fatalf("erwarte zweiten Aufruf %v, habe %v", wantPrefix, prefixCall)
		}
	}
	textCall := runner.calls[2]
	wantText := []string{"agent", "prompt", "pane-self", "mein prompt"}
	for i := range wantText {
		if textCall[i] != wantText[i] {
			t.Fatalf("erwarte dritten Aufruf %v, habe %v", wantText, textCall)
		}
	}
}

func TestGateway_ResolveAndPromptWithPrefix_WithPrefix_FailsWhenHerdrDisabled(t *testing.T) {
	t.Setenv("HERDR_ENV", "")
	g := NewGatewayWithRunner(&fakeRunner{})

	if err := g.ResolveAndPromptWithPrefix(context.Background(), "", []string{"/plan"}, "text"); err == nil {
		t.Fatal("erwarte Fehler, wenn HERDR_ENV nicht gesetzt ist")
	}
}

func TestGateway_ResolveAndPromptWithPrefix_SendsBothPrefixesInOrder_ThenActualText_ResolvesPaneOnce(t *testing.T) {
	t.Setenv("HERDR_ENV", "1")
	old := prefixSettleDelay
	prefixSettleDelay = time.Millisecond
	defer func() { prefixSettleDelay = old }()

	runner := &fakeRunner{responses: map[string][]byte{
		"pane:current": []byte(`{"result":{"pane":{"pane_id":"pane-self"}}}`),
	}}
	g := NewGatewayWithRunner(runner)

	if err := g.ResolveAndPromptWithPrefix(context.Background(), "", []string{"/clear", "/plan"}, "mein prompt"); err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}

	// calls[0] = "pane current" (nur einmal aufgelöst), calls[1] = /clear, calls[2] = /plan, calls[3] = Text.
	if len(runner.calls) != 4 {
		t.Fatalf("erwarte 4 Aufrufe (pane current, /clear, /plan, text), habe %d: %v", len(runner.calls), runner.calls)
	}
	wantSequence := [][]string{
		{"pane", "current"},
		{"agent", "prompt", "pane-self", "/clear"},
		{"agent", "prompt", "pane-self", "/plan"},
		{"agent", "prompt", "pane-self", "mein prompt"},
	}
	for i, want := range wantSequence {
		got := runner.calls[i]
		for j := range want {
			if got[j] != want[j] {
				t.Fatalf("erwarte Aufruf %d %v, habe %v", i, want, got)
			}
		}
	}
}
