#!/bin/sh
# Copies honeybadger's generated OpenAPI bundle into openapi/ and records where
# it came from. Never edit openapi/bundled.yaml by hand.
set -eu

root=$(cd "$(dirname "$0")/.." && pwd)
# Resolve from the main checkout, so this works from a git worktree too.
main=$(dirname "$(git -C "$root" rev-parse --path-format=absolute --git-common-dir)")
hb=${HONEYBADGER_DIR:-$main/../../honeybadger}
src=$hb/openapi/v3/bundled.yaml

if [ ! -f "$src" ]; then
	echo "update-spec: $src not found. Run bin/rails openapi:bundle in honeybadger, or set HONEYBADGER_DIR." >&2
	exit 1
fi

# The bundle is a gitignored build artifact, so a pull or checkout in honeybadger
# doesn't rebuild it. A source newer than the bundle means the copy would be
# stale while recording the current commit as its source.
newer=$(find "$hb/openapi/v3" -name "*.yaml" ! -name bundled.yaml -newer "$src" | head -1)
if [ -n "$newer" ]; then
	echo "update-spec: $src is older than $newer. Run bin/rails openapi:bundle in honeybadger first." >&2
	exit 1
fi

sha256() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | cut -d' ' -f1
	else
		shasum -a 256 "$1" | cut -d' ' -f1
	fi
}

cp "$src" "$root/openapi/bundled.yaml"

branch=$(git -C "$hb" rev-parse --abbrev-ref HEAD)
commit=$(git -C "$hb" rev-parse --short=10 HEAD)
if [ -n "$(git -C "$hb" status --porcelain -- openapi/v3)" ]; then
	commit="$commit, plus uncommitted changes under openapi/v3"
fi
sha=$(sha256 "$root/openapi/bundled.yaml")

cat > "$root/openapi/README.md" <<EOF
# OpenAPI bundle

\`bundled.yaml\` is honeybadger's generated v3 bundle (\`bin/rails openapi:bundle\`),
copied by \`make update-spec\`. Don't edit it: fix the spec in honeybadger, then
copy the new bundle into every downstream project (api-go, docs, terraform, this server).

The sha256 identifies the bundle. In honeybadger the bundle is a generated,
gitignored file, so the branch and commit are honeybadger's checkout when the
bundle was copied, not a record of what built it. update-spec refuses a bundle
older than its sources, which catches the usual stale copy.

| | |
| --- | --- |
| Source | \`honeybadger/openapi/v3/bundled.yaml\` |
| Checkout when copied | \`$branch\` @ \`$commit\` |
| Copied | $(date +%Y-%m-%d) |
| sha256 | \`$sha\` |
EOF

echo "update-spec: copied $sha from $branch@$commit"
