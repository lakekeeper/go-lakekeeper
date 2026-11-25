#!/usr/bin/env bash
# Report the breaking changes in each OpenAPI spec between a base ref and the
# working tree, as Markdown: one section per spec.
#
# Usage: OASDIFF="<oasdiff command>" scripts/spec-breaking-changes.sh <base-ref> <spec> ...
#
# Run it through `make spec-breaking-changes`, which sets OASDIFF to the pinned
# oasdiff and passes both specs. Spec paths are relative to the repository root
# and the working directory. A spec that is not on the base ref is reported as
# new. The script exits 0 whether or not there are breaking changes: it
# reports, it does not gate.
set -euo pipefail

: "${OASDIFF:?set OASDIFF to the oasdiff command, e.g. via make spec-breaking-changes}"

if (($# < 2)); then
	echo "usage: $0 <base-ref> <spec> ..." >&2
	exit 2
fi
base=$1
shift

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

for spec in "$@"; do
	echo "### \`$spec\`"
	echo
	if ! git cat-file -e "$base:$spec" 2>/dev/null; then
		echo "New: the spec is not on $base, so there is nothing to compare."
		echo
		continue
	fi
	git show "$base:$spec" >"$tmp/base.yaml"
	# `go run` maps every non-zero exit to 1, so the exit code cannot tell
	# "breaking changes found" from a failure; the JSON list can.
	# OASDIFF is a command line ("go run ...@version"), so split it into words.
	# shellcheck disable=SC2086
	changes=$($OASDIFF breaking --format json "$tmp/base.yaml" "$spec")
	if [[ $changes =~ ^[[:space:]]*\[\][[:space:]]*$ ]]; then
		echo "Compared with $base: no breaking changes."
		echo
		continue
	fi
	echo "Compared with $base:"
	# Drop oasdiff's "# API Changelog" title and nest its headings under the
	# spec's section.
	# shellcheck disable=SC2086
	$OASDIFF breaking --format markdown "$tmp/base.yaml" "$spec" |
		sed -e '/^# API Changelog/d' -e 's/^##/####/' |
		cat -s

done
