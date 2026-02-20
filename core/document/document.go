// Package document handles metadata for all document formats:
// PDF, DOCX, XLSX, PPTX, ODT, EPUB
package document

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ankit-chaubey/media-metadata-surgery/core"
)

// Handler implements core.Handler for document formats.
type Handler struct {
	format core.FormatID
}

// New returns a document Handler for the given format.
func New(fmt core.FormatID) *Handler { return &Handler{format: fmt} }

func (h *Handler) Info() core.FormatInfo {
	return formatInfo[h.format]
}

var formatInfo = map[core.FormatID]core.FormatInfo{
	core.FmtPDF: {
		Name:        "PDF",
		Extensions:  []string{".pdf"},
		MediaType:   "document",
		MIMETypes:   []string{"application/pdf"},
		CanView:     true,
		CanEdit:     true,
		CanStrip:    true,
		Notes:       "Info dict and XMP stream metadata.",
		EditableFields: []string{
			"Title", "Author", "Subject", "Keywords",
			"Creator", "Producer",
		},
	},
	core.FmtDOCX: {
		Name:        "DOCX",
		Extensions:  []string{".docx", ".docm"},
		MediaType:   "document",
		MIMETypes:   []string{"application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		CanView:     true,
		CanEdit:     true,
		CanStrip:    true,
		Notes:       "OPC ZIP container. Reads docProps/core.xml and docProps/app.xml.",
		EditableFields: []string{
			"Title", "Subject", "Author", "Keywords",
			"Description", "LastModifiedBy", "Category",
		},
	},
	core.FmtXLSX: {
		Name:        "XLSX",
		Extensions:  []string{".xlsx", ".xlsm"},
		MediaType:   "document",
		MIMETypes:   []string{"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
		CanView:     true,
		CanEdit:     true,
		CanStrip:    true,
		Notes:       "OPC ZIP container. Reads docProps/core.xml and docProps/app.xml.",
		EditableFields: []string{
			"Title", "Subject", "Author", "Keywords",
			"Description", "LastModifiedBy", "Category",
		},
	},
	core.FmtPPTX: {
		Name:        "PPTX",
		Extensions:  []string{".pptx", ".pptm"},
		MediaType:   "document",
		MIMETypes:   []string{"application/vnd.openxmlformats-officedocument.presentationml.presentation"},
		CanView:     true,
		CanEdit:     true,
		CanStrip:    true,
		Notes:       "OPC ZIP container. Reads docProps/core.xml and docProps/app.xml.",
		EditableFields: []string{
			"Title", "Subject", "Author", "Keywords",
			"Description", "LastModifiedBy", "Category",
		},
	},
	core.FmtODT: {
		Name:        "ODT/ODS/ODP",
		Extensions:  []string{".odt", ".ods", ".odp"},
		MediaType:   "document",
		MIMETypes:   []string{"application/vnd.oasis.opendocument.text"},
		CanView:     true,
		CanEdit:     false,
		CanStrip:    false,
		Notes:       "ODF ZIP container. View only in v0.1.2.",
	},
	core.FmtEPUB: {
		Name:        "EPUB",
		Extensions:  []string{".epub"},
		MediaType:   "document",
		MIMETypes:   []string{"application/epub+zip"},
		CanView:     true,
		CanEdit:     false,
		CanStrip:    false,
		Notes:       "OPS ZIP container. View only in v0.1.2.",
	},
}

// ──────────────────────────────────────────────────────────────────────────────
// View
// ──────────────────────────────────────────────────────────────────────────────

func (h *Handler) View(path string) (*core.Metadata, error) {
	m := &core.Metadata{FilePath: path}
	ext := strings.ToLower(filepath.Ext(path))

	switch h.format {
	case core.FmtPDF:
		m.Format = "PDF"
		return viewPDF(path, m)
	case core.FmtDOCX, core.FmtXLSX, core.FmtPPTX:
		m.Format = formatInfo[h.format].Name
		return viewOPC(path, m, true)
	case core.FmtODT:
		m.Format = "ODT"
		return viewODF(path, m)
	case core.FmtEPUB:
		m.Format = "EPUB"
		return viewEPUB(path, m)
	default:
		m.Format = strings.ToUpper(strings.TrimPrefix(ext, "."))
		return m, fmt.Errorf("unsupported document format: %s", ext)
	}
}

// ─── PDF ─────────────────────────────────────────────────────────────────────

// pdfInfoFields are the standard Info dict keys.
var pdfInfoFields = []string{
	"Title", "Author", "Subject", "Keywords",
	"Creator", "Producer", "CreationDate", "ModDate", "Trapped",
}

func viewPDF(path string, m *core.Metadata) (*core.Metadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return m, err
	}

	// 1. Info dict — scan for "/Info" dictionary
	infoFields := parsePDFInfoDict(data)
	for _, k := range pdfInfoFields {
		if v, ok := infoFields[k]; ok {
			editable := k != "CreationDate" && k != "ModDate" && k != "Trapped"
			m.Fields = append(m.Fields, core.MetaField{
				Key:      k,
				Value:    v,
				Category: "PDF Info",
				Editable: editable,
			})
		}
	}
	// Any extra fields
	for k, v := range infoFields {
		found := false
		for _, sf := range pdfInfoFields {
			if sf == k {
				found = true
				break
			}
		}
		if !found {
			m.Fields = append(m.Fields, core.MetaField{
				Key:      k,
				Value:    v,
				Category: "PDF Info",
				Editable: false,
			})
		}
	}

	// 2. XMP metadata stream
	xmpData := extractPDFXMP(data)
	if len(xmpData) > 0 {
		parseXMPIntoPDF(xmpData, m)
	}

	// 3. PDF version
	if len(data) >= 8 {
		m.Fields = append(m.Fields, core.MetaField{
			Key:      "PDFVersion",
			Value:    string(data[5:8]),
			Category: "PDF Header",
			Editable: false,
		})
	}

	return m, nil
}

// parsePDFInfoDict finds the /Info dictionary in the PDF cross-reference.
// This is a heuristic scanner — not a full PDF parser.
func parsePDFInfoDict(data []byte) map[string]string {
	result := map[string]string{}
	// Find all PDF string patterns like: /Title (value) or /Title <hexvalue>
	// Pattern: /Key (string) or /Key (string with parens)
	re := regexp.MustCompile(`/(\w+)\s*\(([^)]*)\)`)
	matches := re.FindAllSubmatch(data, -1)
	for _, m := range matches {
		key := string(m[1])
		val := decodePDFString(string(m[2]))
		for _, f := range pdfInfoFields {
			if f == key {
				result[key] = val
				break
			}
		}
	}

	// Also handle hex strings: /Key <HEXHEX>
	reHex := regexp.MustCompile(`/(\w+)\s*<([0-9A-Fa-f\s]+)>`)
	hexMatches := reHex.FindAllSubmatch(data, -1)
	for _, m := range hexMatches {
		key := string(m[1])
		hex := strings.ReplaceAll(string(m[2]), " ", "")
		if len(hex)%2 != 0 {
			continue
		}
		val := hexToString(hex)
		for _, f := range pdfInfoFields {
			if f == key {
				if _, exists := result[key]; !exists {
					result[key] = val
				}
				break
			}
		}
	}

	return result
}

func decodePDFString(s string) string {
	// Handle basic PDF escape sequences
	s = strings.ReplaceAll(s, `\n`, "\n")
	s = strings.ReplaceAll(s, `\r`, "\r")
	s = strings.ReplaceAll(s, `\t`, "\t")
	s = strings.ReplaceAll(s, `\\`, "\\")
	s = strings.ReplaceAll(s, `\(`, "(")
	s = strings.ReplaceAll(s, `\)`, ")")
	// Strip BOM for UTF-16
	if len(s) >= 2 && s[0] == '\xFE' && s[1] == '\xFF' {
		return utf16BEToString([]byte(s[2:]))
	}
	return s
}

func utf16BEToString(b []byte) string {
	var runes []rune
	for i := 0; i+1 < len(b); i += 2 {
		r := rune(uint16(b[i])<<8 | uint16(b[i+1]))
		if r == 0 {
			break
		}
		runes = append(runes, r)
	}
	return string(runes)
}

func hexToString(h string) string {
	b := make([]byte, len(h)/2)
	for i := range b {
		var byt byte
		fmt.Sscanf(h[i*2:i*2+2], "%02x", &byt)
		b[i] = byt
	}
	// Check for UTF-16 BOM
	if len(b) >= 2 && b[0] == 0xFE && b[1] == 0xFF {
		return utf16BEToString(b[2:])
	}
	return string(b)
}

func extractPDFXMP(data []byte) []byte {
	// Find XMP packet: <?xpacket begin=...>...</<?xpacket end=...>
	start := bytes.Index(data, []byte("<?xpacket begin="))
	if start < 0 {
		start = bytes.Index(data, []byte("<x:xmpmeta"))
	}
	if start < 0 {
		return nil
	}
	end := bytes.Index(data[start:], []byte("<?xpacket end="))
	if end < 0 {
		end = bytes.Index(data[start:], []byte("</x:xmpmeta>"))
		if end >= 0 {
			end += len("</x:xmpmeta>")
		}
	} else {
		end += len("<?xpacket end=")
		endClose := bytes.Index(data[start+end:], []byte(">"))
		if endClose >= 0 {
			end += endClose + 1
		}
	}
	if end < 0 {
		return nil
	}
	return data[start : start+end]
}

func parseXMPIntoPDF(data []byte, m *core.Metadata) {
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
		case xml.CharData:
			val := strings.TrimSpace(string(t))
			if val != "" && current != "" &&
				current != "xmpmeta" && current != "RDF" && current != "Description" {
				m.Fields = append(m.Fields, core.MetaField{
					Key:      "xmp:" + current,
					Value:    val,
					Category: "PDF XMP",
					Editable: false,
				})
			}
		}
	}
}

// ─── OPC (DOCX / XLSX / PPTX) ─────────────────────────────────────────────────

// OPC core properties XML
type opcCoreProps struct {
	XMLName        xml.Name `xml:"coreProperties"`
	Title          string   `xml:"title"`
	Subject        string   `xml:"subject"`
	Creator        string   `xml:"creator"`
	Keywords       string   `xml:"keywords"`
	Description    string   `xml:"description"`
	LastModifiedBy string   `xml:"lastModifiedBy"`
	Revision       string   `xml:"revision"`
	Created        string   `xml:"created"`
	Modified       string   `xml:"modified"`
	Category       string   `xml:"category"`
	ContentStatus  string   `xml:"contentStatus"`
}

type opcAppProps struct {
	XMLName       xml.Name `xml:"Properties"`
	Application   string   `xml:"Application"`
	Company       string   `xml:"Company"`
	DocSecurity   string   `xml:"DocSecurity"`
	AppVersion    string   `xml:"AppVersion"`
	Manager       string   `xml:"Manager"`
	Template      string   `xml:"Template"`
	TotalTime     string   `xml:"TotalTime"`
	Pages         string   `xml:"Pages"`
	Words         string   `xml:"Words"`
	Characters    string   `xml:"Characters"`
	Paragraphs    string   `xml:"Paragraphs"`
	Slides        string   `xml:"Slides"`
	Notes         string   `xml:"Notes"`
	HiddenSlides  string   `xml:"HiddenSlides"`
}

func viewOPC(path string, m *core.Metadata, editable bool) (*core.Metadata, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return m, fmt.Errorf("cannot open as ZIP: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		switch f.Name {
		case "docProps/core.xml":
			rc, err := f.Open()
			if err != nil {
				continue
			}
			data, _ := io.ReadAll(rc)
			rc.Close()
			parseCoreProps(data, m, editable)

		case "docProps/app.xml":
			rc, err := f.Open()
			if err != nil {
				continue
			}
			data, _ := io.ReadAll(rc)
			rc.Close()
			parseAppProps(data, m)
		}
	}
	return m, nil
}

func parseCoreProps(data []byte, m *core.Metadata, editable bool) {
	var props opcCoreProps
	// Remove namespace prefixes for simpler parsing
	cleaned := stripXMLNamespaces(data)
	if err := xml.Unmarshal(cleaned, &props); err != nil {
		// Fallback: regex extraction
		extractXMLFields(data, m, "Core Properties", editable)
		return
	}
	add := func(k, v string) {
		if v != "" {
			m.Fields = append(m.Fields, core.MetaField{Key: k, Value: v, Category: "Core Properties", Editable: editable})
		}
	}
	add("Title", props.Title)
	add("Subject", props.Subject)
	add("Author", props.Creator)
	add("Keywords", props.Keywords)
	add("Description", props.Description)
	add("LastModifiedBy", props.LastModifiedBy)
	add("Revision", props.Revision)
	add("Created", props.Created)
	add("Modified", props.Modified)
	add("Category", props.Category)
	add("ContentStatus", props.ContentStatus)
}

func parseAppProps(data []byte, m *core.Metadata) {
	var props opcAppProps
	cleaned := stripXMLNamespaces(data)
	if err := xml.Unmarshal(cleaned, &props); err != nil {
		extractXMLFields(data, m, "App Properties", false)
		return
	}
	add := func(k, v string) {
		if v != "" {
			m.Fields = append(m.Fields, core.MetaField{Key: k, Value: v, Category: "App Properties", Editable: false})
		}
	}
	add("Application", props.Application)
	add("Company", props.Company)
	add("AppVersion", props.AppVersion)
	add("Manager", props.Manager)
	add("Template", props.Template)
	add("TotalEditTime", props.TotalTime)
	add("Pages", props.Pages)
	add("Words", props.Words)
	add("Characters", props.Characters)
	add("Paragraphs", props.Paragraphs)
	add("Slides", props.Slides)
}

// stripXMLNamespaces removes namespace prefixes to simplify xml.Unmarshal.
func stripXMLNamespaces(data []byte) []byte {
	re := regexp.MustCompile(`\s+xmlns[^"]*"[^"]*"`)
	data = re.ReplaceAll(data, nil)
	re2 := regexp.MustCompile(`<(/?)[\w]+:`)
	data = re2.ReplaceAll(data, []byte("<$1"))
	return data
}

// extractXMLFields is a fallback XML field extractor using regex.
func extractXMLFields(data []byte, m *core.Metadata, category string, editable bool) {
	re := regexp.MustCompile(`<[^:>]+:?(\w+)[^>]*>([^<]+)</`)
	matches := re.FindAllSubmatch(data, -1)
	for _, match := range matches {
		key := string(match[1])
		val := strings.TrimSpace(string(match[2]))
		if val != "" {
			m.Fields = append(m.Fields, core.MetaField{
				Key:      key,
				Value:    val,
				Category: category,
				Editable: editable,
			})
		}
	}
}

// ─── ODF ─────────────────────────────────────────────────────────────────────

func viewODF(path string, m *core.Metadata) (*core.Metadata, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return m, fmt.Errorf("cannot open as ZIP: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == "meta.xml" {
			rc, err := f.Open()
			if err != nil {
				continue
			}
			data, _ := io.ReadAll(rc)
			rc.Close()

			// Parse ODF meta.xml
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
				case xml.CharData:
					val := strings.TrimSpace(string(t))
					if val != "" && current != "" && current != "meta" && current != "document-meta" {
						m.Fields = append(m.Fields, core.MetaField{
							Key:      current,
							Value:    val,
							Category: "ODF Metadata",
							Editable: false,
						})
					}
				}
			}
			break
		}
	}
	return m, nil
}

// ─── EPUB ─────────────────────────────────────────────────────────────────────

func viewEPUB(path string, m *core.Metadata) (*core.Metadata, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return m, fmt.Errorf("cannot open as ZIP: %w", err)
	}
	defer r.Close()

	// Find OPF file (container.xml points to it)
	opfPath := ""
	for _, f := range r.File {
		if f.Name == "META-INF/container.xml" {
			rc, _ := f.Open()
			data, _ := io.ReadAll(rc)
			rc.Close()
			re := regexp.MustCompile(`full-path="([^"]+\.opf)"`)
			if match := re.FindSubmatch(data); match != nil {
				opfPath = string(match[1])
			}
			break
		}
	}

	for _, f := range r.File {
		if f.Name == opfPath || strings.HasSuffix(f.Name, ".opf") {
			rc, _ := f.Open()
			data, _ := io.ReadAll(rc)
			rc.Close()
			parseEPUBOPF(data, m)
			break
		}
	}
	return m, nil
}

type epubOPF struct {
	XMLName  xml.Name `xml:"package"`
	Metadata struct {
		Title       []string `xml:"title"`
		Creator     []string `xml:"creator"`
		Subject     []string `xml:"subject"`
		Description []string `xml:"description"`
		Publisher   []string `xml:"publisher"`
		Contributor []string `xml:"contributor"`
		Date        []string `xml:"date"`
		Type        []string `xml:"type"`
		Format      []string `xml:"format"`
		Identifier  []string `xml:"identifier"`
		Source      []string `xml:"source"`
		Language    []string `xml:"language"`
		Rights      []string `xml:"rights"`
	} `xml:"metadata"`
}

func parseEPUBOPF(data []byte, m *core.Metadata) {
	cleaned := stripXMLNamespaces(data)
	var opf epubOPF
	if err := xml.Unmarshal(cleaned, &opf); err != nil {
		extractXMLFields(data, m, "EPUB Metadata", false)
		return
	}
	md := opf.Metadata
	addAll := func(k string, vals []string) {
		for _, v := range vals {
			if v = strings.TrimSpace(v); v != "" {
				m.Fields = append(m.Fields, core.MetaField{Key: k, Value: v, Category: "EPUB Metadata", Editable: false})
			}
		}
	}
	addAll("Title", md.Title)
	addAll("Author", md.Creator)
	addAll("Subject", md.Subject)
	addAll("Description", md.Description)
	addAll("Publisher", md.Publisher)
	addAll("Contributor", md.Contributor)
	addAll("Date", md.Date)
	addAll("Language", md.Language)
	addAll("Rights", md.Rights)
	addAll("Identifier", md.Identifier)
}

// ──────────────────────────────────────────────────────────────────────────────
// Edit
// ──────────────────────────────────────────────────────────────────────────────

func (h *Handler) Edit(path string, outPath string, opts core.EditOptions) error {
	out := core.ResolveOutPath(path, outPath)
	switch h.format {
	case core.FmtPDF:
		return editPDF(path, out, opts)
	case core.FmtDOCX, core.FmtXLSX, core.FmtPPTX:
		return editOPC(path, out, opts)
	default:
		info := formatInfo[h.format]
		if !info.CanEdit {
			return fmt.Errorf("%s does not support metadata editing in v0.1.2", info.Name)
		}
		return fmt.Errorf("edit not yet implemented for %s", info.Name)
	}
}

// ─── PDF Edit ─────────────────────────────────────────────────────────────────

func editPDF(path, outPath string, opts core.EditOptions) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	if opts.DryRun {
		fmt.Println("Dry-run: PDF Info dict would be updated:")
		for k, v := range opts.Set {
			fmt.Printf("  /%s (%s)\n", k, v)
		}
		return nil
	}

	// Replace existing Info fields using regex substitution
	for k, v := range opts.Set {
		re := regexp.MustCompile(`/` + regexp.QuoteMeta(k) + `\s*\([^)]*\)`)
		newEntry := fmt.Sprintf("/%s (%s)", k, v)
		if re.Match(data) {
			data = re.ReplaceAll(data, []byte(newEntry))
		} else {
			// Inject into Info dict
			infoIdx := bytes.Index(data, []byte("<< /"))
			if infoIdx >= 0 {
				inject := []byte("\n/" + k + " (" + v + ")")
				data = append(data[:infoIdx+3], append(inject, data[infoIdx+3:]...)...)
			}
		}
	}

	// Handle deletes
	for _, k := range opts.Delete {
		re := regexp.MustCompile(`/` + regexp.QuoteMeta(k) + `\s*\([^)]*\)\s*\n?`)
		data = re.ReplaceAll(data, nil)
	}

	return os.WriteFile(outPath, data, 0644)
}

// ─── OPC Edit (DOCX/XLSX/PPTX) ──────────────────────────────────────────────

func editOPC(path, outPath string, opts core.EditOptions) error {
	if opts.DryRun {
		fmt.Println("Dry-run: OPC core.xml properties would be updated:")
		for k, v := range opts.Set {
			fmt.Printf("  %s = %s\n", k, v)
		}
		return nil
	}

	r, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("cannot open as ZIP: %w", err)
	}
	defer r.Close()

	// Write to output, patching core.xml
	outFile, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	w := zip.NewWriter(outFile)
	defer w.Close()

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return err
		}

		if f.Name == "docProps/core.xml" {
			content = patchCoreXML(content, opts.Set, opts.Delete)
		}

		fw, err := w.Create(f.Name)
		if err != nil {
			return err
		}
		fw.Write(content)
	}

	return nil
}

func patchCoreXML(data []byte, set map[string]string, del []string) []byte {
	// Map friendly names to XML element names
	xmlNames := map[string]string{
		"Title":          "dc:title",
		"Subject":        "dc:subject",
		"Author":         "dc:creator",
		"Keywords":       "cp:keywords",
		"Description":    "dc:description",
		"LastModifiedBy": "cp:lastModifiedBy",
		"Category":       "cp:category",
		"ContentStatus":  "cp:contentStatus",
	}

	for k, v := range set {
		xmlTag := xmlNames[k]
		if xmlTag == "" {
			xmlTag = k
		}
		// Try to replace existing element
		re := regexp.MustCompile(`<` + regexp.QuoteMeta(xmlTag) + `[^>]*>[^<]*</` + regexp.QuoteMeta(xmlTag) + `>`)
		newEl := fmt.Sprintf("<%s>%s</%s>", xmlTag, xmlEscape(v), xmlTag)
		if re.Match(data) {
			data = re.ReplaceAll(data, []byte(newEl))
		} else {
			// Insert before closing tag
			closingRe := regexp.MustCompile(`</cp:coreProperties>`)
			data = closingRe.ReplaceAll(data, []byte("\n  "+newEl+"\n</cp:coreProperties>"))
		}
	}

	for _, k := range del {
		xmlTag := xmlNames[k]
		if xmlTag == "" {
			xmlTag = k
		}
		re := regexp.MustCompile(`\s*<` + regexp.QuoteMeta(xmlTag) + `[^>]*>[^<]*</` + regexp.QuoteMeta(xmlTag) + `>`)
		data = re.ReplaceAll(data, nil)
	}

	return data
}

func xmlEscape(s string) string {
	var buf bytes.Buffer
	xml.EscapeText(&buf, []byte(s))
	return buf.String()
}

// ──────────────────────────────────────────────────────────────────────────────
// Strip
// ──────────────────────────────────────────────────────────────────────────────

func (h *Handler) Strip(path string, outPath string, opts core.StripOptions) error {
	out := core.ResolveOutPath(path, outPath)
	switch h.format {
	case core.FmtPDF:
		return stripPDF(path, out, opts)
	case core.FmtDOCX, core.FmtXLSX, core.FmtPPTX:
		return stripOPC(path, out, opts)
	default:
		info := formatInfo[h.format]
		if !info.CanStrip {
			return fmt.Errorf("%s does not support strip in v0.1.2", info.Name)
		}
		return fmt.Errorf("strip not yet implemented for %s", info.Name)
	}
}

func stripPDF(path, outPath string, opts core.StripOptions) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	if opts.DryRun {
		fmt.Println("Dry-run: PDF Info dict and XMP would be cleared")
		return nil
	}

	keepSet := make(map[string]bool)
	for _, k := range opts.KeepFields {
		keepSet[k] = true
	}

	// Remove Info dict entries
	for _, k := range pdfInfoFields {
		if keepSet[k] {
			continue
		}
		re := regexp.MustCompile(`/` + regexp.QuoteMeta(k) + `\s*\([^)]*\)\s*`)
		data = re.ReplaceAll(data, nil)
		reHex := regexp.MustCompile(`/` + regexp.QuoteMeta(k) + `\s*<[^>]*>\s*`)
		data = reHex.ReplaceAll(data, nil)
	}

	// Remove XMP stream
	if !keepSet["xmp"] && !keepSet["XMP"] {
		xmpRe := regexp.MustCompile(`(?s)<\?xpacket begin.*?<\?xpacket end[^>]*>`)
		data = xmpRe.ReplaceAll(data, nil)
		xmpRe2 := regexp.MustCompile(`(?s)<x:xmpmeta.*?</x:xmpmeta>`)
		data = xmpRe2.ReplaceAll(data, nil)
	}

	return os.WriteFile(outPath, data, 0644)
}

func stripOPC(path, outPath string, opts core.StripOptions) error {
	if opts.DryRun {
		fmt.Println("Dry-run: OPC docProps would be cleared")
		return nil
	}

	r, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("cannot open as ZIP: %w", err)
	}
	defer r.Close()

	outFile, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	w := zip.NewWriter(outFile)
	defer w.Close()

	blankCoreXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties"
  xmlns:dc="http://purl.org/dc/elements/1.1/"
  xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
</cp:coreProperties>`

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return err
		}

		if f.Name == "docProps/core.xml" && len(opts.KeepFields) == 0 {
			content = []byte(blankCoreXML)
		}

		fw, err := w.Create(f.Name)
		if err != nil {
			return err
		}
		fw.Write(content)
	}

	return nil
}
