//go:build pocketbook

package main

import (
	"fmt"
	"image"
	"image/color"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	ink "github.com/dennwc/inkview"

	"github.com/michael/songbook/internal/songbook"
)

const (
	tabH        = 72
	headerH     = 96
	minFont     = 27
	maxFont     = 72
	defaultFont = 39 // ~1.5× previous 26
	tabCount    = 5
)

type tab int

const (
	tabSearch tab = iota
	tabSongs
	tabAuthors
	tabPlaylists
	tabMenu
)

type keyboardMode int

const (
	kbSearch keyboardMode = iota
	kbImportFile
)

type screen int

const (
	screenList screen = iota
	screenAuthorSongs
	screenPlaylistSongs
	screenSong
	screenSongMenu
)

var songMenuActions = []string{
	"Transpose up",
	"Transpose down",
	"Save transposed",
	"Edit song…",
}

type app struct {
	state *appState

	screen screen
	tab    tab

	filter string
	songs  []songbook.Song // current list content
	scroll int
	sel    int

	authors   []songbook.AuthorCount
	playlists []songbook.Playlist
	ctxTitle  string // drill-down title

	song       songbook.Song
	lines      []songbook.DisplayLine
	songScroll int
	transpose  int // session offset on top of song.Step
	fontSize   int
	fromScreen screen // screen to return to from song view

	status string
	font   *ink.Font
	mono   *ink.Font
	bold   *ink.Font

	kbMode            keyboardMode
	pendingImportPath string
}

func newApp() *app {
	rand.Seed(time.Now().UnixNano())
	return &app{
		state:    newAppState(),
		fontSize: defaultFont,
		tab:      tabSearch,
		screen:   screenList,
	}
}

func (a *app) Init() error {
	// Keep portrait so framebuffer Y grows downward on the physical screen.
	ink.SetDefaultOrientation(ink.Orientation0)
	ink.SetOrientation(ink.Orientation0)
	setPanelType(0) // fullscreen; we draw our own top tabs
	a.openFonts()
	ink.SetKeyboardHandler(a.onKeyboard)
	if err := a.state.load(); err != nil {
		a.status = "No library. Place XML at " + a.state.LibraryPath + " or tap Import."
		// settings may still have loaded; library empty
		if a.state.Lib == nil {
			a.state.Lib = songbook.NewEmpty()
		}
	} else {
		a.status = fmt.Sprintf("%d songs", a.state.Lib.SongCount())
		a.applyListFontFromSettings()
	}
	a.refreshList()
	return nil
}

func (a *app) applyListFontFromSettings() {
	a.fontSize = songbook.FontSizePixels(a.state.Settings.ListFontSize, defaultFont)
	a.fontSize = clamp(a.fontSize, minFont, maxFont)
	a.openFonts()
}

func (a *app) applySongFontFromSettings(songID string) {
	step := a.state.Settings.SongFontSize
	if m := a.state.Settings.SongMeta(songID); m.FontSize != "" {
		step = m.FontSize
	}
	a.fontSize = songbook.FontSizePixels(step, defaultFont)
	a.fontSize = clamp(a.fontSize, minFont, maxFont)
	a.openFonts()
}

// layout returns screen size and vertical bands: tabs | header | content.
// Panel is disabled in Init (setPanelType(0)), so Y starts at 0.
func (a *app) layout() (sz image.Point, tabs, header, content image.Rectangle) {
	sz = ink.ScreenSize()
	tabs = image.Rect(0, 0, sz.X, tabH)
	header = image.Rect(0, tabs.Max.Y, sz.X, tabs.Max.Y+headerH)
	content = image.Rect(0, header.Max.Y, sz.X, sz.Y)
	return sz, tabs, header, content
}

func (a *app) Close() error {
	if a.font != nil {
		a.font.Close()
	}
	if a.mono != nil {
		a.mono.Close()
	}
	if a.bold != nil {
		a.bold.Close()
	}
	return nil
}

func (a *app) openFonts() {
	if a.font != nil {
		a.font.Close()
	}
	if a.mono != nil {
		a.mono.Close()
	}
	if a.bold != nil {
		a.bold.Close()
	}
	face := ink.DefaultFont
	if a.state != nil && a.state.Settings.FontFamily == "mono" {
		face = ink.DefaultFontMono
	}
	a.font = ink.OpenFont(face, a.fontSize, true)
	a.mono = ink.OpenFont(ink.DefaultFontMono, a.fontSize, true)
	a.bold = ink.OpenFont(ink.DefaultFontBold, a.fontSize, true)
}

func (a *app) refreshList() {
	lib := a.state.Lib
	if lib == nil {
		a.songs = nil
		a.authors = nil
		a.playlists = nil
		return
	}
	a.authors = lib.Authors()
	a.playlists = lib.Playlists()
	switch a.screen {
	case screenList:
		switch a.tab {
		case tabSearch:
			if a.filter == "" {
				a.songs = nil
			} else {
				a.songs = lib.Search(a.filter, songbook.SearchOptions{
					Fulltext: a.state.Settings.SearchFulltext,
				})
			}
		case tabSongs:
			a.songs = lib.SongsSorted(a.state.Settings.Sort, a.state.Settings.Songs)
		}
	}
	a.scroll = 0
	a.sel = 0
}

func (a *app) Draw() {
	ink.ClearScreen()
	_, tabs, header, content := a.layout()

	a.drawTabs(tabs)

	switch a.screen {
	case screenSong:
		a.drawSong(header, content)
	case screenSongMenu:
		a.drawActionList(header, content, "Song menu", songMenuActions)
	case screenAuthorSongs, screenPlaylistSongs:
		a.drawSongList(header, content, a.ctxTitle)
	default:
		a.drawMain(header, content)
	}
	ink.FullUpdate()
}

func (a *app) drawMain(header, list image.Rectangle) {
	a.font.SetActive(ink.Black)
	title := "Songbook"
	status := a.status
	switch a.tab {
	case tabSearch:
		ft := "[ ]"
		if a.state.Settings.SearchFulltext {
			ft = "[x]"
		}
		if a.filter == "" {
			title = "Search  " + ft + " fulltext"
			status = "Tap title to type; tap fulltext to toggle"
		} else {
			title = "Search: " + a.filter + "  " + ft
			status = fmt.Sprintf("%d results", len(a.songs))
		}
	case tabSongs:
		title = "Songs"
	case tabAuthors:
		title = "Authors"
	case tabPlaylists:
		title = "Playlists"
	case tabMenu:
		title = "Menu"
		status = "Actions"
	}
	ink.DrawString(image.Pt(12, header.Min.Y+6), title)
	if status != "" {
		ink.DrawString(image.Pt(12, header.Min.Y+6+a.fontSize), status)
	}
	ink.DrawLine(image.Pt(0, header.Max.Y-1), image.Pt(header.Max.X, header.Max.Y-1), ink.Black)

	rowH := a.fontSize + 12
	visible := max(1, list.Dy()/rowH)
	switch a.tab {
	case tabSearch, tabSongs:
		a.drawSongRows(list, rowH, visible, a.songs)
	case tabAuthors:
		a.drawAuthorRows(list, rowH, visible)
	case tabPlaylists:
		a.drawPlaylistRows(list, rowH, visible)
	case tabMenu:
		a.drawMenuRows(list, rowH, visible)
	}
}

func (a *app) drawMenuRows(list image.Rectangle, rowH, visible int) {
	a.drawStringRows(list, rowH, visible, a.menuLabels())
}

func (a *app) drawActionList(header, list image.Rectangle, title string, items []string) {
	a.font.SetActive(ink.Black)
	ink.DrawString(image.Pt(12, header.Min.Y+6), title)
	ink.DrawLine(image.Pt(0, header.Max.Y-1), image.Pt(header.Max.X, header.Max.Y-1), ink.Black)
	rowH := a.fontSize + 12
	visible := max(1, list.Dy()/rowH)
	a.drawStringRows(list, rowH, visible, items)
}

func (a *app) drawStringRows(list image.Rectangle, rowH, visible int, items []string) {
	a.font.SetActive(ink.Black)
	ink.SetClip(list)
	defer ink.SetClip(ink.Screen())
	if a.sel < a.scroll {
		a.scroll = a.sel
	}
	if a.sel >= a.scroll+visible {
		a.scroll = a.sel - visible + 1
	}
	end := min(len(items), a.scroll+visible)
	for i := a.scroll; i < end; i++ {
		y := list.Min.Y + (i-a.scroll)*rowH
		if y+rowH > list.Max.Y {
			break
		}
		if i == a.sel {
			ink.FillArea(image.Rect(list.Min.X, y, list.Max.X, min(y+rowH, list.Max.Y)), ink.LightGray)
		}
		ink.DrawString(image.Pt(12, y+2), truncate(items[i], list.Dx()-24))
	}
}

func (a *app) drawSongList(header, list image.Rectangle, title string) {
	a.font.SetActive(ink.Black)
	ink.DrawString(image.Pt(12, header.Min.Y+6), title)
	ink.DrawLine(image.Pt(0, header.Max.Y-1), image.Pt(header.Max.X, header.Max.Y-1), ink.Black)
	rowH := a.fontSize + 12
	visible := max(1, list.Dy()/rowH)
	a.drawSongRows(list, rowH, visible, a.songs)
}

func (a *app) drawSongRows(list image.Rectangle, rowH, visible int, songs []songbook.Song) {
	a.font.SetActive(ink.Black)
	ink.SetClip(list)
	defer ink.SetClip(ink.Screen())
	if len(songs) == 0 {
		ink.DrawString(image.Pt(12, list.Min.Y+2), "(empty)")
		return
	}
	if a.sel < a.scroll {
		a.scroll = a.sel
	}
	if a.sel >= a.scroll+visible {
		a.scroll = a.sel - visible + 1
	}
	end := min(len(songs), a.scroll+visible)
	for i := a.scroll; i < end; i++ {
		y := list.Min.Y + (i-a.scroll)*rowH
		if y+rowH > list.Max.Y {
			break
		}
		label := songs[i].DisplayLabel()
		if i == a.sel {
			ink.FillArea(image.Rect(list.Min.X, y, list.Max.X, min(y+rowH, list.Max.Y)), ink.LightGray)
		}
		ink.DrawString(image.Pt(12, y+2), truncate(label, list.Dx()-24))
	}
}

func (a *app) drawAuthorRows(list image.Rectangle, rowH, visible int) {
	a.font.SetActive(ink.Black)
	ink.SetClip(list)
	defer ink.SetClip(ink.Screen())
	items := a.authors
	if len(items) == 0 {
		ink.DrawString(image.Pt(12, list.Min.Y+2), "(empty)")
		return
	}
	if a.sel < a.scroll {
		a.scroll = a.sel
	}
	if a.sel >= a.scroll+visible {
		a.scroll = a.sel - visible + 1
	}
	end := min(len(items), a.scroll+visible)
	for i := a.scroll; i < end; i++ {
		y := list.Min.Y + (i-a.scroll)*rowH
		if y+rowH > list.Max.Y {
			break
		}
		label := fmt.Sprintf("%s  (%d)", items[i].Name, items[i].Count)
		if i == a.sel {
			ink.FillArea(image.Rect(list.Min.X, y, list.Max.X, min(y+rowH, list.Max.Y)), ink.LightGray)
		}
		ink.DrawString(image.Pt(12, y+2), truncate(label, list.Dx()-24))
	}
}

func (a *app) drawPlaylistRows(list image.Rectangle, rowH, visible int) {
	a.font.SetActive(ink.Black)
	ink.SetClip(list)
	defer ink.SetClip(ink.Screen())
	items := a.playlists
	if len(items) == 0 {
		ink.DrawString(image.Pt(12, list.Min.Y+2), "(empty)")
		return
	}
	if a.sel < a.scroll {
		a.scroll = a.sel
	}
	if a.sel >= a.scroll+visible {
		a.scroll = a.sel - visible + 1
	}
	end := min(len(items), a.scroll+visible)
	for i := a.scroll; i < end; i++ {
		y := list.Min.Y + (i-a.scroll)*rowH
		if y+rowH > list.Max.Y {
			break
		}
		n := len(a.state.Lib.PlaylistSongs(items[i].ID))
		label := fmt.Sprintf("%s  (%d)", items[i].Title, n)
		if i == a.sel {
			ink.FillArea(image.Rect(list.Min.X, y, list.Max.X, min(y+rowH, list.Max.Y)), ink.LightGray)
		}
		ink.DrawString(image.Pt(12, y+2), truncate(label, list.Dx()-24))
	}
}

func (a *app) drawSong(header, body image.Rectangle) {
	a.font.SetActive(ink.Black)
	hdr := a.song.DisplayLabel()
	step := a.song.Step + a.transpose
	ink.DrawString(image.Pt(12, header.Min.Y+4), truncate(hdr, header.Dx()-12))
	ink.DrawString(image.Pt(12, header.Min.Y+4+a.fontSize), fmt.Sprintf("transpose %+d  font %d", step, a.fontSize))
	ink.DrawLine(image.Pt(0, header.Max.Y-1), image.Pt(header.Max.X, header.Max.Y-1), ink.Black)

	ink.SetClip(body)
	defer ink.SetClip(ink.Screen())
	y := body.Min.Y + 4
	lineH := a.fontSize + 6
	for i := a.songScroll; i < len(a.lines); i++ {
		if y+lineH > body.Max.Y {
			break
		}
		line := a.lines[i]
		a.setSongLineStyle(line.Kind)
		ink.DrawString(image.Pt(12, y), truncate(line.Text, body.Dx()-24))
		y += lineH
	}
}

func (a *app) setSongLineStyle(kind songbook.LineKind) {
	var cl color.Color = ink.Black
	switch a.state.Settings.ChordColor {
	case songbook.ChordColorYellow, songbook.ChordColorGreen:
		cl = ink.DarkGray // e-ink stand-in for named colors
	case songbook.ChordColorRed:
		cl = ink.Black
	}
	if kind == songbook.LineChord {
		if a.state.Settings.ChordHighlight == songbook.ChordHighlightBold && a.bold != nil {
			a.bold.SetActive(cl)
			return
		}
		if a.mono != nil {
			a.mono.SetActive(cl)
			return
		}
	}
	a.font.SetActive(ink.Black)
}

func (a *app) drawTabs(r image.Rectangle) {
	ink.DrawLine(image.Pt(r.Min.X, r.Min.Y), image.Pt(r.Max.X, r.Min.Y), ink.Black)
	labels := []string{"Search", "Songs", "Authors", "Lists", "Menu"}
	w := r.Dx() / len(labels)
	a.font.SetActive(ink.Black)
	for i, lab := range labels {
		x0 := r.Min.X + i*w
		cell := image.Rect(x0, r.Min.Y, x0+w, r.Max.Y)
		if tab(i) == a.tab && a.screen != screenSong {
			ink.FillArea(cell, ink.LightGray)
		}
		ink.DrawString(image.Pt(x0+6, r.Min.Y+(r.Dy()-a.fontSize)/2), lab)
	}
}

func (a *app) Key(e ink.KeyEvent) bool {
	if e.State != ink.KeyStatePress && e.State != ink.KeyStateDown {
		return false
	}
	switch e.Key {
	case ink.KeyBack:
		return a.goBack()
	case ink.KeyUp, ink.KeyPrev, ink.KeyPrev2:
		a.moveSel(-1)
		ink.Repaint()
		return true
	case ink.KeyDown, ink.KeyNext, ink.KeyNext2:
		a.moveSel(1)
		ink.Repaint()
		return true
	case ink.KeyOk:
		a.activate()
		ink.Repaint()
		return true
	case ink.KeyLeft:
		if a.screen == screenSong {
			a.changeTranspose(-1)
			ink.Repaint()
			return true
		}
		a.switchTab(-1)
		ink.Repaint()
		return true
	case ink.KeyRight:
		if a.screen == screenSong {
			a.changeTranspose(1)
			ink.Repaint()
			return true
		}
		a.switchTab(1)
		ink.Repaint()
		return true
	case ink.KeyMinus:
		if a.screen == screenSong {
			a.changeFont(-2)
			ink.Repaint()
			return true
		}
	case ink.KeyPlus:
		if a.screen == screenSong {
			a.changeFont(2)
			ink.Repaint()
			return true
		}
	case ink.KeyMenu:
		if a.screen == screenSong {
			a.openSongMenu()
		} else if a.screen == screenSongMenu {
			a.screen = screenSong
			ink.Repaint()
		} else {
			a.openSearch()
		}
		return true
	}
	return false
}

func (a *app) openSearch() {
	a.screen = screenList
	a.tab = tabSearch
	a.ctxTitle = ""
	a.refreshList()
	ink.OpenKeyboard(a.filter, 128)
}

func (a *app) openSongMenu() {
	a.sel = 0
	a.scroll = 0
	a.screen = screenSongMenu
	ink.Repaint()
}

func (a *app) toggleFulltextSearch() {
	a.state.Settings.SearchFulltext = !a.state.Settings.SearchFulltext
	_ = a.state.saveSettings()
	a.refreshList()
	if a.state.Settings.SearchFulltext {
		a.status = "Fulltext search on"
	} else {
		a.status = "Fulltext search off"
	}
}

func (a *app) Pointer(e ink.PointerEvent) bool {
	if e.State != ink.PointerUp && e.State != ink.PointerDown {
		return false
	}
	if e.State != ink.PointerUp {
		return true
	}
	_, tabs, header, list := a.layout()
	if e.Y >= tabs.Min.Y && e.Y < tabs.Max.Y {
		w := tabs.Dx() / tabCount
		if w <= 0 {
			w = 1
		}
		a.setTab(tab((e.X - tabs.Min.X) / w))
		ink.Repaint()
		return true
	}
	if a.screen == screenSong {
		mid := (list.Min.Y + list.Max.Y) / 2
		if e.Y < mid {
			a.songScroll = max(0, a.songScroll-8)
		} else {
			a.songScroll = min(max(0, len(a.lines)-1), a.songScroll+8)
		}
		ink.Repaint()
		return true
	}
	rowH := a.fontSize + 12
	if a.screen == screenSongMenu {
		if e.Y >= list.Min.Y {
			a.sel = a.scroll + (e.Y-list.Min.Y)/rowH
			a.activate()
			ink.Repaint()
		}
		return true
	}
	if e.Y < list.Min.Y {
		if a.screen == screenList && a.tab == tabSearch && e.Y >= header.Min.Y {
			// Right third of header toggles fulltext; left opens keyboard.
			if e.X >= header.Dx()*2/3 {
				a.toggleFulltextSearch()
				ink.Repaint()
				return true
			}
			ink.OpenKeyboard(a.filter, 128)
			return true
		}
		return true
	}
	idx := a.scroll + (e.Y-list.Min.Y)/rowH
	a.sel = idx
	a.activate()
	ink.Repaint()
	return true
}

func (a *app) Touch(e ink.TouchEvent) bool { return false }

func (a *app) Orientation(o ink.Orientation) bool {
	ink.Repaint()
	return true
}

func (a *app) onKeyboard(text string) {
	text = strings.TrimSpace(text)
	switch a.kbMode {
	case kbImportFile:
		a.kbMode = kbSearch
		a.onImportFilename(text)
	default:
		a.filter = text
		a.screen = screenList
		a.tab = tabSearch
		a.refreshList()
	}
	ink.Repaint()
}

func (a *app) listLen() int {
	switch a.screen {
	case screenSong:
		return len(a.lines)
	case screenSongMenu:
		return len(songMenuActions)
	case screenAuthorSongs, screenPlaylistSongs:
		return len(a.songs)
	default:
		switch a.tab {
		case tabSearch, tabSongs:
			return len(a.songs)
		case tabAuthors:
			return len(a.authors)
		case tabPlaylists:
			return len(a.playlists)
		case tabMenu:
			return len(a.menuLabels())
		}
	}
	return 0
}

func (a *app) moveSel(delta int) {
	if a.screen == screenSong {
		a.songScroll = clamp(a.songScroll+delta*3, 0, max(0, len(a.lines)-1))
		return
	}
	n := a.listLen()
	if n == 0 {
		return
	}
	a.sel = clamp(a.sel+delta, 0, n-1)
}

func (a *app) activate() {
	switch a.screen {
	case screenList:
		switch a.tab {
		case tabSearch, tabSongs:
			if a.sel >= 0 && a.sel < len(a.songs) {
				a.openSong(a.songs[a.sel])
			}
		case tabAuthors:
			if a.sel >= 0 && a.sel < len(a.authors) {
				au := a.authors[a.sel]
				a.songs = a.state.Lib.SongsByAuthor(au.Name)
				a.ctxTitle = fmt.Sprintf("%s (%d)", au.Name, au.Count)
				a.screen = screenAuthorSongs
				a.sel = 0
				a.scroll = 0
			}
		case tabPlaylists:
			if a.sel >= 0 && a.sel < len(a.playlists) {
				pl := a.playlists[a.sel]
				a.songs = a.state.Lib.PlaylistSongs(pl.ID)
				a.ctxTitle = fmt.Sprintf("%s (%d)", pl.Title, len(a.songs))
				a.screen = screenPlaylistSongs
				a.sel = 0
				a.scroll = 0
			}
		case tabMenu:
			a.runMenuAction(a.sel)
		}
	case screenSongMenu:
		a.runSongMenuAction(a.sel)
	case screenAuthorSongs, screenPlaylistSongs:
		if a.sel >= 0 && a.sel < len(a.songs) {
			a.openSong(a.songs[a.sel])
		}
	}
}

func (a *app) runSongMenuAction(i int) {
	if i < 0 || i >= len(songMenuActions) {
		return
	}
	switch songMenuActions[i] {
	case "Transpose up":
		a.changeTranspose(1)
		a.screen = screenSong
	case "Transpose down":
		a.changeTranspose(-1)
		a.screen = screenSong
	case "Save transposed":
		a.saveTransposed()
		a.screen = screenSong
	case "Edit song…":
		a.startEditSong()
	}
}

func (a *app) saveTransposed() {
	if a.transpose == 0 {
		ink.Message(ink.Info, "Save transposed", "Nothing to save (offset 0).", 0)
		return
	}
	s := a.song
	s.Text = songbook.TransposeSongtext(s.Text, a.transpose)
	s.Step += a.transpose
	if err := a.state.upsertSong(s); err != nil {
		ink.Message(ink.Error, "Save transposed", err.Error(), 0)
		return
	}
	a.song = s
	a.transpose = 0
	a.rebuildSongLines()
	a.status = "Transposition saved"
	ink.Message(ink.Info, "Save transposed", a.status, 0)
}

func (a *app) startEditSong() {
	// Wired in the Edit song feature; placeholder keeps menu index stable.
	ink.Message(ink.Info, "Edit song", "Coming next.", 0)
	a.screen = screenSong
}

func (a *app) openSong(s songbook.Song) {
	a.fromScreen = a.screen
	a.song = s
	a.transpose = 0
	a.songScroll = 0
	a.applySongFontFromSettings(s.ID)
	a.rebuildSongLines()
	a.screen = screenSong
}

func (a *app) rebuildSongLines() {
	a.lines = songbook.Render(a.song.Text, a.song.Step+a.transpose)
}

func (a *app) changeTranspose(d int) {
	a.transpose += d
	a.rebuildSongLines()
}

func (a *app) changeFont(d int) {
	a.fontSize = clamp(a.fontSize+d, minFont, maxFont)
	a.openFonts()
}

func (a *app) goBack() bool {
	switch a.screen {
	case screenSongMenu:
		a.screen = screenSong
		ink.Repaint()
		return true
	case screenSong:
		a.screen = a.fromScreen
		a.applyListFontFromSettings()
		ink.Repaint()
		return true
	case screenAuthorSongs, screenPlaylistSongs:
		a.ctxTitle = ""
		a.screen = screenList
		a.refreshList()
		ink.Repaint()
		return true
	default:
		ink.Exit()
		return true
	}
}

func (a *app) switchTab(delta int) {
	if a.screen != screenList {
		a.screen = screenList
		a.ctxTitle = ""
	}
	a.setTab(tab(clamp(int(a.tab)+delta, 0, tabCount-1)))
}

func (a *app) setTab(t tab) {
	if t < 0 {
		t = 0
	}
	if t >= tabCount {
		t = tabCount - 1
	}
	a.tab = t
	a.screen = screenList
	a.ctxTitle = ""
	a.refreshList()
	a.sel = 0
	a.scroll = 0
	a.status = fmt.Sprintf("%d songs", a.state.Lib.SongCount())
}

func (a *app) promptImportFilename() {
	a.kbMode = kbImportFile
	ink.OpenKeyboard("export.zpk.xml", 256)
}

func (a *app) onImportFilename(name string) {
	if name == "" {
		a.status = "Import cancelled"
		return
	}
	path := name
	if !filepath.IsAbs(path) {
		path = filepath.Join(songbookDir, path)
	}
	if st, err := os.Stat(path); err != nil || st.IsDir() {
		a.status = "File not found: " + path
		ink.Message(ink.Warning, "Import", a.status, 0)
		return
	}
	a.pendingImportPath = path
	a.promptImportMode()
}

func (a *app) openRandomSong() {
	songs := a.state.Lib.Songs()
	if len(songs) == 0 {
		ink.Message(ink.Warning, "Random song", "Library is empty.", 0)
		return
	}
	a.openSong(songs[rand.Intn(len(songs))])
}

func (a *app) promptImportMode() {
	ink.SetDialogHandler(func(button int) {
		if button == 1 {
			a.doImport(true)
		} else if button == 2 {
			a.doImport(false)
		}
		ink.Repaint()
	})
	ink.Dialog(ink.Question, "Import", "Replace library or merge with current songs?", "Replace", "Merge")
}

func (a *app) doImport(replace bool) {
	src := a.pendingImportPath
	a.pendingImportPath = ""
	if src == "" {
		a.status = "No import file selected"
		ink.Message(ink.Warning, "Import", a.status, 0)
		return
	}
	incoming, err := songbook.LoadFile(src)
	if err != nil {
		a.status = err.Error()
		ink.Message(ink.Error, "Import", err.Error(), 0)
		return
	}
	if replace {
		err = a.state.replaceLibrary(incoming)
	} else {
		err = a.state.mergeLibrary(incoming)
	}
	if err != nil {
		a.status = err.Error()
		ink.Message(ink.Error, "Import", err.Error(), 0)
		return
	}
	a.screen = screenList
	a.refreshList()
	mode := "Merged"
	if replace {
		mode = "Replaced"
	}
	a.status = fmt.Sprintf("%s: %d songs", mode, a.state.Lib.SongCount())
	ink.Message(ink.Info, "Import", a.status, 0)
}

func truncate(s string, maxPx int) string {
	// approximate: ~0.5*font width; StringWidth needs active font
	if ink.StringWidth(s) <= maxPx {
		return s
	}
	runes := []rune(s)
	for i := len(runes); i > 0; i-- {
		cand := string(runes[:i]) + "…"
		if ink.StringWidth(cand) <= maxPx {
			return cand
		}
	}
	return "…"
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}