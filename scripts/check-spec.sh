#!/bin/sh
# Checks that openapi/bundled.yaml is the bundle make update-spec copied from
# honeybadger: its sha256 must match the one recorded in openapi/README.md. A
# mismatch means the file was changed by hand.
#
# It doesn't compare with api-go's bundle. The generated code uses api-go's
# types by name, so a build against the api-go in go.mod (GOWORK=off) already
# fails on a field the two disagree on, and reworded spec text shouldn't need
# an api-go release.
set -eu

root=$(cd "$(dirname "$0")/.." && pwd)
bundle=$root/openapi/bundled.yaml
readme=$root/openapi/README.md

sha256() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | cut -d' ' -f1
	else
		shasum -a 256 "$1" | cut -d' ' -f1
	fi
}

if [ ! -f "$bundle" ]; then
	echo "check-spec: openapi/bundled.yaml is missing; run make update-spec" >&2
	exit 1
fi
want=$(sed -n 's/^| sha256 | `\([0-9a-f]\{64\}\)` |$/\1/p' "$readme")
if [ -z "$want" ]; then
	echo "check-spec: openapi/README.md records no sha256; run make update-spec" >&2
	exit 1
fi
got=$(sha256 "$bundle")

if [ "$got" != "$want" ]; then
	echo "check-spec: openapi/bundled.yaml ($got) isn't the bundle make update-spec recorded in openapi/README.md ($want)." >&2
	echo "Don't edit it by hand: change the spec in honeybadger, then run make update-spec." >&2
	exit 1
fi
echo "check-spec: bundle matches openapi/README.md ($got)"
