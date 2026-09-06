#!/usr/bin/env bash
# Cross-compiles the Snipp Linux binary and its installer, mirroring the
# linux job in .github/workflows/build.yml. No Linux toolchain needed (pure
# Go, no cgo) -- this can run from any bash, including Git Bash on Windows.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

mkdir -p artifacts
export GOOS=linux GOARCH=amd64 CGO_ENABLED=0
go build -o artifacts/snipp .
echo "Built $root/artifacts/snipp"

payload="cmd/installer/payload/linux.bin"
cp artifacts/snipp "$payload"
trap 'git checkout -- "$payload"' EXIT

go build -o artifacts/snipp-installer ./cmd/installer
echo "Built $root/artifacts/snipp-installer"
