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

The easiest way is the installer — download it, run it, done: it installs
Snipp (or updates an existing install to the new version), registers it to
autostart at login, and starts it immediately. No prompts, no flags.

- Windows: https://github.com/tonybogdanov/snipp/releases/latest/download/snipp-installer.exe
  (installs to `%LOCALAPPDATA%\Snipp`, autostarts via the
  `HKCU\...\CurrentVersion\Run` registry key)
- Linux: https://github.com/tonybogdanov/snipp/releases/latest/download/snipp-installer
  (installs to `~/.local/bin`, autostarts via an XDG `~/.config/autostart`
  entry)

If you'd rather manage the binary yourself, plain (non-installing) builds
are also published:

- Windows: https://github.com/tonybogdanov/snipp/releases/latest/download/snipp.exe
- Linux: https://github.com/tonybogdanov/snipp/releases/latest/download/snipp

Or build from source:

```sh
git clone git@github.com:tonybogdanov/snipp.git
cd snipp
go build -o snipp .        # Linux
# or, on Windows:
./bin/build-windows.ps1    # -> artifacts/snipp.exe, artifacts/snipp-installer.exe
```

Run the resulting binary — it has no window, just a tray icon. Right-click
(or on Linux, however your DE surfaces the menu) for "Take Screenshot" and
"Quit".

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
