#!/bin/bash
set -e

DIST_DIR="dist"
mkdir -p $DIST_DIR

echo "Building for macOS..."
go build -o $DIST_DIR/ClickAssist_mac main.go

