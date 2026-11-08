//go:build pocketbook

package main

/*
#include <stdlib.h>

typedef void (*iv_keyboardhandler)(char *text);
void OpenKeyboard(const char *title, char *buffer, int maxlen, int flags, iv_keyboardhandler hproc);

extern void goAppKeyboardHandler(char *text);

static void c_app_keyboard_handler(char *text) {
	goAppKeyboardHandler(text);
}

static void c_open_keyboard(const char *title, char *buffer, int maxlen, int flags) {
	OpenKeyboard(title, buffer, maxlen, flags, c_app_keyboard_handler);
}
*/
import "C"

import "unsafe"

// Keep title + edit buffer alive while the native keyboard is open.
var (
	kbTitleC  *C.char
	kbBuf     []byte
	kbPrefill string // text shown when the keyboard opened (for cancel detection)
)

// openKeyboard shows the PocketBook keyboard with text prefilled for editing.
// The first argument is the dialog title (not the edit string).
func openKeyboard(title, text string, maxlen int) {
	if maxlen < 2 {
		maxlen = 1024
	}
	if kbTitleC != nil {
		C.free(unsafe.Pointer(kbTitleC))
		kbTitleC = nil
	}
	kbPrefill = text
	kbTitleC = C.CString(title)
	kbBuf = make([]byte, maxlen)
	copy(kbBuf[:maxlen-1], text) // leaves trailing NUL
	C.c_open_keyboard(kbTitleC, (*C.char)(unsafe.Pointer(&kbBuf[0])), C.int(maxlen), 0)
}

// keyboardCancelled reports firmware cancel: handler gets "" while we had prefilled text.
func keyboardCancelled(text string) bool {
	return text == "" && kbPrefill != ""
}

//export goAppKeyboardHandler
func goAppKeyboardHandler(text *C.char) {
	if appRef == nil {
		return
	}
	appRef.onKeyboard(C.GoString(text))
}
