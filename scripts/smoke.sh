#!/usr/bin/env bash
# Smoke test of a running backend (plan §22.4, T19): the health endpoints, the public reads, then
# sign in as the demo reader (reader@waraqah.test / SEED_DEMO_PASSWORD) and call one endpoint per
# feature, and one staff endpoint as admin@. Every call must answer 200 with JSON that is not an
# error. Usage: make smoke [API_BASE_URL=https://<service>.onrender.com/v1]
set -euo pipefail
BASE="${API_BASE_URL:-http://localhost:8080/v1}"
ROOT="${BASE%/v1}"
PASSWORD="${SEED_DEMO_PASSWORD:?set SEED_DEMO_PASSWORD, the password of the demo accounts}"
fails=0
nl=$'\n'

check() { # check <label> <status> <body>
  if [ "$2" = "200" ] && ! grep -q '"error"' <<<"$3"; then
    echo "ok   $1"
  else
    echo "FAIL $1 -> $2 $(head -c 200 <<<"$3")"
    fails=$((fails + 1))
  fi
}

get() { # get <path> [token]
  local out
  out=$(curl -s -w '\n%{http_code}' ${2:+-H "Authorization: Bearer $2"} "$BASE$1")
  check "GET $1" "${out##*$nl}" "${out%$nl*}"
}

post() { # post <path> <json> [token]
  local out
  out=$(curl -s -w '\n%{http_code}' -X POST -H 'Content-Type: application/json' ${3:+-H "Authorization: Bearer $3"} -d "$2" "$BASE$1")
  check "POST $1" "${out##*$nl}" "${out%$nl*}"
}

login() { # login <email>: prints the access token
  curl -s -X POST -H 'Content-Type: application/json' -d "{\"email\":\"$1\",\"password\":\"$PASSWORD\"}" "$BASE/auth/login" |
    sed -n 's/.*"accessToken":"\([^"]*\)".*/\1/p'
}

for p in /healthz /readyz /version; do
  code=$(curl -s -o /dev/null -w '%{http_code}' "$ROOT$p")
  check "GET $p" "$code" "{}"
done

# public reads
for p in /home/banners /home/season /islamic/ayah-of-the-day "/books?q=sapiens" "/books/detail?id=bk-sapiens" /categories \
  /collections /deals /donate/recipients /p2p/listings "/bites?feed=forYou" "/reviews?bookId=bk-sapiens" \
  "/readers/detail?id=p-nabila" "/assistant/greeting?lang=en" /geo "/scan/lookup?isbn=9789840001774"; do
  get "$p"
done

TOKEN=$(login reader@waraqah.test)
[ -n "$TOKEN" ] || { echo "FAIL sign in as reader@waraqah.test (is the database seeded?)"; exit 1; }
echo "ok   POST /auth/login"
# one endpoint per feature, signed in
for p in /profile /addresses /notifications /cart /wishlist /alerts /orders /wallet /points /p2p/listings/mine \
  /blocks /inbox /requests/mine /sales/mine /sales/earnings /sell-back/mine /shelves /reading/stats; do
  get "$p" "$TOKEN"
done
post /assistant/ask '{"prompt":"Suggest a book about habits","history":[],"lang":"en"}' "$TOKEN"

ADMIN=$(login admin@waraqah.test)
[ -n "$ADMIN" ] || { echo "FAIL sign in as admin@waraqah.test"; exit 1; }
for p in /admin/dashboard /moderation/listings /admin/orders /admin/catalog/low-stock /sell-back/queue /sales/disputes; do
  get "$p" "$ADMIN"
done

if [ "$fails" -gt 0 ]; then
  echo "smoke: $fails failed"
  exit 1
fi
echo "smoke ok"
