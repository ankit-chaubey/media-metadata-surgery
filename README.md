# Media Metadata Surgery v0.1.2

**Precision-focused, fully offline CLI for viewing, editing, and stripping metadata from any media or document file.**

The core engine is **Go** — fast, correct, zero dependencies. Distribution via **pip**.

---

## What's new in v0.1.2

v0.1.1 supported only JPEG read-only.  
v0.1.2 expands to **28 formats** across **4 media categories**:

| Category   | Formats |
|------------|---------|
| 🖼 Image    | JPEG, PNG, GIF, WebP, TIFF, BMP, HEIC/HEIF, SVG |
| 🎵 Audio    | MP3, FLAC, OGG, Opus, M4A/AAC, WAV, AIFF |
| 🎬 Video    | MP4, MOV, MKV, WebM, AVI, WMV, FLV |
| 📄 Document | PDF, DOCX, XLSX, PPTX, ODT, EPUB |

---

## Installation

```bash
pip install surgery
```

Or build from source (requires Go 1.21+):

```bash
git clone https://github.com/ankit-chaubey/media-metadata-surgery
cd media-metadata-surgery
go build -o surgery ./cli
```

---

## Commands

| Command   | Description |
|-----------|-------------|
| `view`    | View all metadata for a file |
| `edit`    | Add or update metadata fields |
| `strip`   | Remove metadata from a file |
| `info`    | Show format detection and capabilities |
| `formats` | List all supported formats |
| `batch`   | Process all files in a directory |
| `version` | Print version |

---

## view — read metadata

```bash
surgery view photo.jpg
surgery view --json audio.mp3
surgery view --verbose document.pdf
```

**Output (JPEG):**
```
File  : photo.jpg
Format: JPEG

── EXIF ──
  Make:                          vivo               [editable]
  Model:                         vivo T1 5G         [editable]
  DateTimeOriginal:              2026:02:04 18:44:10
  GPSLatitude:                   18 deg 20' 47.19"
  GPSLongitude:                  84 deg 25' 25.39"

── IPTC ──
  Keywords:                      travel, india
```

**Output (MP3):**
```
File  : song.mp3
Format: MP3

── ID3v2.4.0 ──
  Title:                         Bohemian Rhapsody    [editable]
  Artist:                        Queen                [editable]
  Album:                         A Night at the Opera [editable]
  Year:                          1975
  Genre:                         Rock
```

---

## edit — update metadata

```bash
# Set fields (in-place)
surgery edit --set "Artist=John Doe" --set "Title=My Song" audio.mp3

# Write to new file
surgery edit --set "Make=Canon" --out edited.jpg photo.jpg

# Delete a field
surgery edit --delete UserComment photo.jpg

# Preview without writing
surgery edit --dry-run --set "Title=Report 2024" document.docx
```

### Editable fields by format

| Format | Fields |
|--------|--------|
| **JPEG** | Make, Model, Software, Artist, Copyright, ImageDescription, UserComment, DateTime, DateTimeOriginal, DateTimeDigitized |
| **PNG** | Title, Author, Description, Copyright, Comment, Creation Time, Source, Software |
| **MP3** | Title, Artist, Album, Year, Genre, Comment, TrackNumber, AlbumArtist, Composer, Lyrics, Copyright |
| **FLAC** | TITLE, ARTIST, ALBUM, DATE, GENRE, COMMENT, TRACKNUMBER, ALBUMARTIST, COMPOSER, COPYRIGHT |
| **MP4/MOV** | title, artist, album, comment, year, genre, description, copyright |
| **PDF** | Title, Author, Subject, Keywords, Creator, Producer |
| **DOCX/XLSX/PPTX** | Title, Subject, Author, Keywords, Description, LastModifiedBy, Category |

---

## strip — remove metadata

```bash
# Remove all metadata (in-place)
surgery strip photo.jpg

# Remove to new file
surgery strip --out clean.jpg photo.jpg

# Remove only GPS coordinates
surgery strip --gps-only photo.jpg

# Remove all EXCEPT EXIF
surgery strip --keep exif photo.jpg

# Preview
surgery strip --dry-run audio.mp3
```

**Privacy use-case — strip location before uploading:**
```bash
surgery strip --gps-only holiday_photo.jpg
```

---

## info — detect format

```bash
surgery info video.mkv
```
```
File            : video.mkv
Detected Format : Matroska MKV  (id: mkv)
Media Type      : video
Can View        : true
Can Edit        : false
Can Strip       : false
Notes           : EBML-based container. View only in v0.1.2.
```

---

## formats — list all formats

```bash
surgery formats
surgery formats --type audio
```

```
Format ID    Name                   Type        View  Edit  Strip  Extensions
──────────────────────────────────────────────────────────────────────────────
jpeg         JPEG                   image       ✓     ✓     ✓      .jpg .jpeg
png          PNG                    image       ✓     ✓     ✓      .png
mp3          MP3                    audio       ✓     ✓     ✓      .mp3
flac         FLAC                   audio       ✓     ✓     ✓      .flac
mp4          MP4                    video       ✓     ✓     ✓      .mp4
pdf          PDF                    document    ✓     ✓     ✓      .pdf
docx         DOCX                   document    ✓     ✓     ✓      .docx
...
             TOTAL                              28    9     13     (28 formats)
```

---

## batch — process directories

```bash
# View all files
surgery batch view ./photos

# View recursively as JSON
surgery batch view --json --recursive ./media

# Strip all files, output to new directory
surgery batch strip --out ./clean ./photos

# Strip recursively in-place
surgery batch strip --recursive ./photos

# Apply copyright to all editable files
surgery batch edit --set "Copyright=ACME Corp 2024" ./docs

# Dry-run
surgery batch edit --dry-run --set "Author=Ankit" ./documents
```

---

## Capability matrix

| Format | View | Edit | Strip | Metadata types |
|--------|------|------|-------|----------------|
| JPEG   | ✓    | ✓    | ✓     | EXIF, XMP, IPTC |
| PNG    | ✓    | ✓    | ✓     | tEXt, iTXt, eXIf |
| GIF    | ✓    | —    | ✓     | Comment blocks |
| WebP   | ✓    | —    | ✓     | EXIF, XMP |
| TIFF   | ✓    | —    | —     | EXIF IFDs |
| BMP    | ✓    | —    | —     | Header fields |
| HEIC   | ✓    | —    | —     | EXIF (ISOBMFF) |
| SVG    | ✓    | —    | —     | title, desc, XMP |
| MP3    | ✓    | ✓    | ✓     | ID3v1, ID3v2 |
| FLAC   | ✓    | ✓    | ✓     | Vorbis Comments |
| OGG    | ✓    | —    | —     | Vorbis Comments |
| Opus   | ✓    | —    | —     | Vorbis Comments |
| M4A    | ✓    | —    | —     | iTunes atoms |
| WAV    | ✓    | —    | ✓     | LIST INFO |
| AIFF   | ✓    | —    | —     | NAME, AUTH, ANNO |
| MP4    | ✓    | ✓    | ✓     | iTunes atoms |
| MOV    | ✓    | —    | ✓     | udta atoms |
| MKV    | ✓    | —    | —     | EBML tags |
| WebM   | ✓    | —    | —     | EBML tags |
| AVI    | ✓    | —    | —     | RIFF INFO |
| WMV    | ✓    | —    | —     | ASF Content Desc |
| FLV    | ✓    | —    | —     | onMetaData AMF |
| PDF    | ✓    | ✓    | ✓     | Info dict, XMP |
| DOCX   | ✓    | ✓    | ✓     | OPC core/app props |
| XLSX   | ✓    | ✓    | ✓     | OPC core/app props |
| PPTX   | ✓    | ✓    | ✓     | OPC core/app props |
| ODT    | ✓    | —    | —     | ODF meta.xml |
| EPUB   | ✓    | —    | —     | OPF package metadata |

---

## Security & privacy

- All operations are **fully offline** — no network access
- No background processes, no telemetry
- Viewing never modifies files
- `--out` always writes to a **new** file
- `--dry-run` previews changes before any write

---

## Project structure

```
media-metadata-surgery/
├── cli/main.go              # Commands: view, edit, strip, info, formats, batch
├── core/
│   ├── types.go             # Handler interface, Metadata, MetaField, options
│   ├── detect.go            # Magic-byte + extension format detection (28 formats)
│   ├── output.go            # Text + JSON printer
│   ├── image/image.go       # JPEG/PNG/GIF/WebP/TIFF/BMP/HEIC/SVG handlers
│   ├── audio/audio.go       # MP3/FLAC/OGG/Opus/M4A/WAV/AIFF handlers
│   ├── video/video.go       # MP4/MOV/MKV/WebM/AVI/WMV/FLV handlers
│   └── document/document.go # PDF/DOCX/XLSX/PPTX/ODT/EPUB handlers
├── surgery/
│   ├── __init__.py
│   ├── __main__.py
│   └── bin/surgery          # Compiled binary (bundled at release)
├── go.mod / go.sum
├── setup.py / pyproject.toml
└── README.md
```

---

## License

Apache License 2.0

## Author

**Ankit Chaubey** — <https://github.com/ankit-chaubey>

## Philosophy

> Precision over features. Correctness over speed. Transparency over magic.
