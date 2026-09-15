package clipboard

import (
	"encoding/base64"
	"io"
	"os"
	"strings"
	"testing"
)

func TestClipboard_WriteAll_EmitsOSC52AndNeverErrors(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe erstellen: %v", err)
	}
	original := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = original }()

	writeErr := New().WriteAll("hallo welt")

	w.Close()
	os.Stdout = original
	var buf strings.Builder
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("pipe lesen: %v", err)
	}

	if writeErr != nil {
		t.Fatalf("erwarte nil trotz evtl. fehlendem lokalem Clipboard-Tool, habe: %v", writeErr)
	}

	want := "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte("hallo welt")) + "\x07"
	if !strings.Contains(buf.String(), want) {
		t.Fatalf("erwarte OSC52-Sequenz %q in Ausgabe, habe: %q", want, buf.String())
	}
}
