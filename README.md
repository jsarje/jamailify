# Jamailify

A small, self-hosted "Gmailify" service written in Go: it polls POP3 accounts and pushes new messages into Gmail inboxes using pre-provisioned OAuth2 refresh tokens.

> [!note]
> This repository implements a production-minded daemon that syncs one or more POP3 accounts to Gmail. It expects pre-generated OAuth2 refresh tokens and local configuration (no interactive OAuth flow).

## Features

- Poll multiple POP3 accounts concurrently (one goroutine per account).
- Persist processed POP3 UIDs in SQLite to avoid duplicate delivery.
- Push raw RFC 2822 messages to Gmail using the `Users.Messages.Insert` API (bypasses filters).
- Simple configuration via `config.json` and container-friendly runtime paths.

## Quickstart

Clone the repo and build locally:

```bash
git clone <repo-url>
cd jamailify
go build -o jamailify ./src
```

Run locally (reads `/app/config/config.json` by default):

```bash
go run ./src
```

Build and run with Docker:

```bash
docker build -t jamailify .
docker-compose up -d
```

## Configuration

The application reads configuration from `/app/config/config.json` (see `config/config.go`). The expected structure is:

```json
{
	"poll_interval_minutes": 10,
	"google_client_id": "YOUR_CLIENT_ID",
	"google_client_secret": "YOUR_CLIENT_SECRET",
	"accounts": [
		{
			"name": "User 1",
			"pop3_server": "pop.example.com:995",
			"pop3_user": "me@example.com",
			"pop3_pass": "supersecret",
			"gmail_refresh_token": "1//0eabc123..."
		}
	]
}

Additional optional configuration keys:

- `max_messages_to_check` (int): maximum number of newest messages to inspect per sync run. Defaults to `2000` when omitted or zero. This prevents scanning very large mailboxes on each poll.
- `sync_window_days` (int): number of days to look back for messages to sync. Defaults to `7` when omitted or zero. Only messages with a `Date` header within this window are considered.

Behavior notes:

- The sync worker now inspects messages newest→oldest and stops scanning when it encounters a message older than the configured window. This reduces bandwidth and processing for large mailboxes.
- To determine message age the service uses POP3 `TOP` to fetch headers only. If a server does not support `TOP`, the service falls back to `RETR` (full message) and parses the `Date` header.
- UID tracking in SQLite (`sync_state.db`) remains the mechanism to avoid duplicate processing.
```

Runtime file locations used by the service:

- Config: `/app/config/config.json`
- SQLite DB: `/app/data/sync_state.db`

> [!warning]
> Do not commit real credentials to the repository. Provide `config.json` via secrets, mounts, or environment-specific tooling.

## Tests

Run the unit tests with:

```bash
go test ./...
```

Notes:

- Unit tests are designed to mock external services (POP3/Gmail). Integration tests that exercise a real POP3 server are present under `src/pop3` and require Docker (see `specs/spec-02-testing.md`).

> [!note]
> Integration tests use containers (e.g., GreenMail) and should be run only when Docker is available on the host.

## Project Structure

- `src/` — application sources (`main.go`, `config`, `database`, `pop3`, `gmail`). See the main entry point at [src/main.go](src/main.go#L1).
- `config/` — configuration loader and validation (`config/config.go`).
- `database/` — SQLite-based sync state manager.
- `pop3/` — POP3 client implementation and integration tests.
- `gmail/` — Gmail API wrapper (uses OAuth2 refresh tokens).
- `specs/` — design & testing specifications used while building the project.

## How it Works (brief)

1. Load `config.json`.
2. Initialize SQLite DB (`sync_state.db`).
3. For each configured account, start a goroutine that:
	 - Connects to POP3, lists UIDs, and downloads new messages.
	 - Pushes raw messages to Gmail using the `Users.Messages.Insert` API (base64url encoded without padding).
	 - Marks processed UIDs in the DB to avoid reprocessing.

The detailed sync flow is implemented in `src/main.go` and the testable single-sync function is `RunSingleSync`.

## Development notes

- Follow the specs in `specs/spec-01.md` and `specs/spec-02-testing.md` when modifying core behavior or tests.
- Use interfaces for external services so tests can inject mocks (the codebase already follows this pattern).

## Where to look next

- Start reading the main logic: [src/main.go](src/main.go#L1).
- Config loader: [src/config/config.go](src/config/config.go#L1).
- POP3 client and integration tests: [src/pop3/pop3.go](src/pop3/pop3.go#L1) and [src/pop3/pop3_integration_test.go](src/pop3/pop3_integration_test.go#L1).
