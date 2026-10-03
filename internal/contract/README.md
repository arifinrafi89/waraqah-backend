# contract

Checks that every endpoint answers in the same shape as the fake API (BACKEND_PLAN.md section 15).

- `testdata/contract/` holds one golden per request, recorded from the fake API by `make contract-export`
  (the exporter lives in `tools/contract-export/`). `_index.json` lists them in the order they ran.
- `TestContract` replays each golden, in that order, against the router on a seeded database and compares
  **shapes**: same keys, same JSON kinds, `null` where the golden is `null`. Extra keys in our answer are fine.
- `pending.txt` lists endpoints not built yet. A feature task removes its lines; T19 needs the file empty.
- `TestRouteCoverage` reads Appendix A of the plan and fails when an endpoint is neither registered nor pending.
