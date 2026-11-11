package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"qdrover/internal/adapters/sysclock"
	"qdrover/internal/domain"
	"qdrover/internal/ports"
)

// runEditCommand verdrahtet `qdrover edit` mit echtem Dateisystem und Uhr.
func runEditCommand(args []string, stdin io.Reader, out io.Writer) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("arbeitsverzeichnis ermitteln: %w", err)
	}
	stateRoot, err := defaultStateDir()
	if err != nil {
		return err
	}
	return runEdit(args, stdin, out, stateRoot, cwd, sysclock.New())
}

// runEdit ersetzt den Inhalt eines lokalen Prompts (per ID oder Roadmap-Nummer)
// und stößt laufende TUIs zum Abgleich an.
func runEdit(args []string, stdin io.Reader, out io.Writer, stateRoot, cwd string, clock ports.Clock) error {
	if len(args) == 0 {
		return errors.New("aufruf: qdrover edit <prompt-id|nummer> [text]")
	}
	text, err := resolveQuery(args[1:], stdin)
	if err != nil {
		return err
	}
	board, store, err := bootBoard(stateRoot, cwd)
	if err != nil {
		return err
	}
	id, err := resolveLocalPromptRef(board, args[0])
	if err != nil {
		return err
	}
	board, err = board.EditPrompt(id, strings.TrimSpace(text), clock.Now())
	if err != nil {
		return err
	}
	if _, err := store.Save(board); err != nil {
		return fmt.Errorf("prompt bearbeiten: %w", err)
	}
	fmt.Fprintln(out, "geändert:", id)
	notifyInstances(filepath.Join(stateRoot, runDirName))
	return nil
}

// resolveLocalPromptRef löst eine Prompt-ID oder eine Roadmap-Nummer ("3", "#3" bzw. "3.")
// zur ID eines lebenden lokalen Prompts auf. Die Nummer steht als "<n>. " (Roadmap-Skill)
// oder als älteres "#<n> " am Anfang der ersten Zeile.
func resolveLocalPromptRef(board domain.Board, ref string) (domain.PromptID, error) {
	number := strings.TrimSuffix(strings.TrimPrefix(ref, "#"), ".")
	if _, err := strconv.Atoi(number); err != nil {
		if !hasLocalPromptID(board, domain.PromptID(ref)) {
			return "", fmt.Errorf("kein lokaler prompt mit id %s", ref)
		}
		return domain.PromptID(ref), nil
	}
	return findLocalPromptByNumber(board, number)
}

func findLocalPromptByNumber(board domain.Board, number string) (domain.PromptID, error) {
	prefix := "#" + number
	var matches []domain.PromptID
	for _, p := range board.LivePrompts() {
		headline, _, _ := strings.Cut(p.Content, "\n")
		if p.IsLocal() && (hasNumberPrefix(headline, prefix) || hasNumberPrefix(headline, number+".")) {
			matches = append(matches, p.ID)
		}
	}
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("kein lokaler prompt mit nummer %s", prefix)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("nummer %s ist mehrdeutig (%d prompts), bitte per id angeben", prefix, len(matches))
	}
}

// hasNumberPrefix prüft, ob die Headline genau mit prefix beginnt — "12." soll nicht auf "1." passen.
func hasNumberPrefix(headline, prefix string) bool {
	return headline == prefix || strings.HasPrefix(headline, prefix+" ")
}
