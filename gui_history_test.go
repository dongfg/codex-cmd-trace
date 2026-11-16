package main

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"testing"
)

func TestHistoryRetiresCardsAndLateResultReferences(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	feed := container.NewVBox()
	groups := map[string][]*traceCard{}
	var active []*traceCard
	for i := 0; i < 25; i++ {
		c := newTraceCard(fmt.Sprint(i), "12:00:00", "echo test")
		feed.Objects = append([]fyne.CanvasObject{c.box}, feed.Objects...)
		groups[fmt.Sprint(i)] = []*traceCard{c}
		active = append(active, c)
		active = trimCardHistory(feed, groups, active)
	}
	if len(feed.Objects) != 20 || len(active) != 20 || len(groups) != 20 {
		t.Fatalf("history leaked: cards=%d animations=%d pending=%d", len(feed.Objects), len(active), len(groups))
	}
	for i := 0; i < 5; i++ {
		if _, ok := groups[fmt.Sprint(i)]; ok {
			t.Fatal("late result can resurrect retired card", i)
		}
	}
	if feed.Objects[0] != groups["24"][0].box || feed.Objects[19] != groups["5"][0].box {
		t.Fatal("latest-first ordering lost")
	}
	// Keep original result indexes when only part of a multi-command call survives.
	old := newTraceCard("old", "", "old")
	kept := groups["24"][0]
	groups["batch"] = []*traceCard{old, kept}
	active = trimCardHistory(feed, groups, active)
	if groups["batch"][0] != nil || groups["batch"][1] != kept {
		t.Fatal("partial batch result indexes changed")
	}
}
