<p align="center">
  <h1 align="center">md2docx</h1>
  <p align="center">Convert Markdown to Word documents — locally, instantly, no cloud.</p>
  <p align="center">
    <a href="https://github.com/hritupitu/md-docx/releases/latest"><img src="https://img.shields.io/github/v/release/hritupitu/md-docx?style=flat-square&color=00d4ff" alt="Latest release"></a>
    <a href="https://github.com/hritupitu/md-docx/actions/workflows/release.yml"><img src="https://img.shields.io/github/actions/workflow/status/hritupitu/md-docx/release.yml?style=flat-square&label=build" alt="Build"></a>
    <a href="LICENSE"><img src="https://img.shields.io/github/license/hritupitu/md-docx?style=flat-square&color=22c55e" alt="License"></a>
    <a href="https://github.com/hritupitu/md-docx/releases/latest"><img src="https://img.shields.io/github/downloads/hritupitu/md-docx/total?style=flat-square&color=f59e0b" alt="Downloads"></a>
  </p>
</p>

---

Pick your flavour:

| | What | Best for |
|---|---|---|
| 🖥 **[macOS App](#-macos-app)** | Native SwiftUI app, Pandoc bundled | Anyone on Mac — no setup |
| 🌐 **[Web GUI](#-web-gui)** | Local browser UI, drag & drop | Cross-platform, quick use |
| ⌨️ **[CLI](#️-cli)** | Terminal binary, batch support | Scripts, CI, power users |

All three run **100% locally** — your files never leave your machine.

---

## 🖥 macOS App

The easiest way. Pandoc is bundled — nothing else to install.

**[⬇ Download latest .dmg](https://github.com/hritupitu/md-docx/releases/latest)**

1. Open the DMG and drag **md2docx** to Applications
2. Launch it — click to pick `.md` files, hit **Convert**, save your `.docx`

> First launch: macOS will say the app is unverified. Go to **System Settings → Privacy & Security → Open Anyway**.

**Requires macOS 13+.** Works on Apple Silicon and Intel.

---

## 🌐 Web GUI

A local HTTP server that opens in your browser. Drag & drop files, convert, download. The server shuts itself down automatically when you close the tab.

Requires [Pandoc](https://pandoc.org/installing.html) installed on your system.

### Install

<details>
<summary><strong>macOS</strong></summary>

```bash
# Apple Silicon
curl -L https://github.com/hritupitu/md-docx/releases/latest/download/md2docx-gui-macos-arm64 -o md2docx-gui

# Intel
curl -L https://github.com/hritupitu/md-docx/releases/latest/download/md2docx-gui-macos-amd64 -o md2docx-gui

chmod +x md2docx-gui
xattr -d com.apple.quarantine md2docx-gui
sudo mv md2docx-gui /usr/local/bin/md2docx-gui

brew install pandoc
```
</details>

<details>
<summary><strong>Linux</strong></summary>

```bash
curl -L https://github.com/hritupitu/md-docx/releases/latest/download/md2docx-gui-linux-amd64 -o md2docx-gui
chmod +x md2docx-gui
sudo mv md2docx-gui /usr/local/bin/md2docx-gui

sudo apt install pandoc  # or dnf, pacman, etc.
```
</details>

<details>
<summary><strong>Windows</strong></summary>

Download [`md2docx-gui-windows-amd64.exe`](https://github.com/hritupitu/md-docx/releases/latest/download/md2docx-gui-windows-amd64.exe) and run it directly.

Install Pandoc: `winget install JohnMacFarlane.Pandoc`
</details>

### Run

```bash
md2docx-gui
# Your browser opens automatically at http://127.0.0.1:<port>
```

---

## ⌨️ CLI

Batch convert, scriptable, works everywhere. Offers to auto-install Pandoc if it's not found.

### Install

<details>
<summary><strong>macOS</strong></summary>

```bash
# Apple Silicon
curl -L https://github.com/hritupitu/md-docx/releases/latest/download/md2docx-macos-arm64 -o md2docx

# Intel
curl -L https://github.com/hritupitu/md-docx/releases/latest/download/md2docx-macos-amd64 -o md2docx

chmod +x md2docx
xattr -d com.apple.quarantine md2docx
sudo mv md2docx /usr/local/bin/md2docx
```
</details>

<details>
<summary><strong>Linux</strong></summary>

```bash
curl -L https://github.com/hritupitu/md-docx/releases/latest/download/md2docx-linux-amd64 -o md2docx
chmod +x md2docx
sudo mv md2docx /usr/local/bin/md2docx
```
</details>

<details>
<summary><strong>Windows</strong></summary>

Download [`md2docx-windows-amd64.exe`](https://github.com/hritupitu/md-docx/releases/latest/download/md2docx-windows-amd64.exe), rename to `md2docx.exe`, and add to your PATH.
</details>

### Usage

```bash
md2docx notes.md                        # convert a single file
md2docx *.md                            # batch convert a folder
md2docx -o ~/Desktop report.md          # specify output directory
md2docx -ref template.docx *.md         # apply Word styles from a reference doc
md2docx -j 8 *.md                       # batch with 8 parallel workers
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `-o <dir>` | next to input file | Output directory for `.docx` files |
| `-ref <file>` | — | Reference `.docx` whose styles and fonts are applied |
| `-j <n>` | `4` | Number of parallel workers for batch jobs |
| `-q` | off | Quiet — only print output paths, no progress UI |
| `--version` | | Print version and exit |

---

## What converts

| Markdown | Word |
|---|---|
| `# H1` / `## H2` / `### H3` | Heading styles |
| `**bold**` / `*italic*` | Bold / Italic |
| `- item` / `1. item` | Bullet / Numbered list |
| `` `inline code` `` | Inline code |
| ```` ```lang ```` block | Code block |
| `> blockquote` | Block quote |
| `\| table \|` | Word table |
| `[text](url)` | Hyperlink |
| `![alt](img)` | Embedded image |

---

## Building from source

**Requirements:** Go 1.21+ for the CLI and web GUI. Xcode 15+ for the macOS app.

```bash
git clone https://github.com/hritupitu/md-docx.git
cd md-docx
```

```bash
# CLI
cd md2docx && go build -o md2docx .

# Web GUI
cd md2docx-gui && go build -o md2docx-gui .

# macOS DMG (downloads and bundles Pandoc automatically)
bash scripts/build-dmg.sh
```

Releases are built automatically by [GitHub Actions](.github/workflows/release.yml) on every version tag. To cut a release:

```bash
git tag v2.1.0 && git push origin v2.1.0
```

---

## Contributing

PRs are welcome. Open an issue first for anything beyond small fixes so we can agree on the approach.

---

## License

[MIT](LICENSE)
