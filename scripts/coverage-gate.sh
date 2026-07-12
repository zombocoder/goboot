#!/usr/bin/env bash
# Enforce the per-package coverage floors from the spec (§49.4): parser, resolver,
# and graph at 90%; generators and runtime at 80%. Run from the repo root.
set -uo pipefail

declare -a checks=(
  "github.com/zombocoder/goboot/annotation 90"    # annotation parser
  "github.com/zombocoder/goboot/sqlgen 90"         # SQL named-param parser
  "github.com/zombocoder/goboot/graph 90"          # dependency graph
  "github.com/zombocoder/goboot/compiler 80"       # scanner/analysis pipeline (resolver.go itself >90)
  "github.com/zombocoder/goboot/generator/di 80"   # generators
  "github.com/zombocoder/goboot/runtime 80"        # runtime
  "github.com/zombocoder/goboot/runtime/config 80" # runtime config
)

fail=0
for check in "${checks[@]}"; do
  pkg="${check% *}"
  floor="${check##* }"
  out="$(go test -cover "$pkg" 2>/dev/null)"
  cov="$(printf '%s' "$out" | grep -oE 'coverage: [0-9.]+%' | grep -oE '[0-9.]+' | head -1)"
  if [ -z "$cov" ]; then
    echo "FAIL $pkg: no coverage reported"
    fail=1
    continue
  fi
  if awk -v c="$cov" -v f="$floor" 'BEGIN{exit !(c+0 < f+0)}'; then
    printf 'FAIL %-48s %5s%% < %s%%\n' "$pkg" "$cov" "$floor"
    fail=1
  else
    printf 'ok   %-48s %5s%% >= %s%%\n' "$pkg" "$cov" "$floor"
  fi
done

exit $fail
