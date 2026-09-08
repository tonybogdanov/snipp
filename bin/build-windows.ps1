#!/usr/bin/env pwsh
# Builds the Snipp Windows exe and its installer locally, mirroring
# .github/workflows/build.yml.

$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
Push-Location $root
try {
    New-Item -ItemType Directory -Force -Path artifacts | Out-Null

    # The release built from this commit is tagged with its short hash, so
    # stamping the same value in gives the in-app updater something to
    # compare against.
    # Sliced from the full hash rather than via --short, which widens the
    # abbreviation when it would be ambiguous; the release tag is a plain
    # 7-character slice of the same hash and the two have to match exactly.
    $version = (git rev-parse HEAD).Trim().Substring(0, 7)

    go build -ldflags "-H=windowsgui -X main.version=$version" -o artifacts/snipp.exe .
    Write-Host "Built $root\artifacts\snipp.exe"

    # The installer carries no payload — it downloads snipp.exe from the
    # latest release at run time — so it doesn't depend on the exe built
    # above and needs no version stamp of its own.
    go build -ldflags "-H=windowsgui" -o artifacts/snipp-installer.exe ./cmd/installer
    Write-Host "Built $root\artifacts\snipp-installer.exe"
} finally {
    Pop-Location
}
