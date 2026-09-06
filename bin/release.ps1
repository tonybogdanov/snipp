#!/usr/bin/env pwsh
# Bumps the semver tag (X.Y.Z), then creates and pushes it. The build
# workflow reacts to the new tag and publishes a GitHub release with the
# built artifacts, updating the "latest" release pointer.

$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
Push-Location $root
try {
    $status = git status --porcelain
    if ($status) {
        throw "Working tree is not clean. Commit or stash your changes first."
    }

    $branch = git rev-parse --abbrev-ref HEAD
    if ($branch -ne "master") {
        throw "Releases must be cut from master (currently on '$branch')."
    }

    git fetch --tags --quiet

    $tags = git tag --list | Where-Object { $_ -match '^\d+\.\d+\.\d+$' }
    $latest = [version]"0.0.0"
    foreach ($t in $tags) {
        $v = [version]$t
        if ($v -gt $latest) { $latest = $v }
    }
    Write-Host "Current version: $latest"

    $kind = $null
    while ($kind -notin @("major", "minor", "patch")) {
        $kind = Read-Host "Release type (major/minor/patch)"
    }

    switch ($kind) {
        "major" { $next = [version]::new($latest.Major + 1, 0, 0) }
        "minor" { $next = [version]::new($latest.Major, $latest.Minor + 1, 0) }
        "patch" { $next = [version]::new($latest.Major, $latest.Minor, $latest.Build + 1) }
    }
    $nextStr = "$($next.Major).$($next.Minor).$($next.Build)"

    git tag -a $nextStr -m "Release $nextStr"
    git push origin $nextStr
    Write-Host "Pushed tag $nextStr"
} finally {
    Pop-Location
}
