package songbook

import (
	"strings"
	"unicode"
)

var diacriticReplacer = strings.NewReplacer(
	"á", "a", "ä", "a", "ą", "a",
	"č", "c", "ć", "c",
	"ď", "d",
	"é", "e", "ě", "e", "ë", "e", "ę", "e",
	"í", "i", "ï", "i",
	"ľ", "l", "ĺ", "l", "ł", "l",
	"ň", "n", "ń", "n",
	"ó", "o", "ô", "o", "ö", "o", "ő", "o",
	"ř", "r",
	"š", "s", "ś", "s",
	"ť", "t",
	"ú", "u", "ů", "u", "ü", "u", "ű", "u",
	"ý", "y",
	"ž", "z", "ź", "z", "ż", "z",
)

// fold lowercases and strips common Czech/Slovak diacritics for search.
func fold(s string) string {
	s = strings.ToLower(s)
	s = diacriticReplacer.Replace(s)
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// ContainsFold reports whether s contains sub, ignoring case and diacritics.
func ContainsFold(s, sub string) bool {
	return strings.Contains(fold(s), fold(sub))
}