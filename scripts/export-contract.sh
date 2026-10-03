#!/usr/bin/env bash
# Runs the contract exporter against the fake API in the frontend checkout, then copies the
# goldens to testdata/contract/ and the seeds to seed/ (BACKEND_PLAN.md section 14).
# The exporter lives in tools/contract-export/ and is copied into the frontend's test/tool/ for
# the run; nothing is committed to the frontend.
set -euo pipefail
cd "$(dirname "$0")/.."
[ -f .env ] && set -a && . ./.env && set +a
FRONTEND_DIR="${FRONTEND_DIR:-../waraqah-frontend}"
[ -d "$FRONTEND_DIR/lib" ] || { echo "FRONTEND_DIR=$FRONTEND_DIR is not a frontend checkout"; exit 1; }

tool="$FRONTEND_DIR/test/tool"
mkdir -p "$tool"
cp tools/contract-export/*.dart "$tool/"
cleanup() {
  # flutter regenerates tracked plugin registrants; put them back so the frontend stays untouched.
  git -C "$FRONTEND_DIR" checkout -- '*generated_plugin*' '*GeneratedPluginRegistrant*' 2>/dev/null || true
  rm -f "$tool"/harness.dart "$tool"/samples_*.dart "$tool"/seeds*.dart "$tool"/export_contract_test.dart
  rmdir "$tool" 2>/dev/null || true
}
trap cleanup EXIT

(cd "$FRONTEND_DIR" && flutter test test/tool/export_contract_test.dart)

mkdir -p testdata/contract seed
rm -f testdata/contract/*.json
cp "$FRONTEND_DIR"/build/contract/*.json testdata/contract/
cp "$FRONTEND_DIR"/build/seed/*.json seed/
echo "goldens: $(ls testdata/contract/*.json | wc -l), seed files: $(ls seed/*.json | wc -l)"
