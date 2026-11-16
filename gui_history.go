package main

import (
	"fyne.io/fyne/v2"
)

const maxHistoryCards = 20

// Retire visual cards, their animation data, and pending result references together.
func trimCardHistory(feed *fyne.Container, groups map[string][]*traceCard, active []*traceCard) []*traceCard {
	if len(feed.Objects) > maxHistoryCards {
		feed.Objects = append([]fyne.CanvasObject(nil), feed.Objects[:maxHistoryCards]...)
		feed.Refresh()
	}
	alive := map[fyne.CanvasObject]bool{}
	for _, o := range feed.Objects {
		alive[o] = true
	}
	return pruneCardReferences(groups, active, alive)
}

func pruneCardReferences(groups map[string][]*traceCard, active []*traceCard, alive map[fyne.CanvasObject]bool) []*traceCard {
	kept := make([]*traceCard, 0, maxHistoryCards)
	for _, c := range active {
		if alive[c.box] {
			kept = append(kept, c)
		}
	}
	for key, group := range groups {
		any := false
		for i, c := range group {
			if c != nil && alive[c.box] {
				any = true
			} else {
				group[i] = nil
			}
		}
		if !any {
			delete(groups, key)
		}
	}
	return kept
}
