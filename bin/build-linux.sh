#!/usr/bin/env bash
# Cross-compiles the Snipp Linux binary and its installer, mirroring the
# linux job in .github/workflows/build.yml. No Linux toolchain needed (pure
# Go, no cgo) -- this can run from any bash, including Git Bash on Windows.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

mkdir -p artifacts
export GOOS=linux GOARCH=amd64 CGO_ENABLED=0

# The release built from this commit is tagged with its short hash, so
# stamping the same value in gives the in-app updater something to compare
# against.
# Sliced from the full hash rather than via --short, which widens the
# abbreviation when it would be ambiguous; the release tag is a plain
# 7-character slice of the same hash and the two have to match exactly.
version="$(git rev-parse HEAD | cut -c1-7)"

go build -ldflags "-X main.version=$version" -o artifacts/snipp .
echo "Built $root/artifacts/snipp"

# The installer carries no payload — it downloads the snipp binary from the
# latest release at run time — so it doesn't depend on the binary built
# above and needs no version stamp of its own.
go build -o artifacts/snipp-installer ./cmd/installer
echo "Built $root/artifacts/snipp-installer"
