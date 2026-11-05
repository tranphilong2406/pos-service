
# POS-SERVICE

API server


## Installation

Install all modules

```bash
  go mod tidy
```
Install air:
```bash
    go install github.com/air-verse/air@latest
```
Configure: copy `.env.example` to `.env` and fill in `MONGO_URL` and `JWT_SECRET`.

Run Server
```bash
    make run
```

Create the first admin (the API needs an admin token to create users):
```bash
    ADMIN_PASSWORD='<password>' go run ./cmd/createadmin -name admin -full-name "Store owner"
```
