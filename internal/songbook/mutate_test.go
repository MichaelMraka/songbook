package songbook_test

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/michael/songbook/internal/songbook"
)

func TestXMLRoundTripCounts(t *testing.T) {
	lib, err := songbook.LoadFile(filepath.Join("..", "..", "test-data", "export.zpk.xml"))
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := lib.Save(&buf); err != nil {
		t.Fatal(err)
	}
	lib2, err := songbook.Load(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if lib2.SongCount() != lib.SongCount() {
		t.Fatalf("songs %d -> %d", lib.SongCount(), lib2.SongCount())
	}
	if len(lib2.Playlists()) != len(lib.Playlists()) {
		t.Fatalf("playlists %d -> %d", len(lib.Playlists()), len(lib2.Playlists()))
	}
	s1, ok := lib.SongByID("qz1t63zo")
	if !ok {
		t.Fatal("missing sample song")
	}
	s2, ok := lib2.SongByID("qz1t63zo")
	if !ok {
		t.Fatal("missing after round-trip")
	}
	if s1.Title != s2.Title || s1.Author != s2.Author || s1.Text != s2.Text {
		t.Fatalf("song mismatch:\n%q/%q\n%q/%q", s1.Author, s1.Title, s2.Author, s2.Title)
	}
}

func TestUpsertAndDelete(t *testing.T) {
	lib := songbook.NewEmpty()
	if err := lib.Upsert(songbook.Song{ID: "a", Title: "T", Author: "A", Text: "[C]x"}); err != nil {
		t.Fatal(err)
	}
	if lib.SongCount() != 1 {
		t.Fatal(lib.SongCount())
	}
	if err := lib.Upsert(songbook.Song{ID: "a", Title: "T2", Author: "A", Text: "[D]y"}); err != nil {
		t.Fatal(err)
	}
	s, _ := lib.SongByID("a")
	if s.Title != "T2" || s.Text != "[D]y" {
		t.Fatalf("%+v", s)
	}
	if !lib.Delete("a") || lib.SongCount() != 0 {
		t.Fatal("delete failed")
	}
}

func TestMergeAndReplace(t *testing.T) {
	base := songbook.NewEmpty()
	_ = base.Upsert(songbook.Song{ID: "1", Title: "Old", Author: "A"})
	_ = base.Upsert(songbook.Song{ID: "2", Title: "Keep", Author: "B"})

	inc := songbook.NewEmpty()
	_ = inc.Upsert(songbook.Song{ID: "1", Title: "New", Author: "A"})
	_ = inc.Upsert(songbook.Song{ID: "3", Title: "Added", Author: "C"})

	base.MergeFrom(inc)
	if base.SongCount() != 3 {
		t.Fatalf("merge count %d", base.SongCount())
	}
	s, _ := base.SongByID("1")
	if s.Title != "New" {
		t.Fatalf("merge overwrite: %q", s.Title)
	}

	base.ReplaceFrom(inc)
	if base.SongCount() != 2 {
		t.Fatalf("replace count %d", base.SongCount())
	}
	if _, ok := base.SongByID("2"); ok {
		t.Fatal("replace should drop id 2")
	}
}