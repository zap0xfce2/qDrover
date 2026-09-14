package application

import (
	"context"
	"testing"

	"qdrover/internal/domain"
	"qdrover/internal/ports"
)

type fakeHerdr struct {
	current        ports.PaneInfo
	neighbor       ports.PaneInfo
	promptedTo     string
	promptedTxt    string
	calledPrefixes []string
}

func (f *fakeHerdr) CurrentPane(ctx context.Context) (ports.PaneInfo, error) {
	return f.current, nil
}

func (f *fakeHerdr) NeighborPane(ctx context.Context, paneID string, direction ports.Direction) (ports.PaneInfo, error) {
	return f.neighbor, nil
}

func (f *fakeHerdr) AgentPrompt(ctx context.Context, paneID string, text string) error {
	f.promptedTo = paneID
	f.promptedTxt = text
	return nil
}

func (f *fakeHerdr) ResolveAndPromptWithPrefix(ctx context.Context, direction ports.Direction, prefixCommands []string, text string) error {
	f.calledPrefixes = prefixCommands
	target := f.current
	if direction != "" {
		target = f.neighbor
	}
	return f.AgentPrompt(ctx, target.ID, text)
}

func TestExecutor_SendDispatch_DiscoversNeighborThenSendsPrompt(t *testing.T) {
	store := newFakeStore()
	herdr := &fakeHerdr{
		current:  ports.PaneInfo{ID: "pane-self"},
		neighbor: ports.PaneInfo{ID: "pane-neighbor"},
	}
	executor := NewExecutor(store, herdr)

	err := executor.Execute(context.Background(), []Effect{
		SendDispatch{Direction: ports.DirectionUp, Text: "mein prompt"},
	})
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if herdr.promptedTo != "pane-neighbor" {
		t.Fatalf("erwarte Prompt an pane-neighbor, ging an %q", herdr.promptedTo)
	}
	if herdr.promptedTxt != "mein prompt" {
		t.Fatalf("erwarte Text 'mein prompt', habe %q", herdr.promptedTxt)
	}
}

func TestExecutor_SendDispatch_WithPrefixCommands_PassesPrefixesToGateway(t *testing.T) {
	store := newFakeStore()
	herdr := &fakeHerdr{current: ports.PaneInfo{ID: "pane-self"}}
	executor := NewExecutor(store, herdr)

	err := executor.Execute(context.Background(), []Effect{
		SendDispatch{Text: "mein prompt", PrefixCommands: []string{"/clear", "/plan"}},
	})
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	want := []string{"/clear", "/plan"}
	if len(herdr.calledPrefixes) != len(want) || herdr.calledPrefixes[0] != want[0] || herdr.calledPrefixes[1] != want[1] {
		t.Fatalf("erwarte Präfixe %v an Gateway übergeben, habe %v", want, herdr.calledPrefixes)
	}
}

func TestExecutor_SendDispatch_WithoutPrefixCommands_PassesEmptyPrefixesToGateway(t *testing.T) {
	store := newFakeStore()
	herdr := &fakeHerdr{current: ports.PaneInfo{ID: "pane-self"}}
	executor := NewExecutor(store, herdr)

	err := executor.Execute(context.Background(), []Effect{
		SendDispatch{Text: "mein prompt"},
	})
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if len(herdr.calledPrefixes) != 0 {
		t.Fatalf("erwarte keine Präfixe an Gateway übergeben, habe %v", herdr.calledPrefixes)
	}
}

func TestExecutor_PersistBoard_SavesToStore(t *testing.T) {
	store := newFakeStore()
	herdr := &fakeHerdr{}
	executor := NewExecutor(store, herdr)

	board := domain.Board{Session: domain.Session{ID: "s1"}}
	err := executor.Execute(context.Background(), []Effect{PersistBoard{Board: board}})
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if _, ok := store.boards["s1"]; !ok {
		t.Fatal("erwarte gespeicherte Session s1")
	}
}
