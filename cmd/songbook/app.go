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
	uiFontSize  = 36 // fixed chrome/menu face; independent of list/song fonts
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
	kbEditAuthor
	kbEditTitle
	kbEditText
)

type screen int

const (
	screenList screen = iota
	screenAuthorSongs
	screenPlaylistSongs
	screenSong
	screenSongMenu
	screenEditSong
)

var songMenuActions = []string{
	"Transpose up",
	"Transpose down",
	"Save transposed",
	"Edit song…",
	"Song font size…",
	"Song font family…",
	"Reset song font",
}

var editSongActions = []string{
	"Edit author",
	"Edit title",
	"Edit song text",
	"Done",
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
	font   *ink.Font // list / content face (follows list or song settings)
	mono   *ink.Font
	bold   *ink.Font
	uiFont *ink.Font // fixed chrome + OpenMenu; never follows list/song fonts

	kbMode            keyboardMode
	pendingImportPath string

	menuReturn tab         // tab shown under / after native OpenMenu (Menu is a trigger)
	menuKind   appMenuKind // main OpenMenu vs a settings submenu
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
	setAppRef(a)
	// Keep portrait so framebuffer Y grows downward on the physical screen.
	ink.SetDefaultOrientation(ink.Orientation0)
	ink.SetOrientation(ink.Orientation0)
	enableSystemPanel() // clock/wifi/battery; no FB offset (see panel.go)
	a.openUIFont()
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
	// List chrome always uses the default face (same family as UI); only size varies.
	a.fontSize = songbook.FontSizePixels(a.state.Settings.ListFontSize, defaultFont)
	a.fontSize = clamp(a.fontSize, minFont, maxFont)
	a.openFontsWithFace(ink.DefaultFont)
}

func (a *app) applySongFontFromSettings(songID string) {
	step := a.state.Settings.SongFontSize
	family := a.state.Settings.FontFamily
	if m := a.state.Settings.SongMeta(songID); m.FontSize != "" || m.FontFamily != "" {
		if m.FontSize != "" {
			step = m.FontSize
		}
		if m.FontFamily != "" {
			family = m.FontFamily
		}
	}
	a.fontSize = songbook.FontSizePixels(step, defaultFont)
	a.fontSize = clamp(a.fontSize, minFont, maxFont)
	a.openFontsWithFace(songFace(family))
}

func songFace(family string) string {
	if family == "mono" {
		return ink.DefaultFontMono
	}
	return ink.DefaultFont
}

// layout returns screen size and vertical bands below the system panel:
// tabs | header | content. List pages omit the header band.
func (a *app) layout() (sz image.Point, tabs, header, content image.Rectangle) {
	sz = ink.ScreenSize()
	top := appChromeTop()
	tabs = image.Rect(0, top, sz.X, top+tabH)
	hh := a.contentHeaderHeight()
	header = image.Rect(0, tabs.Max.Y, sz.X, tabs.Max.Y+hh)
	content = image.Rect(0, header.Max.Y, sz.X, sz.Y)
	return sz, tabs, header, content
}

func (a *app) contentHeaderHeight() int {
	switch a.screen {
	case screenList, screenAuthorSongs, screenPlaylistSongs, screenSong:
		// Lists and song body start under the tab bar (song title scrolls with text).
		return 0
	default:
		return headerH
	}
}

func (a *app) listPage() bool {
	switch a.screen {
	case screenList, screenAuthorSongs, screenPlaylistSongs:
		return true
	default:
		return false
	}
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
	if a.uiFont != nil {
		a.uiFont.Close()
	}
	return nil
}

// openUIFont loads the fixed chrome/menu face once (not tied to list/song settings).
func (a *app) openUIFont() {
	if a.uiFont != nil {
		a.uiFont.Close()
	}
	a.uiFont = ink.OpenFont(ink.DefaultFont, uiFontSize, true)
}

func (a *app) openFonts() {
	a.openFontsWithFace(ink.DefaultFont)
}

func (a *app) openFontsWithFace(face string) {
	if a.font != nil {
		a.font.Close()
	}
	if a.mono != nil {
		a.mono.Close()
	}
	if a.bold != nil {
		a.bold.Close()
	}
	a.font = ink.OpenFont(face, a.fontSize, true)
	a.mono = ink.OpenFont(ink.DefaultFontMono, a.fontSize, true)
	a.bold = ink.OpenFont(ink.DefaultFontBold, a.fontSize, true)
}

func (a *app) activateUIFont() {
	if a.uiFont != nil {
		a.uiFont.SetActive(ink.Black)
	}
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
			q, fulltext := songbook.ParseSearchQuery(a.filter)
			if q == "" {
				a.songs = nil
			} else {
				a.songs = lib.Search(q, songbook.SearchOptions{Fulltext: fulltext})
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
	case screenEditSong:
		a.drawActionList(header, content, "Edit: "+a.song.Title, editSongActions)
	case screenAuthorSongs, screenPlaylistSongs:
		a.drawSongList(header, content, a.ctxTitle)
	default:
		a.drawMain(header, content)
	}
	// Panel last so it owns 0..PanelHeight exclusively (no race with tab bar).
	ink.DrawTopPanel()
	ink.FullUpdate()
}

func (a *app) drawMain(_, list image.Rectangle) {
	// List pages have no title/status header — content starts under the tab bar.
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

func (a *app) drawSongList(_, list image.Rectangle, _ string) {
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

func (a *app) drawSong(_, body image.Rectangle) {
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
		pt := image.Pt(12, y)
		text := truncate(line.Text, body.Dx()-24)
		ink.DrawString(pt, text)
		// Faux-bold chord highlight: same mono metrics as lyrics (real Bold
		// face would shift columns), drawn 1px right for weight on e-ink.
		if line.Kind == songbook.LineChord && a.chordHighlightOn() {
			ink.DrawString(image.Pt(pt.X+1, pt.Y), text)
		}
		y += lineH
	}
}

func (a *app) chordHighlightOn() bool {
	return a.state != nil && a.state.Settings.ChordHighlight != songbook.ChordHighlightOff
}

func (a *app) setSongLineStyle(kind songbook.LineKind) {
	// Chord + lyric share the song face (default or mono from settings).
	face := a.font
	if face == nil {
		face = a.mono
	}
	if kind == songbook.LineChord {
		var cl color.Color = ink.Black
		if a.chordHighlightOn() {
			switch a.state.Settings.ChordColor {
			case songbook.ChordColorYellow, songbook.ChordColorGreen:
				cl = ink.DarkGray // e-ink stand-in for named colors
			case songbook.ChordColorRed:
				cl = ink.Black
			default:
				cl = ink.Black
			}
		}
		face.SetActive(cl)
		return
	}
	face.SetActive(ink.Black)
}

func (a *app) drawTabs(r image.Rectangle) {
	labels := []string{" Search", " Songs", " Authors", " Playlists", " Menu"}
	n := len(labels)
	w := r.Dx() / n
	a.activateUIFont()

	// Fill bar white so panel/content don't bleed through.
	ink.FillArea(r, ink.White)

	for i, lab := range labels {
		x0 := r.Min.X + i*w
		x1 := x0 + w
		if i == n-1 {
			x1 = r.Max.X
		}
		// Inset 1px for top/bottom rules so the selection box sits between them.
		cell := image.Rect(x0, r.Min.Y+1, x1, r.Max.Y-1)
		selected := tab(i) == a.tab && a.screen != screenSong && a.screen != screenSongMenu && a.screen != screenEditSong
		if selected {
			ink.FillArea(cell, ink.LightGray)
			ink.DrawRect(cell, ink.Black)
		}
		ink.DrawString(image.Pt(x0+6, cell.Min.Y+(cell.Dy()-uiFontSize)/2), lab)

		// Vertical divider between items (PocketBook-style hairline).
		if i > 0 {
			ink.DrawLine(image.Pt(x0, r.Min.Y+4), image.Pt(x0, r.Max.Y-5), ink.DarkGray)
		}
	}

	// Hairlines separating the bar from the system panel above and content below.
	ink.DrawLine(image.Pt(r.Min.X, r.Min.Y), image.Pt(r.Max.X, r.Min.Y), ink.DarkGray)
	ink.DrawLine(image.Pt(r.Min.X, r.Max.Y-1), image.Pt(r.Max.X, r.Max.Y-1), ink.DarkGray)
}

func (a *app) Key(e ink.KeyEvent) bool {
	if e.State != ink.KeyStatePress && e.State != ink.KeyStateDown {
		return false
	}
	switch e.Key {
	case ink.KeyBack:
		return a.goBack()
	case ink.KeyUp:
		a.moveSel(-1)
		ink.Repaint()
		return true
	case ink.KeyDown:
		a.moveSel(1)
		ink.Repaint()
		return true
	case ink.KeyPrev, ink.KeyPrev2:
		// Hardware < : page lists; half-page scroll in song view.
		if a.pagingList() {
			a.pageSel(-1)
		} else if a.screen == screenSong {
			a.pageSong(-1)
		} else {
			a.moveSel(-1)
		}
		ink.Repaint()
		return true
	case ink.KeyNext, ink.KeyNext2:
		// Hardware > : page lists; half-page scroll in song view.
		if a.pagingList() {
			a.pageSel(1)
		} else if a.screen == screenSong {
			a.pageSong(1)
		} else {
			a.moveSel(1)
		}
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
		} else if a.screen == screenSongMenu || a.screen == screenEditSong {
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
	a.kbMode = kbSearch
	ink.OpenKeyboard(a.filter, 128)
}

// openSongMenu is implemented in openmenu.go (native OpenMenu).

func (a *app) Pointer(e ink.PointerEvent) bool {
	if e.State != ink.PointerUp && e.State != ink.PointerDown {
		return false
	}
	if e.State != ink.PointerUp {
		return true
	}
	_, tabs, _, list := a.layout()
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
	if a.screen == screenSongMenu || a.screen == screenEditSong {
		if e.Y >= list.Min.Y {
			a.sel = a.scroll + (e.Y-list.Min.Y)/rowH
			a.activate()
			ink.Repaint()
		}
		return true
	}
	if e.Y < list.Min.Y {
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
	mode := a.kbMode
	a.kbMode = kbSearch
	switch mode {
	case kbImportFile:
		a.onImportFilename(strings.TrimSpace(text))
	case kbEditAuthor:
		a.song.Author = strings.TrimSpace(text)
		a.saveSongEdits()
		a.screen = screenEditSong
	case kbEditTitle:
		a.song.Title = strings.TrimSpace(text)
		a.saveSongEdits()
		a.screen = screenEditSong
	case kbEditText:
		// Preserve internal whitespace; only trim outer newlines.
		a.song.Text = strings.Trim(text, "\r\n")
		a.saveSongEdits()
		a.rebuildSongLines()
		a.screen = screenEditSong
	default:
		a.filter = strings.TrimSpace(text)
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
	case screenEditSong:
		return len(editSongActions)
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

func (a *app) pagingList() bool {
	switch a.screen {
	case screenList, screenAuthorSongs, screenPlaylistSongs:
		return true
	default:
		return false
	}
}

func (a *app) listVisibleRows() int {
	_, _, _, list := a.layout()
	rowH := a.fontSize + 12
	return max(1, list.Dy()/rowH)
}

// pageSel jumps by one screen of rows (hardware < / > on song/author lists).
func (a *app) pageSel(dir int) {
	n := a.listLen()
	if n == 0 {
		return
	}
	page := a.listVisibleRows()
	a.sel = clamp(a.sel+dir*page, 0, n-1)
}

func (a *app) songVisibleRows() int {
	_, _, _, body := a.layout()
	lineH := a.fontSize + 6
	return max(1, (body.Dy()-4)/lineH)
}

// pageSong scrolls the song view by half a screen of lines.
func (a *app) pageSong(dir int) {
	if len(a.lines) == 0 {
		return
	}
	half := max(1, a.songVisibleRows()/2)
	a.songScroll = clamp(a.songScroll+dir*half, 0, max(0, len(a.lines)-1))
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
	case screenEditSong:
		a.runEditSongAction(a.sel)
	case screenAuthorSongs, screenPlaylistSongs:
		if a.sel >= 0 && a.sel < len(a.songs) {
			a.openSong(a.songs[a.sel])
		}
	}
}

func (a *app) runSongMenuAction(i int) {
	// Legacy list-UI path; native OpenMenu uses runSongOpenMenuAction.
	a.runSongOpenMenuAction(i)
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
	a.sel = 0
	a.scroll = 0
	a.screen = screenEditSong
}

func (a *app) runEditSongAction(i int) {
	if i < 0 || i >= len(editSongActions) {
		return
	}
	switch editSongActions[i] {
	case "Edit author":
		a.kbMode = kbEditAuthor
		ink.OpenKeyboard(a.song.Author, 256)
	case "Edit title":
		a.kbMode = kbEditTitle
		ink.OpenKeyboard(a.song.Title, 256)
	case "Edit song text":
		a.kbMode = kbEditText
		ink.OpenKeyboard(a.song.Text, 16384)
	case "Done":
		a.screen = screenSong
		a.rebuildSongLines()
	}
}

func (a *app) saveSongEdits() {
	if err := a.state.upsertSong(a.song); err != nil {
		ink.Message(ink.Error, "Edit song", err.Error(), 0)
		a.status = err.Error()
		return
	}
	a.status = "Song saved"
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
	body := songbook.Render(a.song.Text, a.song.Step+a.transpose)
	// Author/title scroll with the song (no fixed header status line).
	head := []songbook.DisplayLine{
		{Kind: songbook.LineLyric, Text: a.song.Author},
		{Kind: songbook.LineLyric, Text: a.song.Title},
		{Kind: songbook.LineLyric, Text: ""},
	}
	a.lines = append(head, body...)
}

func (a *app) changeTranspose(d int) {
	a.transpose += d
	a.rebuildSongLines()
}

func (a *app) changeFont(d int) {
	a.fontSize = clamp(a.fontSize+d, minFont, maxFont)
	if a.screen == screenSong || a.screen == screenSongMenu {
		a.openFontsWithFace(a.currentSongFace())
		a.persistSongFontSizeFromPixels()
		return
	}
	a.openFontsWithFace(ink.DefaultFont)
}

func (a *app) currentSongFace() string {
	family := ""
	if a.state != nil {
		family = a.state.Settings.FontFamily
		if m := a.state.Settings.SongMeta(a.song.ID); m.FontFamily != "" {
			family = m.FontFamily
		}
	}
	return songFace(family)
}

func (a *app) persistSongFontSizeFromPixels() {
	step := nearestFontSize(a.fontSize, defaultFont)
	m := a.state.Settings.SongMeta(a.song.ID)
	m.FontSize = step
	if a.state.Settings.Songs == nil {
		a.state.Settings.Songs = map[string]songbook.SongMeta{}
	}
	a.state.Settings.Songs[a.song.ID] = m
	_ = a.state.saveSettings()
}

func nearestFontSize(px, base int) songbook.FontSizeStep {
	best := songbook.FontMedium
	bestDiff := int(^uint(0) >> 1)
	for _, step := range []songbook.FontSizeStep{
		songbook.FontSmallest, songbook.FontSmall, songbook.FontMedium,
		songbook.FontBig, songbook.FontBigger,
	} {
		d := songbook.FontSizePixels(step, base) - px
		if d < 0 {
			d = -d
		}
		if d < bestDiff {
			bestDiff = d
			best = step
		}
	}
	return best
}

func (a *app) resetSongMetaFont() {
	m := a.state.Settings.SongMeta(a.song.ID)
	m.FontFamily = ""
	m.FontSize = ""
	if a.state.Settings.Songs == nil {
		a.state.Settings.Songs = map[string]songbook.SongMeta{}
	}
	a.state.Settings.Songs[a.song.ID] = m
	_ = a.state.saveSettings()
	a.applySongFontFromSettings(a.song.ID)
	a.status = "Song font reset"
}

func (a *app) goBack() bool {
	switch a.screen {
	case screenSongMenu, screenEditSong:
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
	a.setTab(tab(clamp(int(a.tab)+delta, 0, tabCount-1)))
}

func (a *app) leavingSongView() bool {
	return a.screen == screenSong || a.screen == screenSongMenu || a.screen == screenEditSong
}

func (a *app) setTab(t tab) {
	if t < 0 {
		t = 0
	}
	if t >= tabCount {
		t = tabCount - 1
	}
	fromSong := a.leavingSongView()
	if t == tabMenu {
		// On song detail, Menu opens the song-specific menu — not the app menu.
		if fromSong {
			a.openSongMenu()
			return
		}
		// Menu tab opens native OpenMenu; keep previous tab content underneath.
		if a.tab != tabMenu {
			a.menuReturn = a.tab
		}
		if a.menuReturn == tabMenu {
			a.menuReturn = tabSongs
		}
		a.tab = a.menuReturn
		a.screen = screenList
		a.ctxTitle = ""
		a.refreshList()
		a.sel = 0
		a.scroll = 0
		a.status = fmt.Sprintf("%d songs", a.state.Lib.SongCount())
		a.openAppMenu()
		return
	}
	// Song view swaps in song font size/face; restore list settings when leaving.
	if fromSong {
		a.applyListFontFromSettings()
	}
	a.tab = t
	a.menuReturn = t
	a.screen = screenList
	a.ctxTitle = ""
	a.refreshList()
	a.sel = 0
	a.scroll = 0
	a.status = fmt.Sprintf("%d songs", a.state.Lib.SongCount())
	if t == tabSearch {
		a.kbMode = kbSearch
		ink.OpenKeyboard(a.filter, 128)
	}
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
