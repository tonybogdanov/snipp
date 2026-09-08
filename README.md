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

## Updating

Use the tray menu's **Update Snipp** entry: it compares the commit hash the
running build was made from against the latest release's, and if they
differ it downloads that release's installer and hands over to it — the
installer replaces the binary and restarts Snipp. If you're already on the
latest build, or GitHub can't be reached, it says so and changes nothing.

Re-running the installer does the same thing.

## Releases

Every push to `master` builds both platforms and publishes a GitHub release
named after the short commit hash and marked as the `latest` release, with
the installers *and* the app binaries attached. The installer embeds no
payload: it downloads `snipp`/`snipp.exe` from the latest release when it
runs, so an installer saved months ago still installs the current build.
The same short hash is stamped into the app at build time as its version,
which is what the updater compares.

## Requirements

- Windows: none beyond the OS itself.
- Linux: a tray host (most desktop environments provide one). Under
  Wayland, `zenity` or `notify-send` is used to show alerts if available.
