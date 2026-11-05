//go:build pocketbook

package main

import (
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"strings"

	ink "github.com/dennwc/inkview"

	"github.com/michael/songbook/internal/songbook"
)

const (
	defaultLibraryPath = ink.FlashDir + "/songbook/library.xml"
	tabH               = 48
	headerH            = 56
	minFont            = 18
	maxFont            = 48
	defaultFont        = 26
)

type tab int

const (
	tabSongs tab = iota
	tabAuthors
	tabPlaylists
)

type screen int

const (
	screenList screen = iota
	screenAuthorSongs
	screenPlaylistSongs
	screenSong
)

type app struct {
	lib *songbook.Library

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
}

func newApp() *app {
	return &app{
		fontSize: defaultFont,
		tab:      tabSongs,
		screen:   screenList,
	}
}

func (a *app) Init() error {
	a.openFonts()
	ink.SetKeyboardHandler(a.onKeyboard)
	if err := a.loadLibrary(defaultLibraryPath); err != nil {
		a.status = "No library. Place XML at " + defaultLibraryPath + " or tap Import."
		a.lib = &songbook.Library{}
	}
	a.refreshList()
	return nil
}

func (a *app) Close() error {
	if a.font != nil {
		a.font.Close()
	}
	if a.mono != nil {
		a.mono.Close()
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
	a.font = ink.OpenFont(ink.DefaultFont, a.fontSize, true)
	a.mono = ink.OpenFont(ink.DefaultFontMono, a.fontSize, true)
}

func (a *app) loadLibrary(path string) error {
	lib, err := songbook.LoadFile(path)
	if err != nil {
		return err
	}
	a.lib = lib
	a.status = fmt.Sprintf("%d songs", lib.SongCount())
	return nil
}

func (a *app) refreshList() {
	if a.lib == nil {
		a.songs = nil
		a.authors = nil
		a.playlists = nil
		return
	}
	a.authors = a.lib.Authors()
	a.playlists = a.lib.Playlists()
	switch a.screen {
	case screenList:
		if a.tab == tabSongs {
			if a.filter == "" {
				a.songs = a.lib.Songs()
			} else {
				a.songs = a.lib.Search(a.filter)
			}
		}
	}
	a.scroll = 0
	a.sel = 0
}

func (a *app) Draw() {
	ink.ClearScreen()
	sz := ink.ScreenSize()
	content := image.Rect(0, 0, sz.X, sz.Y-tabH)

	switch a.screen {
	case screenSong:
		a.drawSong(content)
	case screenAuthorSongs, screenPlaylistSongs:
		a.drawSongList(content, a.ctxTitle)
	default:
		a.drawMain(content)
	}
	a.drawTabs(image.Rect(0, sz.Y-tabH, sz.X, sz.Y))
	ink.FullUpdate()
}

func (a *app) drawMain(r image.Rectangle) {
	a.font.SetActive(ink.Black)
	title := "Songbook"
	switch a.tab {
	case tabSongs:
		title = "Songs"
		if a.filter != "" {
			title = "Songs: " + a.filter
		}
	case tabAuthors:
		title = "Authors"
	case tabPlaylists:
		title = "Playlists"
	}
	ink.DrawString(image.Pt(12, 8), title)
	if a.status != "" {
		ink.DrawString(image.Pt(12, 8+a.fontSize+4), a.status)
	}
	ink.DrawLine(image.Pt(0, headerH), image.Pt(r.Max.X, headerH), ink.Black)

	listTop := headerH + 4
	rowH := a.fontSize + 12
	visible := max(1, (r.Max.Y-listTop)/rowH)

	switch a.tab {
	case tabSongs:
		a.drawSongRows(r, listTop, rowH, visible, a.songs)
	case tabAuthors:
		a.drawAuthorRows(r, listTop, rowH, visible)
	case tabPlaylists:
		a.drawPlaylistRows(r, listTop, rowH, visible)
	}
}

func (a *app) drawSongList(r image.Rectangle, title string) {
	a.font.SetActive(ink.Black)
	ink.DrawString(image.Pt(12, 8), title)
	ink.DrawLine(image.Pt(0, headerH), image.Pt(r.Max.X, headerH), ink.Black)
	listTop := headerH + 4
	rowH := a.fontSize + 12
	visible := max(1, (r.Max.Y-listTop)/rowH)
	a.drawSongRows(r, listTop, rowH, visible, a.songs)
}

func (a *app) drawSongRows(r image.Rectangle, listTop, rowH, visible int, songs []songbook.Song) {
	a.font.SetActive(ink.Black)
	if len(songs) == 0 {
		ink.DrawString(image.Pt(12, listTop), "(empty)")
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
		y := listTop + (i-a.scroll)*rowH
		label := songs[i].DisplayLabel()
		if i == a.sel {
			ink.FillArea(image.Rect(0, y, r.Max.X, y+rowH), ink.LightGray)
		}
		ink.DrawString(image.Pt(12, y+2), truncate(label, r.Max.X-24))
	}
}

func (a *app) drawAuthorRows(r image.Rectangle, listTop, rowH, visible int) {
	a.font.SetActive(ink.Black)
	items := a.authors
	if len(items) == 0 {
		ink.DrawString(image.Pt(12, listTop), "(empty)")
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
		y := listTop + (i-a.scroll)*rowH
		label := fmt.Sprintf("%s  (%d)", items[i].Name, items[i].Count)
		if i == a.sel {
			ink.FillArea(image.Rect(0, y, r.Max.X, y+rowH), ink.LightGray)
		}
		ink.DrawString(image.Pt(12, y+2), truncate(label, r.Max.X-24))
	}
}

func (a *app) drawPlaylistRows(r image.Rectangle, listTop, rowH, visible int) {
	a.font.SetActive(ink.Black)
	items := a.playlists
	if len(items) == 0 {
		ink.DrawString(image.Pt(12, listTop), "(empty)")
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
		y := listTop + (i-a.scroll)*rowH
		n := len(a.lib.PlaylistSongs(items[i].ID))
		label := fmt.Sprintf("%s  (%d)", items[i].Title, n)
		if i == a.sel {
			ink.FillArea(image.Rect(0, y, r.Max.X, y+rowH), ink.LightGray)
		}
		ink.DrawString(image.Pt(12, y+2), truncate(label, r.Max.X-24))
	}
}

func (a *app) drawSong(r image.Rectangle) {
	a.font.SetActive(ink.Black)
	hdr := a.song.DisplayLabel()
	step := a.song.Step + a.transpose
	ink.DrawString(image.Pt(12, 4), truncate(hdr, r.Max.X-12))
	ink.DrawString(image.Pt(12, 4+a.fontSize), fmt.Sprintf("transpose %+d  font %d", step, a.fontSize))
	ink.DrawLine(image.Pt(0, headerH), image.Pt(r.Max.X, headerH), ink.Black)

	y := headerH + 8
	lineH := a.fontSize + 6
	if a.mono != nil {
		a.mono.SetActive(ink.Black)
	}
	for i := a.songScroll; i < len(a.lines); i++ {
		if y+lineH > r.Max.Y {
			break
		}
		ink.DrawString(image.Pt(12, y), truncate(a.lines[i].Text, r.Max.X-24))
		y += lineH
	}
}

func (a *app) drawTabs(r image.Rectangle) {
	ink.DrawLine(image.Pt(r.Min.X, r.Min.Y), image.Pt(r.Max.X, r.Min.Y), ink.Black)
	labels := []string{"Songs", "Authors", "Playlists"}
	w := r.Dx() / len(labels)
	a.font.SetActive(ink.Black)
	for i, lab := range labels {
		x0 := r.Min.X + i*w
		cell := image.Rect(x0, r.Min.Y, x0+w, r.Max.Y)
		if tab(i) == a.tab && a.screen != screenSong {
			ink.FillArea(cell, ink.LightGray)
		}
		ink.DrawString(image.Pt(x0+12, r.Min.Y+12), lab)
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
		a.openMenu()
		return true
	}
	return false
}

func (a *app) Pointer(e ink.PointerEvent) bool {
	if e.State != ink.PointerUp && e.State != ink.PointerDown {
		return false
	}
	if e.State != ink.PointerUp {
		return true
	}
	sz := ink.ScreenSize()
	if e.Y >= sz.Y-tabH {
		w := sz.X / 3
		a.setTab(tab(e.X / w))
		ink.Repaint()
		return true
	}
	if a.screen == screenSong {
		mid := sz.Y / 2
		if e.Y < mid {
			a.songScroll = max(0, a.songScroll-8)
		} else {
			a.songScroll = min(max(0, len(a.lines)-1), a.songScroll+8)
		}
		ink.Repaint()
		return true
	}
	listTop := headerH + 4
	rowH := a.fontSize + 12
	if e.Y < listTop {
		// header tap: filter on songs tab
		if a.screen == screenList && a.tab == tabSongs {
			ink.OpenKeyboard(a.filter, 128)
			return true
		}
		return true
	}
	idx := a.scroll + (e.Y-listTop)/rowH
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
	a.filter = strings.TrimSpace(text)
	a.screen = screenList
	a.tab = tabSongs
	a.refreshList()
	ink.Repaint()
}

func (a *app) listLen() int {
	switch a.screen {
	case screenSong:
		return len(a.lines)
	case screenAuthorSongs, screenPlaylistSongs:
		return len(a.songs)
	default:
		switch a.tab {
		case tabSongs:
			return len(a.songs)
		case tabAuthors:
			return len(a.authors)
		case tabPlaylists:
			return len(a.playlists)
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
		case tabSongs:
			if a.sel >= 0 && a.sel < len(a.songs) {
				a.openSong(a.songs[a.sel])
			}
		case tabAuthors:
			if a.sel >= 0 && a.sel < len(a.authors) {
				au := a.authors[a.sel]
				a.songs = a.lib.SongsByAuthor(au.Name)
				a.ctxTitle = fmt.Sprintf("%s (%d)", au.Name, au.Count)
				a.screen = screenAuthorSongs
				a.sel = 0
				a.scroll = 0
			}
		case tabPlaylists:
			if a.sel >= 0 && a.sel < len(a.playlists) {
				pl := a.playlists[a.sel]
				a.songs = a.lib.PlaylistSongs(pl.ID)
				a.ctxTitle = fmt.Sprintf("%s (%d)", pl.Title, len(a.songs))
				a.screen = screenPlaylistSongs
				a.sel = 0
				a.scroll = 0
			}
		}
	case screenAuthorSongs, screenPlaylistSongs:
		if a.sel >= 0 && a.sel < len(a.songs) {
			a.openSong(a.songs[a.sel])
		}
	}
}

func (a *app) openSong(s songbook.Song) {
	a.fromScreen = a.screen
	a.song = s
	a.transpose = 0
	a.songScroll = 0
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
	case screenSong:
		a.screen = a.fromScreen
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
	a.setTab(tab(clamp(int(a.tab)+delta, 0, 2)))
}

func (a *app) setTab(t tab) {
	a.tab = t
	a.screen = screenList
	a.ctxTitle = ""
	a.filter = a.filter // keep filter for songs
	a.refreshList()
	a.sel = 0
	a.scroll = 0
	a.status = fmt.Sprintf("%d songs", a.lib.SongCount())
}

func (a *app) openMenu() {
	ink.SetDialogHandler(func(button int) {
		if button == 1 {
			a.doImport()
		}
		ink.Repaint()
	})
	ink.Dialog(ink.Question, "Songbook", "Import library XML from /mnt/ext1/songbook/?", "Import", "Cancel")
}

func (a *app) doImport() {
	candidates := []string{
		ink.FlashDir + "/songbook/import.xml",
		ink.FlashDir + "/songbook/export.zpk.xml",
		ink.FlashDir + "/songbook/export.xml",
		ink.FlashDir + "/export.zpk.xml",
	}
	var src string
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			src = c
			break
		}
	}
	if src == "" {
		a.status = "No import file found under /mnt/ext1/songbook/"
		ink.Message(ink.Warning, "Import", a.status, 0)
		return
	}
	if err := os.MkdirAll(filepath.Dir(defaultLibraryPath), 0o755); err != nil {
		a.status = err.Error()
		return
	}
	if err := copyFile(src, defaultLibraryPath); err != nil {
		a.status = err.Error()
		ink.Message(ink.Error, "Import", err.Error(), 0)
		return
	}
	if err := a.loadLibrary(defaultLibraryPath); err != nil {
		a.status = err.Error()
		ink.Message(ink.Error, "Import", err.Error(), 0)
		return
	}
	a.screen = screenList
	a.refreshList()
	a.status = fmt.Sprintf("Imported %d songs", a.lib.SongCount())
	ink.Message(ink.Info, "Import", a.status, 0)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
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