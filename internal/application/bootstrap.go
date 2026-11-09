package application

import (
	"sort"

	"qdrover/internal/domain"
)

// MigrateSessions fasst alle Alt-Sessions zu einem globalen Board zusammen.
func MigrateSessions(boards []domain.Board) domain.Board {
	byAge := append([]domain.Board{}, boards...)
	sort.SliceStable(byAge, func(i, j int) bool {
		return byAge[i].Session.CreatedAt < byAge[j].Session.CreatedAt
	})
	merged := domain.Board{Session: mostRecentlyOpened(byAge).Session}
	seen := map[string]bool{}
	for _, board := range byAge {
		for _, prompt := range board.LivePrompts() {
			if seen[prompt.Content] {
				continue
			}
			seen[prompt.Content] = true
			prompt.Scope = domain.ScopeGlobal
			prompt.Position = domain.PromptPosition(len(merged.Prompts))
			merged.Prompts = append(merged.Prompts, prompt)
		}
	}
	return merged
}

func mostRecentlyOpened(boards []domain.Board) domain.Board {
	var latest domain.Board
	for i, board := range boards {
		if i == 0 || board.Session.LastOpenedAt > latest.Session.LastOpenedAt {
			latest = board
		}
	}
	return latest
}
