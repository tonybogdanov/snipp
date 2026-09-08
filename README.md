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

Use the tray menu's **Check for Updates** entry: it compares the commit hash
the running build was made from against the latest release's, and if they
differ it downloads that release's installer and hands over to it — the
installer replaces the binary and restarts Snipp. If you're already on the
latest build, or GitHub can't be reached, it says so and changes nothing.

Updating is the app's job, not the installer's: an installer only ever
installs the build it carries, so it needs no network and does the same
thing every time it runs.

## Releases

Every push to `master` builds both platforms and publishes a GitHub release
with the installers attached, named after the short commit hash and marked
as the `latest` release. That same short hash is stamped into the app at
build time as its version, which is what the updater compares.

Only that one release is kept: publishing a new one deletes every earlier
release and its tag, the same way old workflow runs and artifacts are
pruned. There is no download of an older build — the install and update
links always point at the current one.

## Requirements

- Windows: none beyond the OS itself.
- Linux: a tray host (most desktop environments provide one). Under
  Wayland, `zenity` or `notify-send` is used to show alerts if available.
