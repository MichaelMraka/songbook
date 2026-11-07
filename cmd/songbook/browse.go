//go:build pocketbook

package main

import (
	"path/filepath"

	ink "github.com/dennwc/inkview"

	"github.com/michael/songbook/internal/songbook"
)

func (a *app) openImportBrowser() {
	a.pendingImportPath = ""
	a.browsePath = ink.FlashDir
	a.reloadBrowseEntries()
	a.showBrowseMenu()
}

func (a *app) reloadBrowseEntries() {
	ents, err := songbook.ListDir(a.browsePath)
	if err != nil {
		a.status = err.Error()
		ink.Message(ink.Error, "Import", err.Error(), 0)
		a.browseEntries = nil
		return
	}
	var rows []songbook.DirEntry
	root := ink.FlashDir
	if a.browsePath != root && a.browsePath != "/" {
		rows = append(rows, songbook.DirEntry{
			Name:  "..",
			Path:  filepath.Clean(filepath.Join(a.browsePath, "..")),
			IsDir: true,
		})
	}
	a.browseEntries = append(rows, ents...)
}

func (a *app) showBrowseMenu() {
	if len(a.browseEntries) == 0 {
		ink.Message(ink.Warning, "Import", "No files here", 0)
		ink.Repaint()
		return
	}
	labels := make([]string, len(a.browseEntries))
	for i, e := range a.browseEntries {
		labels[i] = songbook.EntryLabel(e)
	}
	// Short title; full path can be long on device menus.
	title := "Import: " + filepath.Base(a.browsePath)
	if a.browsePath == ink.FlashDir || a.browsePath == "/" {
		title = "Import"
	}
	a.openSettingMenu(menuKindBrowse, title, labels, -1)
}

func (a *app) onBrowseMenu(i int) {
	if i < 0 || i >= len(a.browseEntries) {
		a.browseEntries = nil
		ink.Repaint()
		return
	}
	e := a.browseEntries[i]
	if e.IsDir {
		a.browsePath = e.Path
		a.reloadBrowseEntries()
		a.showBrowseMenu()
		return
	}
	a.pendingImportPath = e.Path
	a.browseEntries = nil
	a.promptImportMode()
}
