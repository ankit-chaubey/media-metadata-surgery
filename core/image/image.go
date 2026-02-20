// Package image handles metadata for all image formats:
// JPEG/JPG, PNG, GIF, WebP, TIFF, BMP, HEIC/HEIF
package image

import (
	"bytes"
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/ankit-chaubey/media-metadata-surgery/core"
	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/tiff"
)

// ──────────────────────────────────────────────────────────────────────────────
// Handler
// ──────────────────────────────────────────────────────────────────────────────

// Handler implements core.Handler for all image formats.
type Handler struct {
	format core.FormatID
}

// New returns a Handler for the given format.
func New(fmt core.FormatID) *Handler { return &Handler{format: fmt} }

func (h *Handler) Info() core.FormatInfo {
	info := formatInfo[h.format]
	return info
}

var formatInfo = map[core.FormatID]core.FormatInfo{
	core.FmtJPEG: {
		Name:        "JPEG",
		Extensions:  []string{".jpg", ".jpeg"},
		MediaType:   "image",
		MIMETypes:   []string{"image/jpeg"},
		CanView:     true,
		CanEdit:     true,
		CanStrip:    true,
		Notes:       "EXIF, XMP, IPTC metadata. Edit supports common EXIF text fields.",
		EditableFields: []string{
			"Make", "Model", "Software", "Artist", "Copyright",
			"ImageDescription", "UserComment", "DateTime",
			"DateTimeOriginal", "DateTimeDigitized",
		},
	},
	core.FmtPNG: {
		Name:        "PNG",
		Extensions:  []string{".png"},
		MediaType:   "image",
		MIMETypes:   []string{"image/png"},
		CanView:     true,
		CanEdit:     true,
		CanStrip:    true,
		Notes:       "tEXt, iTXt, zTXt, eXIf chunks.",
		EditableFields: []string{
			"Title", "Author", "Description", "Copyright",
			"Comment", "Creation Time", "Source", "Software",
			"Disclaimer", "Warning",
		},
	},
	core.FmtGIF: {
		Name:       "GIF",
		Extensions: []string{".gif"},
		MediaType:  "image",
		MIMETypes:  []string{"image/gif"},
		CanView:    true,
		CanEdit:    false,
		CanStrip:   true,
		Notes:      "Comment extensions. Strip removes all comment blocks.",
	},
	core.FmtWebP: {
		Name:       "WebP",
		Extensions: []string{".webp"},
		MediaType:  "image",
		MIMETypes:  []string{"image/webp"},
		CanView:    true,
		CanEdit:    false,
		CanStrip:   true,
		Notes:      "EXIF and XMP chunks in RIFF container.",
	},
	core.FmtTIFF: {
		Name:       "TIFF",
		Extensions: []string{".tiff", ".tif"},
		MediaType:  "image",
		MIMETypes:  []string{"image/tiff"},
		CanView:    true,
		CanEdit:    false,
		CanStrip:   false,
		Notes:      "IFD-based metadata. View only in v0.1.2.",
	},
	core.FmtBMP: {
		Name:       "BMP",
		Extensions: []string{".bmp"},
		MediaType:  "image",
		MIMETypes:  []string{"image/bmp"},
		CanView:    true,
		CanEdit:    false,
		CanStrip:   false,
		Notes:      "Limited header metadata only.",
	},
	core.FmtHEIC: {
		Name:       "HEIC/HEIF",
		Extensions: []string{".heic", ".heif"},
		MediaType:  "image",
		MIMETypes:  []string{"image/heic", "image/heif"},
		CanView:    true,
		CanEdit:    false,
		CanStrip:   false,
		Notes:      "EXIF embedded in ISOBMFF container. View only in v0.1.2.",
	},
}

// ──────────────────────────────────────────────────────────────────────────────
// View
// ──────────────────────────────────────────────────────────────────────────────

func (h *Handler) View(path string) (*core.Metadata, error) {
	m := &core.Metadata{FilePath: path}
	ext := strings.ToLower(filepath.Ext(path))

	switch h.format {
	case core.FmtJPEG:
		m.Format = "JPEG"
		return viewJPEG(path, m)
	case core.FmtPNG:
		m.Format = "PNG"
		return viewPNG(path, m)
	case core.FmtGIF:
		m.Format = "GIF"
		return viewGIF(path, m)
	case core.FmtWebP:
		m.Format = "WebP"
		return viewWebP(path, m)
	case core.FmtTIFF:
		m.Format = "TIFF"
		return viewTIFF(path, m)
	case core.FmtBMP:
		m.Format = "BMP"
		return viewBMP(path, m)
	case core.FmtHEIC:
		m.Format = "HEIC/HEIF"
		return viewHEIC(path, m)
	case core.FmtSVG:
		m.Format = "SVG"
		return viewSVG(path, m)
	default:
		m.Format = strings.ToUpper(strings.TrimPrefix(ext, "."))
		return m, fmt.Errorf("unsupported image format: %s", ext)
	}
}

// ─── JPEG ────────────────────────────────────────────────────────────────────

func viewJPEG(path string, m *core.Metadata) (*core.Metadata, error) {
	f, err := os.Open(path)
	if err != nil {
		return m, err
	}
	defer f.Close()

	// EXIF via goexif
	x, err := exif.Decode(f)
	if err == nil {
		editableSet := map[string]bool{
			"Make": true, "Model": true, "Software": true, "Artist": true,
			"Copyright": true, "ImageDescription": true, "UserComment": true,
			"DateTime": true, "DateTimeOriginal": true, "DateTimeDigitized": true,
		}
		x.Walk(exifWalker{m: m, editableSet: editableSet})
	}

	// XMP — scan for APP1 with XMP namespace
	f.Seek(0, io.SeekStart)
	xmpData := extractJPEGSegment(f, 0xE1, []byte("http://ns.adobe.com/xap/1.0/\x00"))
	if len(xmpData) > 0 {
		parseXMPInto(xmpData, m)
	}

	// IPTC — scan APP13
	f.Seek(0, io.SeekStart)
	iptcData := extractJPEGSegment(f, 0xED, []byte("Photoshop 3.0\x00"))
	if len(iptcData) > 0 {
		parseIPTCInto(iptcData, m)
	}

	return m, nil
}

type exifWalker struct {
	m          *core.Metadata
	editableSet map[string]bool
}

func (w exifWalker) Walk(name exif.FieldName, tag *tiff.Tag) error {
	val := tag.String()
	// Remove surrounding quotes from string values
	if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
		val = val[1 : len(val)-1]
	}
	w.m.Fields = append(w.m.Fields, core.MetaField{
		Key:      string(name),
		Value:    val,
		Category: "EXIF",
		Editable: w.editableSet[string(name)],
	})
	return nil
}

// extractJPEGSegment finds a JPEG APP segment by marker byte and optional prefix.
// Returns the segment data (after the prefix), or nil.
func extractJPEGSegment(r io.ReadSeeker, marker byte, prefix []byte) []byte {
	buf := make([]byte, 2)
	// Read SOI
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil
	}
	if buf[0] != 0xFF || buf[1] != 0xD8 {
		return nil
	}
	for {
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil
		}
		if buf[0] != 0xFF {
			return nil
		}
		segMarker := buf[1]
		lenBuf := make([]byte, 2)
		if _, err := io.ReadFull(r, lenBuf); err != nil {
			return nil
		}
		segLen := int(binary.BigEndian.Uint16(lenBuf)) - 2
		if segLen < 0 {
			return nil
		}
		data := make([]byte, segLen)
		if _, err := io.ReadFull(r, data); err != nil {
			return nil
		}
		if segMarker == marker {
			if len(prefix) == 0 || bytes.HasPrefix(data, prefix) {
				if len(prefix) <= len(data) {
					return data[len(prefix):]
				}
			}
		}
		// Stop at SOS (start of scan)
		if segMarker == 0xDA {
			break
		}
	}
	return nil
}

// ─── XMP ─────────────────────────────────────────────────────────────────────

func parseXMPInto(data []byte, m *core.Metadata) {
	// Simple XMP key extraction using a generic approach
	dec := xml.NewDecoder(bytes.NewReader(data))
	var current string
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			current = t.Name.Local
			// Also capture attributes as fields
			for _, attr := range t.Attr {
				if attr.Name.Local == "xmlns" || strings.HasPrefix(attr.Name.Local, "xmlns") {
					continue
				}
				if attr.Value != "" {
					m.Fields = append(m.Fields, core.MetaField{
						Key:      "xmp:" + attr.Name.Local,
						Value:    attr.Value,
						Category: "XMP",
						Editable: false,
					})
				}
			}
		case xml.CharData:
			val := strings.TrimSpace(string(t))
			if val != "" && current != "" && current != "xmpmeta" && current != "RDF" {
				m.Fields = append(m.Fields, core.MetaField{
					Key:      "xmp:" + current,
					Value:    val,
					Category: "XMP",
					Editable: false,
				})
			}
		}
	}

}

// ─── IPTC ─────────────────────────────────────────────────────────────────────

var iptcFieldNames = map[byte]string{
	0x05: "ObjectName",
	0x0F: "Category",
	0x14: "SupplementalCategory",
	0x19: "Keywords",
	0x1E: "DateCreated",
	0x1F: "TimeCreated",
	0x28: "SpecialInstructions",
	0x37: "DigitalCreationDate",
	0x3C: "Byline",
	0x3E: "BylineTitle",
	0x46: "City",
	0x4E: "Province",
	0x55: "Country",
	0x67: "OriginalTransmissionReference",
	0x69: "Headline",
	0x6E: "Credit",
	0x73: "Source",
	0x74: "CopyrightNotice",
	0x76: "Contact",
	0x78: "Caption",
	0x7A: "CaptionWriter",
}

func parseIPTCInto(data []byte, m *core.Metadata) {
	// Skip "8BIM" Photoshop resource blocks to find IPTC resource (0x0404)
	i := 0
	for i+8 < len(data) {
		if !bytes.Equal(data[i:i+4], []byte("8BIM")) {
			i++
			continue
		}
		resType := binary.BigEndian.Uint16(data[i+4 : i+6])
		nameLen := int(data[i+6])
		if nameLen%2 == 0 {
			nameLen++
		}
		i += 7 + nameLen
		if i+4 > len(data) {
			break
		}
		blockLen := int(binary.BigEndian.Uint32(data[i : i+4]))
		i += 4
		if resType == 0x0404 && i+blockLen <= len(data) {
			parseIPTCBlock(data[i:i+blockLen], m)
		}
		i += blockLen
		if blockLen%2 != 0 {
			i++
		}
	}
}

func parseIPTCBlock(data []byte, m *core.Metadata) {
	i := 0
	for i+5 < len(data) {
		if data[i] != 0x1C {
			i++
			continue
		}
		// record := data[i+1]
		dataset := data[i+2]
		length := int(binary.BigEndian.Uint16(data[i+3 : i+5]))
		i += 5
		if i+length > len(data) {
			break
		}
		val := string(data[i : i+length])
		if name, ok := iptcFieldNames[dataset]; ok {
			m.Fields = append(m.Fields, core.MetaField{
				Key:      name,
				Value:    val,
				Category: "IPTC",
				Editable: false,
			})
		}
		i += length
	}
}

// ─── PNG ─────────────────────────────────────────────────────────────────────

func viewPNG(path string, m *core.Metadata) (*core.Metadata, error) {
	f, err := os.Open(path)
	if err != nil {
		return m, err
	}
	defer f.Close()

	chunks, err := readPNGChunks(f)
	if err != nil {
		return m, err
	}

	for _, c := range chunks {
		switch c.typ {
		case "tEXt":
			// Format: keyword\0value
			null := bytes.IndexByte(c.data, 0)
			if null > 0 {
				key := string(c.data[:null])
				val := ""
				if null+1 < len(c.data) {
					val = string(c.data[null+1:])
				}
				m.Fields = append(m.Fields, core.MetaField{
					Key:      key,
					Value:    val,
					Category: "PNG tEXt",
					Editable: true,
				})
			}
		case "iTXt":
			// Format: keyword\0compression_flag\0compression_method\0language\0translated_keyword\0text
			null := bytes.IndexByte(c.data, 0)
			if null > 0 {
				key := string(c.data[:null])
				// Skip flags, method, language, translated keyword (3 more nulls)
				rest := c.data[null+3:]
				for i := 0; i < 2 && len(rest) > 0; i++ {
					n := bytes.IndexByte(rest, 0)
					if n < 0 {
						rest = nil
						break
					}
					rest = rest[n+1:]
				}
				val := ""
				if rest != nil {
					val = string(rest)
				}
				m.Fields = append(m.Fields, core.MetaField{
					Key:      key,
					Value:    val,
					Category: "PNG iTXt",
					Editable: true,
				})
			}
		case "eXIf":
			// EXIF data embedded in PNG — parse with goexif
			x, err := exif.Decode(bytes.NewReader(c.data))
			if err == nil {
				x.Walk(exifWalker{m: m})
			}
		case "tIME":
			if len(c.data) == 7 {
				year := binary.BigEndian.Uint16(c.data[0:2])
				m.Fields = append(m.Fields, core.MetaField{
					Key:      "LastModified",
					Value:    fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d", year, c.data[2], c.data[3], c.data[4], c.data[5], c.data[6]),
					Category: "PNG tIME",
					Editable: false,
				})
			}
		}
	}
	return m, nil
}

type pngChunk struct {
	typ  string
	data []byte
}

func readPNGChunks(r io.Reader) ([]pngChunk, error) {
	sig := make([]byte, 8)
	if _, err := io.ReadFull(r, sig); err != nil {
		return nil, err
	}
	expected := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	if !bytes.Equal(sig, expected) {
		return nil, fmt.Errorf("not a valid PNG")
	}

	var chunks []pngChunk
	for {
		lenBuf := make([]byte, 4)
		if _, err := io.ReadFull(r, lenBuf); err != nil {
			break
		}
		length := binary.BigEndian.Uint32(lenBuf)
		typBuf := make([]byte, 4)
		if _, err := io.ReadFull(r, typBuf); err != nil {
			break
		}
		data := make([]byte, length)
		if _, err := io.ReadFull(r, data); err != nil {
			break
		}
		crcBuf := make([]byte, 4)
		io.ReadFull(r, crcBuf)

		typ := string(typBuf)
		chunks = append(chunks, pngChunk{typ: typ, data: data})
		if typ == "IEND" {
			break
		}
	}
	return chunks, nil
}

// ─── GIF ─────────────────────────────────────────────────────────────────────

func viewGIF(path string, m *core.Metadata) (*core.Metadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return m, err
	}

	if len(data) < 6 {
		return m, fmt.Errorf("file too short")
	}
	version := string(data[3:6])
	m.Fields = append(m.Fields, core.MetaField{
		Key:      "Version",
		Value:    "GIF" + version,
		Category: "GIF Header",
		Editable: false,
	})
	if len(data) >= 10 {
		w := binary.LittleEndian.Uint16(data[6:8])
		h := binary.LittleEndian.Uint16(data[8:10])
		m.Fields = append(m.Fields, core.MetaField{
			Key:      "Dimensions",
			Value:    fmt.Sprintf("%d x %d", w, h),
			Category: "GIF Header",
			Editable: false,
		})
	}

	// Scan for comment extensions (0x21 0xFE)
	i := 13 // skip header (6) + logical screen descriptor (7)
	if len(data) > 10 && data[10]&0x80 != 0 {
		// Global color table present
		colorTableSize := 1 << (int(data[10]&0x07) + 1)
		i += colorTableSize * 3
	}

	commentCount := 0
	for i < len(data)-1 {
		if data[i] == 0x3B { // trailer
			break
		}
		if data[i] == 0x21 && i+1 < len(data) && data[i+1] == 0xFE {
			// Comment extension
			i += 2
			var comment []byte
			for i < len(data) {
				blockSize := int(data[i])
				i++
				if blockSize == 0 {
					break
				}
				if i+blockSize > len(data) {
					break
				}
				comment = append(comment, data[i:i+blockSize]...)
				i += blockSize
			}
			if len(comment) > 0 {
				commentCount++
				m.Fields = append(m.Fields, core.MetaField{
					Key:      fmt.Sprintf("Comment_%d", commentCount),
					Value:    string(comment),
					Category: "GIF Comment",
					Editable: false,
				})
			}
			continue
		}
		i++
	}
	return m, nil
}

// ─── WebP ─────────────────────────────────────────────────────────────────────

func viewWebP(path string, m *core.Metadata) (*core.Metadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return m, err
	}
	if len(data) < 12 {
		return m, fmt.Errorf("file too short")
	}

	// Parse RIFF chunks
	offset := 12 // skip RIFF header
	for offset+8 <= len(data) {
		chunkID := string(data[offset : offset+4])
		chunkSize := int(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		offset += 8
		if offset+chunkSize > len(data) {
			break
		}
		chunkData := data[offset : offset+chunkSize]

		switch chunkID {
		case "EXIF":
			x, err := exif.Decode(bytes.NewReader(chunkData))
			if err == nil {
				x.Walk(exifWalker{m: m})
			}
		case "XMP ":
			if utf8.Valid(chunkData) {
				parseXMPInto(chunkData, m)
			}
		case "VP8 ", "VP8L", "VP8X":
			m.Fields = append(m.Fields, core.MetaField{
				Key:      "Encoding",
				Value:    strings.TrimSpace(chunkID),
				Category: "WebP",
				Editable: false,
			})
		}

		offset += chunkSize
		if chunkSize%2 != 0 {
			offset++ // padding
		}
	}
	return m, nil
}

// ─── TIFF ────────────────────────────────────────────────────────────────────

func viewTIFF(path string, m *core.Metadata) (*core.Metadata, error) {
	f, err := os.Open(path)
	if err != nil {
		return m, err
	}
	defer f.Close()

	x, err := exif.Decode(f)
	if err != nil {
		return m, fmt.Errorf("could not parse TIFF IFDs: %w", err)
	}
	x.Walk(exifWalker{m: m})
	return m, nil
}

// ─── BMP ─────────────────────────────────────────────────────────────────────

func viewBMP(path string, m *core.Metadata) (*core.Metadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return m, err
	}
	if len(data) < 54 {
		return m, fmt.Errorf("file too short for BMP header")
	}
	fileSize := binary.LittleEndian.Uint32(data[2:6])
	width := int32(binary.LittleEndian.Uint32(data[18:22]))
	height := int32(binary.LittleEndian.Uint32(data[22:26]))
	bpp := binary.LittleEndian.Uint16(data[28:30])

	m.Fields = append(m.Fields,
		core.MetaField{Key: "FileSize", Value: fmt.Sprintf("%d bytes", fileSize), Category: "BMP Header", Editable: false},
		core.MetaField{Key: "Width", Value: fmt.Sprintf("%d px", width), Category: "BMP Header", Editable: false},
		core.MetaField{Key: "Height", Value: fmt.Sprintf("%d px", height), Category: "BMP Header", Editable: false},
		core.MetaField{Key: "BitsPerPixel", Value: fmt.Sprintf("%d", bpp), Category: "BMP Header", Editable: false},
	)
	return m, nil
}

// ─── HEIC ─────────────────────────────────────────────────────────────────────

func viewHEIC(path string, m *core.Metadata) (*core.Metadata, error) {
	f, err := os.Open(path)
	if err != nil {
		return m, err
	}
	defer f.Close()

	// HEIC is ISOBMFF — walk boxes looking for 'Exif' or 'uuid' boxes
	parseISOBMFF(f, m)
	return m, nil
}

func parseISOBMFF(r io.ReadSeeker, m *core.Metadata) {
	for {
		hdr := make([]byte, 8)
		if _, err := io.ReadFull(r, hdr); err != nil {
			break
		}
		size := int64(binary.BigEndian.Uint32(hdr[0:4]))
		boxType := string(hdr[4:8])
		dataSize := size - 8
		if size == 1 {
			// 64-bit size
			ext := make([]byte, 8)
			io.ReadFull(r, ext)
			size = int64(binary.BigEndian.Uint64(ext))
			dataSize = size - 16
		}
		if size == 0 {
			break
		}

		switch boxType {
		case "ftyp":
			brand := make([]byte, 4)
			io.ReadFull(r, brand)
			m.Fields = append(m.Fields, core.MetaField{
				Key:      "Brand",
				Value:    strings.TrimSpace(string(brand)),
				Category: "HEIC",
				Editable: false,
			})
			r.Seek(dataSize-4, io.SeekCurrent)
		case "Exif":
			// Skip 4-byte offset prefix
			skip := make([]byte, 4)
			io.ReadFull(r, skip)
			exifData := make([]byte, dataSize-4)
			io.ReadFull(r, exifData)
			x, err := exif.Decode(bytes.NewReader(exifData))
			if err == nil {
				x.Walk(exifWalker{m: m})
			}
		default:
			r.Seek(dataSize, io.SeekCurrent)
		}
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Edit
// ──────────────────────────────────────────────────────────────────────────────

func (h *Handler) Edit(path string, outPath string, opts core.EditOptions) error {
	out := core.ResolveOutPath(path, outPath)
	switch h.format {
	case core.FmtJPEG:
		return editJPEG(path, out, opts)
	case core.FmtPNG:
		return editPNG(path, out, opts)
	default:
		info := formatInfo[h.format]
		if !info.CanEdit {
			return fmt.Errorf("%s does not support metadata editing in v0.1.2", info.Name)
		}
		return fmt.Errorf("edit not yet implemented for %s", info.Name)
	}
}

// ─── JPEG Edit ───────────────────────────────────────────────────────────────
// Approach: Read all JPEG segments, rebuild APP1/EXIF segment with updated IFD
// entries using dsoprea/go-exif.  For simplicity in v0.1.2 we implement a
// targeted string-field replacement by scanning the raw EXIF IFD bytes.

func editJPEG(path, outPath string, opts core.EditOptions) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	segments, err := parseJPEGSegments(data)
	if err != nil {
		return err
	}

	// Find APP1 EXIF segment
	exifSegIdx := -1
	for i, seg := range segments {
		if seg.marker == 0xE1 && bytes.HasPrefix(seg.data, []byte("Exif\x00\x00")) {
			exifSegIdx = i
			break
		}
	}

	if exifSegIdx < 0 && len(opts.Set) > 0 {
		// No EXIF yet — create a minimal one
		newExifData, err := buildMinimalEXIF(opts.Set)
		if err != nil {
			return err
		}
		// Insert APP1 after SOI
		newSeg := jpegSegment{marker: 0xE1, data: newExifData}
		segments = append([]jpegSegment{segments[0], newSeg}, segments[1:]...)
	} else if exifSegIdx >= 0 {
		updated, err := patchEXIFSegment(segments[exifSegIdx].data, opts.Set, opts.Delete)
		if err != nil {
			return err
		}
		segments[exifSegIdx].data = updated
	}

	if opts.DryRun {
		fmt.Println("Dry-run: JPEG EXIF would be updated with:")
		for k, v := range opts.Set {
			fmt.Printf("  %s = %s\n", k, v)
		}
		return nil
	}

	return writeJPEGSegments(outPath, segments)
}

type jpegSegment struct {
	marker byte
	data   []byte
}

func parseJPEGSegments(data []byte) ([]jpegSegment, error) {
	if len(data) < 2 || data[0] != 0xFF || data[1] != 0xD8 {
		return nil, fmt.Errorf("not a JPEG")
	}
	var segs []jpegSegment
	segs = append(segs, jpegSegment{marker: 0xD8}) // SOI

	i := 2
	for i < len(data) {
		if data[i] != 0xFF {
			// Raw scan data (after SOS)
			segs = append(segs, jpegSegment{marker: 0x00, data: data[i:]})
			break
		}
		i++
		if i >= len(data) {
			break
		}
		marker := data[i]
		i++

		if marker == 0xD8 || marker == 0xD9 {
			segs = append(segs, jpegSegment{marker: marker})
			if marker == 0xD9 {
				break
			}
			continue
		}

		if i+2 > len(data) {
			break
		}
		segLen := int(binary.BigEndian.Uint16(data[i:i+2])) - 2
		i += 2
		if segLen < 0 || i+segLen > len(data) {
			break
		}
		segs = append(segs, jpegSegment{marker: marker, data: append([]byte{}, data[i:i+segLen]...)})
		i += segLen
	}
	return segs, nil
}

func writeJPEGSegments(path string, segs []jpegSegment) error {
	var buf bytes.Buffer
	for _, seg := range segs {
		switch seg.marker {
		case 0xD8:
			buf.Write([]byte{0xFF, 0xD8})
		case 0xD9:
			buf.Write([]byte{0xFF, 0xD9})
		case 0x00:
			buf.Write(seg.data)
		default:
			buf.WriteByte(0xFF)
			buf.WriteByte(seg.marker)
			length := uint16(len(seg.data) + 2)
			buf.WriteByte(byte(length >> 8))
			buf.WriteByte(byte(length))
			buf.Write(seg.data)
		}
	}
	return os.WriteFile(path, buf.Bytes(), 0644)
}

// EXIF tag IDs for common editable string fields.
var exifTagIDs = map[string]uint16{
	"ImageDescription":    0x010E,
	"Make":                0x010F,
	"Model":               0x0110,
	"Software":            0x0131,
	"DateTime":            0x0132,
	"Artist":              0x013B,
	"Copyright":           0x8298,
	"UserComment":         0x9286,
	"DateTimeOriginal":    0x9003,
	"DateTimeDigitized":   0x9004,
}

// buildMinimalEXIF creates a bare-bones EXIF APP1 segment data with the given fields.
func buildMinimalEXIF(fields map[string]string) ([]byte, error) {
	var buf bytes.Buffer
	// EXIF header
	buf.WriteString("Exif\x00\x00")
	// TIFF header (little-endian)
	buf.WriteString("II")                   // byte order
	buf.Write([]byte{0x2A, 0x00})           // magic
	buf.Write([]byte{0x08, 0x00, 0x00, 0x00}) // offset to IFD0

	// Collect valid fields
	type ifdEntry struct {
		tag   uint16
		value string
	}
	var entries []ifdEntry
	for k, v := range fields {
		if tid, ok := exifTagIDs[k]; ok {
			entries = append(entries, ifdEntry{tag: tid, value: v})
		}
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("no recognised EXIF fields to write; supported: %v", supportedEditFields())
	}

	// IFD structure (little-endian)
	// Each entry: 2 tag + 2 type + 4 count + 4 value/offset = 12 bytes
	numEntries := uint16(len(entries))
	ifdBase := 8 // offset in TIFF block where IFD starts
	ifdSize := 2 + int(numEntries)*12 + 4
	valOffset := ifdBase + ifdSize

	var ifdBuf bytes.Buffer
	le16 := func(v uint16) { binary.Write(&ifdBuf, binary.LittleEndian, v) }
	le32 := func(v uint32) { binary.Write(&ifdBuf, binary.LittleEndian, v) }

	le16(numEntries)
	var valueBuf bytes.Buffer
	for _, e := range entries {
		val := e.value + "\x00"
		le16(e.tag)
		le16(2) // ASCII type
		le32(uint32(len(val)))
		if len(val) <= 4 {
			// Value fits inline
			padded := make([]byte, 4)
			copy(padded, val)
			ifdBuf.Write(padded)
		} else {
			offset := uint32(valOffset + valueBuf.Len())
			le32(offset)
			valueBuf.WriteString(val)
		}
	}
	le32(0) // next IFD offset = 0

	buf.Write(ifdBuf.Bytes())
	buf.Write(valueBuf.Bytes())
	return buf.Bytes(), nil
}

// patchEXIFSegment updates string IFD entries in an existing EXIF APP1 block.
// It uses a conservative approach: locate string values by tag and overwrite
// them in-place if the new value fits, or add new IFD entries otherwise.
func patchEXIFSegment(data []byte, set map[string]string, del []string) ([]byte, error) {
	if len(data) < 8 {
		return data, nil
	}
	// For v0.1.2, if the segment is present we rebuild with merged fields.
	// Read existing fields, apply changes, rebuild.
	existing := map[string]string{}
	x, err := exif.Decode(bytes.NewReader(data[6:])) // skip "Exif\x00\x00"
	if err == nil {
		x.Walk(exifStringWalker{fields: existing})
	}
	// Apply set
	for k, v := range set {
		existing[k] = v
	}
	// Apply delete
	for _, k := range del {
		delete(existing, k)
	}
	return buildMinimalEXIF(existing)
}

type exifStringWalker struct {
	fields map[string]string
}

func (w exifStringWalker) Walk(name exif.FieldName, tag *tiff.Tag) error {
	if tag.Type == tiff.DTAscii || tag.Type == tiff.DTUndefined {
		val := tag.String()
		if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
			val = val[1 : len(val)-1]
		}
		w.fields[string(name)] = val
	}
	return nil
}

func supportedEditFields() []string {
	fields := make([]string, 0, len(exifTagIDs))
	for k := range exifTagIDs {
		fields = append(fields, k)
	}
	return fields
}

// ─── PNG Edit ────────────────────────────────────────────────────────────────

func editPNG(path, outPath string, opts core.EditOptions) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	chunks, err := readPNGChunks(f)
	if err != nil {
		return err
	}

	delSet := make(map[string]bool)
	for _, k := range opts.Delete {
		delSet[k] = true
	}

	// Update or remove existing tEXt chunks, then add new ones
	setDone := make(map[string]bool)
	var newChunks []pngChunk
	for _, c := range chunks {
		if c.typ == "tEXt" {
			null := bytes.IndexByte(c.data, 0)
			if null > 0 {
				key := string(c.data[:null])
				if delSet[key] {
					continue // delete
				}
				if v, ok := opts.Set[key]; ok {
					// Update
					c.data = append([]byte(key+"\x00"), []byte(v)...)
					setDone[key] = true
				}
			}
		}
		newChunks = append(newChunks, c)
	}

	// Add new fields not yet present
	var addChunks []pngChunk
	for k, v := range opts.Set {
		if !setDone[k] {
			d := append([]byte(k+"\x00"), []byte(v)...)
			addChunks = append(addChunks, pngChunk{typ: "tEXt", data: d})
		}
	}

	// Insert new chunks before IDAT
	var final []pngChunk
	inserted := false
	for _, c := range newChunks {
		if !inserted && c.typ == "IDAT" {
			final = append(final, addChunks...)
			inserted = true
		}
		final = append(final, c)
	}
	if !inserted {
		final = append(final, addChunks...)
	}

	if opts.DryRun {
		fmt.Printf("Dry-run: PNG tEXt chunks would be updated:\n")
		for k, v := range opts.Set {
			fmt.Printf("  %s = %s\n", k, v)
		}
		return nil
	}

	return writePNGChunks(outPath, final)
}

func writePNGChunks(path string, chunks []pngChunk) error {
	var buf bytes.Buffer
	// PNG signature
	buf.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})
	for _, c := range chunks {
		writePNGChunk(&buf, c.typ, c.data)
	}
	return os.WriteFile(path, buf.Bytes(), 0644)
}

func writePNGChunk(w *bytes.Buffer, typ string, data []byte) {
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(data)))
	w.Write(lenBuf)
	w.WriteString(typ)
	w.Write(data)
	// CRC over type + data
	crc := crc32PNG([]byte(typ), data)
	crcBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(crcBuf, crc)
	w.Write(crcBuf)
}

// crc32PNG computes CRC32 for PNG chunk (type + data).
func crc32PNG(typ, data []byte) uint32 {
	// PNG uses CRC32 with polynomial 0xEDB88320 (reflected)
	const poly = 0xEDB88320
	var table [256]uint32
	for i := range table {
		c := uint32(i)
		for j := 0; j < 8; j++ {
			if c&1 != 0 {
				c = poly ^ (c >> 1)
			} else {
				c >>= 1
			}
		}
		table[i] = c
	}
	crc := uint32(0xFFFFFFFF)
	for _, b := range typ {
		crc = table[(crc^uint32(b))&0xFF] ^ (crc >> 8)
	}
	for _, b := range data {
		crc = table[(crc^uint32(b))&0xFF] ^ (crc >> 8)
	}
	return crc ^ 0xFFFFFFFF
}

// ──────────────────────────────────────────────────────────────────────────────
// Strip
// ──────────────────────────────────────────────────────────────────────────────

func (h *Handler) Strip(path string, outPath string, opts core.StripOptions) error {
	out := core.ResolveOutPath(path, outPath)
	switch h.format {
	case core.FmtJPEG:
		return stripJPEG(path, out, opts)
	case core.FmtPNG:
		return stripPNG(path, out, opts)
	case core.FmtGIF:
		return stripGIF(path, out, opts)
	case core.FmtWebP:
		return stripWebP(path, out, opts)
	default:
		info := formatInfo[h.format]
		if !info.CanStrip {
			return fmt.Errorf("%s does not support strip in v0.1.2", info.Name)
		}
		return fmt.Errorf("strip not yet implemented for %s", info.Name)
	}
}

// ─── JPEG Strip ──────────────────────────────────────────────────────────────

// Metadata segments to remove on full strip.
var jpegMetaMarkers = map[byte]bool{
	0xE1: true, // APP1  — EXIF / XMP
	0xE2: true, // APP2  — ICC profile / FlashPix
	0xEC: true, // APP12 — Picture Info
	0xED: true, // APP13 — IPTC / Photoshop
	0xEE: true, // APP14 — Adobe
	0xFE: true, // COM   — comment
}

// GPS EXIF tag IDs
var gpsTagIDs = map[uint16]bool{
	0x0000: true, 0x0001: true, 0x0002: true, 0x0003: true,
	0x0004: true, 0x0005: true, 0x0006: true, 0x0007: true,
	0x0008: true, 0x0009: true, 0x001D: true, 0x001F: true,
}

func stripJPEG(path, outPath string, opts core.StripOptions) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	segments, err := parseJPEGSegments(data)
	if err != nil {
		return err
	}

	keepSet := make(map[string]bool)
	for _, k := range opts.KeepFields {
		keepSet[strings.ToLower(k)] = true
	}

	var out []jpegSegment
	for _, seg := range segments {
		if opts.StripGPS && seg.marker == 0xE1 {
			// Strip only GPS sub-IFD from EXIF, keep rest
			stripped, err := stripGPSFromEXIF(seg.data)
			if err == nil {
				seg.data = stripped
			}
			out = append(out, seg)
			continue
		}
		if jpegMetaMarkers[seg.marker] {
			if opts.StripAll {
				continue // drop segment
			}
			// Check keep list
			if len(keepSet) > 0 {
				if keepSet["exif"] && seg.marker == 0xE1 && bytes.HasPrefix(seg.data, []byte("Exif")) {
					out = append(out, seg)
					continue
				}
				if keepSet["xmp"] && seg.marker == 0xE1 && bytes.Contains(seg.data, []byte("ns.adobe.com")) {
					out = append(out, seg)
					continue
				}
				if keepSet["iptc"] && seg.marker == 0xED {
					out = append(out, seg)
					continue
				}
			}
			continue // drop by default
		}
		out = append(out, seg)
	}

	return writeJPEGSegments(outPath, out)
}

func stripGPSFromEXIF(data []byte) ([]byte, error) {
	// Simply rebuild EXIF without GPS tags
	existing := map[string]string{}
	if len(data) < 6 {
		return data, nil
	}
	x, err := exif.Decode(bytes.NewReader(data[6:]))
	if err != nil {
		return data, err
	}
	x.Walk(exifStringWalker{fields: existing})
	return buildMinimalEXIF(existing)
}

// ─── PNG Strip ───────────────────────────────────────────────────────────────

var pngMetaChunks = map[string]bool{
	"tEXt": true,
	"iTXt": true,
	"zTXt": true,
	"eXIf": true,
	"tIME": true,
	"iCCP": true,
	"sRGB": true,
	"gAMA": true,
	"cHRM": true,
	"bKGD": true,
	"hIST": true,
	"pHYs": true,
	"sBIT": true,
	"sPLT": true,
}

func stripPNG(path, outPath string, opts core.StripOptions) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	chunks, err := readPNGChunks(f)
	if err != nil {
		return err
	}

	keepSet := make(map[string]bool)
	for _, k := range opts.KeepFields {
		keepSet[strings.ToLower(k)] = true
	}

	var final []pngChunk
	for _, c := range chunks {
		if pngMetaChunks[c.typ] {
			if keepSet[strings.ToLower(c.typ)] || keepSet["all"] {
				final = append(final, c)
				continue
			}
			continue // drop
		}
		final = append(final, c)
	}

	return writePNGChunks(outPath, final)
}

// ─── GIF Strip ───────────────────────────────────────────────────────────────

func stripGIF(path, outPath string, opts core.StripOptions) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Walk through GIF blocks, skip comment extensions
	var out bytes.Buffer
	i := 0

	// Header (6 bytes) + Logical Screen Descriptor (7 bytes)
	if len(data) < 13 {
		return fmt.Errorf("GIF too short")
	}
	out.Write(data[:13])
	i = 13

	// Global color table
	if data[10]&0x80 != 0 {
		ctSize := 3 * (1 << (int(data[10]&0x07) + 1))
		if i+ctSize > len(data) {
			return fmt.Errorf("GIF truncated")
		}
		out.Write(data[i : i+ctSize])
		i += ctSize
	}

	for i < len(data) {
		if data[i] == 0x3B { // trailer
			out.WriteByte(0x3B)
			break
		}
		if data[i] == 0x21 && i+1 < len(data) && data[i+1] == 0xFE {
			// Comment extension — skip it
			i += 2
			for i < len(data) {
				blockSize := int(data[i])
				i++
				if blockSize == 0 {
					break
				}
				i += blockSize
			}
			continue
		}
		// Copy everything else
		out.WriteByte(data[i])
		i++
	}

	return os.WriteFile(outPath, out.Bytes(), 0644)
}

// ─── WebP Strip ───────────────────────────────────────────────────────────────

func stripWebP(path, outPath string, opts core.StripOptions) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(data) < 12 {
		return fmt.Errorf("WebP too short")
	}

	// Rebuild RIFF without EXIF and XMP chunks
	var body bytes.Buffer
	offset := 12
	for offset+8 <= len(data) {
		chunkID := string(data[offset : offset+4])
		chunkSize := int(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		offset += 8
		if offset+chunkSize > len(data) {
			break
		}
		chunkData := data[offset : offset+chunkSize]

		skip := false
		if opts.StripAll || len(opts.KeepFields) == 0 {
			if chunkID == "EXIF" || chunkID == "XMP " {
				skip = true
			}
		}
		if !skip {
			body.WriteString(chunkID)
			sizeBuf := make([]byte, 4)
			binary.LittleEndian.PutUint32(sizeBuf, uint32(chunkSize))
			body.Write(sizeBuf)
			body.Write(chunkData)
			if chunkSize%2 != 0 {
				body.WriteByte(0)
			}
		}

		offset += chunkSize
		if chunkSize%2 != 0 {
			offset++
		}
	}

	var out bytes.Buffer
	out.WriteString("RIFF")
	totalSize := make([]byte, 4)
	binary.LittleEndian.PutUint32(totalSize, uint32(body.Len()+4))
	out.Write(totalSize)
	out.WriteString("WEBP")
	out.Write(body.Bytes())

	return os.WriteFile(outPath, out.Bytes(), 0644)
}


// ─── SVG ─────────────────────────────────────────────────────────────────────

func init() {
	formatInfo[core.FmtSVG] = core.FormatInfo{
		Name:       "SVG",
		Extensions: []string{".svg"},
		MediaType:  "image",
		MIMETypes:  []string{"image/svg+xml"},
		CanView:    true,
		CanEdit:    false,
		CanStrip:   false,
		Notes:      "XML-based vector format. Reads title, desc, and metadata elements.",
	}
}

// ─── SVG View ──────────────────────────────────────────────────────────────

func viewSVG(path string, m *core.Metadata) (*core.Metadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return m, err
	}
	type svgMeta struct {
		Title   string `xml:"title"`
		Desc    string `xml:"desc"`
		Width   string `xml:"width,attr"`
		Height  string `xml:"height,attr"`
		ViewBox string `xml:"viewBox,attr"`
	}
	var svg svgMeta
	xml.Unmarshal(data, &svg)

	add := func(k, v string) {
		if v != "" {
			m.Fields = append(m.Fields, core.MetaField{Key: k, Value: v, Category: "SVG", Editable: false})
		}
	}
	add("Title", strings.TrimSpace(svg.Title))
	add("Description", strings.TrimSpace(svg.Desc))
	add("Width", svg.Width)
	add("Height", svg.Height)
	add("ViewBox", svg.ViewBox)

	// Also extract <metadata> children
	metaRe := regexp.MustCompile(`(?s)<metadata[^>]*>(.*?)</metadata>`)
	if match := metaRe.FindSubmatch(data); match != nil {
		parseXMPInto(match[1], m)
	}
	return m, nil
}
