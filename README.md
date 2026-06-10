# md2docx

Convert Markdown files to Word documents (`.docx`) locally — no cloud, no subscriptions. Formatting, headings, bold, lists, tables, and code blocks all come through correctly.

Three ways to use it: a **native macOS app** (DMG), a **web GUI** (browser-based), and a **CLI** (terminal).

---

## macOS App — easiest way (DMG)

No Pandoc install required — it's bundled inside the app.

**[⬇ Download the latest DMG](https://github.com/hritupitu/md-docx/releases/latest)**

Open the DMG, drag **md2docx** to your Applications folder, and launch it.

> **First launch:** macOS may block the app since it's not notarized. Go to **System Settings → Privacy & Security** and click **Open Anyway**.

### How to use

1. Click the drop zone to pick one or more `.md` files
2. Optionally attach a reference `.docx` to use its Word styles and fonts
3. Click **Convert to DOCX**
4. A Save dialog appears for each file — choose where to save it

---

## Web GUI — browser-based

Runs a local server and opens your browser. Drag & drop files, convert, download. Shuts itself down automatically when you close the browser tab.

Requires Pandoc installed: `brew install pandoc` / `sudo apt install pandoc`

### Install

**macOS (M1/M2/M3)**
```bash
curl -L https://github.com/hritupitu/md-docx/releases/latest/download/md2docx-gui-macos-arm64 -o md2docx-gui
chmod +x md2docx-gui && xattr -d com.apple.quarantine md2docx-gui
sudo mv md2docx-gui /usr/local/bin/md2docx-gui
```

**macOS (Intel)**
```bash
curl -L https://github.com/hritupitu/md-docx/releases/latest/download/md2docx-gui-macos-amd64 -o md2docx-gui
chmod +x md2docx-gui && xattr -d com.apple.quarantine md2docx-gui
sudo mv md2docx-gui /usr/local/bin/md2docx-gui
```

**Linux**
```bash
curl -L https://github.com/hritupitu/md-docx/releases/latest/download/md2docx-gui-linux-amd64 -o md2docx-gui
chmod +x md2docx-gui
sudo mv md2docx-gui /usr/local/bin/md2docx-gui
```

**Windows** — download [`md2docx-gui-windows-amd64.exe`](https://github.com/hritupitu/md-docx/releases/latest/download/md2docx-gui-windows-amd64.exe) and run it directly.

Then just run:
```bash
md2docx-gui
```

Your browser opens automatically.

---

## CLI — for terminal users

If Pandoc isn't installed, the CLI will offer to install it automatically on first run.

### Install

**macOS (M1/M2/M3)**
```bash
curl -L https://github.com/hritupitu/md-docx/releases/latest/download/md2docx-macos-arm64 -o md2docx
chmod +x md2docx && xattr -d com.apple.quarantine md2docx
sudo mv md2docx /usr/local/bin/md2docx
```

**macOS (Intel)**
```bash
curl -L https://github.com/hritupitu/md-docx/releases/latest/download/md2docx-macos-amd64 -o md2docx
chmod +x md2docx && xattr -d com.apple.quarantine md2docx
sudo mv md2docx /usr/local/bin/md2docx
```

**Linux**
```bash
curl -L https://github.com/hritupitu/md-docx/releases/latest/download/md2docx-linux-amd64 -o md2docx
chmod +x md2docx
sudo mv md2docx /usr/local/bin/md2docx
```

**Windows** — download [`md2docx-windows-amd64.exe`](https://github.com/hritupitu/md-docx/releases/latest/download/md2docx-windows-amd64.exe), rename to `md2docx.exe`, and add to your PATH.

### Usage

```bash
md2docx notes.md                          # single file, saved next to input
md2docx -o ~/Desktop notes.md             # save to a specific folder
md2docx *.md                              # batch convert
md2docx -ref my-template.docx *.md       # use a reference .docx for Word styles
```

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

```bash
git clone https://github.com/hritupitu/md-docx.git
cd md-docx
```

**macOS App** (requires Xcode 15+)
```bash
bash scripts/build-dmg.sh
# output: dist/md2docx-<version>.dmg
```

**Web GUI** (requires Go 1.21+)
```bash
cd md2docx-gui && go build -o md2docx-gui .
```

**CLI** (requires Go 1.21+)
```bash
cd md2docx && go build -o md2docx .
```
