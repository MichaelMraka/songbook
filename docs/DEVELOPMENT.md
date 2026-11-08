# Developer documentation

Offline songbook for PocketBook InkPad 3 Pro (**PB740**), compatible with [Karel Hovorka’s Android Songbook](https://play.google.com/store/apps/details?id=eu.karelhovorka.zpevnik&hl=en) XML exports (`InetSongDb` / `.xml`).

For end-user install and usage, see the [README](../README.md).

## Layout

| Path | Role |
|------|------|
| `internal/songbook` | Pure Go: XML parse/save, search, chord layout/transpose, settings (host-tested) |
| `cmd/songbook` | InkView UI (`//go:build pocketbook`) |
| `test-data/export.zpk.xml` | Sample Android export |
| `docs/images/` | Device screenshots for docs |
| `docs/superpowers/` | Design/plan notes |

## Architecture

- **Library** lives in `/mnt/ext1/songbook/library.xml` (mutable; import can Replace or Merge).
- **Settings** live in `/mnt/ext1/songbook/settings.json` (sort, fonts, chord highlight, per-song meta).
- UI is PocketBook InkView via [`dennwc/inkview`](https://github.com/dennwc/inkview): menus (`OpenMenu`), keyboard (local cgo wrapper that prefills the edit buffer), file browse for import (also `OpenMenu`), system panel without FB offset.

```text
internal/songbook          cmd/songbook (pocketbook)
─────────────────          ─────────────────────────
Library / Search    <──    app state, tabs, song view
Chords / Transpose  <──    draw + hardware keys
Settings JSON       <──    Menu / song menu
XML load/save       <──    Import browser, edit song
```

## Host tests

```bash
make test
# or: go test ./internal/...
```

## Build for device (Docker/Podman)

Requires Docker or Podman and network to pull the PocketBook Go SDK image:

```bash
make build
# produces ./songbook.app (ARM ELF for PB740)
```

Default image: `docker.io/5keeve/pocketbook-go-sdk:6.3.0-b288-v1` (B288 / PB740). Override with `DOCKER_IMAGE=...` or `CONTAINER=docker|podman`.

## Install paths (device)

On the device, internal storage is `/mnt/ext1`. Over USB that same volume appears as the **drive root** (`applications/`, `books/`, … — not a `mnt` or `ext1` folder).

```text
applications/songbook.app
songbook/library.xml           # preferred library path
songbook/settings.json         # created automatically
```

Import also accepts dropping a file as `songbook/import.xml`, `songbook/export.zpk.xml`, or `songbook/export.xml`, then **Menu → Import library…** (Replace or Merge into `library.xml`). The in-app browser starts at FlashDir (`/mnt/ext1`).

## Notable UI implementation details

| Area | Notes |
|------|--------|
| Keyboard | `cmd/songbook/keyboard.go` — native `OpenKeyboard` with buffer **prefill**; firmware cancel often returns `""` (ignored when prefill was non-empty). |
| Import browser | `OpenMenu`-based directory listing (dirs first); not `OpenList` (unreliable dismiss on this firmware). |
| Search | Default title/author; leading `#@*~!%$.:/?` enables fulltext (lyrics). Last non-empty query remembered for keyboard prefill. |
| Song text edit | Line-oriented list (insert/delete/done); monospace face so spaced chord lines stay aligned. |
| Fonts | UI chrome fixed; list size vs song size/face independent; optional per-song overrides. |
| Panel | `PANEL_NO_FB_OFFSET` so content is not double-padded under the system panel. |

## Device smoke checklist

- [ ] App appears under Applications and launches
- [ ] Library loads or empty/import hint shown
- [ ] Search / Songs / Authors / Playlists / Menu work
- [ ] Open a song: chords above lyrics
- [ ] Transpose (Left/Right) and font (Minus/Plus) redraw correctly
- [ ] Import via file browser (Replace / Merge)
- [ ] Edit author/title; edit song text by line; keyboard cancel does not wipe fields
