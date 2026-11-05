package songbook_test

import (
	"strings"
	"testing"

	"github.com/michael/songbook/internal/songbook"
)

func TestRenderInlineChords(t *testing.T) {
	text := "[Ami]Hello [G]world"
	lines := songbook.Render(text, 0)
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2 (chord+lyric)", len(lines))
	}
	if lines[0].Kind != songbook.LineChord {
		t.Fatalf("line0 kind = %v, want chord", lines[0].Kind)
	}
	if lines[1].Kind != songbook.LineLyric {
		t.Fatalf("line1 kind = %v, want lyric", lines[1].Kind)
	}
	if !strings.Contains(lines[0].Text, "Ami") || !strings.Contains(lines[0].Text, "G") {
		t.Fatalf("chord line %q missing Ami/G", lines[0].Text)
	}
	if lines[1].Text != "Hello world" {
		t.Fatalf("lyric = %q, want %q", lines[1].Text, "Hello world")
	}
	// Ami should start at column 0; G above 'w' of world (after "Hello ")
	amiAt := strings.Index(lines[0].Text, "Ami")
	gAt := strings.Index(lines[0].Text, "G")
	if amiAt != 0 {
		t.Errorf("Ami at %d, want 0", amiAt)
	}
	if gAt != len("Hello ") {
		t.Errorf("G at %d, want %d", gAt, len("Hello "))
	}
}

func TestRenderChordOnlyLine(t *testing.T) {
	text := "[Ami]         [Dmi7]"
	lines := songbook.Render(text, 0)
	if len(lines) != 2 {
		t.Fatalf("got %d lines: %#v", len(lines), lines)
	}
	if lines[0].Kind != songbook.LineChord || lines[1].Kind != songbook.LineLyric {
		t.Fatalf("unexpected kinds: %#v", lines)
	}
}

func TestRenderPlainLine(t *testing.T) {
	lines := songbook.Render("just lyrics", 0)
	if len(lines) != 1 || lines[0].Kind != songbook.LineLyric || lines[0].Text != "just lyrics" {
		t.Fatalf("got %#v", lines)
	}
}

func TestTransposeUp(t *testing.T) {
	got := songbook.TransposeChord("Ami", 2)
	if got != "Hmi" && got != "Bmi" {
		// Czech convention uses H for B natural
		t.Fatalf("TransposeChord(Ami, 2) = %q, want Hmi", got)
	}
	got = songbook.TransposeChord("C", 1)
	if got != "C#" && got != "Db" {
		t.Fatalf("TransposeChord(C, 1) = %q", got)
	}
}

func TestRenderWithTranspose(t *testing.T) {
	lines := songbook.Render("[C]hi", 2)
	if !strings.Contains(lines[0].Text, "D") {
		t.Fatalf("expected D after +2, got %q", lines[0].Text)
	}
}

func TestEmptyAndMultiline(t *testing.T) {
	text := "[Ami]line1\n\n2. verse"
	lines := songbook.Render(text, 0)
	if len(lines) < 3 {
		t.Fatalf("got %d lines, want >= 3", len(lines))
	}
}