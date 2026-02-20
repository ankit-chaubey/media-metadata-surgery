package core

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"strings"
)

// FormatID enumerates every recognised format.
type FormatID string

const (
	FmtJPEG FormatID = "jpeg"
	FmtPNG  FormatID = "png"
	FmtGIF  FormatID = "gif"
	FmtWebP FormatID = "webp"
	FmtTIFF FormatID = "tiff"
	FmtBMP  FormatID = "bmp"
	FmtHEIC FormatID = "heic"
	FmtSVG  FormatID = "svg"

	FmtMP3  FormatID = "mp3"
	FmtFLAC FormatID = "flac"
	FmtOGG  FormatID = "ogg"
	FmtM4A  FormatID = "m4a"
	FmtWAV  FormatID = "wav"
	FmtAIFF FormatID = "aiff"
	FmtOpus FormatID = "opus"

	FmtMP4  FormatID = "mp4"
	FmtMOV  FormatID = "mov"
	FmtMKV  FormatID = "mkv"
	FmtWebM FormatID = "webm"
	FmtAVI  FormatID = "avi"
	FmtWMV  FormatID = "wmv"
	FmtFLV  FormatID = "flv"

	FmtPDF  FormatID = "pdf"
	FmtDOCX FormatID = "docx"
	FmtXLSX FormatID = "xlsx"
	FmtPPTX FormatID = "pptx"
	FmtODT  FormatID = "odt"
	FmtEPUB FormatID = "epub"

	FmtUnknown FormatID = "unknown"
)

// extMap maps lowercase extensions to format IDs.
var extMap = map[string]FormatID{
	".jpg":  FmtJPEG,
	".jpeg": FmtJPEG,
	".png":  FmtPNG,
	".gif":  FmtGIF,
	".webp": FmtWebP,
	".tiff": FmtTIFF,
	".tif":  FmtTIFF,
	".bmp":  FmtBMP,
	".heic": FmtHEIC,
	".heif": FmtHEIC,
	".svg":  FmtSVG,

	".mp3":  FmtMP3,
	".flac": FmtFLAC,
	".ogg":  FmtOGG,
	".oga":  FmtOGG,
	".m4a":  FmtM4A,
	".aac":  FmtM4A,
	".wav":  FmtWAV,
	".wave": FmtWAV,
	".aif":  FmtAIFF,
	".aiff": FmtAIFF,
	".opus": FmtOpus,

	".mp4":  FmtMP4,
	".m4v":  FmtMP4,
	".mov":  FmtMOV,
	".qt":   FmtMOV,
	".mkv":  FmtMKV,
	".webm": FmtWebM,
	".avi":  FmtAVI,
	".wmv":  FmtWMV,
	".flv":  FmtFLV,

	".pdf":  FmtPDF,
	".docx": FmtDOCX,
	".docm": FmtDOCX,
	".xlsx": FmtXLSX,
	".xlsm": FmtXLSX,
	".pptx": FmtPPTX,
	".pptm": FmtPPTX,
	".odt":  FmtODT,
	".ods":  FmtODT,
	".odp":  FmtODT,
	".epub": FmtEPUB,
}

// DetectFormat returns the FormatID for the given file, first by reading
// magic bytes and falling back to extension.
func DetectFormat(path string) (FormatID, error) {
	f, err := os.Open(path)
	if err != nil {
		return FmtUnknown, err
	}
	defer f.Close()

	buf := make([]byte, 16)
	n, err := io.ReadFull(f, buf)
	if err != nil && n == 0 {
		return FmtUnknown, err
	}
	buf = buf[:n]

	if id := detectMagic(buf); id != FmtUnknown {
		return id, nil
	}

	// Fallback to extension
	dot := strings.LastIndex(path, ".")
	if dot >= 0 {
		ext := strings.ToLower(path[dot:])
		if id, ok := extMap[ext]; ok {
			return id, nil
		}
	}
	return FmtUnknown, nil
}

func detectMagic(b []byte) FormatID {
	if len(b) < 4 {
		return FmtUnknown
	}
	switch {
	// JPEG: FF D8 FF
	case b[0] == 0xFF && b[1] == 0xD8 && b[2] == 0xFF:
		return FmtJPEG
	// PNG: 89 50 4E 47 0D 0A 1A 0A
	case bytes.HasPrefix(b, []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}):
		return FmtPNG
	// GIF: GIF87a or GIF89a
	case bytes.HasPrefix(b, []byte("GIF87a")) || bytes.HasPrefix(b, []byte("GIF89a")):
		return FmtGIF
	// WebP: RIFF????WEBP
	case len(b) >= 12 && bytes.Equal(b[0:4], []byte("RIFF")) && bytes.Equal(b[8:12], []byte("WEBP")):
		return FmtWebP
	// TIFF: 49 49 2A 00 (little-endian) or 4D 4D 00 2A (big-endian)
	case bytes.HasPrefix(b, []byte{0x49, 0x49, 0x2A, 0x00}) ||
		bytes.HasPrefix(b, []byte{0x4D, 0x4D, 0x00, 0x2A}):
		return FmtTIFF
	// BMP: 42 4D
	case b[0] == 0x42 && b[1] == 0x4D:
		return FmtBMP
	// MP3: ID3 tag or FF FB / FF F3 / FF F2 sync
	case bytes.HasPrefix(b, []byte("ID3")):
		return FmtMP3
	case b[0] == 0xFF && (b[1]&0xE0 == 0xE0):
		return FmtMP3
	// FLAC: fLaC
	case bytes.HasPrefix(b, []byte("fLaC")):
		return FmtFLAC
	// OGG: OggS
	case bytes.HasPrefix(b, []byte("OggS")):
		return FmtOGG
	// WAV: RIFF????WAVE
	case len(b) >= 12 && bytes.Equal(b[0:4], []byte("RIFF")) && bytes.Equal(b[8:12], []byte("WAVE")):
		return FmtWAV
	// AIFF: FORM????AIFF or AIFC
	case len(b) >= 12 && bytes.Equal(b[0:4], []byte("FORM")) &&
		(bytes.Equal(b[8:12], []byte("AIFF")) || bytes.Equal(b[8:12], []byte("AIFC"))):
		return FmtAIFF
	// MP4/MOV: ftyp box at offset 4 — check ftypM4A , ftypmp42, ftypisom, ftypM4V
	case len(b) >= 8 && bytes.Equal(b[4:8], []byte("ftyp")):
		return detectMP4Subtype(b)
	// MKV/WebM: EBML header 0x1A45DFA3
	case len(b) >= 4 && binary.BigEndian.Uint32(b[0:4]) == 0x1A45DFA3:
		return detectMKVSubtype(b)
	// AVI: RIFF????AVI
	case len(b) >= 12 && bytes.Equal(b[0:4], []byte("RIFF")) && bytes.Equal(b[8:12], []byte("AVI ")):
		return FmtAVI
	// FLV: FLV
	case bytes.HasPrefix(b, []byte("FLV")):
		return FmtFLV
	// PDF: %PDF
	case bytes.HasPrefix(b, []byte("%PDF")):
		return FmtPDF
	// ZIP-based (DOCX/XLSX/PPTX/ODT/EPUB): PK\x03\x04
	case bytes.HasPrefix(b, []byte("PK\x03\x04")):
		return FmtDOCX // resolved more precisely by extension later
	}
	return FmtUnknown
}

func detectMP4Subtype(b []byte) FormatID {
	if len(b) < 12 {
		return FmtMP4
	}
	brand := string(b[8:12])
	switch brand {
	case "M4A ", "M4B ":
		return FmtM4A
	case "qt  ":
		return FmtMOV
	default:
		return FmtMP4
	}
}

func detectMKVSubtype(b []byte) FormatID {
	// We'd need to dig into the EBML DocType to tell MKV vs WebM.
	// For now default to MKV; video/webm extension will override.
	return FmtMKV
}

// MediaTypeFor returns the broad media category for a format.
func MediaTypeFor(id FormatID) string {
	switch id {
	case FmtJPEG, FmtPNG, FmtGIF, FmtWebP, FmtTIFF, FmtBMP, FmtHEIC, FmtSVG:
		return "image"
	case FmtMP3, FmtFLAC, FmtOGG, FmtM4A, FmtWAV, FmtAIFF, FmtOpus:
		return "audio"
	case FmtMP4, FmtMOV, FmtMKV, FmtWebM, FmtAVI, FmtWMV, FmtFLV:
		return "video"
	case FmtPDF, FmtDOCX, FmtXLSX, FmtPPTX, FmtODT, FmtEPUB:
		return "document"
	default:
		return "unknown"
	}
}
