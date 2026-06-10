# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this repo is

Three separate tools that all convert Markdown → DOCX via Pandoc as the backend:

| Directory | Language | What it is |
|---|---|---|
| `md2docx/` | Go | CLI binary |
| `md2docx-gui/` | Go | Web GUI (HTTP server + browser) |
| `md2docx-swift/` | Swift/SwiftUI | Native macOS app (DMG) |
| `scripts/` | Bash + Python | DMG build pipeline + icon generator |

`md2docx-app/` is a leftover Go/webview prototype — ignore it.

## Build commands

### CLI
```bash
cd md2docx
go build -o md2docx .          # dev build
make release                   # cross-compile all platforms → dist/
```

### Web GUI
```bash
cd md2docx-gui
go build -o md2docx-gui .      # dev build
./md2docx-gui                  # runs server, opens browser
make release                   # cross-compile all platforms → dist/
```

### macOS DMG (requires Xcode 15+, macOS)
```bash
bash scripts/build-dmg.sh
# output: dist/md2docx-<VERSION>.dmg
```
The script downloads Pandoc binaries from GitHub releases, builds a universal binary via `lipo`, generates the `.icns` icon with Python/Pillow, assembles the `.app` bundle, and runs `hdiutil`. Pandoc arm64/x86_64 builds are cached in `/tmp/` after the first run.

Set `VERSION` env var to override the hardcoded default: `VERSION=2.1.0 bash scripts/build-dmg.sh`

### Swift app only (no DMG)
```bash
cd md2docx-swift
swift build -c release
# binary: .build/release/md2docx
```

## Architecture notes

### Web GUI flow
The GUI is a single Go binary that embeds `web/index.html` via `//go:embed web`. On launch it binds a random port, starts an HTTP server, and opens the browser. The browser flow is:
1. `POST /upload` — multipart form with `.md` files + optional ref `.docx`, stored to a temp dir, returns a `jobId`
2. `GET /events?job=<id>` — SSE stream; Go runs pandoc per file and streams `converted`/`error`/`complete` events
3. `GET /download?token=<tok>` — serves the output `.docx` as an attachment
4. `POST /heartbeat` — browser pings every 4s; server shuts down after 12s silence (heartbeat timeout is the only shutdown mechanism)

### SwiftUI app flow
No HTTP server. File picking uses `NSOpenPanel` directly (called from button actions — `fileImporter` SwiftUI modifier was tried and is unreliable in SPM-built apps). Conversion calls pandoc as a `Process`. Saving uses `NSSavePanel` via `withCheckedContinuation` to bridge async/await. Pandoc is looked up first at `Bundle.main.executableURL/../pandoc` (bundled), then common PATH locations.

### Release CI
`.github/workflows/release.yml` triggers on `v*` tag push or manual dispatch. Three jobs run in parallel:
- `build-go` (ubuntu): cross-compiles CLI + GUI for macOS/Linux/Windows via `make release`
- `build-dmg` (macos-14): runs `scripts/build-dmg.sh`, uploads DMG artifact
- `release`: downloads all artifacts, creates GitHub release with all 9 assets

To cut a release: `git tag v2.1.0 && git push origin v2.1.0`

### Pandoc detection order
All three tools use the same lookup order: bundled binary next to executable → common hardcoded paths (`/opt/homebrew/bin`, `/usr/local/bin`) → `PATH`. The CLI additionally offers interactive auto-install via brew/apt/dnf/winget.

### Icon generation
`scripts/make-icon.py` generates the `.icns` purely in Python (Pillow) — no design files. It draws the icon programmatically: dark navy rounded square, white document shape, bold "M↓" markdown logo, cyan accent lines, `.docx` badge. Edit that file to change the icon.
