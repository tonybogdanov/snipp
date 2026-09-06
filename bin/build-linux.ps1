#!/usr/bin/env pwsh
# Cross-compiles the Snipp Linux binary and its installer locally, mirroring
# the linux job in .github/workflows/build.yml. No Linux toolchain needed
# (pure Go, no cgo).

$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
Push-Location $root
try {
    New-Item -ItemType Directory -Force -Path artifacts | Out-Null
    $env:GOOS = "linux"
    $env:GOARCH = "amd64"
    $env:CGO_ENABLED = "0"
    go build -o artifacts/snipp .
    Write-Host "Built $root\artifacts\snipp"

    $payload = "cmd/installer/payload/linux.bin"
    Copy-Item artifacts/snipp $payload -Force
    try {
        go build -o artifacts/snipp-installer ./cmd/installer
        Write-Host "Built $root\artifacts\snipp-installer"
    } finally {
        git checkout -- $payload
    }
} finally {
    Remove-Item Env:\GOOS, Env:\GOARCH, Env:\CGO_ENABLED -ErrorAction SilentlyContinue
    Pop-Location
}
