# Media Metadata Surgery

**Media Metadata Surgery** is a precision‑focused, offline CLI tool for inspecting and modifying metadata in media files.

The core engine is written in **Go** for performance, safety, and correctness. A lightweight **Python wrapper** is used only for distribution, making the tool easy to install via `pip` while keeping the actual logic native and fast.

> v0.1.1 is the beginning. The project is intentionally small, strict, and correct. More formats and operations will be added incrementally.

---

## Why Media Metadata Surgery exists

Media files silently carry far more information than most users realize:

* GPS location and timestamps
* Device make, model, and camera internals
* Processing pipelines, filters, and AI hints
* Hidden user comments and sensor data

This project exists to:

* **Expose metadata clearly**
* **Operate fully offline**
* **Avoid heavy external dependencies**
* **Favor correctness over convenience**

No cloud. No telemetry. No shelling out to system tools.

---

## Current status

**Version:** `0.1.1` (initial release)

This version intentionally supports **only one format and one operation**:

* JPG / JPEG
* EXIF metadata inspection (read‑only)

This narrow scope allows the foundation to remain clean and extensible.

---

## Installation

### Install via pip (recommended)

```bash
pip install surgery
```

This installs the `surgery` command globally. The Go binary is bundled inside the Python package.

---

### Build from source (Go)

Requirements:

* Go 1.20+

```bash
git clone https://github.com/ankit-chaubey/media-metadata-surgery
cd media-metadata-surgery
go build -o surgery ./cli
```

Run directly:

```bash
./surgery view image.jpg
```

---

## Usage

### View EXIF metadata from a JPG image

```bash
surgery view image.jpg
```

Example output:

````text
EXIF Metadata:
Make: "vivo"
Model: "vivo T1 5G"
DateTimeOriginal: "2026:02:04 18:44:10"
GPSLatitude: ["18/1","20/1","4247/900"]
GPSLongitude: ["84/1","25/1","4824/190"]
UserComment: "filter: 104; ..."
````

---

## Supported formats (v0.1.1)

| Media type | Format     | Operation          |
| ---------- | ---------- | ------------------ |
| Image      | JPG / JPEG | View EXIF metadata |

More formats will be added only when they can be handled correctly.

---

## Project structure

```text
media-metadata-surgery/
├── cli/                # Go CLI entry point
├── core/               # Core metadata logic
│   └── jpg/            # JPG / EXIF handling
├── surgery/            # Python wrapper package
│   └── bin/surgery     # Go binary (bundled)
├── setup.py            # PyPI configuration
├── pyproject.toml
└── README.md
```

Design principles:

* Go does all real work
* Python only launches the binary
* No CGO
* No external system tools

---

## Security & privacy notes

* All operations are **offline**
* No network access
* No background processes
* Viewing metadata does not modify files

Metadata removal, when added, will be explicit and opt‑in.

---

## Roadmap

Planned, in order of correctness priority:

* GPS metadata removal (JPG)
* Structured JSON output
* Audio metadata (MP3 / FLAC)
* Video containers (MP4 / MKV)
* Batch processing
* Integrity and diff reporting

The project will grow conservatively.

---

## License

Apache License 2.0

---

## Developed by

**Ankit Chaubey**  
GitHub: https://github.com/ankit-chaubey  
Project repository: https://github.com/ankit-chaubey/media-metadata-surgery

---

## Philosophy

> Precision over features.
> Correctness over speed.
> Transparency over magic.

Media Metadata Surgery is built as a long‑term, trustworthy utility — not a shortcut tool.
