#!/bin/sh
# Installs the latest rd release:
#
#   curl -fsSL https://raw.githubusercontent.com/alileza/rd/main/install.sh | sh
#
# Environment:
#   RD_VERSION      release tag to install, e.g. v0.1.0 (default: latest)
#   RD_INSTALL_DIR  directory to install into (default: /usr/local/bin)
set -eu

repo="alileza/rd"

# Wrapped in a function so a partially downloaded script never runs.
main() {
	version="${RD_VERSION:-latest}"
	dir="${RD_INSTALL_DIR:-/usr/local/bin}"

	os=$(uname -s | tr '[:upper:]' '[:lower:]')
	case "$os" in
	darwin | linux) ;;
	*) fail "unsupported OS: $os" ;;
	esac

	arch=$(uname -m)
	case "$arch" in
	x86_64 | amd64) arch=amd64 ;;
	arm64 | aarch64) arch=arm64 ;;
	*) fail "unsupported architecture: $arch" ;;
	esac

	if [ "$version" = latest ]; then
		base="https://github.com/$repo/releases/latest/download"
	else
		base="https://github.com/$repo/releases/download/$version"
	fi

	tmp=$(mktemp -d)
	trap 'rm -rf "$tmp"' EXIT

	asset="rd_${os}_${arch}"
	echo "Downloading $asset ($version)..."
	curl -fsSL -o "$tmp/$asset" "$base/$asset" || fail "download failed: $base/$asset"
	curl -fsSL -o "$tmp/checksums.txt" "$base/checksums.txt" || fail "download failed: $base/checksums.txt"

	want=$(grep " $asset\$" "$tmp/checksums.txt" | cut -d' ' -f1)
	if command -v sha256sum >/dev/null 2>&1; then
		got=$(sha256sum "$tmp/$asset" | cut -d' ' -f1)
	else
		got=$(shasum -a 256 "$tmp/$asset" | cut -d' ' -f1)
	fi
	if [ -z "$want" ] || [ "$want" != "$got" ]; then
		fail "checksum mismatch for $asset"
	fi

	sudo=""
	if ! mkdir -p "$dir" 2>/dev/null || [ ! -w "$dir" ]; then
		echo "Installing to $dir requires sudo (set RD_INSTALL_DIR to pick another directory)."
		sudo="sudo"
	fi
	$sudo mkdir -p "$dir"
	$sudo install -m 0755 "$tmp/$asset" "$dir/rd"

	echo "Installed $("$dir/rd" -version) to $dir/rd"
	# shellcheck disable=SC2016 # $PATH is meant to be printed literally.
	case ":$PATH:" in
	*":$dir:"*) ;;
	*) printf '\n%s is not on your PATH. Add it with:\n  export PATH="%s:$PATH"\n' "$dir" "$dir" ;;
	esac
}

fail() {
	echo "rd install: $*" >&2
	exit 1
}

main "$@"
