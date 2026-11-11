#!/usr/bin/env bash
# Renders a Go coverage profile as a Markdown table of statement coverage per
# package, plus a total row. Usage: coverage-summary.sh <profile>

set -euo pipefail

profile="${1:?usage: $0 <coverage profile>}"
module="$(go list -m)"

echo "## Unit test coverage"
echo
echo "Generated code under \`pkg/apis/management/v1/\` is excluded."
echo
echo "| Package | Statements covered | Coverage |"
echo "| --- | ---: | ---: |"

awk -v module="${module}/" '
NR == 1 && /^mode:/ { next }
{
    # Line format: <file>:<startLine>.<startCol>,<endLine>.<endCol> <numStmts> <count>
    # A block can appear more than once (e.g. with -coverpkg); count it once,
    # as covered if any occurrence was hit.
    block = $1
    if (!(block in stmts)) {
        stmts[block] = $2
        dir = substr(block, 1, index(block, ":") - 1)
        sub(/\/[^\/]*$/, "", dir)
        if (index(dir, module) == 1) dir = substr(dir, length(module) + 1)
        pkg[block] = dir
    }
    if ($3 > 0) hit[block] = 1
}
END {
    for (b in stmts) {
        p = pkg[b]
        total[p] += stmts[b]
        all += stmts[b]
        if (b in hit) { covered[p] += stmts[b]; allCovered += stmts[b] }
    }
    # Prefix each row with a sort key so packages sort by name and the total
    # row lands last; the key is stripped below.
    for (p in total) {
        printf "0 %s\t| `%s` | %d / %d | %.1f%% |\n", p, p, covered[p], total[p], 100 * covered[p] / total[p]
    }
    printf "1\t| **total** | **%d / %d** | **%.1f%%** |\n", allCovered, all, (all ? 100 * allCovered / all : 0)
}
' "${profile}" | sort | cut -f2-
