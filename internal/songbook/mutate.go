package songbook

import (
	"fmt"
	"sort"
)

// NewEmpty returns an empty mutable library.
func NewEmpty() *Library {
	return &Library{
		songs:     nil,
		byID:      map[string]*Song{},
		playlists: nil,
	}
}

func (l *Library) ensure() {
	if l.byID == nil {
		l.byID = map[string]*Song{}
	}
}

func (l *Library) reindex() {
	l.ensure()
	l.byID = make(map[string]*Song, len(l.songs))
	for i := range l.songs {
		id := l.songs[i].ID
		if id == "" {
			continue
		}
		l.byID[id] = &l.songs[i]
	}
}

func (l *Library) sortSongsAuthorTitle() {
	sort.Slice(l.songs, func(i, j int) bool {
		ai := fold(l.songs[i].Author + "\x00" + l.songs[i].Title)
		aj := fold(l.songs[j].Author + "\x00" + l.songs[j].Title)
		return ai < aj
	})
	l.reindex()
}

func (l *Library) sortPlaylists() {
	sort.Slice(l.playlists, func(i, j int) bool {
		return fold(l.playlists[i].Title) < fold(l.playlists[j].Title)
	})
}

// Upsert inserts or replaces a song by ID. Empty ID is rejected.
func (l *Library) Upsert(s Song) error {
	if l == nil {
		return fmt.Errorf("nil library")
	}
	if s.ID == "" {
		return fmt.Errorf("song ID required")
	}
	l.ensure()
	if existing, ok := l.byID[s.ID]; ok {
		*existing = s
		return nil
	}
	l.songs = append(l.songs, s)
	l.sortSongsAuthorTitle()
	return nil
}

// Delete removes a song by ID. Playlist entries referencing it are left as-is
// (resolved as missing at read time).
func (l *Library) Delete(id string) bool {
	if l == nil || id == "" {
		return false
	}
	l.ensure()
	if _, ok := l.byID[id]; !ok {
		return false
	}
	out := l.songs[:0]
	for _, s := range l.songs {
		if s.ID != id {
			out = append(out, s)
		}
	}
	l.songs = out
	l.reindex()
	return true
}

// ReplaceFrom replaces all songs and playlists with other's content.
func (l *Library) ReplaceFrom(other *Library) {
	if l == nil {
		return
	}
	if other == nil {
		l.songs = nil
		l.playlists = nil
		l.reindex()
		return
	}
	l.songs = append([]Song(nil), other.songs...)
	l.playlists = append([]Playlist(nil), other.playlists...)
	l.sortSongsAuthorTitle()
	l.sortPlaylists()
}

// MergeFrom merges songs and playlists from other.
// Songs with the same ID are overwritten by incoming; new IDs are appended.
// Playlists with the same ID are overwritten; new playlist IDs are appended.
func (l *Library) MergeFrom(other *Library) {
	if l == nil || other == nil {
		return
	}
	l.ensure()
	for _, s := range other.songs {
		if s.ID == "" {
			continue
		}
		if existing, ok := l.byID[s.ID]; ok {
			*existing = s
			continue
		}
		l.songs = append(l.songs, s)
	}
	l.reindex()
	plByID := map[string]int{}
	for i := range l.playlists {
		plByID[l.playlists[i].ID] = i
	}
	for _, p := range other.playlists {
		if p.ID == "" {
			continue
		}
		if i, ok := plByID[p.ID]; ok {
			l.playlists[i] = p
			continue
		}
		l.playlists = append(l.playlists, p)
		plByID[p.ID] = len(l.playlists) - 1
	}
	l.sortSongsAuthorTitle()
	l.sortPlaylists()
}