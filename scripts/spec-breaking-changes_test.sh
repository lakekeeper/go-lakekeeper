#!/usr/bin/env bash
# Tests scripts/spec-breaking-changes.sh against fixture specs in a throwaway
# git repository. Run it through `make test-spec-breaking-changes`, which sets
# OASDIFF to the pinned oasdiff.
set -euo pipefail

: "${OASDIFF:?set OASDIFF to the oasdiff command, e.g. via make test-spec-breaking-changes}"

script=$(cd "$(dirname "$0")" && pwd)/spec-breaking-changes.sh
repo=$(mktemp -d)
trap 'rm -rf "$repo"' EXIT

failures=0

# expect <name> <pattern> <output>: the output contains the pattern.
expect() {
	if grep -qF -- "$2" <<<"$3"; then
		echo "ok   $1"
	else
		echo "FAIL $1: output lacks '$2'. Output:"
		sed 's/^/     /' <<<"$3"
		failures=$((failures + 1))
	fi
}

# reject <name> <pattern> <output>: the output does not contain the pattern.
reject() {
	if grep -qF -- "$2" <<<"$3"; then
		echo "FAIL $1: output has '$2'. Output:"
		sed 's/^/     /' <<<"$3"
		failures=$((failures + 1))
	else
		echo "ok   $1"
	fi
}

# spec <title> <properties...>: an OpenAPI spec whose one operation returns a
# Widget with the given properties.
spec() {
	local title=$1
	shift
	cat <<EOF
openapi: 3.1.0
info:
  title: $title
  version: 1.0.0
paths:
  /widget:
    get:
      operationId: getWidget
      responses:
        '200':
          description: The widget.
          content:
            application/json:
              schema:
                \$ref: '#/components/schemas/Widget'
components:
  schemas:
    Widget:
      type: object
      required: [$(
		IFS=,
		echo "$*"
	)]
      properties:
EOF
	for p in "$@"; do
		printf '        %s:\n          type: string\n' "$p"
	done
}

cd "$repo"
git init -q
git config user.email test@example.com
git config user.name test
mkdir specs
spec A name color >specs/a.yaml
spec B name size >specs/b.yaml
git add specs
git commit -qm base

out=$("$script" HEAD specs/a.yaml specs/b.yaml)
expect "unchanged spec: no breaking changes" "no breaking changes" "$out"
expect "report names each spec" "specs/a.yaml" "$out"
expect "report names each spec" "specs/b.yaml" "$out"

spec A name >specs/a.yaml
out=$("$script" HEAD specs/a.yaml specs/b.yaml)
a_section=$(sed -n '/specs\/a.yaml/,/specs\/b.yaml/p' <<<"$out")
b_section=$(sed -n '/specs\/b.yaml/,$p' <<<"$out")
expect "removed property is listed" "color" "$a_section"
reject "removed property: spec is not reported clean" "no breaking changes" "$a_section"
expect "other spec reported on its own" "no breaking changes" "$b_section"
reject "other spec does not inherit the change" "color" "$b_section"

spec C name >specs/c.yaml
out=$("$script" HEAD specs/c.yaml)
expect "spec missing on the base is reported as new" "not on HEAD" "$out"

if ((failures > 0)); then
	echo "$failures check(s) failed"
	exit 1
fi
echo "all checks passed"
