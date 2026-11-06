//go:build pocketbook

package main

import (
	"github.com/michael/songbook/internal/songbook"
)

type appMenuKind int

const (
	menuKindMain appMenuKind = iota
	menuKindSort
	menuKindListFont
	menuKindSongFont
	menuKindFont
	menuKindChords
	menuKindSongMenu
	menuKindSongMetaFontSize
	menuKindSongMetaFontFamily
)

func (a *app) menuLabels() []string {
	s := a.state.Settings
	return []string{
		"Import library…",
		"Random song",
		"Sort: " + sortLabel(s.Sort),
		"List font: " + string(s.ListFontSize),
		"Song font: " + string(s.SongFontSize),
		"Font: " + fontFamilyLabel(s.FontFamily),
		"Chords: " + chordStyleLabel(s.ChordHighlight, s.ChordColor),
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

func fontFamilyLabel(fam string) string {
	if fam == "mono" {
		return "mono"
	}
	return "default"
}

func chordStyleLabel(h songbook.ChordHighlightStyle, c songbook.ChordColor) string {
	if h == songbook.ChordHighlightOff {
		return "off"
	}
	if c == "" {
		return "bold"
	}
	return "bold/" + string(c)
}

func sortChoices() []songbook.SortMode {
	return []songbook.SortMode{
		songbook.SortByAuthorName,
		songbook.SortBySongName,
		songbook.SortByUpdated,
		songbook.SortByCreated,
	}
}

func sortChoiceLabels() []string {
	out := make([]string, 0, 4)
	for _, m := range sortChoices() {
		out = append(out, sortLabel(m))
	}
	return out
}

func fontSizeChoices() []songbook.FontSizeStep {
	return []songbook.FontSizeStep{
		songbook.FontSmallest,
		songbook.FontSmall,
		songbook.FontMedium,
		songbook.FontBig,
		songbook.FontBigger,
	}
}

func fontSizeChoiceLabels() []string {
	out := make([]string, 0, 5)
	for _, s := range fontSizeChoices() {
		out = append(out, string(s))
	}
	return out
}

func fontFamilyChoices() []string {
	return []string{"", "mono"}
}

func fontFamilyChoiceLabels() []string {
	return []string{"default", "mono"}
}

type chordChoice struct {
	h songbook.ChordHighlightStyle
	c songbook.ChordColor
}

func chordChoices() []chordChoice {
	return []chordChoice{
		{songbook.ChordHighlightBold, songbook.ChordColorDefault},
		{songbook.ChordHighlightBold, songbook.ChordColorYellow},
		{songbook.ChordHighlightBold, songbook.ChordColorGreen},
		{songbook.ChordHighlightBold, songbook.ChordColorRed},
		{songbook.ChordHighlightOff, songbook.ChordColorDefault},
	}
}

func chordChoiceLabels() []string {
	out := make([]string, 0, 5)
	for _, p := range chordChoices() {
		out = append(out, chordStyleLabel(p.h, p.c))
	}
	return out
}

func indexOfSort(cur songbook.SortMode) int {
	for i, m := range sortChoices() {
		if m == cur {
			return i
		}
	}
	return 0
}

func indexOfFontSize(cur songbook.FontSizeStep) int {
	for i, s := range fontSizeChoices() {
		if s == cur {
			return i
		}
	}
	return 2
}

func indexOfFontFamily(cur string) int {
	if cur == "mono" {
		return 1
	}
	return 0
}

func indexOfChord(h songbook.ChordHighlightStyle, c songbook.ChordColor) int {
	for i, p := range chordChoices() {
		if p.h == h && p.c == c {
			return i
		}
	}
	return 0
}

func (a *app) runMainMenuAction(i int) {
	switch i {
	case 0:
		a.promptImportFilename()
		inkRepaint()
	case 1:
		a.openRandomSong()
		inkRepaint()
	case 2:
		a.openSettingMenu(menuKindSort, "Sort", sortChoiceLabels(), indexOfSort(a.state.Settings.Sort))
	case 3:
		a.openSettingMenu(menuKindListFont, "List font", fontSizeChoiceLabels(), indexOfFontSize(a.state.Settings.ListFontSize))
	case 4:
		a.openSettingMenu(menuKindSongFont, "Song font", fontSizeChoiceLabels(), indexOfFontSize(a.state.Settings.SongFontSize))
	case 5:
		a.openSettingMenu(menuKindFont, "Font", fontFamilyChoiceLabels(), indexOfFontFamily(a.state.Settings.FontFamily))
	case 6:
		a.openSettingMenu(menuKindChords, "Chords", chordChoiceLabels(), indexOfChord(a.state.Settings.ChordHighlight, a.state.Settings.ChordColor))
	}
}

func (a *app) applySettingChoice(kind appMenuKind, i int) {
	switch kind {
	case menuKindSort:
		choices := sortChoices()
		if i < 0 || i >= len(choices) {
			return
		}
		a.state.Settings.Sort = choices[i]
		a.persistSettings()
		a.refreshList()
	case menuKindListFont:
		choices := fontSizeChoices()
		if i < 0 || i >= len(choices) {
			return
		}
		a.state.Settings.ListFontSize = choices[i]
		a.persistSettings()
		a.applyListFontFromSettings()
	case menuKindSongFont:
		choices := fontSizeChoices()
		if i < 0 || i >= len(choices) {
			return
		}
		a.state.Settings.SongFontSize = choices[i]
		a.persistSettings()
	case menuKindFont:
		choices := fontFamilyChoices()
		if i < 0 || i >= len(choices) {
			return
		}
		a.state.Settings.FontFamily = choices[i]
		a.persistSettings()
		a.applyListFontFromSettings()
	case menuKindChords:
		choices := chordChoices()
		if i < 0 || i >= len(choices) {
			return
		}
		a.state.Settings.ChordHighlight = choices[i].h
		a.state.Settings.ChordColor = choices[i].c
		a.persistSettings()
	}
}

func (a *app) persistSettings() {
	if err := a.state.saveSettings(); err != nil {
		a.status = err.Error()
		return
	}
	a.status = "Settings saved"
}

// runMenuAction keeps the old list-UI path working if Menu rows are shown.
func (a *app) runMenuAction(i int) {
	a.runMainMenuAction(i)
}

func (a *app) songMenuLabels() []string {
	sz := a.songMetaFontSize()
	fam := a.songMetaFontFamily()
	return []string{
		"Transpose up",
		"Transpose down",
		"Save transposed",
		"Edit song…",
		"Song font size: " + string(sz),
		"Song font family: " + fontFamilyLabel(fam),
		"Reset song font",
	}
}

func (a *app) songMetaFontSize() songbook.FontSizeStep {
	m := a.state.Settings.SongMeta(a.song.ID)
	if m.FontSize != "" {
		return m.FontSize
	}
	if a.state.Settings.SongFontSize != "" {
		return a.state.Settings.SongFontSize
	}
	return songbook.FontMedium
}

func (a *app) songMetaFontFamily() string {
	m := a.state.Settings.SongMeta(a.song.ID)
	if m.FontFamily != "" {
		return m.FontFamily
	}
	return a.state.Settings.FontFamily
}

func (a *app) runSongOpenMenuAction(i int) {
	switch i {
	case 0:
		a.changeTranspose(1)
		inkRepaint()
	case 1:
		a.changeTranspose(-1)
		inkRepaint()
	case 2:
		a.saveTransposed()
		inkRepaint()
	case 3:
		a.startEditSong()
		inkRepaint()
	case 4:
		a.openSettingMenu(menuKindSongMetaFontSize, "Song font size", fontSizeChoiceLabels(), indexOfFontSize(a.songMetaFontSize()))
	case 5:
		a.openSettingMenu(menuKindSongMetaFontFamily, "Song font family", fontFamilyChoiceLabels(), indexOfFontFamily(a.songMetaFontFamily()))
	case 6:
		a.resetSongMetaFont()
		inkRepaint()
	}
}

func (a *app) applySongMetaChoice(kind appMenuKind, i int) {
	m := a.state.Settings.SongMeta(a.song.ID)
	switch kind {
	case menuKindSongMetaFontSize:
		choices := fontSizeChoices()
		if i < 0 || i >= len(choices) {
			return
		}
		m.FontSize = choices[i]
	case menuKindSongMetaFontFamily:
		choices := fontFamilyChoices()
		if i < 0 || i >= len(choices) {
			return
		}
		m.FontFamily = choices[i]
	default:
		return
	}
	if a.state.Settings.Songs == nil {
		a.state.Settings.Songs = map[string]songbook.SongMeta{}
	}
	a.state.Settings.Songs[a.song.ID] = m
	_ = a.state.saveSettings()
	a.applySongFontFromSettings(a.song.ID)
	a.status = "Song font updated"
}
