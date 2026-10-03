# loyalty (points)

Points balance is always the sum of `points_entries` (migration `0006`). 1 point per 100 taka of books paid; at least 50 points to spend, at most 20% of the books.

- **Endpoint:** `GET /points`.
- **Ledger** (`loyalty.Ledger`): `Balance`, `Spend`, `Earn`, `Undo` (cancel gives back what was spent, takes back what was earned).
- The rules are in `rules.go`, ported with their tests from `loyalty_rules.dart`.
