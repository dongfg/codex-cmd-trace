package main

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"testing"
	"time"
)

func testDetachedDeck(t *testing.T) (*detachedDeck, func()) {
	t.Helper()
	a := test.NewApp()
	main := a.NewWindow("main")
	main.Resize(fyne.NewSize(300, 500))
	feed := container.NewVBox()
	d := newDetachedDeck(a, main, feed, nil, func() float32 { return 200 })
	d.desktop = func() (floatingDesktop, error) {
		return floatingDesktop{Main: screenRect{600, 0, 900, 500}, Areas: []floatingArea{{screenRect{0, 0, 900, 600}, 1}}}, nil
	}
	return d, func() { d.Shutdown(); a.Quit() }
}

func TestDetachedRapidToggleAndGlobalHistoryLimit(t *testing.T) {
	d, cleanup := testDetachedDeck(t)
	defer cleanup()
	if err := d.SetEnabled(true); err != nil {
		t.Fatal(err)
	}
	old := newTraceCard("old", "", "echo")
	d.Add(old)
	now := time.Now()
	d.Tick(now)
	if err := d.SetEnabled(false); err != nil {
		t.Fatal(err)
	}
	d.Tick(now.Add(time.Millisecond))
	if err := d.SetEnabled(true); err != nil {
		t.Fatal(err)
	}
	if d.entries[0].detached || d.entries[0].window != nil || len(d.closing) != 0 || d.feed.Objects[0] != old.box {
		t.Fatal("rapid re-enable converted or stranded an existing card")
	}
	for i := 0; i < 25; i++ {
		d.Add(newTraceCard(fmt.Sprint(i), "", "echo"))
	}
	if len(d.entries) != maxHistoryCards {
		t.Fatal("global history cap exceeded", len(d.entries))
	}
	for _, e := range d.entries {
		if !e.detached {
			t.Fatal("new command did not follow enabled mode")
		}
	}
}

func TestDetachedModeKeepsExistingAndRestoresChronology(t *testing.T) {
	d, cleanup := testDetachedDeck(t)
	defer cleanup()
	old := newTraceCard("old", "", "old command")
	d.Add(old)
	if d.enabled {
		t.Fatal("mode must default off")
	}
	if err := d.SetEnabled(true); err != nil {
		t.Fatal(err)
	}
	first := newTraceCard("one", "", "first new command")
	second := newTraceCard("two", "", "second new command")
	d.Add(first)
	d.Add(second)
	if len(d.feed.Objects) != 1 || d.feed.Objects[0] != old.box {
		t.Fatal("existing card moved or new cards remained in main")
	}
	now := time.Now()
	d.Tick(now)
	for _, e := range d.entries[:2] {
		if e.window == nil {
			t.Fatal("missing independent window")
		}
		root := e.window.Content().(*fyne.Container)
		if root.Objects[1].(*container.Scroll).Content != e.card.box {
			t.Fatal("independent content is not the exact card")
		}
	}
	if err := d.SetEnabled(false); err != nil {
		t.Fatal(err)
	}
	latest := newTraceCard("latest", "", "arrived while closing")
	d.Add(latest)
	for i := 0; i < 8; i++ {
		d.Tick(now.Add(time.Duration(i+1) * 100 * time.Millisecond))
	}
	want := []*traceCard{latest, second, first, old}
	if len(d.feed.Objects) != len(want) {
		t.Fatal(len(d.feed.Objects))
	}
	for i, c := range want {
		if d.feed.Objects[i] != c.box {
			t.Fatal("restoration order changed", i)
		}
	}
	for _, e := range d.entries {
		if e.window != nil || e.detached {
			t.Fatal("window remained after disabling")
		}
	}
	// Re-enabling only applies to commands created after that transition.
	if err := d.SetEnabled(true); err != nil {
		t.Fatal(err)
	}
	for _, e := range d.entries {
		if e.detached {
			t.Fatal("existing command converted on re-enable")
		}
	}
}

func TestFullDesktopEvictsOldestAndDropsLateResults(t *testing.T) {
	d, cleanup := testDetachedDeck(t)
	defer cleanup()
	prototype := newTraceCard("prototype", "", "echo")
	prototype.box.Resize(fyne.NewSize(200, prototype.box.MinSize().Height))
	height := int32(prototype.box.MinSize().Height) + 20
	d.desktop = func() (floatingDesktop, error) {
		return floatingDesktop{Main: screenRect{1000, 0, 1200, 500}, Areas: []floatingArea{{screenRect{0, 0, 220, height}, 1}}}, nil
	}
	if err := d.SetEnabled(true); err != nil {
		t.Fatal(err)
	}
	one := newTraceCard("one", "", "echo")
	two := newTraceCard("two", "", "echo")
	three := newTraceCard("three", "", "echo")
	d.Add(one)
	d.Add(two)
	d.Add(three)
	d.Tick(time.Now())
	if len(d.entries) != 1 || d.entries[0].card != three || d.entries[0].window == nil {
		t.Fatal("oldest windows not evicted")
	}
	groups := map[string][]*traceCard{"one": {one}, "two": {two}, "three": {three}}
	active := d.Trim(groups, []*traceCard{one, two, three})
	if len(groups) != 1 || len(active) != 1 || active[0] != three {
		t.Fatal("evicted result or animation references retained")
	}
}
