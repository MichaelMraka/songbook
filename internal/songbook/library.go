package songbook

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"unicode/utf8"
)

// Song is one Zpěvník song entry.
type Song struct {
	ID        string
	Title     string
	Author    string
	AuthorID  string
	Text      string
	Lang      string
	GroupName string
	Step      int // stored transposition from XML; 0 if absent
	YouTube   string
}

// Playlist is an ordered list of song IDs.
type Playlist struct {
	ID    string
	Title string
	Songs []string // song IDs in order
}

// AuthorCount is an author name with song count.
type AuthorCount struct {
	Name  string
	Count int
}

// Library holds the parsed song database in memory.
type Library struct {
	songs     []Song
	byID      map[string]*Song
	playlists []Playlist
}

// LoadFile parses a Zpěvník InetSongDb XML export.
func LoadFile(path string) (*Library, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Load(f)
}

// Load parses InetSongDb XML from r.
func Load(r io.Reader) (*Library, error) {
	dec := xml.NewDecoder(r)
	var root inetSongDb
	if err := dec.Decode(&root); err != nil {
		return nil, fmt.Errorf("decode InetSongDb: %w", err)
	}
	lib := &Library{
		songs:     make([]Song, 0, len(root.Songs)),
		byID:      make(map[string]*Song, len(root.Songs)),
		playlists: make([]Playlist, 0, len(root.Playlists.Items)),
	}
	for _, s := range root.Songs {
		song := Song{
			ID:        strings.TrimSpace(s.ID),
			Title:     strings.TrimSpace(s.Title),
			Author:    strings.TrimSpace(s.Author),
			AuthorID:  strings.TrimSpace(s.AuthorID),
			Text:      s.SongText,
			Lang:      strings.TrimSpace(s.Lang),
			GroupName: strings.TrimSpace(s.GroupName),
			YouTube:   strings.TrimSpace(s.YouTube),
		}
		if strings.TrimSpace(s.Step) != "" {
			var step int
			if _, err := fmt.Sscanf(strings.TrimSpace(s.Step), "%d", &step); err == nil {
				song.Step = step
			}
		}
		lib.songs = append(lib.songs, song)
	}
	for i := range lib.songs {
		lib.byID[lib.songs[i].ID] = &lib.songs[i]
	}
	for _, p := range root.Playlists.Items {
		pl := Playlist{
			ID:    strings.TrimSpace(p.ID),
			Title: strings.TrimSpace(p.Title),
		}
		for _, ps := range p.Songs.Items {
			id := strings.TrimSpace(ps.ID)
			if id != "" {
				pl.Songs = append(pl.Songs, id)
			}
		}
		lib.playlists = append(lib.playlists, pl)
	}
	sort.Slice(lib.songs, func(i, j int) bool {
		ai := fold(lib.songs[i].Author + "\x00" + lib.songs[i].Title)
		aj := fold(lib.songs[j].Author + "\x00" + lib.songs[j].Title)
		return ai < aj
	})
	// rebuild byID after sort (pointers into songs slice)
	lib.byID = make(map[string]*Song, len(lib.songs))
	for i := range lib.songs {
		lib.byID[lib.songs[i].ID] = &lib.songs[i]
	}
	sort.Slice(lib.playlists, func(i, j int) bool {
		return fold(lib.playlists[i].Title) < fold(lib.playlists[j].Title)
	})
	return lib, nil
}

func (l *Library) SongCount() int {
	if l == nil {
		return 0
	}
	return len(l.songs)
}

// Songs returns all songs sorted by author, then title.
func (l *Library) Songs() []Song {
	if l == nil {
		return nil
	}
	out := make([]Song, len(l.songs))
	copy(out, l.songs)
	return out
}

// SongByID looks up a song.
func (l *Library) SongByID(id string) (Song, bool) {
	if l == nil {
		return Song{}, false
	}
	s, ok := l.byID[id]
	if !ok || s == nil {
		return Song{}, false
	}
	return *s, true
}

// Playlists returns playlists sorted by title.
func (l *Library) Playlists() []Playlist {
	if l == nil {
		return nil
	}
	out := make([]Playlist, len(l.playlists))
	copy(out, l.playlists)
	return out
}

// PlaylistSongs resolves playlist entries to songs (skips missing IDs).
func (l *Library) PlaylistSongs(playlistID string) []Song {
	if l == nil {
		return nil
	}
	var pl *Playlist
	for i := range l.playlists {
		if l.playlists[i].ID == playlistID {
			pl = &l.playlists[i]
			break
		}
	}
	if pl == nil {
		return nil
	}
	out := make([]Song, 0, len(pl.Songs))
	for _, id := range pl.Songs {
		if s, ok := l.byID[id]; ok {
			out = append(out, *s)
		}
	}
	return out
}

// Authors returns authors with counts, sorted by name.
func (l *Library) Authors() []AuthorCount {
	if l == nil {
		return nil
	}
	counts := map[string]int{}
	for _, s := range l.songs {
		name := s.Author
		if name == "" {
			name = "Neznámý"
		}
		counts[name]++
	}
	out := make([]AuthorCount, 0, len(counts))
	for name, n := range counts {
		out = append(out, AuthorCount{Name: name, Count: n})
	}
	sort.Slice(out, func(i, j int) bool {
		return fold(out[i].Name) < fold(out[j].Name)
	})
	return out
}

// SongsByAuthor returns songs for an exact author name.
func (l *Library) SongsByAuthor(author string) []Song {
	if l == nil {
		return nil
	}
	var out []Song
	for _, s := range l.songs {
		if s.Author == author {
			out = append(out, s)
		}
	}
	return out
}

// SearchOptions controls Search behavior.
type SearchOptions struct {
	Fulltext bool // also match songtext
}

// FulltextSearchPrefixes mark a query as song-text search when used as the first rune.
// Example: "#neuroním" searches lyrics; "wabi" searches title/author only.
const FulltextSearchPrefixes = "#@*~!%$.:/?"

// ParseSearchQuery trims raw input and detects a leading fulltext prefix symbol.
// The prefix is stripped from the returned query.
func ParseSearchQuery(raw string) (query string, fulltext bool) {
	q := strings.TrimSpace(raw)
	if q == "" {
		return "", false
	}
	r, size := utf8.DecodeRuneInString(q)
	if size > 0 && strings.ContainsRune(FulltextSearchPrefixes, r) {
		return strings.TrimSpace(q[size:]), true
	}
	return q, false
}

// Search filters songs by title or author (diacritics-insensitive).
// When opts.Fulltext is set, songtext is searched too.
func (l *Library) Search(query string, opts SearchOptions) []Song {
	if l == nil {
		return nil
	}
	q := fold(strings.TrimSpace(query))
	if q == "" {
		return l.Songs()
	}
	var out []Song
	for _, s := range l.songs {
		if strings.Contains(fold(s.Title), q) || strings.Contains(fold(s.Author), q) {
			out = append(out, s)
			continue
		}
		if opts.Fulltext && strings.Contains(fold(s.Text), q) {
			out = append(out, s)
		}
	}
	return out
}

// SongsSorted returns songs ordered by mode. meta supplies local created/updated.
func (l *Library) SongsSorted(mode SortMode, meta map[string]SongMeta) []Song {
	if l == nil {
		return nil
	}
	out := l.Songs()
	switch mode {
	case SortBySongName:
		sort.SliceStable(out, func(i, j int) bool {
			return fold(out[i].Title+"\x00"+out[i].Author) < fold(out[j].Title+"\x00"+out[j].Author)
		})
	case SortByUpdated:
		sort.SliceStable(out, func(i, j int) bool {
			ui, uj := metaTime(meta, out[i].ID, true), metaTime(meta, out[j].ID, true)
			if ui != uj {
				return ui > uj
			}
			return fold(out[i].Author+"\x00"+out[i].Title) < fold(out[j].Author+"\x00"+out[j].Title)
		})
	case SortByCreated:
		sort.SliceStable(out, func(i, j int) bool {
			ci, cj := metaTime(meta, out[i].ID, false), metaTime(meta, out[j].ID, false)
			if ci != cj {
				return ci > cj
			}
			return fold(out[i].Author+"\x00"+out[i].Title) < fold(out[j].Author+"\x00"+out[j].Title)
		})
	default: // SortByAuthorName
		sort.SliceStable(out, func(i, j int) bool {
			return fold(out[i].Author+"\x00"+out[i].Title) < fold(out[j].Author+"\x00"+out[j].Title)
		})
	}
	return out
}

func metaTime(meta map[string]SongMeta, id string, updated bool) int64 {
	if meta == nil {
		return 0
	}
	m, ok := meta[id]
	if !ok {
		return 0
	}
	if updated {
		return m.LocalUpdated
	}
	return m.LocalCreated
}

// DisplayLabel returns "Author - Title".
func (s Song) DisplayLabel() string {
	return s.Author + " - " + s.Title
}

// XML structures matching Zpěvník export.
type inetSongDb struct {
	XMLName   xml.Name     `xml:"InetSongDb"`
	Songs     []xmlSong    `xml:"song"`
	Playlists xmlPlaylists `xml:"playlists"`
}

type xmlSong struct {
	ID        string `xml:"ID"`
	Lang      string `xml:"lang"`
	SongText  string `xml:"songtext"`
	Author    string `xml:"author"`
	AuthorID  string `xml:"authorId"`
	GroupName string `xml:"groupname"`
	Title     string `xml:"title"`
	Step      string `xml:"step"`
	YouTube   string `xml:"youtube"`
}

type xmlPlaylists struct {
	Items []xmlPlaylist `xml:"playlist"`
}

type xmlPlaylist struct {
	ID    string           `xml:"ID"`
	Title string           `xml:"title"`
	Songs xmlPlaylistSongs `xml:"playlist_songs"`
}

type xmlPlaylistSongs struct {
	Items []xmlPlaylistSong `xml:"playlist_song"`
}

type xmlPlaylistSong struct {
	ID         string `xml:"id,attr"`
	SourceType string `xml:"source_type,attr"`
}