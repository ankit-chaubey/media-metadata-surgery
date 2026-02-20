// Media Metadata Surgery — CLI entry point
// Version: 0.1.2
//
// Usage:
//   surgery <command> [flags] <file|directory>
//
// Commands:
//   view     View all metadata for a file
//   edit     Add or update metadata fields
//   strip    Remove metadata from a file
//   info     Show format detection and capabilities for a file
//   formats  List all supported formats and their capabilities
//   batch    Run view/strip/edit on all files in a directory
//   version  Print version information
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	audpkg "github.com/ankit-chaubey/media-metadata-surgery/core/audio"
	docpkg "github.com/ankit-chaubey/media-metadata-surgery/core/document"
	imgpkg "github.com/ankit-chaubey/media-metadata-surgery/core/image"
	vidpkg "github.com/ankit-chaubey/media-metadata-surgery/core/video"

	"github.com/ankit-chaubey/media-metadata-surgery/core"
)

const Version = "0.1.2"

// ──────────────────────────────────────────────────────────────────────────────
// kvFlags — multi-value flag for --set KEY=VALUE and --delete KEY
// ──────────────────────────────────────────────────────────────────────────────

type kvFlags []string

func (k *kvFlags) String() string  { return strings.Join(*k, ", ") }
func (k *kvFlags) Set(v string) error { *k = append(*k, v); return nil }

// ──────────────────────────────────────────────────────────────────────────────
// main
// ──────────────────────────────────────────────────────────────────────────────

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "view":
		runView(args)
	case "edit":
		runEdit(args)
	case "strip":
		runStrip(args)
	case "info":
		runInfo(args)
	case "formats":
		runFormats(args)
	case "batch":
		runBatch(args)
	case "version", "--version", "-v":
		fmt.Printf("Media Metadata Surgery v%s\n", Version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf(`Media Metadata Surgery v%s

USAGE
  surgery <command> [flags] <file>

COMMANDS
  view      View all metadata embedded in a file
  edit      Add or update metadata fields in a file
  strip     Remove metadata from a file
  info      Show format detection and capabilities for a file
  formats   List all supported formats and their capabilities
  batch     Run view/strip/edit on all files in a directory
  version   Print version information

QUICK EXAMPLES
  surgery view photo.jpg
  surgery view --json audio.mp3
  surgery edit --set "Artist=John Doe" --set "Title=My Song" audio.mp3
  surgery edit --set "Title=Report 2024" document.docx
  surgery strip photo.jpg
  surgery strip --out clean.jpg --keep xmp photo.jpg
  surgery info video.mp4
  surgery formats --type image
  surgery batch view ./photos
  surgery batch strip --out ./clean ./photos

Run 'surgery <command> --help' for command-specific help.
`, Version)
}

// ──────────────────────────────────────────────────────────────────────────────
// view
// ──────────────────────────────────────────────────────────────────────────────

func runView(args []string) {
	fs := flag.NewFlagSet("view", flag.ExitOnError)
	jsonOut := fs.Bool("json", false, "Output metadata as JSON")
	verbose := fs.Bool("verbose", false, "Include raw/low-level fields")
	fs.Usage = func() {
		fmt.Println("Usage: surgery view [--json] [--verbose] <file>")
		fmt.Println()
		fmt.Println("View all metadata embedded in a file.")
		fmt.Println()
		fmt.Println("Flags:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  surgery view photo.jpg")
		fmt.Println("  surgery view --json audio.mp3")
		fmt.Println("  surgery view --verbose document.pdf")
	}
	fs.Parse(args)

	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(1)
	}

	path := fs.Arg(0)
	p := core.NewPrinter(*jsonOut, *verbose)

	m, err := viewFile(path)
	if err != nil {
		core.PrintError(err.Error())
		os.Exit(1)
	}
	p.PrintMetadata(m)
}

// ──────────────────────────────────────────────────────────────────────────────
// edit
// ──────────────────────────────────────────────────────────────────────────────

func runEdit(args []string) {
	fs := flag.NewFlagSet("edit", flag.ExitOnError)
	var setFlags kvFlags
	var delFlags kvFlags
	outPath := fs.String("out", "", "Output file path (default: edit in-place)")
	dryRun := fs.Bool("dry-run", false, "Preview changes without writing to disk")
	fs.Var(&setFlags, "set", "Set a metadata field:  KEY=VALUE  (repeatable)")
	fs.Var(&delFlags, "delete", "Delete a metadata field by key (repeatable)")
	fs.Usage = func() {
		fmt.Println("Usage: surgery edit [flags] <file>")
		fmt.Println()
		fmt.Println("Add, update, or delete metadata fields in a file.")
		fmt.Println()
		fmt.Println("Flags:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println(`  surgery edit --set "Artist=John Doe" --set "Title=My Song" audio.mp3`)
		fmt.Println(`  surgery edit --set "Title=Report" --delete Author document.pdf`)
		fmt.Println(`  surgery edit --set "Make=Canon" --out out.jpg photo.jpg`)
		fmt.Println(`  surgery edit --dry-run --set "Title=Test" video.mp4`)
		fmt.Println()
		fmt.Println("Editable fields by format:")
		fmt.Println("  JPEG/TIFF : Make, Model, Software, Artist, Copyright, ImageDescription,")
		fmt.Println("              UserComment, DateTime, DateTimeOriginal, DateTimeDigitized")
		fmt.Println("  PNG       : Title, Author, Description, Copyright, Comment,")
		fmt.Println("              Creation Time, Source, Software")
		fmt.Println("  MP3       : Title, Artist, Album, Year, Genre, Comment,")
		fmt.Println("              TrackNumber, AlbumArtist, Composer, Lyrics, Copyright")
		fmt.Println("  FLAC      : TITLE, ARTIST, ALBUM, DATE, GENRE, COMMENT,")
		fmt.Println("              TRACKNUMBER, ALBUMARTIST, COMPOSER, COPYRIGHT")
		fmt.Println("  MP4/MOV   : title, artist, album, comment, year, genre,")
		fmt.Println("              description, copyright")
		fmt.Println("  PDF       : Title, Author, Subject, Keywords, Creator, Producer")
		fmt.Println("  DOCX/XLSX/PPTX: Title, Subject, Author, Keywords, Description,")
		fmt.Println("              LastModifiedBy, Category")
	}
	fs.Parse(args)

	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(1)
	}
	if len(setFlags) == 0 && len(delFlags) == 0 {
		fmt.Fprintln(os.Stderr, "Error: provide at least one --set or --delete flag")
		fmt.Fprintln(os.Stderr, "Run 'surgery edit --help' for usage.")
		os.Exit(1)
	}

	path := fs.Arg(0)

	setMap := map[string]string{}
	for _, kv := range setFlags {
		k, v, ok := core.ParseKV(kv)
		if !ok {
			core.PrintError(fmt.Sprintf("invalid --set value %q — expected KEY=VALUE", kv))
			os.Exit(1)
		}
		setMap[k] = v
	}

	opts := core.EditOptions{
		Set:    setMap,
		Delete: []string(delFlags),
		DryRun: *dryRun,
	}

	h, err := getHandler(path)
	if err != nil {
		core.PrintError(err.Error())
		os.Exit(1)
	}

	info := h.Info()
	if !info.CanEdit {
		core.PrintError(fmt.Sprintf(
			"%s does not support metadata editing in v%s\n"+
				"Formats that support editing: JPEG, PNG, MP3, FLAC, MP4, PDF, DOCX, XLSX, PPTX",
			info.Name, Version))
		os.Exit(1)
	}

	if err := h.Edit(path, *outPath, opts); err != nil {
		core.PrintError(err.Error())
		os.Exit(1)
	}

	if !*dryRun {
		out := core.ResolveOutPath(path, *outPath)
		if out == path {
			fmt.Printf("✓ Metadata updated in-place: %s\n", path)
		} else {
			fmt.Printf("✓ Metadata updated → %s\n", out)
		}
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// strip
// ──────────────────────────────────────────────────────────────────────────────

func runStrip(args []string) {
	fs := flag.NewFlagSet("strip", flag.ExitOnError)
	outPath := fs.String("out", "", "Output file path (default: strip in-place)")
	dryRun := fs.Bool("dry-run", false, "Preview without writing to disk")
	gpsOnly := fs.Bool("gps-only", false, "Remove only GPS location fields (keep rest)")
	var keepFlags kvFlags
	fs.Var(&keepFlags, "keep", "Keep a metadata section (repeatable): exif, xmp, iptc, id3")
	fs.Usage = func() {
		fmt.Println("Usage: surgery strip [flags] <file>")
		fmt.Println()
		fmt.Println("Remove metadata from a file. Default: remove all metadata.")
		fmt.Println()
		fmt.Println("Flags:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  surgery strip photo.jpg")
		fmt.Println("  surgery strip --out clean.jpg photo.jpg")
		fmt.Println("  surgery strip --keep exif photo.jpg        # remove XMP+IPTC, keep EXIF")
		fmt.Println("  surgery strip --gps-only photo.jpg         # remove GPS only")
		fmt.Println("  surgery strip --dry-run audio.mp3")
		fmt.Println()
		fmt.Println("Formats that support strip: JPEG, PNG, GIF, WebP, MP3, FLAC, WAV, MP4, MOV, PDF, DOCX, XLSX, PPTX")
	}
	fs.Parse(args)

	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(1)
	}

	path := fs.Arg(0)

	opts := core.StripOptions{
		KeepFields: []string(keepFlags),
		StripGPS:   *gpsOnly,
		StripAll:   len(keepFlags) == 0 && !*gpsOnly,
	}

	h, err := getHandler(path)
	if err != nil {
		core.PrintError(err.Error())
		os.Exit(1)
	}

	info := h.Info()
	if !info.CanStrip {
		core.PrintError(fmt.Sprintf(
			"%s does not support metadata stripping in v%s", info.Name, Version))
		os.Exit(1)
	}

	if err := h.Strip(path, *outPath, opts); err != nil {
		core.PrintError(err.Error())
		os.Exit(1)
	}

	if !*dryRun {
		out := core.ResolveOutPath(path, *outPath)
		if out == path {
			fmt.Printf("✓ Metadata stripped from: %s\n", path)
		} else {
			fmt.Printf("✓ Stripped → %s\n", out)
		}
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// info
// ──────────────────────────────────────────────────────────────────────────────

func runInfo(args []string) {
	fs := flag.NewFlagSet("info", flag.ExitOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	fs.Usage = func() {
		fmt.Println("Usage: surgery info [--json] <file>")
		fmt.Println()
		fmt.Println("Show format detection result and capabilities for a file.")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  surgery info photo.jpg")
		fmt.Println("  surgery info --json audio.mp3")
	}
	fs.Parse(args)

	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(1)
	}

	path := fs.Arg(0)
	fmtID, err := core.DetectFormat(path)
	if err != nil {
		core.PrintError(err.Error())
		os.Exit(1)
	}

	h, err := getHandler(path)
	if err != nil {
		core.PrintError(err.Error())
		os.Exit(1)
	}

	info := h.Info()

	if *jsonOut {
		fmt.Printf("{\n")
		fmt.Printf("  \"file\": %q,\n", path)
		fmt.Printf("  \"format_id\": %q,\n", fmtID)
		fmt.Printf("  \"name\": %q,\n", info.Name)
		fmt.Printf("  \"media_type\": %q,\n", info.MediaType)
		fmt.Printf("  \"extensions\": %q,\n", strings.Join(info.Extensions, ", "))
		fmt.Printf("  \"mime_types\": %q,\n", strings.Join(info.MIMETypes, ", "))
		fmt.Printf("  \"can_view\": %v,\n", info.CanView)
		fmt.Printf("  \"can_edit\": %v,\n", info.CanEdit)
		fmt.Printf("  \"can_strip\": %v,\n", info.CanStrip)
		fmt.Printf("  \"editable_fields\": %q,\n", strings.Join(info.EditableFields, ", "))
		fmt.Printf("  \"notes\": %q\n", info.Notes)
		fmt.Printf("}\n")
	} else {
		fmt.Printf("File            : %s\n", path)
		fmt.Printf("Detected Format : %s  (id: %s)\n", info.Name, fmtID)
		fmt.Printf("Media Type      : %s\n", info.MediaType)
		fmt.Printf("Extensions      : %s\n", strings.Join(info.Extensions, ", "))
		fmt.Printf("MIME Types      : %s\n", strings.Join(info.MIMETypes, ", "))
		fmt.Printf("Can View        : %v\n", info.CanView)
		fmt.Printf("Can Edit        : %v\n", info.CanEdit)
		fmt.Printf("Can Strip       : %v\n", info.CanStrip)
		if len(info.EditableFields) > 0 {
			fmt.Printf("Editable Fields : %s\n", strings.Join(info.EditableFields, ", "))
		}
		if info.Notes != "" {
			fmt.Printf("Notes           : %s\n", info.Notes)
		}
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// formats
// ──────────────────────────────────────────────────────────────────────────────

type namedFormatInfo struct {
	id core.FormatID
	core.FormatInfo
}

func runFormats(args []string) {
	fs := flag.NewFlagSet("formats", flag.ExitOnError)
	mediaType := fs.String("type", "", "Filter by media type: image|audio|video|document")
	fs.Usage = func() {
		fmt.Println("Usage: surgery formats [--type image|audio|video|document]")
		fmt.Println()
		fmt.Println("List all supported formats and their capabilities.")
	}
	fs.Parse(args)

	all := getAllFormatInfos()

	fmt.Printf("\n%-12s %-22s %-10s  %-5s %-5s %-5s  %s\n",
		"Format ID", "Name", "Type", "View", "Edit", "Strip", "Extensions")
	fmt.Println(strings.Repeat("─", 82))

	viewCount, editCount, stripCount := 0, 0, 0
	total := 0

	for _, f := range all {
		if *mediaType != "" && f.MediaType != *mediaType {
			continue
		}
		total++
		v := tick(f.CanView)
		e := tick(f.CanEdit)
		s := tick(f.CanStrip)
		if f.CanView { viewCount++ }
		if f.CanEdit { editCount++ }
		if f.CanStrip { stripCount++ }
		fmt.Printf("%-12s %-22s %-10s  %-5s %-5s %-5s  %s\n",
			string(f.id), f.Name, f.MediaType, v, e, s,
			strings.Join(f.Extensions, " "))
	}

	fmt.Println(strings.Repeat("─", 82))
	fmt.Printf("%-12s %-22s %-10s  %-5d %-5d %-5d  (%d formats total)\n",
		"", "TOTAL", "", viewCount, editCount, stripCount, total)
	fmt.Println()
}

func tick(b bool) string {
	if b {
		return "✓"
	}
	return "—"
}

func getAllFormatInfos() []namedFormatInfo {
	var all []namedFormatInfo
	// Images
	for _, id := range []core.FormatID{
		core.FmtJPEG, core.FmtPNG, core.FmtGIF, core.FmtWebP,
		core.FmtTIFF, core.FmtBMP, core.FmtHEIC, core.FmtSVG,
	} {
		h := imgpkg.New(id)
		all = append(all, namedFormatInfo{id: id, FormatInfo: h.Info()})
	}
	// Audio
	for _, id := range []core.FormatID{
		core.FmtMP3, core.FmtFLAC, core.FmtOGG, core.FmtOpus,
		core.FmtM4A, core.FmtWAV, core.FmtAIFF,
	} {
		h := audpkg.New(id)
		all = append(all, namedFormatInfo{id: id, FormatInfo: h.Info()})
	}
	// Video
	for _, id := range []core.FormatID{
		core.FmtMP4, core.FmtMOV, core.FmtMKV, core.FmtWebM,
		core.FmtAVI, core.FmtWMV, core.FmtFLV,
	} {
		h := vidpkg.New(id)
		all = append(all, namedFormatInfo{id: id, FormatInfo: h.Info()})
	}
	// Documents
	for _, id := range []core.FormatID{
		core.FmtPDF, core.FmtDOCX, core.FmtXLSX, core.FmtPPTX,
		core.FmtODT, core.FmtEPUB,
	} {
		h := docpkg.New(id)
		all = append(all, namedFormatInfo{id: id, FormatInfo: h.Info()})
	}
	return all
}

// ──────────────────────────────────────────────────────────────────────────────
// batch
// ──────────────────────────────────────────────────────────────────────────────

func runBatch(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: surgery batch <view|strip|edit> [flags] <directory>")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  surgery batch view ./photos")
		fmt.Println("  surgery batch view --json ./music")
		fmt.Println("  surgery batch strip --out ./clean ./photos")
		fmt.Println("  surgery batch strip --recursive ./media")
		fmt.Println(`  surgery batch edit --set "Copyright=ACME Corp" ./docs`)
		os.Exit(1)
	}

	subcmd := args[0]
	subargs := args[1:]

	switch subcmd {
	case "view":
		runBatchView(subargs)
	case "strip":
		runBatchStrip(subargs)
	case "edit":
		runBatchEdit(subargs)
	default:
		fmt.Fprintf(os.Stderr, "Unknown batch sub-command: %s\n", subcmd)
		fmt.Println("Valid sub-commands: view, strip, edit")
		os.Exit(1)
	}
}

func runBatchView(args []string) {
	fs := flag.NewFlagSet("batch view", flag.ExitOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	recursive := fs.Bool("recursive", false, "Recurse into subdirectories")
	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Println("Usage: surgery batch view [--json] [--recursive] <directory>")
		os.Exit(1)
	}

	dir := fs.Arg(0)
	p := core.NewPrinter(*jsonOut, false)
	files := collectFiles(dir, *recursive)
	errs := 0

	for _, f := range files {
		m, err := viewFile(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "✗ %s: %s\n", f, err)
			errs++
			continue
		}
		if !*jsonOut {
			fmt.Println(strings.Repeat("═", 60))
		}
		p.PrintMetadata(m)
	}

	if !*jsonOut {
		fmt.Printf("\nProcessed %d files", len(files))
		if errs > 0 {
			fmt.Printf(", %d errors", errs)
		}
		fmt.Println()
	}
}

func runBatchStrip(args []string) {
	fs := flag.NewFlagSet("batch strip", flag.ExitOnError)
	outDir := fs.String("out", "", "Output directory (default: in-place)")
	dryRun := fs.Bool("dry-run", false, "Preview without writing")
	recursive := fs.Bool("recursive", false, "Recurse into subdirectories")
	var keepFlags kvFlags
	fs.Var(&keepFlags, "keep", "Keep a metadata section (repeatable)")
	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Println("Usage: surgery batch strip [--out <dir>] [--recursive] [--dry-run] <directory>")
		os.Exit(1)
	}

	dir := fs.Arg(0)
	files := collectFiles(dir, *recursive)
	opts := core.StripOptions{
		KeepFields: []string(keepFlags),
		StripAll:   len(keepFlags) == 0,
	}

	ok, errs, skipped := 0, 0, 0
	for _, f := range files {
		outPath := ""
		if *outDir != "" {
			rel, _ := filepath.Rel(dir, f)
			outPath = filepath.Join(*outDir, rel)
			os.MkdirAll(filepath.Dir(outPath), 0755)
		}

		h, err := getHandler(f)
		if err != nil || !h.Info().CanStrip {
			skipped++
			continue
		}

		if *dryRun {
			fmt.Printf("[dry-run] would strip: %s\n", f)
			continue
		}

		if err := h.Strip(f, outPath, opts); err != nil {
			fmt.Fprintf(os.Stderr, "✗ %s: %s\n", f, err)
			errs++
		} else {
			fmt.Printf("✓ %s\n", f)
			ok++
		}
	}
	if !*dryRun {
		fmt.Printf("\nStripped: %d  |  Errors: %d  |  Skipped (unsupported): %d\n", ok, errs, skipped)
	}
}

func runBatchEdit(args []string) {
	fs := flag.NewFlagSet("batch edit", flag.ExitOnError)
	var setFlags kvFlags
	outDir := fs.String("out", "", "Output directory (default: in-place)")
	dryRun := fs.Bool("dry-run", false, "Preview without writing")
	recursive := fs.Bool("recursive", false, "Recurse into subdirectories")
	fs.Var(&setFlags, "set", "Set KEY=VALUE (repeatable)")
	fs.Parse(args)

	if fs.NArg() < 1 || len(setFlags) == 0 {
		fmt.Println("Usage: surgery batch edit --set KEY=VALUE [--recursive] [--out <dir>] <directory>")
		os.Exit(1)
	}

	dir := fs.Arg(0)
	setMap := map[string]string{}
	for _, kv := range setFlags {
		k, v, ok := core.ParseKV(kv)
		if !ok {
			continue
		}
		setMap[k] = v
	}

	opts := core.EditOptions{Set: setMap, DryRun: *dryRun}
	files := collectFiles(dir, *recursive)
	ok, errs, skipped := 0, 0, 0

	for _, f := range files {
		outPath := ""
		if *outDir != "" {
			rel, _ := filepath.Rel(dir, f)
			outPath = filepath.Join(*outDir, rel)
			os.MkdirAll(filepath.Dir(outPath), 0755)
		}

		h, err := getHandler(f)
		if err != nil || !h.Info().CanEdit {
			skipped++
			continue
		}

		if *dryRun {
			fmt.Printf("[dry-run] would edit: %s\n", f)
			continue
		}

		if err := h.Edit(f, outPath, opts); err != nil {
			fmt.Fprintf(os.Stderr, "✗ %s: %s\n", f, err)
			errs++
		} else {
			fmt.Printf("✓ %s\n", f)
			ok++
		}
	}
	if !*dryRun {
		fmt.Printf("\nEdited: %d  |  Errors: %d  |  Skipped (unsupported): %d\n", ok, errs, skipped)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Core helpers
// ──────────────────────────────────────────────────────────────────────────────

// getHandler returns the appropriate Handler for the given file path.
func getHandler(path string) (core.Handler, error) {
	fmtID, err := core.DetectFormat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot detect format of %s: %w", path, err)
	}
	if fmtID == core.FmtUnknown {
		return nil, fmt.Errorf("unknown or unsupported format: %s", path)
	}

	switch core.MediaTypeFor(fmtID) {
	case "image":
		return imgpkg.New(fmtID), nil
	case "audio":
		return audpkg.New(fmtID), nil
	case "video":
		return vidpkg.New(fmtID), nil
	case "document":
		return docpkg.New(fmtID), nil
	default:
		return nil, fmt.Errorf("no handler for format: %s", fmtID)
	}
}

// viewFile is a convenience wrapper.
func viewFile(path string) (*core.Metadata, error) {
	h, err := getHandler(path)
	if err != nil {
		return nil, err
	}
	return h.View(path)
}

// collectFiles gathers all regular files under dir.
// If recursive is true, subdirectories are descended into.
func collectFiles(dir string, recursive bool) []string {
	var files []string
	entries, err := os.ReadDir(dir)
	if err != nil {
		core.PrintError(fmt.Sprintf("cannot read directory %q: %s", dir, err))
		return nil
	}
	for _, e := range entries {
		full := filepath.Join(dir, e.Name())
		if e.IsDir() {
			if recursive {
				files = append(files, collectFiles(full, true)...)
			}
			continue
		}
		// Only include files with recognised extensions
		if _, err := core.DetectFormat(full); err == nil {
			fid, _ := core.DetectFormat(full)
			if fid != core.FmtUnknown {
				files = append(files, full)
			}
		}
	}
	return files
}
