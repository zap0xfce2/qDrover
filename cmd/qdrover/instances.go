package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	socketSuffix = ".sock"
	unixNetwork  = "unix"
	// notifyTimeout begrenzt das Warten auf eine hängende Instanz pro Socket.
	notifyTimeout = 500 * time.Millisecond
)

// listenForRefresh öffnet für eine laufende TUI einen Unix-Socket in runDir;
// jede Verbindung darauf (von `qdrover add`) löst onRefresh aus. Socket statt
// Signal an eine PID: eine wiederverwendete PID eines abgestürzten Prozesses
// würde sonst ein fremdes Programm treffen.
func listenForRefresh(runDir string, pid int, onRefresh func()) (func(), error) {
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return nil, fmt.Errorf("run-verzeichnis anlegen: %w", err)
	}
	path := filepath.Join(runDir, strconv.Itoa(pid)+socketSuffix)
	_ = os.Remove(path) // Rest eines abgestürzten Prozesses mit derselben PID
	listener, err := net.Listen(unixNetwork, path)
	if err != nil {
		return nil, fmt.Errorf("refresh-socket öffnen: %w", err)
	}
	go acceptRefreshes(listener, onRefresh)
	return func() { _ = listener.Close() }, nil
}

func acceptRefreshes(listener net.Listener, onRefresh func()) {
	for {
		conn, err := listener.Accept()
		if errors.Is(err, net.ErrClosed) {
			return
		}
		if err != nil {
			continue
		}
		_ = conn.Close()
		onRefresh()
	}
}

// notifyInstances stößt alle laufenden TUIs zum Abgleich an und räumt
// Sockets weg, hinter denen kein Prozess mehr lauscht. Best effort: ein
// fehlgeschlagener Anstoß darf `qdrover add` nicht scheitern lassen.
func notifyInstances(runDir string) {
	entries, err := os.ReadDir(runDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), socketSuffix) {
			continue
		}
		path := filepath.Join(runDir, entry.Name())
		conn, err := net.DialTimeout(unixNetwork, path, notifyTimeout)
		if err != nil {
			_ = os.Remove(path)
			continue
		}
		_ = conn.Close()
	}
}
