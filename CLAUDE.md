# snipp

Cross-platform screenshot app, written in Go. Targets Windows and most
mainstream Linux distros.

## Cross-platform parity

Any change to how the app behaves must be implemented on both Windows and
Linux — and on Linux, across its variants (X11 and Wayland; with and
without zenity available; GNOME/gsettings-based desktops and others).
Never land a behavior change on one platform only.

## Git commits

Never add a `Co-Authored-By` (or any co-author) line to commit messages
in this repo.

Use Conventional Commits prefixes (`feat: `, `fix: `, `chore: `, etc.),
lower-case throughout — including the subject after the prefix.
