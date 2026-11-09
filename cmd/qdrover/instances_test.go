package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

const refreshWait = 2 * time.Second

// shortRunDir: Unix-Socket-Pfade sind auf ~104 Zeichen begrenzt, t.TempDir() unter macOS ist zu lang.
func shortRunDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "qdr")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

func TestNotifyInstances_TriggersRefreshOfListeningInstance(t *testing.T) {
	runDir := shortRunDir(t)
	refreshed := make(chan struct{}, 1)
	cleanup, err := listenForRefresh(runDir, os.Getpid(), func() { refreshed <- struct{}{} })
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	notifyInstances(runDir)

	select {
	case <-refreshed:
	case <-time.After(refreshWait):
		t.Fatal("erwarte Refresh der lauschenden Instanz")
	}
}

func TestListenForRefresh_CleanupRemovesSocket(t *testing.T) {
	runDir := shortRunDir(t)
	cleanup, err := listenForRefresh(runDir, os.Getpid(), func() {})
	if err != nil {
		t.Fatal(err)
	}

	cleanup()

	entries, _ := os.ReadDir(runDir)
	if len(entries) != 0 {
		t.Fatalf("erwarte leeres run-Verzeichnis nach Cleanup, habe %v", entries)
	}
}

func TestNotifyInstances_RemovesStaleSocket(t *testing.T) {
	runDir := shortRunDir(t)
	stale := filepath.Join(runDir, "999999"+socketSuffix)
	if err := os.WriteFile(stale, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	notifyInstances(runDir)

	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("erwarte entfernten verwaisten Socket, habe err=%v", err)
	}
}
