package domain

import (
	"fmt"
	"sort"
)

type Board struct {
	Session Session
	Prompts []Prompt
}

func (b Board) LivePrompts() []Prompt {
	live := make([]Prompt, 0, len(b.Prompts))
	for _, t := range b.Prompts {
		if t.IsLive() {
			live = append(live, t)
		}
	}
	sort.Slice(live, func(i, j int) bool { return live[i].Position < live[j].Position })
	return live
}

func (b Board) AddPrompt(id PromptID, content string, now Timestamp) Board {
	live := b.LivePrompts()
	newPrompts := append([]Prompt{}, b.Prompts...)
	newPrompts = append(newPrompts, Prompt{
		ID:        id,
		Content:   content,
		Position:  PromptPosition(len(live)),
		CreatedAt: now,
		UpdatedAt: now,
	})
	b.Prompts = newPrompts
	return b
}

func (b Board) EditPrompt(id PromptID, content string, now Timestamp) (Board, error) {
	newPrompts := append([]Prompt{}, b.Prompts...)
	for i, t := range newPrompts {
		if t.ID == id && t.IsLive() {
			newPrompts[i].Content = content
			newPrompts[i].UpdatedAt = now
			b.Prompts = newPrompts
			return b, nil
		}
	}
	return b, fmt.Errorf("prompt %s nicht gefunden", id)
}

func (b Board) ToggleMarked(id PromptID) (Board, error) {
	newPrompts := append([]Prompt{}, b.Prompts...)
	for i, t := range newPrompts {
		if t.ID == id && t.IsLive() {
			newPrompts[i].Marked = !newPrompts[i].Marked
			b.Prompts = newPrompts
			return b, nil
		}
	}
	return b, fmt.Errorf("prompt %s nicht gefunden", id)
}

func (b Board) DeletePrompt(id PromptID, now Timestamp) (Board, error) {
	newPrompts := append([]Prompt{}, b.Prompts...)
	found := false
	for i, t := range newPrompts {
		if t.ID == id && t.IsLive() {
			deletedAt := now
			newPrompts[i].DeletedAt = &deletedAt
			newPrompts[i].UpdatedAt = now
			found = true
			break
		}
	}
	if !found {
		return b, fmt.Errorf("prompt %s nicht gefunden", id)
	}
	b.Prompts = reindexLivePositions(newPrompts)
	return b, nil
}

func reindexLivePositions(prompts []Prompt) []Prompt {
	liveIdx := make([]int, 0, len(prompts))
	for i, t := range prompts {
		if t.IsLive() {
			liveIdx = append(liveIdx, i)
		}
	}
	sort.Slice(liveIdx, func(i, j int) bool {
		return prompts[liveIdx[i]].Position < prompts[liveIdx[j]].Position
	})
	for pos, idx := range liveIdx {
		prompts[idx].Position = PromptPosition(pos)
	}
	return prompts
}

func (b Board) MovePrompt(id PromptID, to PromptPosition, now Timestamp) (Board, error) {
	live := b.LivePrompts()
	fromIdx := -1
	for i, t := range live {
		if t.ID == id {
			fromIdx = i
			break
		}
	}
	if fromIdx == -1 {
		return b, fmt.Errorf("prompt %s nicht gefunden", id)
	}

	target := int(to)
	if target < 0 {
		target = 0
	}
	if target > len(live)-1 {
		target = len(live) - 1
	}

	moved := live[fromIdx]
	live = append(live[:fromIdx], live[fromIdx+1:]...)
	tail := append([]Prompt{moved}, live[target:]...)
	live = append(live[:target], tail...)

	newPrompts := append([]Prompt{}, b.Prompts...)
	for pos, t := range live {
		for i := range newPrompts {
			if newPrompts[i].ID == t.ID {
				newPrompts[i].Position = PromptPosition(pos)
				if t.ID == id {
					newPrompts[i].UpdatedAt = now
				}
			}
		}
	}
	b.Prompts = newPrompts
	return b, nil
}
