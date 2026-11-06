
# POS-SERVICE

API server


## Installation

Requires Go 1.25 or newer.

Install all modules

```bash
  go mod tidy
```
Install air:
```bash
    go install github.com/air-verse/air@latest
```
Configure: copy `.env.example` to `.env` and fill in `MONGO_URL` and `JWT_SECRET`.

Set `ANTHROPIC_API_KEY` to pre-fill products from photos. Without it the server still runs and only that route is off. `CLAUDE_MODEL` picks the model.

Run Server
```bash
    make run
```

Create the first admin (the API needs an admin token to create users):
```bash
    ADMIN_PASSWORD='<password>' go run ./cmd/createadmin -name admin -full-name "Store owner"
```

Products from before reference prices may still hold a `sell_price`. Count them, then copy it into `reference_prices`:
```bash
    go run ./cmd/backfillprices
    go run ./cmd/backfillprices -apply
```
Run `-apply` only after the frontend that sends `reference_prices` is deployed. The old frontend does not send them, so saving a product there clears the list.
