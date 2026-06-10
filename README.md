# docx2pdf

Convert Word documents to PDF locally — no cloud, no subscriptions. Formatting stays intact because it uses LibreOffice under the hood (the same engine Word uses on Linux).

The binary handles everything: if LibreOffice isn't installed it will offer to install it for you automatically.

---

## Installation

### Step 1 — Download the binary

Go to the [Releases](https://github.com/hritupitu/docx-pdf/releases/latest) page and download the file for your system:

| System | File |
|---|---|
| macOS (M1/M2/M3) | `docx2pdf-macos-arm64` |
| macOS (Intel) | `docx2pdf-macos-amd64` |
| Linux | `docx2pdf-linux-amd64` |
| Windows | `docx2pdf-windows-amd64.exe` |

### Step 2 — Make it runnable (macOS / Linux only)

Open Terminal, `cd` to your Downloads folder, then run:

```bash
chmod +x docx2pdf-macos-arm64        # use your filename here
sudo mv docx2pdf-macos-arm64 /usr/local/bin/docx2pdf
```

On **Windows**: just move the `.exe` somewhere convenient and run it from there.

### Step 3 — First run (installs LibreOffice if needed)

```bash
docx2pdf myfile.docx
```

If LibreOffice isn't on your machine yet, it will ask:

```
  LibreOffice not found.

  → Will run: brew install --cask libreoffice

  Install LibreOffice now? [y/N]
```

Type `y` and hit Enter. It installs LibreOffice and then converts your file — all in one go. You only need to do this once.

> **macOS note:** After downloading, macOS may warn that the binary is from an unidentified developer. To bypass: right-click the file → Open → Open anyway. Or run `xattr -d com.apple.quarantine docx2pdf-macos-arm64` before moving it.

---

## Usage

```bash
# Convert a single file (PDF saved next to the docx)
docx2pdf report.docx

# Save to a specific folder
docx2pdf -o ~/Desktop report.docx

# Convert multiple files at once
docx2pdf *.docx

# Batch convert into a folder, 8 files at a time
docx2pdf -j 8 -o ./pdfs *.docx

# Quiet mode — only prints output paths (good for scripting)
docx2pdf -q report.docx
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `-o <dir>` | same folder as input | Where to save the PDF(s) |
| `-j <n>` | `4` | Parallel workers for batch conversion |
| `-q` | off | Quiet mode, only print output paths |
| `--version` | | Print version and exit |

---

## Requirements

- **LibreOffice** — the binary installs it for you on first run via `brew` (macOS), `apt` / `dnf` / `pacman` (Linux), or `winget` (Windows).
- No internet connection needed after that. Conversion is fully local.

---

## Build from source

Requires [Go 1.21+](https://go.dev/dl/).

```bash
git clone https://github.com/hritupitu/docx-pdf.git
cd docx-pdf/docx2pdf
go build -o docx2pdf .
sudo mv docx2pdf /usr/local/bin/docx2pdf
```
