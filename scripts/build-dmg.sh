#!/usr/bin/env bash
# Creates md2docx.app (with pandoc bundled) and packages it into a DMG.
set -euo pipefail

PANDOC_VERSION="3.9.0.2"
APP_NAME="md2docx"
VERSION="2.0.0"
BUNDLE_ID="io.github.hritupitu.md2docx"

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
GUI_DIR="$REPO_ROOT/md2docx-gui"
BUILD_DIR="$REPO_ROOT/build"
DIST_DIR="$REPO_ROOT/dist"

# ── colours ───────────────────────────────────────────────────────────────────
CYAN='\033[36m' GREEN='\033[32m' YELLOW='\033[33m' BOLD='\033[1m' NC='\033[0m'
log()  { echo -e "${CYAN}→${NC}  $1"; }
ok()   { echo -e "${GREEN}✔${NC}  $1"; }
warn() { echo -e "${YELLOW}⚠${NC}  $1"; }
step() { echo -e "\n${BOLD}$1${NC}"; }

step "── 1 / 5  Build Go binary (universal) ──────────────────────────────"
mkdir -p "$BUILD_DIR" "$DIST_DIR"
TMPDIR_GO=/tmp/gobuild; mkdir -p $TMPDIR_GO

log "Compiling arm64…"
TMPDIR=$TMPDIR_GO GOOS=darwin GOARCH=arm64 \
  go build -C "$GUI_DIR" -ldflags="-s -w" -o "$BUILD_DIR/${APP_NAME}-arm64" .

log "Compiling amd64…"
TMPDIR=$TMPDIR_GO GOOS=darwin GOARCH=amd64 \
  go build -C "$GUI_DIR" -ldflags="-s -w" -o "$BUILD_DIR/${APP_NAME}-amd64" .

log "Merging into universal binary…"
lipo -create \
  "$BUILD_DIR/${APP_NAME}-arm64" \
  "$BUILD_DIR/${APP_NAME}-amd64" \
  -output "$BUILD_DIR/${APP_NAME}-universal"
ok "Go universal binary ready"

step "── 2 / 5  Download & bundle Pandoc ${PANDOC_VERSION} ─────────────────────"

download_pandoc() {
  local arch="$1"   # arm64 or x86_64
  local outbin="$2"
  local pkg_arch="${arch}"    # same in URL

  local url="https://github.com/jgm/pandoc/releases/download/${PANDOC_VERSION}/pandoc-${PANDOC_VERSION}-${pkg_arch}-macOS.pkg"
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

  # Find the Payload (location varies between pandoc versions)
  local payload
  payload=$(find "$expanded" -name "Payload" | head -1)
  if [ -z "$payload" ]; then
    echo "ERROR: Could not find Payload in expanded pkg"; exit 1
  fi

  mkdir -p "$extracted"
  cd "$extracted"
  # Try gunzip+cpio first, fall back to tar
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
APP_BUNDLE="$BUILD_DIR/${APP_NAME}.app"
rm -rf "$APP_BUNDLE"
mkdir -p "$APP_BUNDLE/Contents/MacOS"
mkdir -p "$APP_BUNDLE/Contents/Resources"

cp "$BUILD_DIR/${APP_NAME}-universal" "$APP_BUNDLE/Contents/MacOS/${APP_NAME}"
cp "$BUILD_DIR/pandoc-universal"      "$APP_BUNDLE/Contents/MacOS/pandoc"

cat > "$APP_BUNDLE/Contents/Info.plist" << PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleDisplayName</key>   <string>md2docx</string>
    <key>CFBundleExecutable</key>    <string>${APP_NAME}</string>
    <key>CFBundleIdentifier</key>    <string>${BUNDLE_ID}</string>
    <key>CFBundleName</key>          <string>md2docx</string>
    <key>CFBundlePackageType</key>   <string>APPL</string>
    <key>CFBundleShortVersionString</key> <string>${VERSION}</string>
    <key>CFBundleVersion</key>       <string>${VERSION}</string>
    <key>LSMinimumSystemVersion</key><string>11.0</string>
    <key>LSUIElement</key>           <true/>
    <key>NSHighResolutionCapable</key><true/>
</dict>
</plist>
PLIST

log "Ad-hoc signing…"
codesign --force --deep --sign - "$APP_BUNDLE"
ok "App bundle ready  →  $APP_BUNDLE"

step "── 4 / 5  Create DMG ───────────────────────────────────────────────"
DMG_FINAL="$DIST_DIR/${APP_NAME}-${VERSION}.dmg"
DMG_TMP="/tmp/${APP_NAME}-rw.dmg"
VOLUME_NAME="md2docx ${VERSION}"
MOUNT_POINT="/Volumes/${VOLUME_NAME}"

rm -f "$DMG_TMP" "$DMG_FINAL"

# Create a writable image, size slightly larger than app
APP_MB=$(du -sm "$APP_BUNDLE" | cut -f1)
DMG_MB=$(( APP_MB + 30 ))

hdiutil create \
  -size "${DMG_MB}m" \
  -fs HFS+ \
  -volname "$VOLUME_NAME" \
  "$DMG_TMP" > /dev/null

hdiutil attach "$DMG_TMP" -mountpoint "$MOUNT_POINT" > /dev/null

cp -r "$APP_BUNDLE" "$MOUNT_POINT/"
ln -s /Applications "$MOUNT_POINT/Applications"

# Tell Finder to show the window nicely (best-effort — fails silently without Finder)
osascript << ASCRIPT 2>/dev/null || true
tell application "Finder"
  tell disk "${VOLUME_NAME}"
    open
    set current view of container window to icon view
    set toolbar visible of container window to false
    set statusbar visible of container window to false
    set the bounds of container window to {200, 200, 720, 440}
    set icon size of icon view options of container window to 100
    set arrangement of icon view options of container window to not arranged
    set position of item "md2docx.app" of container window to {130, 120}
    set position of item "Applications" of container window to {390, 120}
    close
  end tell
end tell
ASCRIPT

hdiutil detach "$MOUNT_POINT" > /dev/null

log "Compressing DMG…"
hdiutil convert "$DMG_TMP" \
  -format UDZO \
  -imagekey zlib-level=9 \
  -o "$DMG_FINAL" > /dev/null
rm "$DMG_TMP"
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
