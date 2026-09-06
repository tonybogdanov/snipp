# Snipp

A tiny cross-platform screenshot app for Windows and Linux. It runs quietly
in the system tray — click the icon (or use the "Take Screenshot" menu
entry) to capture your full screen and save it as a PNG to
`~/Pictures/Snipp/`.

On Linux it captures via the X server directly under X11, and via the
xdg-desktop-portal Screenshot API under Wayland (GNOME, KDE, wlroots
compositors like Sway/Hyprland, and anything else implementing the portal
spec).

## Install

- Windows: https://github.com/tonybogdanov/snipp/releases/latest/download/snipp-installer.exe
- Linux: https://github.com/tonybogdanov/snipp/releases/latest/download/snipp-installer

Download and run in one step:

```powershell
curl.exe -Lo snipp-installer.exe https://github.com/tonybogdanov/snipp/releases/latest/download/snipp-installer.exe
.\snipp-installer.exe
```

```bash
curl -Lo snipp-installer https://github.com/tonybogdanov/snipp/releases/latest/download/snipp-installer && chmod +x snipp-installer && ./snipp-installer
```

Installs (or updates) Snipp, registers it to autostart at login, and starts
it right away — no prompts.

## Releases

Each push of an `X.Y.Z` tag builds both platforms and publishes a GitHub
release with the plain binaries and installers attached, marked as the
`latest` release. To cut one, run `bin/release.ps1` — it asks whether it's
a major, minor, or bugfix release, bumps the version accordingly, and
tags/pushes it.

## Requirements

- Windows: none beyond the OS itself.
- Linux: a tray host (most desktop environments provide one). Under
  Wayland, `zenity` or `notify-send` is used to show alerts if available.
