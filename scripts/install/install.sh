#!/bin/sh
# Installs the 'bild' command line tool on macOS or Linux from a GitHub release
# of shdeen/bildomat, after verifying the download against the release's
# SHA-256 checksums file.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/shdeen/bildomat/main/scripts/install/install.sh | sh
#
# Environment variables:
#   BILD_VERSION      release to install, such as 0.1.0 (default: the latest release)
#   BILD_INSTALL_DIR  directory to install bild into (default: $HOME/.local/bin)

set -eu

releases_url="https://github.com/shdeen/bildomat/releases"
latest_release_api_url="https://api.github.com/repos/shdeen/bildomat/releases/latest"

print_notice() {
	printf 'bild install: %s\n' "$*"
}

print_warning() {
	printf 'Warning: %s\n' "$*" >&2
}

abort_install() {
	printf 'Error: %s\n' "$*" >&2
	exit 1
}

detect_os() {
	case "$(uname -s)" in
	Darwin) echo darwin ;;
	Linux) echo linux ;;
	MINGW* | MSYS* | CYGWIN*) abort_install "This script is for macOS and Linux. On Windows, run install.ps1 in PowerShell." ;;
	*) abort_install "No bild release exists for this operating system ($(uname -s))." ;;
	esac
}

detect_arch() {
	machine_arch="$(uname -m)"
	# A shell running under Rosetta 2 on an Apple silicon Mac reports x86_64.
	# The sysctl key exists only on macOS 11 or later, so its "unknown oid"
	# error elsewhere is expected and discarded.
	if [ "$machine_arch" = x86_64 ] && [ "$(sysctl -n sysctl.proc_translated 2>/dev/null || true)" = 1 ]; then
		machine_arch=arm64
	fi
	case "$machine_arch" in
	x86_64 | amd64) echo amd64 ;;
	arm64 | aarch64) echo arm64 ;;
	*) abort_install "No bild release exists for this processor ($machine_arch)." ;;
	esac
}

# download_file writes the file at the URL in $1 to the path in $2, or to standard
# output when $2 is "-".
download_file() {
	if command -v curl >/dev/null 2>&1; then
		curl -fsSL -o "$2" "$1"
	else
		wget -q -O "$2" "$1"
	fi
}

hash_file() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | cut -d ' ' -f 1
	elif command -v shasum >/dev/null 2>&1; then
		shasum -a 256 "$1" | cut -d ' ' -f 1
	else
		abort_install "Unable to verify the download without running sha256sum or shasum, but neither is installed. Install one of them first, then run the installer again."
	fi
}

target_os="$(detect_os)"
target_arch="$(detect_arch)"
install_dir="${BILD_INSTALL_DIR:-$HOME/.local/bin}"
command -v curl >/dev/null 2>&1 || command -v wget >/dev/null 2>&1 || abort_install "Unable to download without curl or wget; install one of them first, then run the installer again."

if [ -n "${BILD_VERSION:-}" ]; then
	release_version="${BILD_VERSION#v}"
else
	release_version="$(download_file "$latest_release_api_url" - | sed -n 's/.*"tag_name": *"v\([^"]*\)".*/\1/p' | head -n 1)"
	[ -n "$release_version" ] || abort_install "Unable to look up the latest release at $releases_url"
fi

archive_name="bild_${release_version}_${target_os}_${target_arch}.tar.gz"
checksums_name="bild_${release_version}_checksums.txt"
download_url="$releases_url/download/v$release_version"

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT
trap 'exit 1' INT TERM

print_notice "downloading bild $release_version for $target_os/$target_arch"
download_file "$download_url/$archive_name" "$tmp_dir/$archive_name" ||
	abort_install "Unable to download $download_url/$archive_name"
download_file "$download_url/$checksums_name" "$tmp_dir/$checksums_name" ||
	abort_install "Unable to download $download_url/$checksums_name"

expected_sha256="$(awk -v name="$archive_name" '$2 == name { print $1 }' "$tmp_dir/$checksums_name")"
[ -n "$expected_sha256" ] || abort_install "Unable to verify $archive_name because the release has no checksum for it. Nothing was installed."
archive_sha256="$(hash_file "$tmp_dir/$archive_name")"
[ "$archive_sha256" = "$expected_sha256" ] ||
	abort_install "$archive_name failed verification and may be corrupted. Nothing was installed."

tar -xzf "$tmp_dir/$archive_name" -C "$tmp_dir" bild
mkdir -p "$install_dir"
# Moving gives the installed file a new inode. Copying over an existing signed
# binary on macOS can leave the kernel's cached signature attached to the new
# bytes, and the system then kills the program on launch.
mv -f "$tmp_dir/bild" "$install_dir/bild"
chmod 755 "$install_dir/bild"

installed_version="$("$install_dir/bild" --version)" ||
	abort_install "bild was installed at $install_dir/bild, but could not be run; try running it directly to see if the problem persists or for more detailed error info, then try correcting the issue and run the installer again."
print_notice "installed $installed_version at $install_dir/bild"

case ":$PATH:" in
*":$install_dir:"*)
	active_bild_path="$(command -v bild || true)"
	if [ "$active_bild_path" != "$install_dir/bild" ]; then
		print_warning "Running \`bild\` without an explicit path would run a different copy of \`bild\`, at '$active_bild_path', since its path is earlier in the PATH environment variable. To use the current \`bild\` installation, remove that copy from your system or move '$install_dir' ahead of '$(dirname "$active_bild_path")' on your PATH."
	fi
	;;
*)
	print_notice "$install_dir is not on your PATH. To run bild by name, add the following line to your shell startup file, such as ~/.zshrc or ~/.bashrc:"
	# shellcheck disable=SC2016 # $PATH must print literally, for the user's startup file.
	printf '  export PATH="%s:$PATH"\n' "$install_dir"
	;;
esac
