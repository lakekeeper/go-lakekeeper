#!/bin/sh
# Re-render diagrams from docs/diagrams/<slug>.json into docs/diagrams/.
#
# Usage: scripts/render-diagrams.sh [slug ...]
#
# Without slugs, it renders every docs/diagrams/*.json. For each slug it:
#
#   1. creates .archify/<diagram_type>-<slug>-<YYYYMMDD-HHMMSS>/
#   2. writes candidate.json there: docs/diagrams/<slug>.json with meta.output set
#      to that folder's <slug>.html and meta.repository.revision set to HEAD
#   3. runs `archify finalize`, which fails when a cited source file or line
#      range no longer exists at HEAD
#   4. exports the light and dark SVGs (scripts/export-diagram-svg.mjs)
#
# Then it copies the new renders into docs/diagrams/ (scripts/copy-diagrams.sh).
# Nothing is copied unless every slug rendered.
#
# Needs Node, Chrome and Archify. Set ARCHIFY to the path of Archify's
# bin/archify.mjs, or put an `archify` command on PATH.
set -eu

fail() {
	echo "render-diagrams: $*" >&2
	exit 1
}

root=$(git rev-parse --show-toplevel)
cd "$root"

if [ $# -eq 0 ]; then
	for f in docs/diagrams/*.json; do
		[ -f "$f" ] || continue
		slug=$(basename "$f" .json)
		set -- "$@" "$slug"
	done
	[ $# -gt 0 ] || fail "no docs/diagrams/*.json to render."
fi
for slug in "$@"; do
	[ -f "docs/diagrams/$slug.json" ] || fail "docs/diagrams/$slug.json not found; known diagrams: $(cd docs/diagrams && ls -- *.json 2>/dev/null | sed 's/\.json$//' | tr '\n' ' ')"
done

command -v node >/dev/null 2>&1 || fail "node not found. Rendering needs Node, Chrome and Archify."

archify_help="Set ARCHIFY to the path of Archify's bin/archify.mjs (for example inside the Archify Claude skill), or put an \`archify\` command on PATH. Rendering also needs Chrome or Chromium. See https://github.com/tt-a1i/archify."
if [ -n "${ARCHIFY:-}" ]; then
	[ -f "$ARCHIFY" ] || fail "ARCHIFY=$ARCHIFY is not a file. $archify_help"
	archify() { node "$ARCHIFY" "$@"; }
elif ! command -v archify >/dev/null 2>&1; then
	fail "Archify not found. $archify_help"
fi

revision=$(git rev-parse HEAD)

# The render cites HEAD, so uncommitted edits to cited files are not what it checks.
for slug in "$@"; do
	dirty=$(node -e '
		const fs = require("fs");
		const paths = new Set();
		const walk = (v) => {
			if (Array.isArray(v)) return v.forEach(walk);
			if (v && typeof v === "object") {
				if (Array.isArray(v.sources)) v.sources.forEach((s) => s && s.path && paths.add(s.path));
				Object.values(v).forEach(walk);
			}
		};
		walk(JSON.parse(fs.readFileSync(process.argv[1], "utf8")));
		console.log([...paths].join("\n"));
	' "docs/diagrams/$slug.json" | while read -r path; do
		[ -n "$path" ] || continue
		git status --porcelain -- "$path"
	done)
	if [ -n "$dirty" ]; then
		echo "render-diagrams: warning: $slug cites files with uncommitted changes; the render checks HEAD ($revision), not your edits:" >&2
		printf '%s\n' "$dirty" | sed 's/^/  /' >&2
	fi
done

for slug in "$@"; do
	type=$(node -e 'console.log(JSON.parse(require("fs").readFileSync(process.argv[1], "utf8")).diagram_type || "")' "docs/diagrams/$slug.json")
	[ -n "$type" ] || fail "docs/diagrams/$slug.json has no diagram_type."
	dir=.archify/$type-$slug-$(date +%Y%m%d-%H%M%S)
	mkdir -p "$dir"
	# Change only the two values in place so the JSON keeps its hand formatting;
	# fall back to re-serialising when either key is missing.
	node -e '
		const fs = require("fs");
		const assert = require("assert");
		const [src, dst, output, revision] = process.argv.slice(1);
		const text = fs.readFileSync(src, "utf8");
		const want = JSON.parse(text);
		want.meta = want.meta || {};
		want.meta.output = output;
		want.meta.repository = want.meta.repository || {};
		want.meta.repository.revision = revision;
		const set = (t, key, value) => t.replace(new RegExp(`("${key}"\\s*:\\s*)"(?:[^"\\\\]|\\\\.)*"`), (_, k) => k + JSON.stringify(value));
		let out = set(set(text, "output", output), "revision", revision);
		try {
			assert.deepStrictEqual(JSON.parse(out), want);
		} catch {
			out = JSON.stringify(want, null, 2) + "\n";
		}
		fs.writeFileSync(dst, out);
	' "docs/diagrams/$slug.json" "$dir/candidate.json" "$dir/$slug.html" "$revision"
	echo "$slug: rendering into $dir"
	archify finalize "$type" "$dir/candidate.json" "$dir/$slug.html" --repo-root "$root" --quality showcase ||
		fail "archify finalize failed for $slug; fix docs/diagrams/$slug.json and rerun."
	node scripts/export-diagram-svg.mjs "$dir"
done

./scripts/copy-diagrams.sh "$@"
