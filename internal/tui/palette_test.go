package tui

import (
	"strings"
	"testing"

	"github.com/aeon022/missionctl-core/tuitest"
)

func TestPaletteFiltersAndJumpsToATab(t *testing.T) {
	m := layoutModel(t, 120, 40)
	opened, _ := tuitest.Keys(m, ":")
	om := opened.(Model)
	if !om.inPalette {
		t.Fatal(": must open the palette")
	}
	if text := tuitest.Text(om); !strings.Contains(text, "Commands") || !strings.Contains(text, "calendar") {
		t.Errorf("palette lists the commands:\n%s", text)
	}
	typed, _ := tuitest.Keys(om, "c", "a", "l")
	if got := typed.(Model).paletteMatches(); len(got) != 1 || got[0].Name != "calendar" {
		t.Errorf("'cal' matches only calendar, got %v", got)
	}
	done, _ := tuitest.Keys(typed.(Model), "enter")
	dm := done.(Model)
	if dm.inPalette || dm.activeTab != tabCalendar {
		t.Errorf("enter must close the palette and open the calendar tab (palette=%v tab=%d)", dm.inPalette, dm.activeTab)
	}
}

func TestPaletteReplaysKeyShortcutsAndEscCloses(t *testing.T) {
	m := layoutModel(t, 120, 40)
	opened, _ := tuitest.Keys(m, ":", "n", "e", "w", "enter")
	if !opened.(Model).isEditing {
		t.Error("':new' must behave like pressing n and open the editor")
	}
	closed, _ := tuitest.Keys(m, ":", "esc")
	if cm := closed.(Model); cm.inPalette || cm.isEditing {
		t.Error("esc closes the palette without side effects")
	}
}
