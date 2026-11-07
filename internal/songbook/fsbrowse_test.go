package songbook_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/michael/songbook/internal/songbook"
)

func TestListDirDirsFirstAndLabels(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "b.xml"), []byte("<x/>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "a_dir"), 0o755); err != nil {
		t.Fatal(err)
	}
	ents, err := songbook.ListDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(ents) != 2 {
		t.Fatalf("got %d entries: %#v", len(ents), ents)
	}
	if !ents[0].IsDir || ents[0].Name != "a_dir" {
		t.Fatalf("first want dir a_dir, got %#v", ents[0])
	}
	if ents[1].IsDir || ents[1].Name != "b.xml" {
		t.Fatalf("second want file b.xml, got %#v", ents[1])
	}
	if got := songbook.EntryLabel(ents[0]); got != "• a_dir/" {
		t.Fatalf("dir label %q", got)
	}
	if got := songbook.EntryLabel(ents[1]); got != "b.xml" {
		t.Fatalf("file label %q", got)
	}
}
