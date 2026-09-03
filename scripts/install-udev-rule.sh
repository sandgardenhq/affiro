#!/usr/bin/env bash
# scripts/install-udev-rule.sh
#
# Installs the udev rule that lets affiro's `monitor` command read raw keyboard input under
# Wayland (see packaging/udev/70-affiro-input.rules and cmd/affiro/README.md). Run once, with
# root privileges.
set -euo pipefail

if [[ "$(id -u)" -ne 0 ]]; then
	echo "error: this script must be run as root (it installs a rule under /usr/lib/udev/rules.d)" >&2
	exit 1
fi

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
rule_src="${script_dir}/../packaging/udev/70-affiro-input.rules"
rule_dest="/usr/lib/udev/rules.d/70-affiro-input.rules"

if [[ ! -f "${rule_src}" ]]; then
	echo "error: expected udev rule at ${rule_src}, not found" >&2
	exit 1
fi

cp "${rule_src}" "${rule_dest}"
udevadm control --reload-rules
udevadm trigger

cat <<'EOF'
Installed the affiro keyboard-input udev rule.

If you are already logged in, log out and back in (or reboot) so your current desktop session
picks up the new access grant -- the rule only applies to sessions started after it's installed.
EOF
