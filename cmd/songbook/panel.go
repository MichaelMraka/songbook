//go:build pocketbook

package main

/*
#cgo LDFLAGS: -pthread -lpthread -linkview
int PanelHeight(void);
void SetPanelType(int type);
*/
import "C"

func panelHeight() int {
	h := int(C.PanelHeight())
	if h < 0 {
		return 0
	}
	return h
}

// setPanelType configures the firmware status panel (0 = disabled / fullscreen).
func setPanelType(t int) {
	C.SetPanelType(C.int(t))
}