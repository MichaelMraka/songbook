package songbook

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// LineKind distinguishes chord rows from lyric rows.
type LineKind int

const (
	LineLyric LineKind = iota
	LineChord
)

// DisplayLine is one rendered row for the song view.
type DisplayLine struct {
	Kind LineKind
	Text string
}

// Render turns songtext into chord/lyric display lines.
// transpose is semitones relative to the written chords (usually song.Step + session).
func Render(songtext string, transpose int) []DisplayLine {
	raw := strings.ReplaceAll(songtext, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")
	var out []DisplayLine
	for _, line := range strings.Split(raw, "\n") {
		out = append(out, renderLine(line, transpose)...)
	}
	return out
}

func renderLine(line string, transpose int) []DisplayLine {
	type token struct {
		chord string // empty = lyric run
		text  string
	}
	var tokens []token
	for len(line) > 0 {
		if line[0] == '[' {
			end := strings.IndexByte(line, ']')
			if end < 0 {
				tokens = append(tokens, token{text: line})
				break
			}
			chord := line[1:end]
			tokens = append(tokens, token{chord: chord})
			line = line[end+1:]
			continue
		}
		next := strings.IndexByte(line, '[')
		if next < 0 {
			tokens = append(tokens, token{text: line})
			break
		}
		tokens = append(tokens, token{text: line[:next]})
		line = line[next:]
	}

	hasChord := false
	for _, t := range tokens {
		if t.chord != "" {
			hasChord = true
			break
		}
	}
	if !hasChord {
		var b strings.Builder
		for _, t := range tokens {
			b.WriteString(t.text)
		}
		return []DisplayLine{{Kind: LineLyric, Text: b.String()}}
	}

	var lyric strings.Builder
	type placed struct {
		col   int
		chord string
	}
	var placedChords []placed
	for _, t := range tokens {
		if t.chord != "" {
			ch := t.chord
			if transpose != 0 {
				ch = TransposeChord(ch, transpose)
			}
			placedChords = append(placedChords, placed{col: utf8.RuneCountInString(lyric.String()), chord: ch})
			continue
		}
		lyric.WriteString(t.text)
	}

	lyricStr := lyric.String()
	chordRunes := []rune(strings.Repeat(" ", max(utf8.RuneCountInString(lyricStr), 1)))
	for _, p := range placedChords {
		writeChordAt(&chordRunes, p.col, p.chord)
	}
	return []DisplayLine{
		{Kind: LineChord, Text: strings.TrimRight(string(chordRunes), " ")},
		{Kind: LineLyric, Text: lyricStr},
	}
}

func writeChordAt(buf *[]rune, col int, chord string) {
	need := col + utf8.RuneCountInString(chord)
	for len(*buf) < need {
		*buf = append(*buf, ' ')
	}
	for i, r := range []rune(chord) {
		(*buf)[col+i] = r
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// sharp-oriented chromatic scale using Czech H for B natural.
var scale = []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "H"}

var rootAliases = map[string]int{
	"C": 0, "C#": 1, "Db": 1, "D": 2, "D#": 3, "Eb": 3,
	"E": 4, "Fb": 4, "E#": 5, "F": 5, "F#": 6, "Gb": 6,
	"G": 7, "G#": 8, "Ab": 8, "A": 9, "A#": 10, "Bb": 10, "B": 11, "H": 11,
	"Cb": 11, "Hb": 10, // Hb ~ B flat in some Czech tabs
}

// TransposeChord shifts a chord name by semitones. Empty/invalid chords pass through.
func TransposeChord(chord string, semitones int) string {
	chord = strings.TrimSpace(chord)
	if chord == "" || semitones == 0 {
		return chord
	}
	root, suffix := splitChord(chord)
	idx, ok := rootAliases[root]
	if !ok {
		return chord
	}
	semitones %= 12
	if semitones < 0 {
		semitones += 12
	}
	return scale[(idx+semitones)%12] + suffix
}

func splitChord(chord string) (root, suffix string) {
	if chord == "" {
		return "", ""
	}
	// Multi-char roots first
	for _, cand := range []string{"C#", "Db", "D#", "Eb", "F#", "Gb", "G#", "Ab", "A#", "Bb", "E#", "Fb", "Cb", "Hb"} {
		if strings.HasPrefix(chord, cand) {
			return cand, chord[len(cand):]
		}
	}
	r, size := utf8.DecodeRuneInString(chord)
	if r == unicode.ReplacementChar && size == 1 {
		return chord, ""
	}
	root = string(r)
	// optional #/b after single letter already handled; single letter root
	rest := chord[size:]
	if strings.HasPrefix(rest, "#") || strings.HasPrefix(rest, "b") {
		// already covered by multi-char; if we get here it's lone letter
	}
	return root, rest
}