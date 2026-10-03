#!/usr/bin/env bash
# Curls a handful of endpoints against API_BASE_URL (T19 extends this).
set -eu
BASE="${API_BASE_URL:-http://localhost:8080/v1}"
ROOT="${BASE%/v1}"
for p in /healthz /readyz /version; do
  code=$(curl -s -o /dev/null -w '%{http_code}' "$ROOT$p")
  echo "$p -> $code"
  [ "$code" = "200" ] || exit 1
done
echo "smoke ok"
