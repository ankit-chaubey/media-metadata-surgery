// Package audio handles metadata for all audio formats:
// MP3 (ID3v1/v2), FLAC (Vorbis Comments), OGG, WAV, AIFF, M4A, Opus
package audio

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ankit-chaubey/media-metadata-surgery/core"
	"github.com/bogem/id3v2/v2"
	"github.com/dhowden/tag"
)

// Handler implements core.Handler for audio formats.
type Handler struct {
	format core.FormatID
}

// New returns an audio Handler for the given format.
func New(fmt core.FormatID) *Handler { return &Handler{format: fmt} }

func (h *Handler) Info() core.FormatInfo {
	return formatInfo[h.format]
}

var formatInfo = map[core.FormatID]core.FormatInfo{
	core.FmtMP3: {
		Name:        "MP3",
		Extensions:  []string{".mp3"},
		MediaType:   "audio",
		MIMETypes:   []string{"audio/mpeg"},
		CanView:     true,
		CanEdit:     true,
		CanStrip:    true,
		Notes:       "ID3v1 and ID3v2 tags. Full edit + strip support.",
		EditableFields: []string{
			"Title", "Artist", "Album", "Year", "Genre",
			"Comment", "TrackNumber", "AlbumArtist", "Composer",
			"Lyrics", "Copyright",
		},
	},
	core.FmtFLAC: {
		Name:        "FLAC",
		Extensions:  []string{".flac"},
		MediaType:   "audio",
		MIMETypes:   []string{"audio/flac"},
		CanView:     true,
		CanEdit:     true,
		CanStrip:    true,
		Notes:       "Vorbis Comment metadata blocks.",
		EditableFields: []string{
			"TITLE", "ARTIST", "ALBUM", "DATE", "GENRE",
			"COMMENT", "TRACKNUMBER", "ALBUMARTIST", "COMPOSER", "COPYRIGHT",
		},
	},
	core.FmtOGG: {
		Name:        "OGG",
		Extensions:  []string{".ogg", ".oga"},
		MediaType:   "audio",
		MIMETypes:   []string{"audio/ogg"},
		CanView:     true,
		CanEdit:     false,
		CanStrip:    false,
		Notes:       "Vorbis Comment blocks. View only in v0.1.2.",
	},
	core.FmtOpus: {
		Name:        "Opus",
		Extensions:  []string{".opus"},
		MediaType:   "audio",
		MIMETypes:   []string{"audio/opus"},
		CanView:     true,
		CanEdit:     false,
		CanStrip:    false,
		Notes:       "Vorbis Comment blocks. View only in v0.1.2.",
	},
	core.FmtM4A: {
		Name:        "M4A/AAC",
		Extensions:  []string{".m4a", ".aac"},
		MediaType:   "audio",
		MIMETypes:   []string{"audio/mp4", "audio/aac"},
		CanView:     true,
		CanEdit:     false,
		CanStrip:    false,
		Notes:       "iTunes-style MP4 atoms (©nam, ©ART, etc.). View only in v0.1.2.",
	},
	core.FmtWAV: {
		Name:        "WAV",
		Extensions:  []string{".wav"},
		MediaType:   "audio",
		MIMETypes:   []string{"audio/wav"},
		CanView:     true,
		CanEdit:     false,
		CanStrip:    true,
		Notes:       "LIST INFO and ID3 chunks.",
	},
	core.FmtAIFF: {
		Name:        "AIFF",
		Extensions:  []string{".aif", ".aiff"},
		MediaType:   "audio",
		MIMETypes:   []string{"audio/aiff"},
		CanView:     true,
		CanEdit:     false,
		CanStrip:    false,
		Notes:       "FORM/AIFF metadata chunks. View only in v0.1.2.",
	},
}

// ──────────────────────────────────────────────────────────────────────────────
// View
// ──────────────────────────────────────────────────────────────────────────────

func (h *Handler) View(path string) (*core.Metadata, error) {
	m := &core.Metadata{FilePath: path}
	ext := strings.ToLower(filepath.Ext(path))
	_ = ext

	switch h.format {
	case core.FmtMP3, core.FmtFLAC, core.FmtOGG, core.FmtOpus, core.FmtM4A:
		m.Format = formatInfo[h.format].Name
		return viewWithDhowden(path, m)
	case core.FmtWAV:
		m.Format = "WAV"
		return viewWAV(path, m)
	case core.FmtAIFF:
		m.Format = "AIFF"
		return viewAIFF(path, m)
	default:
		m.Format = strings.ToUpper(strings.TrimPrefix(ext, "."))
		return viewWithDhowden(path, m)
	}
}

// viewWithDhowden uses the dhowden/tag library to read audio metadata.
func viewWithDhowden(path string, m *core.Metadata) (*core.Metadata, error) {
	f, err := os.Open(path)
	if err != nil {
		return m, err
	}
	defer f.Close()

	t, err := tag.ReadFrom(f)
	if err != nil {
		return m, fmt.Errorf("could not read tags: %w", err)
	}

	// Determine category label
	cat := string(t.Format())
	if cat == "" {
		cat = "Audio Tags"
	}

	add := func(key, val string, editable bool) {
		if val != "" {
			m.Fields = append(m.Fields, core.MetaField{
				Key:      key,
				Value:    val,
				Category: cat,
				Editable: editable,
			})
		}
	}

	editable := (m.Format == "MP3" || m.Format == "FLAC")

	add("Title", t.Title(), editable)
	add("Artist", t.Artist(), editable)
	add("Album", t.Album(), editable)
	add("AlbumArtist", t.AlbumArtist(), editable)
	add("Composer", t.Composer(), editable)
	add("Genre", t.Genre(), editable)
	add("Comment", t.Comment(), editable)
	if t.Year() != 0 {
		add("Year", fmt.Sprintf("%d", t.Year()), editable)
	}
	track, total := t.Track()
	if track != 0 {
		trackStr := fmt.Sprintf("%d", track)
		if total != 0 {
			trackStr = fmt.Sprintf("%d/%d", track, total)
		}
		add("TrackNumber", trackStr, editable)
	}
	disc, totalDisc := t.Disc()
	if disc != 0 {
		discStr := fmt.Sprintf("%d", disc)
		if totalDisc != 0 {
			discStr = fmt.Sprintf("%d/%d", disc, totalDisc)
		}
		add("DiscNumber", discStr, editable)
	}
	if t.Lyrics() != "" {
		add("Lyrics", t.Lyrics(), editable)
	}

	// Raw tags
	for k, v := range t.Raw() {
		if v == nil {
			continue
		}
		// Skip keys already displayed
		switch strings.ToLower(k) {
		case "title", "artist", "album", "albumartist", "composer",
			"genre", "comment", "year", "date", "track", "tracknumber",
			"disc", "discnumber", "lyrics":
			continue
		}
		valStr := ""
		switch vt := v.(type) {
		case string:
			valStr = vt
		case []string:
			valStr = strings.Join(vt, "; ")
		case int:
			valStr = fmt.Sprintf("%d", vt)
		default:
			b, _ := json.Marshal(v)
			valStr = string(b)
		}
		if valStr != "" && len(valStr) < 512 {
			m.Fields = append(m.Fields, core.MetaField{
				Key:      k,
				Value:    valStr,
				Category: cat + " (raw)",
				Editable: false,
			})
		}
	}

	return m, nil
}

// ─── WAV ─────────────────────────────────────────────────────────────────────

// WAV INFO field IDs → human names
var infoChunkNames = map[string]string{
	"IARL": "ArchivalLocation",
	"IART": "Artist",
	"ICMS": "Commissioned",
	"ICMT": "Comment",
	"ICOP": "Copyright",
	"ICRD": "DateCreated",
	"ICRP": "Cropped",
	"IDIM": "Dimensions",
	"IDPI": "DotsPerInch",
	"IENG": "Engineer",
	"IGNR": "Genre",
	"IKEY": "Keywords",
	"ILGT": "Lightness",
	"IMED": "Medium",
	"INAM": "Title",
	"IPLT": "NumberOfColors",
	"IPRD": "Product",
	"ISBJ": "Subject",
	"ISFT": "Software",
	"ISHP": "Sharpness",
	"ISRC": "Source",
	"ISRF": "SourceForm",
	"ITCH": "Technician",
}

func viewWAV(path string, m *core.Metadata) (*core.Metadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return m, err
	}
	if len(data) < 12 {
		return m, fmt.Errorf("WAV too short")
	}

	// RIFF header info
	sampleRate := binary.LittleEndian.Uint32(data[24:28])
	channels := binary.LittleEndian.Uint16(data[22:24])
	bitsPerSample := binary.LittleEndian.Uint16(data[34:36])

	m.Fields = append(m.Fields,
		core.MetaField{Key: "SampleRate", Value: fmt.Sprintf("%d Hz", sampleRate), Category: "WAV Header", Editable: false},
		core.MetaField{Key: "Channels", Value: fmt.Sprintf("%d", channels), Category: "WAV Header", Editable: false},
		core.MetaField{Key: "BitsPerSample", Value: fmt.Sprintf("%d", bitsPerSample), Category: "WAV Header", Editable: false},
	)

	// Scan LIST INFO chunk
	offset := 12
	for offset+8 <= len(data) {
		chunkID := string(data[offset : offset+4])
		chunkSize := int(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		offset += 8
		if offset+chunkSize > len(data) {
			break
		}
		if chunkID == "LIST" && offset+4 <= len(data) && string(data[offset:offset+4]) == "INFO" {
			pos := offset + 4
			end := offset + chunkSize
			for pos+8 <= end {
				infoID := string(data[pos : pos+4])
				infoSize := int(binary.LittleEndian.Uint32(data[pos+4 : pos+8]))
				pos += 8
				if pos+infoSize > end {
					break
				}
				val := strings.TrimRight(string(data[pos:pos+infoSize]), "\x00")
				if val != "" {
					name := infoChunkNames[infoID]
					if name == "" {
						name = infoID
					}
					m.Fields = append(m.Fields, core.MetaField{
						Key:      name,
						Value:    val,
						Category: "WAV INFO",
						Editable: false,
					})
				}
				pos += infoSize
				if infoSize%2 != 0 {
					pos++
				}
			}
		}
		// Also check for ID3 chunk
		if chunkID == "id3 " || chunkID == "ID3 " {
			f2, err := os.Open(path)
			if err == nil {
				defer f2.Close()
				t, err := tag.ReadFrom(f2)
				if err == nil && t.Title() != "" {
					m.Fields = append(m.Fields, core.MetaField{
						Key:      "Title",
						Value:    t.Title(),
						Category: "WAV ID3",
						Editable: false,
					})
				}
			}
		}
		offset += chunkSize
		if chunkSize%2 != 0 {
			offset++
		}
	}
	return m, nil
}

// ─── AIFF ────────────────────────────────────────────────────────────────────

func viewAIFF(path string, m *core.Metadata) (*core.Metadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return m, err
	}
	if len(data) < 12 {
		return m, fmt.Errorf("AIFF too short")
	}

	// FORM/AIFF structure
	offset := 12
	for offset+8 <= len(data) {
		chunkID := string(data[offset : offset+4])
		chunkSize := int(binary.BigEndian.Uint32(data[offset+4 : offset+8]))
		offset += 8
		if offset+chunkSize > len(data) {
			break
		}
		switch chunkID {
		case "COMM":
			if chunkSize >= 18 {
				channels := binary.BigEndian.Uint16(data[offset : offset+2])
				bitsPerSample := binary.BigEndian.Uint16(data[offset+6 : offset+8])
				m.Fields = append(m.Fields,
					core.MetaField{Key: "Channels", Value: fmt.Sprintf("%d", channels), Category: "AIFF", Editable: false},
					core.MetaField{Key: "BitsPerSample", Value: fmt.Sprintf("%d", bitsPerSample), Category: "AIFF", Editable: false},
				)
			}
		case "NAME":
			m.Fields = append(m.Fields, core.MetaField{
				Key:      "Title",
				Value:    strings.TrimRight(string(data[offset:offset+chunkSize]), "\x00"),
				Category: "AIFF",
				Editable: false,
			})
		case "AUTH":
			m.Fields = append(m.Fields, core.MetaField{
				Key:      "Author",
				Value:    strings.TrimRight(string(data[offset:offset+chunkSize]), "\x00"),
				Category: "AIFF",
				Editable: false,
			})
		case "(c) ":
			m.Fields = append(m.Fields, core.MetaField{
				Key:      "Copyright",
				Value:    strings.TrimRight(string(data[offset:offset+chunkSize]), "\x00"),
				Category: "AIFF",
				Editable: false,
			})
		case "ANNO":
			m.Fields = append(m.Fields, core.MetaField{
				Key:      "Annotation",
				Value:    strings.TrimRight(string(data[offset:offset+chunkSize]), "\x00"),
				Category: "AIFF",
				Editable: false,
			})
		case "ID3 ":
			// ID3 tag embedded in AIFF
			sub, err := tag.ReadFrom(bytes.NewReader(data[offset : offset+chunkSize]))
			if err == nil {
				addFromTag(sub, m, "AIFF ID3")
			}
		}
		offset += chunkSize
		if chunkSize%2 != 0 {
			offset++
		}
	}
	return m, nil
}

func addFromTag(t tag.Metadata, m *core.Metadata, cat string) {
	add := func(k, v string) {
		if v != "" {
			m.Fields = append(m.Fields, core.MetaField{Key: k, Value: v, Category: cat})
		}
	}
	add("Title", t.Title())
	add("Artist", t.Artist())
	add("Album", t.Album())
	add("Genre", t.Genre())
}

// ──────────────────────────────────────────────────────────────────────────────
// Edit
// ──────────────────────────────────────────────────────────────────────────────

func (h *Handler) Edit(path string, outPath string, opts core.EditOptions) error {
	out := core.ResolveOutPath(path, outPath)
	switch h.format {
	case core.FmtMP3:
		return editMP3(path, out, opts)
	case core.FmtFLAC:
		return editFLAC(path, out, opts)
	default:
		info := formatInfo[h.format]
		if !info.CanEdit {
			return fmt.Errorf("%s does not support metadata editing in v0.1.2", info.Name)
		}
		return fmt.Errorf("edit not yet implemented for %s", info.Name)
	}
}

// ─── MP3 Edit ─────────────────────────────────────────────────────────────────

func editMP3(path, outPath string, opts core.EditOptions) error {
	if opts.DryRun {
		fmt.Println("Dry-run: MP3 ID3 tags would be updated:")
		for k, v := range opts.Set {
			fmt.Printf("  %s = %s\n", k, v)
		}
		return nil
	}

	// Copy to outPath first if different
	if path != outPath {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.WriteFile(outPath, data, 0644); err != nil {
			return err
		}
	}

	t, err := id3v2.Open(outPath, id3v2.Options{Parse: true})
	if err != nil {
		return fmt.Errorf("could not open MP3: %w", err)
	}
	defer t.Close()

	// Apply deletions
	for _, k := range opts.Delete {
		// id3v2 uses 4-char frame IDs; map friendly names
		if fid := mp3FrameID(k); fid != "" {
			t.DeleteFrames(fid)
		}
	}

	// Apply sets
	for k, v := range opts.Set {
		switch strings.ToLower(k) {
		case "title":
			t.SetTitle(v)
		case "artist":
			t.SetArtist(v)
		case "album":
			t.SetAlbum(v)
		case "year":
			t.SetYear(v)
		case "genre":
			t.SetGenre(v)
		case "comment":
			t.AddCommentFrame(id3v2.CommentFrame{
				Encoding:    id3v2.EncodingUTF8,
				Language:    "eng",
				Description: "",
				Text:        v,
			})
		case "tracknumber":
			t.AddTextFrame(t.CommonID("Track number/Position in set"), id3v2.EncodingUTF8, v)
		case "albumartist":
			t.AddTextFrame("TPE2", id3v2.EncodingUTF8, v)
		case "composer":
			t.AddTextFrame(t.CommonID("Composer"), id3v2.EncodingUTF8, v)
		case "lyrics":
			t.AddUnsynchronisedLyricsFrame(id3v2.UnsynchronisedLyricsFrame{
				Encoding:          id3v2.EncodingUTF8,
				Language:          "eng",
				ContentDescriptor: "",
				Lyrics:            v,
			})
		case "copyright":
			t.AddTextFrame("TCOP", id3v2.EncodingUTF8, v)
		default:
			// Try as raw frame ID (e.g. "TIT2")
			if len(k) == 4 {
				t.AddTextFrame(k, id3v2.EncodingUTF8, v)
			} else {
				fmt.Printf("  Warning: unknown MP3 field %q — skipped\n", k)
			}
		}
	}

	return t.Save()
}

// mp3FrameID maps friendly names to ID3v2 frame IDs.
func mp3FrameID(name string) string {
	m := map[string]string{
		"title":        "TIT2",
		"artist":       "TPE1",
		"album":        "TALB",
		"year":         "TDRC",
		"genre":        "TCON",
		"comment":      "COMM",
		"tracknumber":  "TRCK",
		"albumartist":  "TPE2",
		"composer":     "TCOM",
		"lyrics":       "USLT",
		"copyright":    "TCOP",
	}
	return m[strings.ToLower(name)]
}

// ─── FLAC Edit ────────────────────────────────────────────────────────────────
// FLAC Vorbis Comment block format:
//   - 4 bytes vendor length (LE) + vendor string
//   - 4 bytes comment count (LE)
//   - for each comment: 4 bytes length (LE) + "KEY=VALUE"

func editFLAC(path, outPath string, opts core.EditOptions) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(data) < 4 || !bytes.Equal(data[0:4], []byte("fLaC")) {
		return fmt.Errorf("not a valid FLAC file")
	}

	if opts.DryRun {
		fmt.Println("Dry-run: FLAC Vorbis comments would be updated:")
		for k, v := range opts.Set {
			fmt.Printf("  %s = %s\n", k, v)
		}
		return nil
	}

	// Parse metadata blocks
	blocks, audioStart, err := parseFLACBlocks(data)
	if err != nil {
		return err
	}

	// Find VORBIS_COMMENT block (type 4)
	vcIdx := -1
	for i, b := range blocks {
		if b.blockType == 4 {
			vcIdx = i
			break
		}
	}

	// Build updated Vorbis comment block
	existing := map[string]string{}
	if vcIdx >= 0 {
		parseVorbisComments(blocks[vcIdx].data, existing)
	}

	// Apply sets (uppercase keys per Vorbis spec)
	for k, v := range opts.Set {
		existing[strings.ToUpper(k)] = v
	}
	// Apply deletes
	for _, k := range opts.Delete {
		delete(existing, strings.ToUpper(k))
	}

	newVC := buildVorbisComment(existing)
	if vcIdx >= 0 {
		blocks[vcIdx].data = newVC
	} else {
		// Insert a new VORBIS_COMMENT block after STREAMINFO
		newBlock := flacBlock{blockType: 4, data: newVC}
		blocks = append([]flacBlock{blocks[0], newBlock}, blocks[1:]...)
	}

	return writeFLAC(outPath, blocks, data[audioStart:])
}

type flacBlock struct {
	blockType byte
	data      []byte
}

func parseFLACBlocks(data []byte) ([]flacBlock, int, error) {
	var blocks []flacBlock
	i := 4 // skip "fLaC"
	for i+4 <= len(data) {
		header := binary.BigEndian.Uint32(data[i : i+4])
		isLast := (header >> 31) == 1
		blockType := byte((header >> 24) & 0x7F)
		length := int(header & 0xFFFFFF)
		i += 4
		if i+length > len(data) {
			return nil, i, fmt.Errorf("FLAC block truncated")
		}
		blockData := append([]byte{}, data[i:i+length]...)
		blocks = append(blocks, flacBlock{blockType: blockType, data: blockData})
		i += length
		if isLast {
			break
		}
	}
	return blocks, i, nil
}

func parseVorbisComments(data []byte, out map[string]string) {
	if len(data) < 4 {
		return
	}
	vendorLen := int(binary.LittleEndian.Uint32(data[0:4]))
	pos := 4 + vendorLen
	if pos+4 > len(data) {
		return
	}
	count := int(binary.LittleEndian.Uint32(data[pos : pos+4]))
	pos += 4
	for i := 0; i < count && pos+4 <= len(data); i++ {
		cLen := int(binary.LittleEndian.Uint32(data[pos : pos+4]))
		pos += 4
		if pos+cLen > len(data) {
			break
		}
		comment := string(data[pos : pos+cLen])
		pos += cLen
		eq := strings.Index(comment, "=")
		if eq > 0 {
			out[strings.ToUpper(comment[:eq])] = comment[eq+1:]
		}
	}
}

func buildVorbisComment(fields map[string]string) []byte {
	vendor := "Media Metadata Surgery v0.1.2"
	var buf bytes.Buffer
	le := binary.LittleEndian

	// Vendor string
	vendorBytes := []byte(vendor)
	vLen := make([]byte, 4)
	le.PutUint32(vLen, uint32(len(vendorBytes)))
	buf.Write(vLen)
	buf.Write(vendorBytes)

	// Comment count
	cntBuf := make([]byte, 4)
	le.PutUint32(cntBuf, uint32(len(fields)))
	buf.Write(cntBuf)

	for k, v := range fields {
		comment := k + "=" + v
		cLen := make([]byte, 4)
		le.PutUint32(cLen, uint32(len(comment)))
		buf.Write(cLen)
		buf.WriteString(comment)
	}
	return buf.Bytes()
}

func writeFLAC(path string, blocks []flacBlock, audioData []byte) error {
	var buf bytes.Buffer
	buf.WriteString("fLaC")
	for i, b := range blocks {
		isLast := i == len(blocks)-1
		header := uint32(b.blockType)<<24 | uint32(len(b.data))
		if isLast {
			header |= 1 << 31
		}
		hBuf := make([]byte, 4)
		binary.BigEndian.PutUint32(hBuf, header)
		buf.Write(hBuf)
		buf.Write(b.data)
	}
	buf.Write(audioData)
	return os.WriteFile(path, buf.Bytes(), 0644)
}

// ──────────────────────────────────────────────────────────────────────────────
// Strip
// ──────────────────────────────────────────────────────────────────────────────

func (h *Handler) Strip(path string, outPath string, opts core.StripOptions) error {
	out := core.ResolveOutPath(path, outPath)
	switch h.format {
	case core.FmtMP3:
		return stripMP3(path, out, opts)
	case core.FmtFLAC:
		return stripFLAC(path, out, opts)
	case core.FmtWAV:
		return stripWAV(path, out, opts)
	default:
		info := formatInfo[h.format]
		if !info.CanStrip {
			return fmt.Errorf("%s does not support strip in v0.1.2", info.Name)
		}
		return fmt.Errorf("strip not yet implemented for %s", info.Name)
	}
}

func stripMP3(path, outPath string, opts core.StripOptions) error {
	if opts.DryRun {
		fmt.Println("Dry-run: MP3 ID3 tags would be removed")
		return nil
	}
	if path != outPath {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.WriteFile(outPath, data, 0644); err != nil {
			return err
		}
	}

	t, err := id3v2.Open(outPath, id3v2.Options{Parse: true})
	if err != nil {
		return err
	}
	defer t.Close()

	if len(opts.KeepFields) > 0 {
		keep := make(map[string]bool)
		for _, k := range opts.KeepFields {
			keep[strings.ToLower(k)] = true
		}
		// Delete all except kept
		all := []string{"TIT2", "TPE1", "TALB", "TDRC", "TCON", "COMM",
			"TRCK", "TPE2", "TCOM", "USLT", "TCOP", "TALB"}
		for _, fid := range all {
			name := mp3FrameNameFromID(fid)
			if !keep[strings.ToLower(name)] && !keep[strings.ToLower(fid)] {
				t.DeleteFrames(fid)
			}
		}
	} else {
		t.DeleteAllFrames()
	}

	return t.Save()
}

func mp3FrameNameFromID(fid string) string {
	m := map[string]string{
		"TIT2": "title", "TPE1": "artist", "TALB": "album",
		"TDRC": "year", "TCON": "genre", "COMM": "comment",
		"TRCK": "tracknumber", "TPE2": "albumartist", "TCOM": "composer",
		"USLT": "lyrics", "TCOP": "copyright",
	}
	return m[fid]
}

func stripFLAC(path, outPath string, opts core.StripOptions) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if opts.DryRun {
		fmt.Println("Dry-run: FLAC Vorbis comment block would be cleared")
		return nil
	}

	blocks, audioStart, err := parseFLACBlocks(data)
	if err != nil {
		return err
	}

	if len(opts.KeepFields) > 0 {
		// Clear only specified fields
		for i, b := range blocks {
			if b.blockType == 4 {
				existing := map[string]string{}
				parseVorbisComments(b.data, existing)
				keep := make(map[string]bool)
				for _, k := range opts.KeepFields {
					keep[strings.ToUpper(k)] = true
				}
				for k := range existing {
					if !keep[k] {
						delete(existing, k)
					}
				}
				blocks[i].data = buildVorbisComment(existing)
			}
		}
	} else {
		// Clear entire Vorbis comment block
		for i, b := range blocks {
			if b.blockType == 4 {
				blocks[i].data = buildVorbisComment(map[string]string{})
			}
		}
		// Also remove PICTURE blocks (type 6)
		var filtered []flacBlock
		for _, b := range blocks {
			if b.blockType == 6 {
				continue // drop embedded pictures
			}
			filtered = append(filtered, b)
		}
		blocks = filtered
	}

	return writeFLAC(outPath, blocks, data[audioStart:])
}

func stripWAV(path, outPath string, opts core.StripOptions) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(data) < 12 {
		return fmt.Errorf("WAV too short")
	}

	var body bytes.Buffer
	offset := 12
	for offset+8 <= len(data) {
		chunkID := string(data[offset : offset+4])
		chunkSize := int(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		offset += 8
		if offset+chunkSize > len(data) {
			break
		}

		skip := false
		if chunkID == "LIST" || chunkID == "id3 " || chunkID == "ID3 " {
			skip = true
		}
		if !skip {
			body.Write(data[offset-8 : offset+chunkSize])
		}
		offset += chunkSize
		if chunkSize%2 != 0 {
			offset++
		}
	}

	var out bytes.Buffer
	out.WriteString("RIFF")
	sizeBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(sizeBuf, uint32(body.Len()+4))
	out.Write(sizeBuf)
	out.WriteString("WAVE")
	out.Write(body.Bytes())

	return os.WriteFile(outPath, out.Bytes(), 0644)
}

// ─── io.ReadSeeker adapter ────────────────────────────────────────────────────
// dhowden/tag requires io.ReadSeeker — bytes.NewReader satisfies this.
var _ io.ReadSeeker = (*bytes.Reader)(nil)
