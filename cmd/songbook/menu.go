//go:build pocketbook

package main

import (
	"github.com/michael/songbook/internal/songbook"
)

func (a *app) menuLabels() []string {
	s := a.state.Settings
	font := "default"
	if s.FontFamily == "mono" {
		font = "mono"
	}
	chords := string(s.ChordHighlight)
	if s.ChordColor != "" {
		chords += "/" + string(s.ChordColor)
	}
	return []string{
		"Import library…",
		"Random song",
		"Sort: " + sortLabel(s.Sort),
		"List font: " + string(s.ListFontSize),
		"Song font: " + string(s.SongFontSize),
		"Font: " + font,
		"Chords: " + chords,
	}
}

func sortLabel(m songbook.SortMode) string {
	switch m {
	case songbook.SortBySongName:
		return "song name"
	case songbook.SortByUpdated:
		return "updated"
	case songbook.SortByCreated:
		return "created"
	default:
		return "author name"
	}
}

func (a *app) runMenuAction(i int) {
	labels := a.menuLabels()
	if i < 0 || i >= len(labels) {
		return
	}
	switch i {
	case 0:
		a.promptImportFilename()
	case 1:
		a.openRandomSong()
	case 2:
		a.cycleSort()
	case 3:
		a.cycleListFont()
	case 4:
		a.cycleSongFont()
	case 5:
		a.cycleFontFamily()
	case 6:
		a.cycleChordStyle()
	}
}

func (a *app) persistSettings() {
	if err := a.state.saveSettings(); err != nil {
		a.status = err.Error()
		return
	}
	a.status = "Settings saved"
}

func (a *app) cycleSort() {
	order := []songbook.SortMode{
		songbook.SortByAuthorName,
		songbook.SortBySongName,
		songbook.SortByUpdated,
		songbook.SortByCreated,
	}
	cur := a.state.Settings.Sort
	next := order[0]
	for i, m := range order {
		if m == cur {
			next = order[(i+1)%len(order)]
			break
		}
	}
	a.state.Settings.Sort = next
	a.persistSettings()
	a.refreshList()
}

func (a *app) cycleListFont() {
	a.state.Settings.ListFontSize = nextFontSize(a.state.Settings.ListFontSize)
	a.persistSettings()
	a.applyListFontFromSettings()
}

func (a *app) cycleSongFont() {
	a.state.Settings.SongFontSize = nextFontSize(a.state.Settings.SongFontSize)
	a.persistSettings()
}

func (a *app) cycleFontFamily() {
	if a.state.Settings.FontFamily == "mono" {
		a.state.Settings.FontFamily = ""
	} else {
		a.state.Settings.FontFamily = "mono"
	}
	a.persistSettings()
	a.applyListFontFromSettings()
}

func (a *app) cycleChordStyle() {
	s := &a.state.Settings
	type pair struct {
		h songbook.ChordHighlightStyle
		c songbook.ChordColor
	}
	seq := []pair{
		{songbook.ChordHighlightBold, songbook.ChordColorDefault},
		{songbook.ChordHighlightBold, songbook.ChordColorYellow},
		{songbook.ChordHighlightBold, songbook.ChordColorGreen},
		{songbook.ChordHighlightBold, songbook.ChordColorRed},
		{songbook.ChordHighlightOff, songbook.ChordColorDefault},
	}
	next := seq[0]
	for i, p := range seq {
		if p.h == s.ChordHighlight && p.c == s.ChordColor {
			next = seq[(i+1)%len(seq)]
			break
		}
	}
	s.ChordHighlight = next.h
	s.ChordColor = next.c
	a.persistSettings()
}

func nextFontSize(cur songbook.FontSizeStep) songbook.FontSizeStep {
	order := []songbook.FontSizeStep{
		songbook.FontSmallest,
		songbook.FontSmall,
		songbook.FontMedium,
		songbook.FontBig,
		songbook.FontBigger,
	}
	for i, s := range order {
		if s == cur {
			return order[(i+1)%len(order)]
		}
	}
	return songbook.FontMedium
}