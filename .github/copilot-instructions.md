# GitHub Copilot / Agent Instructions

Purpose
-------
This file tells automated chat agents (Copilot Chat) how to be immediately productive in this repository: where to find code, how to build and test, important runtime conventions, and warnings about integration tests and secrets.

Quick links
-----------
- **Repo root**: [README.md](README.md)
- **Main entry**: [src/main.go](src/main.go)
- **Config code**: [src/config/config.go](src/config/config.go)
- **Database code**: [src/database/database.go](src/database/database.go)
- **Gmail client**: [src/gmail/gmail.go](src/gmail/gmail.go)
- **POP3 client**: [src/pop3/pop3.go](src/pop3/pop3.go)
- **Docker / compose**: [Dockerfile](Dockerfile), [docker-compose.yml](docker-compose.yml)
- **Specs**: [specs/spec-01.md](specs/spec-01.md), [specs/spec-02-testing.md](specs/spec-02-testing.md), [specs/spec-03-containisation.md](specs/spec-03-containisation.md)
- **Tools**: [Tool/TokenGenerator/GETTING-STARTED.md](Tool/TokenGenerator/GETTING-STARTED.md)

How to build & run
-------------------
- Build binary locally: `go build -o jamailify ./src`
- Run app locally (read config first): `go run ./src`
- Build Docker image: `docker build -t jamailify .`
- Docker Compose: `docker-compose up -d` (or `docker compose up -d`)

How to run tests
-----------------
- Run all Go tests: `go test ./...`
- Run package tests only: `go test ./src` or `go test ./src/<package>`
- Run a single test: `go test ./src -run TestName`
- Integration tests: `src/pop3/pop3_integration_test.go` uses Docker/testcontainers; run these individually and ensure Docker is available.

Secrets & credentials
---------------------
- `config/config.json` and `Tool/TokenGenerator/credentials.json` are expected to be provided by operators and are gitignored. Do NOT write secrets to the repo.
- Many runtime paths assume the container layout (see Runtime paths below). For local runs, create `/app/config/config.json` and `/app/data` or adjust code to use a local path.

Runtime paths & conventions
--------------------------
- Container runtime expects config at `/app/config/config.json` and DB at `/app/data/sync_state.db` (see [Dockerfile](Dockerfile) and [specs/spec-03-containisation.md](specs/spec-03-containisation.md)).
- Tests include both unit and integration tests. Integration tests require Docker and may be slow; CI must provide Docker to run them.
- `main` starts long-running workers — prefer helper functions or test hooks (e.g., single-run helpers) for quick manual tests.

Common pitfalls for automation
-----------------------------
- Absolute paths: code expects `/app/...`. Local execution needs those paths created or code adjusted.
- Integration tests will fail without Docker. Avoid running `go test ./...` on systems without Docker unless integration tests are explicitly skipped.
- Sensitive files are excluded from the repo; provide credentials via secrets or local mount when running.

Agent behavior guidance
-----------------------
- Link, don't duplicate: reference specs in `/specs` rather than copying them into chat responses.
- When suggesting changes that affect runtime (paths, containers), ask whether the user prefers modifying code or changing local mount points.
- For code changes that touch infra (Dockerfile, docker-compose), request a human review before merging.
