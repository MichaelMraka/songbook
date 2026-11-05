# PocketBook Songbook (PB740)

Offline songbook for PocketBook InkPad 3 Pro (**PB740**) compatible with [Karel Hovorka’s Android Songbook](https://play.google.com/store/apps/details?id=eu.karelhovorka.zpevnik&hl=en) XML exports (`InetSongDb` / `.zpk.xml`).

## Features (v1)

- Import Zpěvník XML library
- **Songs / Authors / Playlists** browsing
- Offline filter (title/author, diacritics-insensitive)
- Song view with chords above lyrics, transpose, font size, scroll

## Layout

| Path | Role |
|------|------|
| `internal/songbook` | Pure Go: XML parse, search, chord layout/transpose (host-tested) |
| `cmd/songbook` | InkView UI (`//go:build pocketbook`) |
| `test-data/export.zpk.xml` | Sample Android export |
| `test-data/*.jpg` | Reference screenshots |

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

## Install on PB740

On the device, internal storage is `/mnt/ext1`. Over USB that same volume appears as the **drive root** (you will see `applications/`, `books/`, … — not a `mnt` or `ext1` folder).

1. Connect the reader over USB.
2. Copy `songbook.app` into `applications/` on the USB drive.
3. Create `songbook/` next to `applications/` and copy your export:

   ```text
   applications/songbook.app
   songbook/library.xml          # preferred; app reads /mnt/ext1/songbook/library.xml
   ```

   Or drop the export as one of `songbook/import.xml`, `songbook/export.zpk.xml`, or `songbook/export.xml`, then use **Menu → Import** in the app (copies into `library.xml`).

4. Eject/unmount safely, then open **Applications → songbook** on the device.

## Controls

| Action | How |
|--------|-----|
| Switch tab | Bottom tabs or Left/Right (on list) |
| Move / scroll | Up/Down, or tap list row / song body |
| Open | OK or tap |
| Back | Back key |
| Filter (Songs) | Tap header, or use on-screen keyboard |
| Transpose | Left / Right in song view |
| Font size | Minus / Plus in song view |
| Import | Menu key |

## Device smoke checklist

- [ ] App appears under Applications and launches
- [ ] Library loads (status shows song count) or empty hint shown
- [ ] Songs / Authors / Playlists tabs work
- [ ] Harmonika playlist shows 5 songs
- [ ] Open a song: chords above lyrics
- [ ] Transpose and font change redraw correctly
- [ ] Import from `export.zpk.xml` refreshes the library
