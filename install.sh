#!/bin/sh

set -eu

if [ "$(uname -s)" != "Darwin" ]; then
	printf '%s\n' 'psfax can only be installed on macOS.' >&2
	exit 1
fi

repo="jk779/psfax"
install_dir="${HOME}/.local/bin"
destination="${install_dir}/psfax"
temporary_file="$(mktemp "${TMPDIR:-/tmp}/psfax.XXXXXX")"

cleanup() {
	rm -f "$temporary_file"
}
trap cleanup EXIT INT TERM

mkdir -p "$install_dir"
curl --fail --location --retry 3 --silent --show-error \
	"https://github.com/${repo}/releases/latest/download/psfax" \
	-o "$temporary_file"
chmod 755 "$temporary_file"
mv "$temporary_file" "$destination"

printf 'Installed psfax to %s\n' "$destination"
case ":${PATH}:" in
	*:"${install_dir}":*) ;;
	*) printf 'Add %s to PATH to run psfax from any shell.\n' "$install_dir" ;;
esac
