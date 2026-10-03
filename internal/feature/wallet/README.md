# wallet

The wallet balance is always the sum of its ledger entries (`wallet_entries`, migration `0006`).

- **Endpoint:** `GET /wallet`.
- **Ledger** (`wallet.Ledger`): `Balance`, `Credit` (refunds, Sell Back) and `Spend` (checkout, wallet pays last). Every method takes the caller's queries so it runs inside its transaction.
