package core

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Printer handles all display output for the CLI.
type Printer struct {
	JSON    bool
	Verbose bool
	Writer  *os.File
}

// NewPrinter creates a default Printer writing to stdout.
func NewPrinter(jsonMode, verbose bool) *Printer {
	return &Printer{JSON: jsonMode, Verbose: verbose, Writer: os.Stdout}
}

// PrintMetadata renders a Metadata struct to the configured output.
func (p *Printer) PrintMetadata(m *Metadata) {
	if p.JSON {
		p.printJSON(m)
		return
	}
	p.printText(m)
}

func (p *Printer) printText(m *Metadata) {
	fmt.Fprintf(p.Writer, "File  : %s\n", m.FilePath)
	fmt.Fprintf(p.Writer, "Format: %s\n", m.Format)
	if len(m.Fields) == 0 {
		fmt.Fprintln(p.Writer, "(no metadata found)")
		return
	}
	fmt.Fprintln(p.Writer)

	// Group by category
	groups := make(map[string][]MetaField)
	order := []string{}
	seen := map[string]bool{}
	for _, f := range m.Fields {
		if !seen[f.Category] {
			seen[f.Category] = true
			order = append(order, f.Category)
		}
		groups[f.Category] = append(groups[f.Category], f)
	}

	for _, cat := range order {
		fmt.Fprintf(p.Writer, "── %s ──\n", cat)
		for _, f := range groups[cat] {
			edit := ""
			if f.Editable {
				edit = " [editable]"
			}
			fmt.Fprintf(p.Writer, "  %-30s %s%s\n", f.Key+":", f.Value, edit)
		}
		fmt.Fprintln(p.Writer)
	}
}

func (p *Printer) printJSON(m *Metadata) {
	type jsonField struct {
		Key      string `json:"key"`
		Value    string `json:"value"`
		Category string `json:"category"`
		Editable bool   `json:"editable"`
	}
	type jsonOutput struct {
		FilePath string      `json:"file"`
		Format   string      `json:"format"`
		Fields   []jsonField `json:"fields"`
	}

	out := jsonOutput{
		FilePath: m.FilePath,
		Format:   m.Format,
	}
	for _, f := range m.Fields {
		out.Fields = append(out.Fields, jsonField{
			Key:      f.Key,
			Value:    f.Value,
			Category: f.Category,
			Editable: f.Editable,
		})
	}

	b, _ := json.MarshalIndent(out, "", "  ")
	fmt.Fprintln(p.Writer, string(b))
}

// PrintSuccess prints a success message.
func (p *Printer) PrintSuccess(msg string) {
	fmt.Fprintln(p.Writer, "✓ "+msg)
}

// PrintInfo prints an info line (suppressed in JSON mode).
func (p *Printer) PrintInfo(msg string) {
	if !p.JSON {
		fmt.Fprintln(p.Writer, msg)
	}
}

// PrintError prints an error to stderr.
func PrintError(msg string) {
	fmt.Fprintln(os.Stderr, "✗ Error: "+msg)
}

// ParseKV parses a "Key=Value" string.
func ParseKV(s string) (key, value string, ok bool) {
	idx := strings.Index(s, "=")
	if idx < 1 {
		return "", "", false
	}
	return strings.TrimSpace(s[:idx]), strings.TrimSpace(s[idx+1:]), true
}

// ResolveOutPath returns dst if non-empty, otherwise src (in-place).
func ResolveOutPath(src, dst string) string {
	if dst == "" {
		return src
	}
	return dst
}
