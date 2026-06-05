#!/usr/bin/env bash

set -euo pipefail

BINARY_NAME="sys-stat-mqtt"
DIST_DIR="dist"

echo "Cleaning up dist folder..."
rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

# Ensure we use pure Go cross-compilation
export CGO_ENABLED=0

compile_target() {
    local os=$1
    local arch=$2
    local ext=${3:-""}
    local output="${DIST_DIR}/${BINARY_NAME}-${os}-${arch}${ext}"

    echo "Building for ${os}/${arch}..."
    GOOS=$os GOARCH=$arch go build -ldflags="-s -w" -o "$output" main.go
}

# Linux targets
compile_target "linux" "amd64"
compile_target "linux" "arm64"
compile_target "linux" "386"
compile_target "linux" "arm"

# macOS (Darwin) targets
compile_target "darwin" "amd64"
compile_target "darwin" "arm64"

# Windows targets
compile_target "windows" "amd64" ".exe"
compile_target "windows" "386" ".exe"
compile_target "windows" "arm64" ".exe"

echo "All builds completed!"
