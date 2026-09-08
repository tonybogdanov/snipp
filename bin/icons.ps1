#!/usr/bin/env pwsh
# Regenerates every raster icon the app and its installer embed from
# assets/_source.png, the icon's source of truth. Run it after changing that
# file; the generated files are committed, not built on demand.
#
# Icons are only ever regenerated from the Windows host, so unlike the
# build scripts there's no bash counterpart to keep in step.

$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
Push-Location $root
try {
    go run ./cmd/icongen

    Write-Host "Regenerated from $root\assets\_source.png:"
    foreach ($icon in @(
        "assets\icon.png",
        "assets\icon-large.png",
        "assets\icon.ico",
        "assets\icon-white.png",
        "assets\icon-white.ico",
        "cmd\installer\icon.png",
        "cmd\installer\icon.ico"
    )) {
        Write-Host "  $icon"
    }
} finally {
    Pop-Location
}
