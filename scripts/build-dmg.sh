#!/usr/bin/env bash
# Creates md2docx.app (with pandoc bundled) and packages it into a DMG.
set -euo pipefail

PANDOC_VERSION="3.9.0.2"
APP_NAME="md2docx"
VERSION="${VERSION:-2.0.0}"
BUNDLE_ID="io.github.hritupitu.md2docx"

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SWIFT_DIR="$REPO_ROOT/md2docx-swift"
BUILD_DIR="$REPO_ROOT/build"
DIST_DIR="$REPO_ROOT/dist"

# ── colours ───────────────────────────────────────────────────────────────────
CYAN='\033[36m' GREEN='\033[32m' YELLOW='\033[33m' BOLD='\033[1m' NC='\033[0m'
log()  { echo -e "${CYAN}→${NC}  $1"; }
ok()   { echo -e "${GREEN}✔${NC}  $1"; }
warn() { echo -e "${YELLOW}⚠${NC}  $1"; }
step() { echo -e "\n${BOLD}$1${NC}"; }

step "── 1 / 5  Build Swift binary ────────────────────────────────────────"
mkdir -p "$BUILD_DIR" "$DIST_DIR"

log "Compiling SwiftUI app (release)…"
cd "$SWIFT_DIR"
swift build -c release 2>&1
SWIFT_BIN="$SWIFT_DIR/.build/release/${APP_NAME}"
cp "$SWIFT_BIN" "$BUILD_DIR/${APP_NAME}-binary"
ok "Swift binary ready"
cd "$REPO_ROOT"

step "── 2 / 5  Download & bundle Pandoc ${PANDOC_VERSION} ─────────────────────"

download_pandoc() {
  local arch="$1"
  local outbin="$2"

  local url="https://github.com/jgm/pandoc/releases/download/${PANDOC_VERSION}/pandoc-${PANDOC_VERSION}-${arch}-macOS.pkg"
  local pkg="/tmp/pandoc-${arch}.pkg"
  local expanded="/tmp/pandoc-${arch}-expanded"
  local extracted="/tmp/pandoc-${arch}-extracted"

  if [ -f "$outbin" ]; then
    warn "Using cached pandoc-${arch}"
    return
  fi

  log "Downloading pandoc ${PANDOC_VERSION} (${arch})…"
  curl -L --progress-bar "$url" -o "$pkg"

  log "Extracting binary…"
  rm -rf "$expanded" "$extracted"
  pkgutil --expand "$pkg" "$expanded"

  local payload
  payload=$(find "$expanded" -name "Payload" | head -1)
  if [ -z "$payload" ]; then
    echo "ERROR: Could not find Payload in expanded pkg"; exit 1
  fi

  mkdir -p "$extracted"
  cd "$extracted"
  if gunzip -c "$payload" | cpio -id --quiet 2>/dev/null; then
    :
  else
    tar -xf "$payload" -C "$extracted" 2>/dev/null || true
  fi

  local bin
  bin=$(find "$extracted" -type f -name "pandoc" | head -1)
  if [ -z "$bin" ]; then
    echo "ERROR: Could not find pandoc binary after extraction"; exit 1
  fi
  cp "$bin" "$outbin"
  chmod +x "$outbin"
  cd "$REPO_ROOT"
}

download_pandoc "arm64"  "$BUILD_DIR/pandoc-arm64"
download_pandoc "x86_64" "$BUILD_DIR/pandoc-amd64"

log "Merging pandoc into universal binary…"
lipo -create \
  "$BUILD_DIR/pandoc-arm64" \
  "$BUILD_DIR/pandoc-amd64" \
  -output "$BUILD_DIR/pandoc-universal"
chmod +x "$BUILD_DIR/pandoc-universal"
ok "Pandoc universal binary ready ($(du -sh "$BUILD_DIR/pandoc-universal" | cut -f1))"

step "── 3 / 5  Assemble .app bundle ─────────────────────────────────────"

log "Generating app icon…"
python3 "$REPO_ROOT/scripts/make-icon.py" "$BUILD_DIR/md2docx.icns"

APP_BUNDLE="$BUILD_DIR/${APP_NAME}.app"
rm -rf "$APP_BUNDLE"
mkdir -p "$APP_BUNDLE/Contents/MacOS"
mkdir -p "$APP_BUNDLE/Contents/Resources"

cp "$BUILD_DIR/${APP_NAME}-binary"   "$APP_BUNDLE/Contents/MacOS/${APP_NAME}"
cp "$BUILD_DIR/pandoc-universal"     "$APP_BUNDLE/Contents/MacOS/pandoc"
cp "$BUILD_DIR/md2docx.icns"        "$APP_BUNDLE/Contents/Resources/AppIcon.icns"

cat > "$APP_BUNDLE/Contents/Info.plist" << PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleDisplayName</key>        <string>md2docx</string>
    <key>CFBundleExecutable</key>         <string>${APP_NAME}</string>
    <key>CFBundleIconFile</key>           <string>AppIcon</string>
    <key>CFBundleIdentifier</key>         <string>${BUNDLE_ID}</string>
    <key>CFBundleName</key>               <string>md2docx</string>
    <key>CFBundlePackageType</key>        <string>APPL</string>
    <key>CFBundleShortVersionString</key> <string>${VERSION}</string>
    <key>CFBundleVersion</key>            <string>${VERSION}</string>
    <key>LSMinimumSystemVersion</key>     <string>13.0</string>
    <key>NSHighResolutionCapable</key>    <true/>
    <key>NSPrincipalClass</key>           <string>NSApplication</string>
</dict>
</plist>
PLIST

log "Ad-hoc signing…"
codesign --force --deep --sign - "$APP_BUNDLE"
ok "App bundle ready  →  $APP_BUNDLE"

step "── 4 / 5  Create DMG ───────────────────────────────────────────────"
DMG_FINAL="$DIST_DIR/${APP_NAME}-${VERSION}.dmg"
STAGING="$BUILD_DIR/dmg-staging"
VOLUME_NAME="md2docx"

rm -f "$DMG_FINAL"
rm -rf "$STAGING"
mkdir -p "$STAGING"

cp -r "$APP_BUNDLE" "$STAGING/"
ln -s /Applications "$STAGING/Applications"

hdiutil create \
  -volname "$VOLUME_NAME" \
  -srcfolder "$STAGING" \
  -ov \
  -format UDZO \
  -imagekey zlib-level=9 \
  "$DMG_FINAL" > /dev/null

rm -rf "$STAGING"
ok "DMG ready  →  $DMG_FINAL  ($(du -sh "$DMG_FINAL" | cut -f1))"

step "── 5 / 5  Done ─────────────────────────────────────────────────────"
echo ""
echo -e "  ${GREEN}${BOLD}✔ $DMG_FINAL${NC}"
echo ""
echo "  To test locally:"
echo "    open \"$DMG_FINAL\""
echo ""
echo "  To publish to GitHub:"
echo "    gh release upload v2.0.0 \"$DMG_FINAL\""
echo ""
