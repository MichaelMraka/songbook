package songbook_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/michael/songbook/internal/songbook"
)

func TestRenderAdjacentSharpChordsDoNotOverwrite(t *testing.T) {
	// Rosa na kolejích style: one space between [F6] [F#6] [G6] must not smash into FFG6.
	text := "             [F6] [F#6] [G6]       [C]"
	lines := songbook.Render(text, 0)
	if len(lines) < 1 || lines[0].Kind != songbook.LineChord {
		t.Fatalf("got %#v", lines)
	}
	got := lines[0].Text
	if strings.Contains(got, "FFG6") || strings.Contains(got, "FG6") {
		t.Fatalf("chords overwrote each other: %q", got)
	}
	for _, want := range []string{"F6", "F#6", "G6", "C"} {
		if !strings.Contains(got, want) {
			t.Fatalf("chord line %q missing %q", got, want)
		}
	}
}

func TestRenderRosaChordLineDense(t *testing.T) {
	text := "                  [F6] [F#6] [G6] [F#6] [F6] [C]"
	lines := songbook.Render(text, 0)
	got := lines[0].Text
	if strings.Contains(got, "FFGFFC") || strings.Contains(got, "FFG") {
		t.Fatalf("dense chords overwrote: %q", got)
	}
	if !strings.Contains(got, "F#6") {
		t.Fatalf("lost sharp chord in %q", got)
	}
}

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

func TestRenderInlineBracesAreControlOnly(t *testing.T) {
	// A pusť ty holuby style: chords embedded in the lyric line; [ ] never print.
	text := "[C]Tak běž Vašku [F]a pusť ty [G]holuby"
	lines := songbook.Render(text, 0)
	if len(lines) != 2 {
		t.Fatalf("got %#v", lines)
	}
	joined := lines[0].Text + lines[1].Text
	if strings.ContainsAny(joined, "[]") {
		t.Fatalf("braces leaked into output: chord=%q lyric=%q", lines[0].Text, lines[1].Text)
	}
	wantLyric := "Tak běž Vašku a pusť ty holuby"
	if lines[1].Text != wantLyric {
		t.Fatalf("lyric = %q, want %q", lines[1].Text, wantLyric)
	}
	for _, ch := range []string{"C", "F", "G"} {
		if !strings.Contains(lines[0].Text, ch) {
			t.Fatalf("chord line %q missing %s", lines[0].Text, ch)
		}
	}
	// F above "a pusť…", G above "holuby" (rune columns, not bytes).
	wantF := utf8.RuneCountInString("Tak běž Vašku ")
	wantG := utf8.RuneCountInString("Tak běž Vašku a pusť ty ")
	if fAt := strings.Index(lines[0].Text, "F"); fAt != wantF {
		t.Errorf("F at %d, want %d (chord=%q)", fAt, wantF, lines[0].Text)
	}
	if gAt := strings.Index(lines[0].Text, "G"); gAt != wantG {
		t.Errorf("G at %d, want %d (chord=%q)", gAt, wantG, lines[0].Text)
	}
}

func TestRenderChordOnlyLine(t *testing.T) {
	text := "[Ami]         [Dmi7]"
	lines := songbook.Render(text, 0)
	if len(lines) != 1 {
		t.Fatalf("got %d lines (want chord only, no blank lyric): %#v", len(lines), lines)
	}
	if lines[0].Kind != songbook.LineChord {
		t.Fatalf("unexpected kind: %#v", lines)
	}
	if !strings.Contains(lines[0].Text, "Ami") || !strings.Contains(lines[0].Text, "Dmi7") {
		t.Fatalf("chord line %q", lines[0].Text)
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

func TestTransposeSongtext(t *testing.T) {
	in := "[C]hello [Ami]there"
	got := songbook.TransposeSongtext(in, 2)
	if got != "[D]hello [Hmi]there" {
		t.Fatalf("got %q", got)
	}
	if songbook.TransposeSongtext(in, 0) != in {
		t.Fatal("zero should be identity")
	}
}