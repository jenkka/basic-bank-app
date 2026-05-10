# dummy-bank

A banking API service written in Go. Supports user signup and authentication, multi-currency accounts, and atomic money transfers between accounts with concurrency-safe transaction handling.

Built as a deep dive into production Go backend patterns. The architectural skeleton follows [TECH SCHOOL's Backend Master Class](https://www.udemy.com/course/backend-master-class-golang-postgresql-kubernetes/), with several departures from the course noted below.

---

## What it does

- **Users** — Sign up, log in, and receive a JWT access token.
- **Accounts** — Create multiple accounts per user, each in its own currency. A unique `(owner, currency)` constraint prevents a user from accidentally creating two accounts in the same currency.
- **Transfers** — Move money atomically between two accounts in the same currency, with both sides updated under a single database transaction. Each transfer produces a paired ledger entry per account, giving every account an append-only history.
- **Authorization** — Authenticated routes verify the bearer token; transfer requests additionally check that the sender owns the source account before executing.

## Tech stack

| Layer | Tools |
|------|-------|
| Language | Go 1.26 |
| HTTP | Gin |
| Database | PostgreSQL 17, sqlc (type-safe query generation), golang-migrate |
| Auth | JWT (golang-jwt/jwt v5), bcrypt for password hashing |
| Money | `shopspring/decimal` for arbitrary-precision money math |
| Config | Viper (file + env-var, with env overrides) |
| Testing | testify, `go.uber.org/mock` with custom matchers, Postgres service container in CI |
| Container | Multistage Docker build, Docker Compose for local development |
| CI | GitHub Actions running the full test suite against a Postgres service container |
| Image registry | AWS ECR Public via GitHub Actions on push to `main` |

## Architecture

```
Client ──HTTPS──> Gin router
                     │
                     ├── /users, /users/login          (public)
                     │
                     └── authMiddleware (Bearer JWT)
                            │
                            ├── /accounts (create, get, list)
                            └── /transfers (create)
                                     │
                                     └─> Store.TransferTxn
                                            │
                                            └─> single SQL transaction:
                                                  CreateTransfer →
                                                  CreateEntry (from, neg) →
                                                  CreateEntry (to, pos) →
                                                  AddAccountBalance (lower ID first)
```

## Database schema

Four tables: `users`, `accounts`, `transfers`, `entries`.

- `users` — primary key on `username`, unique index on `email`, with `pwd_updated_at` for password rotation tracking.
- `accounts` — owned by a user via `owner → users.username` foreign key, with a unique `(owner, currency)` constraint so each user has at most one account per currency.
- `transfers` — records the intent of moving an amount between two accounts.
- `entries` — records a balance delta on a single account; one negative entry and one positive entry are produced per transfer, giving every account a queryable ledger.

Schema is managed via `golang-migrate` (versioned up/down files), and type-safe Go bindings are generated from raw SQL via `sqlc`. A `decimal` column override in `sqlc.yaml` maps Postgres `numeric` to `shopspring/decimal.Decimal` in Go.

## How concurrent transfers work

Naively, two concurrent transfers between the same two accounts going in opposite directions can deadlock: each transaction grabs the row lock on its "from" account first, then waits on the other transaction to release the "to" account it has locked. Both wait forever, and Postgres aborts one with a deadlock error.

`TransferTxn` avoids this by always updating the lower-ID account's balance first, regardless of the transfer direction. With this consistent lock ordering, even adversarial concurrent transfers serialize cleanly without deadlocking. The store-level test suite exercises this path with a fan-out of concurrent goroutines.

## Notable engineering decisions

These are places where I diverged from the course's defaults:

**`shopspring/decimal` for money instead of `int64` cents.** The course stores all money values as `int64` (in the smallest unit of currency). That works fine for USD but breaks cleanly for currencies with different minor-unit precisions (JPY has none, BHD has three) and is awkward for any future need to represent fractional units. Switching to `shopspring/decimal` — both in the Postgres schema (`numeric`) and Go code (via the `sqlc.yaml` type override) — gives arbitrary-precision arithmetic and a clean API, at the cost of a small per-operation overhead I judged worth paying for a financial system.

**Login defends against username-enumeration timing attacks.** The course's login handler returns `404 Not Found` when the requested username doesn't exist and `401 Unauthorized` when the password is wrong. That leaks two pieces of information: the status code itself, and the response time (since `bcrypt.CompareHashAndPassword` is intentionally slow, the user-not-found path returns measurably faster). I changed both: the not-found branch now runs `CheckPassword` against a precomputed dummy hash to equalize the wall time, and both branches return the same generic `invalid credentials` error with `401`. An attacker can no longer distinguish "this username exists" from "wrong password" by status, body, or timing.

**Username vs email constraint violations get distinct 409 responses.** The course's `createUser` handler treats any unique-constraint violation as a generic `403 Forbidden`. I extended this to inspect `pq.Error.Constraint` and return `409 Conflict` with a specific message — either "username already exists" or "email already exists" — so the client can show the right field-level error. `409 Conflict` is also the semantically correct status for a duplicate-resource collision; `403 Forbidden` implies an authorization failure, which this isn't.

**Docker Compose orchestrates startup natively, without shell scripts.** The course uses `wait-for.sh` and `start.sh` shell scripts inside the container to gate startup on Postgres readiness. I replaced both with native Compose features: a `healthcheck` block on the `postgres` service runs `pg_isready` until the database accepts connections, and the `migrate` and `server` services use `depends_on` with `condition: service_healthy` and `condition: service_completed_successfully` to wait on Postgres health and migration completion respectively. The result is no shell scripts in the image and a clearer dependency graph in one file.

## Running locally

The fastest path is Docker Compose, which brings up Postgres, runs migrations, and starts the server:

```bash
docker-compose up
```

Or run the pieces manually with the Makefile:

```bash
make run-postgres      # start a Postgres 17 container on a shared docker network
make create-db         # create the dummy_bank database
make migrateup         # apply migrations
make sqlc              # regenerate Go bindings from SQL (only after schema changes)
make mock              # regenerate gomock mocks (only after Store interface changes)
make test              # run the full test suite with coverage
make racetest          # run tests with the race detector
make server            # run the API on :8080
```

Configuration is loaded from `app.env` in the working directory, with environment variables overriding file values.

## Continuous integration

Two GitHub Actions workflows run on push to `main`:

1. **`test.yml`** — Spins up a Postgres service container, applies migrations, runs `go vet`, and executes the test suite. Pull requests run the same workflow.
2. **`deploy.yml`** — On a successful push to `main`, builds the Docker image, tags it with the commit SHA, and pushes to AWS ECR Public.

Pushing to ECR is the end of the current automation; deploying the new image to a running cluster is a separate step (and a planned improvement).

## Status and what's next

Implemented: users, accounts, transfers, JWT auth, authorization middleware, transactional balance updates with deadlock-safe ordering, unit-test coverage of the API and store layers with gomock, Dockerized local dev, and CI through ECR push.

Not yet implemented (planned, course covers some of these):
- gRPC endpoints alongside REST
- PASETO as an alternative token format behind the existing `Maker` interface
- Refresh tokens with session storage
- Background workers for async tasks (e.g. welcome emails)
- Structured logging and metrics
- Kubernetes manifests and an automated cluster deploy
- Add app.env to .dockerignore and use Kubernetes Secret instead

---

## Acknowledgement

The architecture follows the structure of [TECH SCHOOL's Backend Master Class](https://www.udemy.com/course/backend-master-class-golang-postgresql-kubernetes/) by Quang Pham. I implemented every section end-to-end and made the divergences listed above where the course's defaults felt incomplete or could be cleaner.
