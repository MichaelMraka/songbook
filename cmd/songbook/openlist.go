//go:build pocketbook

package main

/*
#include <stdlib.h>

typedef int (*iv_listhandler)(int action, int x, int y, int idx, int state);
void OpenList(const char *title, const void *background, int itemw, int itemh, int itemcount, int cpos, iv_listhandler hproc);

enum {
	LIST_BEGINPAINT = 1,
	LIST_PAINT = 2,
	LIST_ENDPAINT = 3,
	LIST_OPEN = 4,
	LIST_MENU = 5,
	LIST_DELETE = 6,
	LIST_EXIT = 7,
	LIST_ORIENTATION = 8,
	LIST_POINTER = 9,
	LIST_INFO = 11,
	LIST_SCROLL = 12
};

extern int goListHandler(int action, int x, int y, int idx, int state);

static int c_list_handler(int action, int x, int y, int idx, int state) {
	return goListHandler(action, x, y, idx, state);
}

static void c_open_list(const char *title, int itemw, int itemh, int itemcount, int cpos) {
	OpenList(title, 0, itemw, itemh, itemcount, cpos, c_list_handler);
}
*/
import "C"

import "unsafe"

const (
	listBeginPaint = 1
	listPaint      = 2
	listEndPaint   = 3
	listOpen       = 4
	listExit       = 7
)

var listTitleC *C.char

func openNativeList(title string, itemW, itemH, itemCount, cpos int) {
	// Keep title alive for the duration of OpenList (may not copy the string).
	if listTitleC != nil {
		C.free(unsafe.Pointer(listTitleC))
		listTitleC = nil
	}
	listTitleC = C.CString(title)
	C.c_open_list(listTitleC, C.int(itemW), C.int(itemH), C.int(itemCount), C.int(cpos))
}

//export goListHandler
func goListHandler(action, x, y, idx, state C.int) C.int {
	if appRef == nil {
		return 0
	}
	return C.int(appRef.onBrowseList(int(action), int(x), int(y), int(idx), int(state)))
}
