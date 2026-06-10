# md2docx

Convert Markdown files to Word documents (`.docx`) locally — no cloud, no subscriptions. Formatting, headings, bold, lists, tables, and code blocks all come through correctly.

Two versions: a **GUI** (browser-based, drag & drop) and a **CLI** (terminal).

---

## GUI — easiest way to use it

Double-click the app, your browser opens, drag your `.md` files in, click **Convert**, download the `.docx` files. Close the tab when done — the app shuts itself down automatically.

Bonus: attach a reference `.docx` to make the output use your own Word styles and fonts.

### Download

Go to the [Releases](https://github.com/hritupitu/docx-pdf/releases/latest) page and grab the GUI binary for your system:

| System | File |
|---|---|
| macOS (M1/M2/M3) | `md2docx-gui-macos-arm64` |
| macOS (Intel) | `md2docx-gui-macos-amd64` |
| Linux | `md2docx-gui-linux-amd64` |
| Windows | `md2docx-gui-windows-amd64.exe` |

### Setup (macOS / Linux)

```bash
chmod +x md2docx-gui-macos-arm64     # use your filename
sudo mv md2docx-gui-macos-arm64 /usr/local/bin/md2docx-gui
```

Then just run:

```bash
md2docx-gui
```

Your browser opens automatically.

> **macOS note:** Right-click → Open → Open anyway, or run `xattr -d com.apple.quarantine md2docx-gui-macos-arm64` before moving it.

### Pandoc (required, one-time)

The GUI shows a warning if Pandoc isn't installed. On macOS:

```bash
brew install pandoc
```

---

## CLI — for terminal users

```bash
# Single file (output saved next to the input)
md2docx notes.md

# Save to a specific folder
md2docx -o ~/Desktop notes.md

# Batch convert
md2docx *.md

# Use a reference .docx for custom Word styles
md2docx -ref my-template.docx *.md
```

If Pandoc isn't installed, the CLI will offer to install it automatically on first run.

### Download

| System | File |
|---|---|
| macOS (M1/M2/M3) | `md2docx-macos-arm64` |
| macOS (Intel) | `md2docx-macos-amd64` |
| Linux | `md2docx-linux-amd64` |
| Windows | `md2docx-windows-amd64.exe` |

### Flags

| Flag | Default | Description |
|---|---|---|
| `-o <dir>` | same folder as input | Where to save the `.docx` files |
| `-ref <file>` | none | Reference `.docx` for custom Word styles/fonts |
| `-j <n>` | `4` | Parallel workers for batch conversion |
| `-q` | off | Quiet mode, only print output paths |
| `--version` | | Print version and exit |

---

## What gets converted

| Markdown | Word output |
|---|---|
| `# Heading 1` | Heading 1 style |
| `**bold**` | Bold |
| `*italic*` | Italic |
| `- list item` | Bulleted list |
| `1. item` | Numbered list |
| `\`code\`` | Inline code |
| ` ``` ` code block | Code block |
| `\| table \|` | Word table |
| `[link](url)` | Hyperlink |

---

## Build from source

Requires [Go 1.21+](https://go.dev/dl/) and [Pandoc](https://pandoc.org/installing.html).

```bash
git clone https://github.com/hritupitu/docx-pdf.git

# GUI
cd docx-pdf/md2docx-gui && go build -o md2docx-gui .

# CLI
cd docx-pdf/md2docx && go build -o md2docx .
```
