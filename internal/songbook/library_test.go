package songbook_test

import (
	"path/filepath"
	"testing"

	"github.com/michael/songbook/internal/songbook"
)

func testExportPath(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..", "test-data", "export.zpk.xml")
}

func TestLoadExportCounts(t *testing.T) {
	lib, err := songbook.LoadFile(testExportPath(t))
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if got := lib.SongCount(); got != 551 {
		t.Fatalf("SongCount = %d, want 551", got)
	}
	if got := len(lib.Playlists()); got != 7 {
		t.Fatalf("Playlists = %d, want 7", got)
	}
}

func TestHarmonikaPlaylist(t *testing.T) {
	lib, err := songbook.LoadFile(testExportPath(t))
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	var pl *songbook.Playlist
	for i := range lib.Playlists() {
		if lib.Playlists()[i].Title == "Harmonika" {
			pl = &lib.Playlists()[i]
			break
		}
	}
	if pl == nil {
		t.Fatal("Harmonika playlist not found")
	}
	songs := lib.PlaylistSongs(pl.ID)
	want := []string{
		"Uhlíř Jaroslav - Severní vítr",
		"Neznámý - Míle  a míle",
		"Malina Brothers - McLaren",
		"Jaromír Nohavica - Převez mě, příteli",
		"Pavel Lohonka Žalman - Barevný šál",
	}
	if len(songs) != len(want) {
		t.Fatalf("len = %d, want %d", len(songs), len(want))
	}
	for i, s := range songs {
		got := s.Author + " - " + s.Title
		if got != want[i] {
			t.Errorf("[%d] = %q, want %q", i, got, want[i])
		}
	}
}

func TestSearchWabi(t *testing.T) {
	lib, err := songbook.LoadFile(testExportPath(t))
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	hits := lib.Search("wabi")
	if len(hits) < 10 {
		t.Fatalf("Search(wabi) got %d hits, want at least 10", len(hits))
	}
	found := false
	for _, s := range hits {
		if s.Title == "Bílá vrána" && containsFold(s.Author, "wabi") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected Bílá vrána by Wabi Daněk in search results")
	}
}

func TestSearchDiacritics(t *testing.T) {
	lib, err := songbook.LoadFile(testExportPath(t))
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	hits := lib.Search("carpe")
	found := false
	for _, s := range hits {
		if s.Title == "Carpe diem" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected Carpe diem")
	}
}

func TestAuthorsIncludeWabi(t *testing.T) {
	lib, err := songbook.LoadFile(testExportPath(t))
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	authors := lib.Authors()
	var count int
	for _, a := range authors {
		if a.Name == "Daněk, Wabi" || a.Name == "Wabi Daněk" {
			count = a.Count
			break
		}
	}
	// Export uses "Daněk, Wabi" in some places and "Wabi Daněk" in others;
	// accept whichever grouping the library uses, but require a sizable count.
	if count < 20 {
		// try alternate name
		for _, a := range authors {
			if containsFold(a.Name, "wabi") && containsFold(a.Name, "dan") {
				count = a.Count
				break
			}
		}
	}
	if count < 20 {
		t.Fatalf("Wabi author count = %d, want >= 20", count)
	}
}

func containsFold(s, sub string) bool {
	return songbook.ContainsFold(s, sub)
}