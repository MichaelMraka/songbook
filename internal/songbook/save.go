package songbook

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strconv"
)

const inetSongDbNS = "http://zpevnik.net/InetSongDb.xsd"

// SaveFile writes the library as InetSongDb XML.
func (l *Library) SaveFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := l.Save(f); err != nil {
		return err
	}
	return f.Close()
}

// Save writes InetSongDb XML to w.
func (l *Library) Save(w io.Writer) error {
	if l == nil {
		return fmt.Errorf("nil library")
	}
	if _, err := io.WriteString(w, xml.Header); err != nil {
		return err
	}
	root := inetSongDbOut{
		Xmlns: inetSongDbNS,
	}
	for _, s := range l.songs {
		xs := xmlSong{
			ID:        s.ID,
			Lang:      s.Lang,
			SongText:  s.Text,
			Author:    s.Author,
			AuthorID:  s.AuthorID,
			GroupName: s.GroupName,
			Title:     s.Title,
			YouTube:   s.YouTube,
		}
		if s.Step != 0 {
			xs.Step = strconv.Itoa(s.Step)
		}
		root.Songs = append(root.Songs, xs)
	}
	for _, p := range l.playlists {
		xp := xmlPlaylist{
			ID:    p.ID,
			Title: p.Title,
		}
		for _, id := range p.Songs {
			xp.Songs.Items = append(xp.Songs.Items, xmlPlaylistSong{ID: id})
		}
		root.Playlists.Items = append(root.Playlists.Items, xp)
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "    ")
	if err := enc.Encode(root); err != nil {
		return err
	}
	if _, err := io.WriteString(w, "\n"); err != nil {
		return err
	}
	return nil
}

// inetSongDbOut is the write shape with xmlns attribute.
type inetSongDbOut struct {
	XMLName   xml.Name     `xml:"InetSongDb"`
	Xmlns     string       `xml:"xmlns,attr"`
	Songs     []xmlSong    `xml:"song"`
	Playlists xmlPlaylists `xml:"playlists"`
}