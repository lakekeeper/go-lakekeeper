#!/bin/sh
# Copy the newest Archify render of each diagram into docs/diagrams/.
#
# Usage: scripts/copy-diagrams.sh [slug ...]
#
# Archify writes one folder per render, .archify/<type>-<slug>-<YYYYMMDD-HHMMSS>/,
# where <type> is an Archify diagram type. For each slug, this copies the newest
# folder's files under stable names:
#
#   candidate.json     -> docs/diagrams/<slug>.json
#   <slug>.html        -> docs/diagrams/<slug>.html
#   <slug>-light.svg   -> docs/diagrams/<slug>-light.svg
#   <slug>-dark.svg    -> docs/diagrams/<slug>-dark.svg
#
# With slugs, it copies only those diagrams; without, every slug in .archify/.
# ARCHIFY_DIR and DIAGRAMS_DIR override the two folders.
#
# The SVGs come from scripts/export-diagram-svg.mjs. This script never runs
# Archify (scripts/render-diagrams.sh does). It checks every render it would
# copy first and copies nothing if one is missing or incomplete.
set -eu

src=${ARCHIFY_DIR:-.archify}
dst=${DIAGRAMS_DIR:-docs/diagrams}

# Archify's diagram types (see `archify --help`), longest first so the longest
# matching type wins. Keep in sync with ARCHIFY_TYPES in
# scripts/export-diagram-svg.mjs.
types="architecture waterfall lifecycle sequence dataflow workflow timeline class tree erd"

fail() {
	echo "copy-diagrams: $*" >&2
	exit 1
}

# render_files <slug> prints a render's files, one per line.
render_files() {
	printf '%s\n' candidate.json "$1.html" "$1-light.svg" "$1-dark.svg"
}

# parse <folder name> prints "<slug> <timestamp>": the name without its leading
# Archify type and trailing timestamp. It prints nothing for any other name.
parse() {
	stamp=$(printf '%s\n' "$1" | sed -n 's/^.*-\([0-9]\{8\}-[0-9]\{6\}\)$/\1/p')
	[ -n "$stamp" ] || return 0
	rest=${1%-"$stamp"}
	for type in $types; do
		slug=${rest#"$type"-}
		if [ "$slug" != "$rest" ] && [ -n "$slug" ]; then
			printf '%s %s\n' "$slug" "$stamp"
			return 0
		fi
	done
}

[ -d "$src" ] || fail "$src/ not found. It holds Archify renders, so this needs Archify; see docs/diagrams/README.md."

# One line per render folder: "<slug> <timestamp> <folder>". Sorting by slug,
# then timestamp, puts each slug's newest render last.
renders=$(
	for dir in "$src"/*/; do
		[ -d "$dir" ] || continue
		name=$(basename "$dir")
		parsed=$(parse "$name")
		if [ -n "$parsed" ]; then
			printf '%s %s\n' "$parsed" "$name"
		else
			echo "copy-diagrams: skipping $src/$name/: not <type>-<slug>-<YYYYMMDD-HHMMSS> with an Archify type" >&2
		fi
	done | sort -k1,1 -k2,2 | awk '{ newest[$1] = $3 } END { for (s in newest) print s, newest[s] }' | sort
)

[ -n "$renders" ] || fail "no <type>-<slug>-<YYYYMMDD-HHMMSS>/ render folders in $src/."

if [ $# -gt 0 ]; then
	selected=
	for want in "$@"; do
		line=$(printf '%s\n' "$renders" | awk -v s="$want" '$1 == s')
		[ -n "$line" ] || fail "no render of $want in $src/."
		selected="$selected$line
"
	done
	renders=$(printf '%s' "$selected" | sort -u)
fi

missing=$(
	printf '%s\n' "$renders" | while read -r slug name; do
		render_files "$slug" | while read -r f; do
			[ -f "$src/$name/$f" ] || printf '  %s\n' "$src/$name/$f"
		done
	done
)
[ -z "$missing" ] || fail "missing files (run Archify finalize, then node scripts/export-diagram-svg.mjs <folder>):
$missing"

mkdir -p "$dst"
printf '%s\n' "$renders" | while read -r slug name; do
	render_files "$slug" | while read -r f; do
		if [ "$f" = candidate.json ]; then target=$slug.json; else target=$f; fi
		cp "$src/$name/$f" "$dst/$target"
	done
	echo "$slug: copied from $src/$name"
done
