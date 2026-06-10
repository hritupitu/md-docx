# md2docx

Convert Markdown files to Word documents (`.docx`) locally — no cloud, no subscriptions. Formatting, headings, bold, lists, tables, and code blocks all come through correctly.

Three ways to use it: a **native macOS app** (DMG), a **web GUI** (browser-based), and a **CLI** (terminal).

---

## macOS App — easiest way (DMG)

Download, drag to Applications, done. No Pandoc install required — it's bundled inside.

### Download

Go to the [Releases](https://github.com/hritupitu/docx-pdf/releases/latest) page and grab:

| File | Notes |
|---|---|
| `md2docx-2.0.0.dmg` | macOS universal (M1/M2/M3 + Intel), macOS 13+ |

Open the DMG, drag **md2docx** to your Applications folder, and launch it.

> **First launch:** macOS may say the app is from an unidentified developer. Go to **System Settings → Privacy & Security** and click **Open Anyway**.

### How to use

1. Click the drop zone to pick one or more `.md` files
2. Optionally attach a reference `.docx` to use its Word styles and fonts
3. Click **Convert to DOCX**
4. A Save dialog appears for each file — choose where to save it

---

## Web GUI — browser-based

Runs a local server and opens your browser. Drag & drop files, convert, download. Shuts itself down automatically when you close the browser tab.

### Download

| System | File |
|---|---|
| macOS (M1/M2/M3) | `md2docx-gui-macos-arm64` |
| macOS (Intel) | `md2docx-gui-macos-amd64` |
| Linux | `md2docx-gui-linux-amd64` |
| Windows | `md2docx-gui-windows-amd64.exe` |

### Setup (macOS / Linux)

```bash
chmod +x md2docx-gui-macos-arm64
sudo mv md2docx-gui-macos-arm64 /usr/local/bin/md2docx-gui
md2docx-gui
```

Your browser opens automatically at a local address.

> **macOS note:** Run `xattr -d com.apple.quarantine md2docx-gui-macos-arm64` before moving it, or right-click → Open → Open Anyway.

Pandoc must be installed separately for the web GUI:

```bash
brew install pandoc      # macOS
sudo apt install pandoc  # Linux
```

---

## CLI — for terminal users

```bash
# Single file
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
| `` `code` `` | Inline code |
| ```` ``` ```` code block | Code block |
| `\| table \|` | Word table |
| `[link](url)` | Hyperlink |

---

## Build from source

### macOS App (requires Xcode 15+ and pandoc cached from prior build)

```bash
git clone https://github.com/hritupitu/docx-pdf.git
cd docx-pdf
bash scripts/build-dmg.sh
# output: dist/md2docx-2.0.0.dmg
```

### Web GUI (requires Go 1.21+)

```bash
cd docx-pdf/md2docx-gui && go build -o md2docx-gui .
./md2docx-gui
```

### CLI (requires Go 1.21+)

```bash
cd docx-pdf/md2docx && go build -o md2docx .
```
