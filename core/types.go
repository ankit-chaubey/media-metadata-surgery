// Package core defines the shared types, interfaces, and format registry
// for Media Metadata Surgery.
package core

// MetaField represents a single metadata key-value pair.
type MetaField struct {
	Key      string // Canonical field name (e.g. "Make", "Artist", "Title")
	Value    string // String representation of the value
	Category string // Category label (e.g. "EXIF", "ID3", "Vorbis", "XMP")
	Editable bool   // Whether this field can be written back by surgery
	Raw      string // Raw / hex representation if different from Value
}

// Metadata holds all metadata extracted from a single file.
type Metadata struct {
	FilePath string
	Format   string // Human-readable format name (e.g. "JPEG", "MP3", "PDF")
	Fields   []MetaField
}

// Summary returns a short string of key fields for quick display.
func (m *Metadata) Summary() string {
	for _, f := range m.Fields {
		if f.Key == "Title" || f.Key == "Make" || f.Key == "Artist" {
			return f.Key + ": " + f.Value
		}
	}
	return m.Format
}

// StripOptions controls which parts of metadata to remove.
type StripOptions struct {
	// KeepFields lists field keys that should NOT be removed.
	// If empty, all metadata is stripped.
	KeepFields []string
	// StripGPS removes GPS coordinates only (for privacy).
	StripGPS bool
	// StripAll removes every possible metadata structure.
	StripAll bool
}

// EditOptions holds field changes for an edit operation.
type EditOptions struct {
	// Set is a map of Key → Value for fields to set or update.
	Set map[string]string
	// Delete is a list of field keys to remove.
	Delete []string
	// DryRun previews changes without writing.
	DryRun bool
}

// FormatInfo describes what a format handler supports.
type FormatInfo struct {
	Name           string   // "JPEG"
	Extensions     []string // [".jpg", ".jpeg"]
	MediaType      string   // "image" | "audio" | "video" | "document"
	MIMETypes      []string
	CanView        bool
	CanEdit        bool
	CanStrip       bool
	EditableFields []string // Names of fields the handler can write
	Notes          string   // Any caveats or notes
}

// Handler is the interface every format must implement.
type Handler interface {
	// View reads and returns all discoverable metadata from path.
	View(path string) (*Metadata, error)
	// Edit writes new/updated fields into path, saving to outPath.
	// outPath == "" means in-place edit.
	Edit(path string, outPath string, opts EditOptions) error
	// Strip removes metadata from path, saving to outPath.
	Strip(path string, outPath string, opts StripOptions) error
	// Info returns format capabilities.
	Info() FormatInfo
}
