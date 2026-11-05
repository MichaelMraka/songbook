package songbook_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/michael/songbook/internal/songbook"
)

func TestSettingsRoundTrip(t *testing.T) {
	s := songbook.DefaultSettings()
	s.SearchFulltext = true
	s.Sort = songbook.SortByUpdated
	s.TouchSongMeta("abc", time.Unix(1700000000, 0))

	var buf bytes.Buffer
	if err := songbook.SaveSettings(&buf, s); err != nil {
		t.Fatal(err)
	}
	got, err := songbook.LoadSettings(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if !got.SearchFulltext || got.Sort != songbook.SortByUpdated {
		t.Fatalf("%+v", got)
	}
	m := got.SongMeta("abc")
	if m.LocalCreated != 1700000000 || m.LocalUpdated != 1700000000 {
		t.Fatalf("%+v", m)
	}
}

func TestLoadSettingsMissingDefaults(t *testing.T) {
	s, err := songbook.LoadSettingsFile("/nonexistent/settings.json")
	if err != nil {
		t.Fatal(err)
	}
	if s.Sort != songbook.SortByAuthorName || s.ListFontSize != songbook.FontMedium {
		t.Fatalf("%+v", s)
	}
}