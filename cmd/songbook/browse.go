//go:build pocketbook

package main

import (
	"image"
	"path/filepath"

	ink "github.com/dennwc/inkview"

	"github.com/michael/songbook/internal/songbook"
)

func (a *app) openImportBrowser() {
	a.browsePath = ink.FlashDir
	a.reloadBrowseEntries()
	a.showBrowseList(0)
}

func (a *app) reloadBrowseEntries() {
	ents, err := songbook.ListDir(a.browsePath)
	if err != nil {
		a.status = err.Error()
		ink.Message(ink.Error, "Import", err.Error(), 0)
		a.browseEntries = nil
		return
	}
	root := ink.FlashDir
	if a.browsePath != root && a.browsePath != "/" {
		parent := songbook.DirEntry{
			Name:  "..",
			Path:  filepath.Clean(filepath.Join(a.browsePath, "..")),
			IsDir: true,
		}
		a.browseEntries = append([]songbook.DirEntry{parent}, ents...)
	} else {
		a.browseEntries = ents
	}
}

func (a *app) showBrowseList(cpos int) {
	a.activateUIFont()
	n := len(a.browseEntries)
	itemH := uiFontSize + 16
	sz := ink.ScreenSize()
	title := "Import: " + a.browsePath
	openNativeList(title, sz.X, itemH, n, cpos)
}

func (a *app) onBrowseList(action, x, y, idx, state int) int {
	switch action {
	case listPaint:
		if idx < 0 || idx >= len(a.browseEntries) {
			return 0
		}
		a.activateUIFont()
		itemH := uiFontSize + 16
		ink.DrawString(image.Pt(x+8, y+(itemH-uiFontSize)/2), songbook.EntryLabel(a.browseEntries[idx]))
		return 0
	case listOpen:
		if idx < 0 || idx >= len(a.browseEntries) {
			return 0
		}
		e := a.browseEntries[idx]
		if e.IsDir {
			a.browsePath = e.Path
			a.reloadBrowseEntries()
			a.showBrowseList(0)
			return 0
		}
		a.pendingImportPath = e.Path
		a.promptImportMode()
		return 0
	case listExit:
		ink.Repaint()
		return 0
	}
	return 0
}
