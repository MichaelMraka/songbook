//go:build pocketbook

package main

/*
#include <stdlib.h>

typedef struct imenu_s {
	short type;
	short index;
	char *text;
	struct imenu_s *submenu;
} imenu;

typedef void (*iv_menuhandler)(int index);
void OpenMenu(imenu *menu, int pos, int x, int y, iv_menuhandler hproc);

extern void goAppMenuHandler(int index);

static void c_app_menu_handler(int index) {
	goAppMenuHandler(index);
}

static void c_open_app_menu(imenu *menu, int pos, int x, int y) {
	OpenMenu(menu, pos, x, y, c_app_menu_handler);
}
*/
import "C"

import (
	"unsafe"

	ink "github.com/dennwc/inkview"
)

const (
	ivItemHeader = 1
	ivItemActive = 2
	ivItemBullet = 7
)

var (
	appRef       *app
	menuCStrings []*C.char
	menuCItems   []C.imenu
)

func setAppRef(a *app) { appRef = a }

func inkRepaint() { ink.Repaint() }

//export goAppMenuHandler
func goAppMenuHandler(index C.int) {
	if appRef == nil {
		return
	}
	appRef.onAppMenu(int(index))
}

func freeMenuCStrings() {
	for _, s := range menuCStrings {
		C.free(unsafe.Pointer(s))
	}
	menuCStrings = nil
}

func (a *app) openAppMenu() {
	a.menuKind = menuKindMain
	a.openIvMenu("Menu", a.menuLabels(), -1)
}

func (a *app) openSongMenu() {
	// Keep song content underneath the native menu.
	a.screen = screenSong
	a.menuKind = menuKindSongMenu
	a.openIvMenu("Song", a.songMenuLabels(), -1)
}

func (a *app) openSettingMenu(kind appMenuKind, title string, labels []string, selected int) {
	a.menuKind = kind
	a.openIvMenu(title, labels, selected)
}

// openIvMenu builds a PocketBook OpenMenu. selected >= 0 marks that choice with a bullet
// and focuses it; use -1 when there is no current selection highlight.
func (a *app) openIvMenu(title string, labels []string, selected int) {
	// OpenMenu inherits the active face; keep chrome independent of list/song fonts.
	a.activateUIFont()
	freeMenuCStrings()

	menuCItems = make([]C.imenu, len(labels)+2)
	hdr := C.CString(title)
	menuCStrings = append(menuCStrings, hdr)
	menuCItems[0]._type = C.short(ivItemHeader)
	menuCItems[0].index = 0
	menuCItems[0].text = hdr

	for i, lab := range labels {
		cs := C.CString(lab)
		menuCStrings = append(menuCStrings, cs)
		typ := ivItemActive
		if selected >= 0 && i == selected {
			typ = ivItemBullet
		}
		menuCItems[i+1]._type = C.short(typ)
		// 1-based so cancel/empty stays distinguishable from first action.
		menuCItems[i+1].index = C.short(i + 1)
		menuCItems[i+1].text = cs
	}

	pos := 0
	if selected >= 0 {
		pos = selected + 1 // OpenMenu pos is index into the imenu array
	}
	sz := ink.ScreenSize()
	x := C.int(sz.X / 2)
	y := C.int(appChromeTop() + tabH + 8)
	C.c_open_app_menu(&menuCItems[0], C.int(pos), x, y)
}

func (a *app) onAppMenu(index int) {
	switch a.menuKind {
	case menuKindMain:
		if index <= 0 {
			ink.Repaint()
			return
		}
		a.runMainMenuAction(index - 1)

	case menuKindSort, menuKindListFont, menuKindSongFont, menuKindFont, menuKindChords:
		if index > 0 {
			a.applySettingChoice(a.menuKind, index-1)
		}
		a.openAppMenu()

	case menuKindSongMenu:
		if index <= 0 {
			ink.Repaint()
			return
		}
		a.runSongOpenMenuAction(index - 1)

	case menuKindSongMetaFontSize, menuKindSongMetaFontFamily:
		if index > 0 {
			a.applySongMetaChoice(a.menuKind, index-1)
		}
		a.openSongMenu()

	default:
		ink.Repaint()
	}
}
