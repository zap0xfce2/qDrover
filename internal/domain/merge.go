package domain

import "sort"

// MergePrompts führt zwei Änderungsstände (mine/theirs) relativ zu base zusammen;
// pro Prompt gewinnt mine nur, wenn ich ihn gegenüber base geändert habe.
func MergePrompts(base, mine, theirs []Prompt) []Prompt {
	baseLive := Board{Prompts: base}.LivePrompts()
	mineLive := Board{Prompts: mine}.LivePrompts()
	theirsLive := Board{Prompts: theirs}.LivePrompts()

	resolved := resolvePrompts(indexByID(base), indexByID(mine), indexByID(theirsLive))

	primary, secondary := mineLive, theirsLive
	if sameCommonOrder(baseLive, mineLive) {
		primary, secondary = theirsLive, mineLive
	}
	result := make([]Prompt, 0, len(resolved))
	seen := map[PromptID]bool{}
	for _, p := range append(append([]Prompt{}, primary...), secondary...) {
		merged, ok := resolved[p.ID]
		if !ok || seen[p.ID] {
			continue
		}
		seen[p.ID] = true
		merged.Position = PromptPosition(len(result))
		result = append(result, merged)
	}
	return result
}

func indexByID(prompts []Prompt) map[PromptID]Prompt {
	index := make(map[PromptID]Prompt, len(prompts))
	for _, p := range prompts {
		index[p.ID] = p
	}
	return index
}

func resolvePrompts(base, mine, theirs map[PromptID]Prompt) map[PromptID]Prompt {
	resolved := map[PromptID]Prompt{}
	for _, index := range []map[PromptID]Prompt{base, mine, theirs} {
		for id := range index {
			if p, ok := resolveOne(id, base, mine, theirs); ok {
				resolved[id] = p
			}
		}
	}
	return resolved
}

func hasKey(index map[PromptID]Prompt, id PromptID) bool {
	_, ok := index[id]
	return ok
}

func resolveOne(id PromptID, base, mine, theirs map[PromptID]Prompt) (Prompt, bool) {
	baseP, inBase := base[id]
	mineP, inMine := mine[id]
	if changedByMe(baseP, inBase, mineP, inMine) {
		return mineP, inMine && mineP.IsLive()
	}
	theirsP, inTheirs := theirs[id]
	return theirsP, inTheirs
}

func changedByMe(base Prompt, inBase bool, mine Prompt, inMine bool) bool {
	if !inBase {
		return inMine
	}
	if !inMine {
		return true
	}
	return mine.Content != base.Content || mine.Marked != base.Marked || mine.IsLive() != base.IsLive()
}

// sameCommonOrder: Reihenfolge der gemeinsamen IDs in a und b stimmt überein.
func sameCommonOrder(a, b []Prompt) bool {
	inA, inB := indexByID(a), indexByID(b)
	var orderA, orderB []PromptID
	for _, p := range a {
		if hasKey(inB, p.ID) {
			orderA = append(orderA, p.ID)
		}
	}
	for _, p := range b {
		if hasKey(inA, p.ID) {
			orderB = append(orderB, p.ID)
		}
	}
	return equalIDs(orderA, orderB)
}

func equalIDs(a, b []PromptID) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// MergeSession übernimmt Modi feldweise (mine nur bei Änderung gegenüber base)
// und vereinigt die Sendehistorie.
func MergeSession(base, mine, theirs Session) Session {
	merged := mine
	merged.PlanModeActive = mergeOptionalBool(base.PlanModeActive, mine.PlanModeActive, theirs.PlanModeActive)
	merged.ClearModeActive = mergeOptionalBool(base.ClearModeActive, mine.ClearModeActive, theirs.ClearModeActive)
	merged.SentHistory = mergeSentHistory(mine.SentHistory, theirs.SentHistory)
	return merged
}

func mergeOptionalBool(base, mine, theirs *bool) *bool {
	if optionalBoolEqual(base, mine) {
		return theirs
	}
	return mine
}

func optionalBoolEqual(a, b *bool) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func mergeSentHistory(mine, theirs []SentHistoryEntry) []SentHistoryEntry {
	seen := map[SentHistoryEntry]bool{}
	union := []SentHistoryEntry{}
	for _, e := range append(append([]SentHistoryEntry{}, mine...), theirs...) {
		if !seen[e] {
			seen[e] = true
			union = append(union, e)
		}
	}
	sort.SliceStable(union, func(i, j int) bool { return union[i].SentAt < union[j].SentAt })
	if len(union) > MaxSentHistoryEntries {
		union = union[len(union)-MaxSentHistoryEntries:]
	}
	return union
}
