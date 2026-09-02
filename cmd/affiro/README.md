# affiro CLI

The `affiro` CLI program offers monitoring functionality to produce signatures based on user input.

Usage: `affiro monitor [-gui]`

The CLI also offers a (to be removed) checking functionality: `affiro check -i input.txt -s <signature>`.

`monitor`'s `-gui` mode works on OSX, Windows, and Linux (X11). Non-GUI monitoring works on Linux (Wayland);
see "Linux Wayland setup" below for the one-time step Wayland needs.

## Linux Wayland setup

Under Wayland, `monitor` captures keystrokes by reading raw keyboard input from `/dev/input/event*` directly (there's no
portal API that can deliver a stream of ordinary keystrokes). Those device nodes aren't readable by an ordinary user
by default, so run this once, as root, before using `monitor` on a Wayland session:

```bash
sudo ./scripts/install-udev-rule.sh
```

Then log out and back in (or reboot) so your current session picks up the new access grant.
X11 sessions need no such setup.

## Publishing

Release process TBD for this repo.
