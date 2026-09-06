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

Grab the latest build for your platform from the
[Actions](https://github.com/tonybogdanov/snipp/actions) tab (each run
uploads `snipp-windows` and `snipp-linux` artifacts), or build it yourself:

```sh
git clone git@github.com:tonybogdanov/snipp.git
cd snipp
go build -o snipp .        # Linux
# or, on Windows:
./bin/build-windows.ps1    # -> artifacts/snipp.exe
```

Run the resulting binary — it has no window, just a tray icon. Right-click
(or on Linux, however your DE surfaces the menu) for "Take Screenshot" and
"Quit".

## Releases

Download the latest build directly:

- Windows: https://github.com/tonybogdanov/snipp/releases/latest/download/snipp.exe
- Linux: https://github.com/tonybogdanov/snipp/releases/latest/download/snipp

Each push of an `X.Y.Z` tag builds both platforms and publishes a GitHub
release with those files attached, and marks it as the `latest` release.
To cut one, run `bin/release.ps1` — it asks whether it's a major, minor,
or bugfix release, bumps the version accordingly, and tags/pushes it.

## Requirements

- Windows: none beyond the OS itself.
- Linux: a tray host (most desktop environments provide one). Under
  Wayland, `zenity` or `notify-send` is used to show alerts if available.
