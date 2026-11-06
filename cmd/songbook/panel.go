//go:build pocketbook

package main

/*
#cgo LDFLAGS: -pthread -lpthread -linkview
int PanelHeight(void);
void SetPanelType(int type);
*/
import "C"

// PANEL_FLAGS from inkview.h — SetPanelType takes a bitset of these.
const (
	panelEnabled     = 1 << 1 // PANEL_ENABLED
	panelNoFBOffset  = 1 << 3 // PANEL_NO_FB_OFFSET: y=0 is top of screen
)

func panelHeight() int {
	h := int(C.PanelHeight())
	if h < 0 {
		return 0
	}
	return h
}

// appChromeTop is where app UI starts: the first row below the system panel.
// Stay at PanelHeight() (not up into its padding). Drawing into the panel
// band overdraws icons and races with firmware panel refreshes (scr0005).
// Requires PANEL_NO_FB_OFFSET so y=0 is the top of the screen.
func appChromeTop() int {
	return panelHeight()
}

// setPanelType configures the firmware status panel (inkview PANEL_FLAGS).
func setPanelType(t int) {
	C.SetPanelType(C.int(t))
}

func enableSystemPanel() {
	// Keep the clock/wifi/battery strip, but do not shift the framebuffer —
	// otherwise layout that uses PanelHeight (or an icon-bottom inset) is
	// applied twice and a large gap appears under the panel (scr0003).
	setPanelType(panelEnabled | panelNoFBOffset)
}