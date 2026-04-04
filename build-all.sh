#!/bin/bash

set -eu

GO_BIN="${GO_BIN:-$(command -v go)}"
APP_NAME="${APP_NAME:-django-stack}"
VERSION="$(git describe --tags --always --dirty 2>/dev/null || echo "dev")"
COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo "")"
BUILD_DATE="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
DIST="dist"
PKG="./cmd/cli"
LDFLAGS="-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${BUILD_DATE}"

echo "Using Go: $GO_BIN"
"$GO_BIN" version

rm -rf "$DIST"
mkdir -p "$DIST"

echo
echo "Building ${APP_NAME} ${VERSION} for multiple platforms..."
echo

platforms=(
  "windows/amd64"
  "linux/amd64"
  "linux/arm64"
  "darwin/amd64"
  "darwin/arm64"
)

success_count=0
fail_count=0

for platform in "${platforms[@]}"; do
  IFS="/" read -r GOOS GOARCH <<EOF
$platform
EOF

  output_name="${APP_NAME}-${GOOS}-${GOARCH}"
  if [ "$GOOS" = "windows" ]; then
    output_name="${output_name}.exe"
  fi

  echo "----------------------------------------"
  echo "Building for ${GOOS}/${GOARCH}..."
  echo "----------------------------------------"

  if GOOS="$GOOS" GOARCH="$GOARCH" "$GO_BIN" build -ldflags "$LDFLAGS" -o "${DIST}/${output_name}" "$PKG"; then
    echo "Built ${output_name}"
    success_count=$((success_count + 1))
  else
    echo "Failed for ${GOOS}/${GOARCH}"
    fail_count=$((fail_count + 1))
  fi
  echo
done

echo "----------------------------------------"
echo "Build Summary"
echo "----------------------------------------"
echo "Successful: ${success_count}"
echo "Failed: ${fail_count}"
echo
ls -la "$DIST"/

chmod +x "${DIST}"/${APP_NAME}-linux-* "${DIST}"/${APP_NAME}-darwin-* 2>/dev/null || true

echo
echo "Done!"
