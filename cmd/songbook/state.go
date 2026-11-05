//go:build pocketbook

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	ink "github.com/dennwc/inkview"

	"github.com/michael/songbook/internal/songbook"
)

const (
	songbookDir        = ink.FlashDir + "/songbook"
	defaultLibraryPath = songbookDir + "/library.xml"
	defaultSettingsPath = songbookDir + "/settings.json"
)

// appState owns durable library + settings paths and load/save helpers.
type appState struct {
	LibraryPath  string
	SettingsPath string
	Lib          *songbook.Library
	Settings     songbook.Settings
}

func newAppState() *appState {
	return &appState{
		LibraryPath:  defaultLibraryPath,
		SettingsPath: defaultSettingsPath,
		Lib:          songbook.NewEmpty(),
		Settings:     songbook.DefaultSettings(),
	}
}

func (s *appState) load() error {
	settings, err := songbook.LoadSettingsFile(s.SettingsPath)
	if err != nil {
		return fmt.Errorf("settings: %w", err)
	}
	s.Settings = settings

	lib, err := songbook.LoadFile(s.LibraryPath)
	if err != nil {
		if os.IsNotExist(err) {
			s.Lib = songbook.NewEmpty()
			return err
		}
		return err
	}
	s.Lib = lib
	return nil
}

func (s *appState) ensureDir() error {
	return os.MkdirAll(filepath.Dir(s.LibraryPath), 0o755)
}

func (s *appState) saveLibrary() error {
	if err := s.ensureDir(); err != nil {
		return err
	}
	return s.Lib.SaveFile(s.LibraryPath)
}

func (s *appState) saveSettings() error {
	if err := s.ensureDir(); err != nil {
		return err
	}
	return songbook.SaveSettingsFile(s.SettingsPath, s.Settings)
}

// replaceLibrary swaps the in-memory library and persists XML.
func (s *appState) replaceLibrary(other *songbook.Library) error {
	s.Lib.ReplaceFrom(other)
	now := time.Now()
	for _, song := range s.Lib.Songs() {
		s.Settings.TouchSongMeta(song.ID, now)
	}
	if err := s.saveLibrary(); err != nil {
		return err
	}
	return s.saveSettings()
}

// mergeLibrary merges other into the current library and persists.
func (s *appState) mergeLibrary(other *songbook.Library) error {
	s.Lib.MergeFrom(other)
	now := time.Now()
	for _, song := range other.Songs() {
		s.Settings.TouchSongMeta(song.ID, now)
	}
	if err := s.saveLibrary(); err != nil {
		return err
	}
	return s.saveSettings()
}

// upsertSong updates one song, touches meta, and saves XML + settings.
func (s *appState) upsertSong(song songbook.Song) error {
	if err := s.Lib.Upsert(song); err != nil {
		return err
	}
	s.Settings.TouchSongMeta(song.ID, time.Now())
	if err := s.saveLibrary(); err != nil {
		return err
	}
	return s.saveSettings()
}