package songbook

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

// FontSizeStep is a named UI size preset.
type FontSizeStep string

const (
	FontSmallest FontSizeStep = "smallest"
	FontSmall    FontSizeStep = "small"
	FontMedium   FontSizeStep = "medium"
	FontBig      FontSizeStep = "big"
	FontBigger   FontSizeStep = "bigger"
)

// SortMode is the default library sort.
type SortMode string

const (
	SortBySongName   SortMode = "song_name"
	SortByAuthorName SortMode = "author_name"
	SortByUpdated    SortMode = "updated"
	SortByCreated    SortMode = "created"
)

// ChordHighlightStyle controls chord rendering emphasis.
type ChordHighlightStyle string

const (
	ChordHighlightBold ChordHighlightStyle = "bold"
	ChordHighlightOff  ChordHighlightStyle = "off"
)

// ChordColor is an e-ink-friendly named chord color (when device supports gray/color).
type ChordColor string

const (
	ChordColorDefault ChordColor = ""
	ChordColorYellow  ChordColor = "yellow"
	ChordColorGreen   ChordColor = "green"
	ChordColorRed     ChordColor = "red"
)

// SongMeta holds per-song UI overrides and local timestamps (not in Zpěvník XML).
type SongMeta struct {
	FontFamily   string       `json:"fontFamily,omitempty"`
	FontSize     FontSizeStep `json:"fontSize,omitempty"`
	LocalCreated int64        `json:"localCreated,omitempty"` // unix seconds
	LocalUpdated int64        `json:"localUpdated,omitempty"`
}

// Settings is the app sidecar stored as settings.json.
type Settings struct {
	Sort              SortMode            `json:"sort"`
	ListFontSize      FontSizeStep        `json:"listFontSize"`
	SongFontSize      FontSizeStep        `json:"songFontSize"`
	FontFamily        string              `json:"fontFamily"`
	ChordHighlight    ChordHighlightStyle `json:"chordHighlight"`
	ChordColor        ChordColor          `json:"chordColor"`
	SearchFulltext    bool                `json:"searchFulltext"`
	Songs             map[string]SongMeta `json:"songs"`
}

// DefaultSettings returns sensible defaults.
func DefaultSettings() Settings {
	return Settings{
		Sort:           SortByAuthorName,
		ListFontSize:   FontMedium,
		SongFontSize:   FontMedium,
		FontFamily:     "",
		ChordHighlight: ChordHighlightBold,
		ChordColor:     ChordColorDefault,
		SearchFulltext: false,
		Songs:          map[string]SongMeta{},
	}
}

// LoadSettingsFile reads settings JSON; missing file yields defaults.
func LoadSettingsFile(path string) (Settings, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultSettings(), nil
		}
		return Settings{}, err
	}
	defer f.Close()
	return LoadSettings(f)
}

// LoadSettings decodes settings from r.
func LoadSettings(r io.Reader) (Settings, error) {
	s := DefaultSettings()
	dec := json.NewDecoder(r)
	if err := dec.Decode(&s); err != nil {
		return Settings{}, fmt.Errorf("settings: %w", err)
	}
	if s.Songs == nil {
		s.Songs = map[string]SongMeta{}
	}
	if s.Sort == "" {
		s.Sort = SortByAuthorName
	}
	if s.ListFontSize == "" {
		s.ListFontSize = FontMedium
	}
	if s.SongFontSize == "" {
		s.SongFontSize = FontMedium
	}
	return s, nil
}

// SaveSettingsFile writes settings JSON.
func SaveSettingsFile(path string, s Settings) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := SaveSettings(f, s); err != nil {
		return err
	}
	return f.Close()
}

// SaveSettings encodes settings to w.
func SaveSettings(w io.Writer, s Settings) error {
	if s.Songs == nil {
		s.Songs = map[string]SongMeta{}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(s)
}

// TouchSongMeta sets localUpdated (and localCreated if unset) for id.
func (s *Settings) TouchSongMeta(id string, now time.Time) {
	if s.Songs == nil {
		s.Songs = map[string]SongMeta{}
	}
	m := s.Songs[id]
	ts := now.Unix()
	if m.LocalCreated == 0 {
		m.LocalCreated = ts
	}
	m.LocalUpdated = ts
	s.Songs[id] = m
}

// SongMeta returns meta for id (zero value if absent).
func (s Settings) SongMeta(id string) SongMeta {
	if s.Songs == nil {
		return SongMeta{}
	}
	return s.Songs[id]
}

// FontSizePixels maps a named step to a pixel size (list/song UI).
func FontSizePixels(step FontSizeStep, base int) int {
	if base <= 0 {
		base = 39
	}
	switch step {
	case FontSmallest:
		return int(float64(base) * 0.7)
	case FontSmall:
		return int(float64(base) * 0.85)
	case FontMedium:
		return base
	case FontBig:
		return int(float64(base) * 1.2)
	case FontBigger:
		return int(float64(base) * 1.4)
	default:
		return base
	}
}