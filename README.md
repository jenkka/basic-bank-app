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
| Deploy | AWS EKS (eksctl-managed), nginx-ingress, cert-manager + Let's Encrypt TLS, config from AWS Secrets Manager |

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

**Automatic TLS and a symmetric, idempotent cluster lifecycle.** The course terminates TLS manually. Here, cert-manager runs in-cluster with a Let's Encrypt `ClusterIssuer`; the ingress is annotated so certificates are issued and renewed automatically with no manual steps. The Makefile lifecycle is deliberately symmetric: `make bootstrap` and `make teardown` are exact inverses (cluster → ingress → cert-manager → issuer → app, and the reverse), with `--ignore-not-found` on every delete so teardown is idempotent and partial states recover cleanly. `teardown` removes the ingress controller — and its cloud load balancer — *before* deleting the cluster, which avoids the classic eksctl failure where an orphaned ELB blocks VPC teardown. The intent is that no required step lives only in someone's shell history.

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

## Continuous integration & deployment

Two GitHub Actions workflows:

1. **`test.yml`** — Runs on every push to `main` and on pull requests. Spins up a Postgres service container, applies migrations, runs `go vet`, and executes the test suite. Never touches AWS.
2. **`deploy.yml`** — Full deploy pipeline: pulls runtime config from AWS Secrets Manager into `app.env`, builds and pushes the image to AWS ECR Public (tagged with the commit SHA), then `kubectl apply`s the deployment, service, issuer, and ingress to EKS and waits on the rollout.

> **Note:** `deploy.yml` is currently `workflow_dispatch`-only (manual). The live cluster is torn down when idle to avoid cloud cost, so auto-deploy-on-push is disabled. The entire environment is reproducible from scratch with `make bootstrap`; restoring the `push` trigger re-arms continuous deployment.

## Deployment

The production target is AWS EKS, fully scripted through the Makefile so the
environment can be created or destroyed in one command.

```bash
make bootstrap   # cluster-up → ingress-install → cert-manager-install → issuer-install → grant-ci → deploy
make teardown    # destroy → issuer-uninstall → cert-manager-uninstall → ingress-uninstall → cluster-down
```

- **Cluster** — `eks/eks.yaml` defines an eksctl-managed cluster (managed node group, NAT gateway disabled to keep cost down).
- **Ingress** — nginx-ingress fronts the API behind an AWS load balancer.
- **TLS** — cert-manager with a Let's Encrypt `ClusterIssuer` (`eks/issuer.yaml`) issues and auto-renews the certificate for the API host; the ingress requests it via annotation. The HTTP-01 challenge is solved through the same nginx ingress.
- **Config & secrets** — runtime config lives in AWS Secrets Manager and is materialized into `app.env` at build time by `deploy.yml`; nothing sensitive is committed.
- **Database** — an AWS RDS PostgreSQL instance, provisioned and managed outside the cluster lifecycle (intentionally not in `make teardown`, so the data layer is decoupled from cluster churn).

`bootstrap` and `teardown` are exact inverses and safe to re-run; `teardown` removes the load balancer before the cluster so no AWS resources are orphaned.

## Status and what's next

Implemented: users, accounts, transfers, JWT auth, authorization middleware, transactional balance updates with deadlock-safe ordering, unit-test coverage of the API and store layers with gomock, Dockerized local dev, CI through the test suite, and a fully scripted EKS deployment with nginx-ingress and automatic Let's Encrypt TLS.

Not yet implemented (planned, course covers some of these):
- gRPC endpoints alongside REST
- PASETO as an alternative token format behind the existing `Maker` interface
- Refresh tokens with session storage
- Background workers for async tasks (e.g. welcome emails)
- Structured logging and metrics
- Add `app.env` to `.dockerignore` and source config from a Kubernetes Secret instead of baking it into the image

---

## Acknowledgement

The architecture follows the structure of [TECH SCHOOL's Backend Master Class](https://www.udemy.com/course/backend-master-class-golang-postgresql-kubernetes/) by Quang Pham. I implemented every section end-to-end and made the divergences listed above where the course's defaults felt incomplete or could be cleaner.
