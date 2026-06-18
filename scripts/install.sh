#!/usr/bin/env sh
set -eu

die() {
	printf '%s\n' "jp-pii-detect install: $*" >&2
	exit 1
}

usage() {
	cat <<'USAGE'
Usage: scripts/install.sh [--version <version>] [--install-dir <dir>] [--repo <owner/repo>] [--print-url]

Downloads a prebuilt jp-pii-detect binary from GitHub Releases.

Environment:
  JP_PII_DETECT_VERSION           Release tag to install (default: latest)
  JP_PII_DETECT_INSTALL_DIR       Destination directory (default: $HOME/.local/bin)
  JP_PII_DETECT_REPO              GitHub repo (default: baneido/jp-pii-detecter)
  JP_PII_DETECT_RELEASE_BASE_URL  Override release base URL for tests/mirrors
  JP_PII_DETECT_OS                Override detected GOOS (linux/darwin/windows)
  JP_PII_DETECT_ARCH              Override detected GOARCH (amd64/arm64)
USAGE
}

normalize_os() {
	case "$1" in
		linux | Linux) printf 'linux' ;;
		darwin | Darwin) printf 'darwin' ;;
		windows | Windows_NT | MINGW* | MSYS* | CYGWIN*) printf 'windows' ;;
		*) die "unsupported OS: $1" ;;
	esac
}

normalize_arch() {
	case "$1" in
		amd64 | x86_64) printf 'amd64' ;;
		arm64 | aarch64) printf 'arm64' ;;
		*) die "unsupported architecture: $1" ;;
	esac
}

download() {
	url=$1
	output=$2
	if command -v curl >/dev/null 2>&1; then
		curl -fsSL "$url" -o "$output"
	elif command -v wget >/dev/null 2>&1; then
		wget -q "$url" -O "$output"
	else
		die "curl or wget is required to download release assets"
	fi
}

repo=${JP_PII_DETECT_REPO:-baneido/jp-pii-detecter}
version=${JP_PII_DETECT_VERSION:-latest}
install_dir=${JP_PII_DETECT_INSTALL_DIR:-}
print_url=0

while [ "$#" -gt 0 ]; do
	case "$1" in
		--version)
			[ "$#" -ge 2 ] || die "--version requires a value"
			version=$2
			shift 2
			;;
		--install-dir)
			[ "$#" -ge 2 ] || die "--install-dir requires a value"
			install_dir=$2
			shift 2
			;;
		--repo)
			[ "$#" -ge 2 ] || die "--repo requires a value"
			repo=$2
			shift 2
			;;
		--print-url)
			print_url=1
			shift
			;;
		-h | --help)
			usage
			exit 0
			;;
		*)
			die "unknown argument: $1"
			;;
	esac
done

if [ -z "$install_dir" ]; then
	install_dir="${HOME:-.}/.local/bin"
fi

goos=$(normalize_os "${JP_PII_DETECT_OS:-$(uname -s)}")
goarch=$(normalize_arch "${JP_PII_DETECT_ARCH:-$(uname -m)}")
asset="jp-pii-detect_${goos}_${goarch}.tar.gz"

if [ -n "${JP_PII_DETECT_RELEASE_BASE_URL:-}" ]; then
	base=${JP_PII_DETECT_RELEASE_BASE_URL%/}
	url="${base}/${version}/${asset}"
elif [ "$version" = "latest" ]; then
	url="https://github.com/${repo}/releases/latest/download/${asset}"
else
	url="https://github.com/${repo}/releases/download/${version}/${asset}"
fi

if [ "$print_url" -eq 1 ]; then
	printf '%s\n' "$url"
	exit 0
fi

bin_name=jp-pii-detect
if [ "$goos" = "windows" ]; then
	bin_name=jp-pii-detect.exe
fi

tmpdir=$(mktemp -d "${TMPDIR:-/tmp}/jp-pii-detect.XXXXXX")
cleanup() {
	rm -rf "$tmpdir"
}
trap cleanup EXIT HUP INT TERM

archive="${tmpdir}/${asset}"
download "$url" "$archive"
tar -xzf "$archive" -C "$tmpdir"

src="${tmpdir}/${bin_name}"
if [ ! -f "$src" ]; then
	src=$(find "$tmpdir" -type f -name "$bin_name" -print | head -n 1)
fi
[ -n "$src" ] && [ -f "$src" ] || die "archive did not contain ${bin_name}"

mkdir -p "$install_dir"
dest="${install_dir}/${bin_name}"
cp "$src" "$dest"
chmod 0755 "$dest"
printf 'Installed %s\n' "$dest"
