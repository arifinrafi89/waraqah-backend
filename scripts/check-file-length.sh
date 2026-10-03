#!/usr/bin/env bash
# Fails when a hand-written .go file is over 300 lines (AGENTS.md rule 7).
set -u
limit=300
bad=0
while IFS= read -r f; do
  if head -n 5 "$f" | grep -q "Code generated"; then continue; fi
  n=$(wc -l < "$f")
  if [ "$n" -gt "$limit" ]; then
    echo "$f has $n lines (limit $limit)"
    bad=1
  fi
done < <(find . -name '*.go' -not -path './.git/*' -not -path './internal/platform/db/sqlc/*')
exit $bad
