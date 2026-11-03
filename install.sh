#!/bin/sh
# Install omut from its GitHub releases on Linux or macOS (WSL included).
#
#   curl -fsSL https://raw.githubusercontent.com/metravod/omut/main/install.sh | sh
#
# OMUT_VERSION=0.4.1   a release other than the latest
# OMUT_DIR=/usr/local/bin   where to put the binary (default: ~/.local/bin)
set -eu

repo=metravod/omut
dir=${OMUT_DIR:-"$HOME/.local/bin"}

say() { printf 'omut install: %s\n' "$*" >&2; }
die() { say "$*"; exit 1; }

if command -v curl >/dev/null 2>&1; then
	fetch() { curl -fsSL "$1" -o "$2"; }
	latest() { curl -fsSI "https://github.com/$repo/releases/latest" | tr -d '\r' | sed -n 's#^[Ll]ocation: .*/tag/v##p'; }
elif command -v wget >/dev/null 2>&1; then
	fetch() { wget -qO "$2" "$1"; }
	latest() { wget -qO- "https://api.github.com/repos/$repo/releases/latest" | sed -n 's#.*"tag_name": *"v\([^"]*\)".*#\1#p'; }
else
	die "needs curl or wget"
fi

case $(uname -s) in
Linux) os=linux ;;
Darwin) os=darwin ;;
*) die "$(uname -s) is not supported here; take a binary from https://github.com/$repo/releases" ;;
esac
case $(uname -m) in
x86_64 | amd64) arch=amd64 ;;
arm64 | aarch64) arch=arm64 ;;
*) die "$(uname -m) is not supported; build from source: go install github.com/$repo/cmd/omut@latest" ;;
esac

version=${OMUT_VERSION:-$(latest)}
version=${version#v}
[ -n "$version" ] || die "could not find the latest release"

archive="omut_${version}_${os}_${arch}.tar.gz"
base="https://github.com/$repo/releases/download/v$version"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

say "downloading omut $version for $os/$arch"
fetch "$base/$archive" "$tmp/$archive" || die "no $archive in release v$version"
fetch "$base/checksums.txt" "$tmp/checksums.txt" || die "no checksums.txt in release v$version"

want=$(awk -v f="$archive" '$2 == f {print $1}' "$tmp/checksums.txt")
if command -v sha256sum >/dev/null 2>&1; then
	got=$(sha256sum "$tmp/$archive" | awk '{print $1}')
else
	got=$(shasum -a 256 "$tmp/$archive" | awk '{print $1}')
fi
[ -n "$want" ] && [ "$want" = "$got" ] || die "checksum mismatch for $archive"

tar -xzf "$tmp/$archive" -C "$tmp" omut
mkdir -p "$dir"
mv "$tmp/omut" "$dir/omut"
chmod 755 "$dir/omut"
say "installed $("$dir/omut" --version) to $dir/omut"

case ":$PATH:" in
*":$dir:"*) ;;
*)
	case ${SHELL:-sh} in
	*/zsh | zsh) rc='~/.zshrc' ;;
	*/bash | bash) rc='~/.bashrc' ;;
	*) rc='~/.profile' ;;
	esac
	say "$dir is not in PATH; add it: echo 'export PATH=\"$dir:\$PATH\"' >> $rc"
	;;
esac
