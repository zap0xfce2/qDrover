package application

import "qdrover/internal/domain"

// History sichert Snapshots von domain.Board.Prompts (nicht des ganzen
// Boards) — Session (Plan-/Clear-Modus, Sendehistory) bleibt damit von
// Undo/Redo unberührt.
type History struct {
	past     [][]domain.Prompt
	future   [][]domain.Prompt
	maxDepth int
}

func NewHistory(maxDepth int) *History {
	return &History{maxDepth: maxDepth}
}

func (h *History) Push(prompts []domain.Prompt) {
	h.past = append(h.past, prompts)
	if len(h.past) > h.maxDepth {
		h.past = h.past[len(h.past)-h.maxDepth:]
	}
	h.future = nil
}

func (h *History) Undo(current []domain.Prompt) ([]domain.Prompt, bool) {
	if len(h.past) == 0 {
		return current, false
	}
	prev := h.past[len(h.past)-1]
	h.past = h.past[:len(h.past)-1]
	h.future = append(h.future, current)
	return prev, true
}

func (h *History) Redo(current []domain.Prompt) ([]domain.Prompt, bool) {
	if len(h.future) == 0 {
		return current, false
	}
	next := h.future[len(h.future)-1]
	h.future = h.future[:len(h.future)-1]
	h.past = append(h.past, current)
	return next, true
}
