# Contributing

Thanks for your interest in contributing to this project — contributions small and large are welcome.

## Code of conduct
Be respectful and collaborative. Treat others with courtesy; abusive or discriminatory behavior will not be tolerated.

## Report an issue
- Search existing issues first.
- Provide a clear title, steps to reproduce, expected vs actual behavior, and relevant hardware/OS/build details.

## Project Structure

- `src/` — application sources (`main.go`, `config`, `database`, `pop3`, `gmail`). See the main entry point at [src/main.go](../src/main.go#L1).
- `src/fetcher/` — interface abstractions for both POP3 and IMAP clients.
- `src/imap/` — IMAP client implementation and integration tests.
- `src/pop3/` — POP3 client implementation and integration tests.
- `src/oauth/` — Microsoft OAuth2 token utilities for Outlook/Hotmail.
- `src/gmail/` — Gmail API wrapper (uses OAuth2 refresh tokens).
- `config/` — configuration loader and validation (`config/config.go`).
- `database/` — SQLite-based sync state manager.
- `specs/` — design & testing specifications used while building the project.
- `Tool/` — standalone tools for generating required OAuth tokens.

## How it Works (brief)

1. Load `config.json`.
2. Initialize SQLite DB (`sync_state.db`).
3. For each configured account, start a goroutine that:
	 - Initializes either an IMAP or POP3 client through the `fetcher.Fetcher` interface.
	 - Connects to the server, iterates messages (newest first), and fetches headers to check dates.
	 - Checks the Gmail `rfc822msgid:` API and local SQLite to prevent duplicate delivery.
	 - Downloads new, undelivered messages.
	 - Pushes raw messages to Gmail using the `Users.Messages.Insert` API (base64url encoded without padding).
	 - Marks processed UIDs in the DB to avoid reprocessing.

The detailed sync flow is implemented in `src/main.go` and the testable single-sync function is `RunSingleSync`.

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

## Tests

Run the unit tests with:

```bash
go test ./...
```

Notes:

- Unit tests are designed to mock external services (POP3/IMAP/Gmail). Integration tests that exercise a real POP3 or IMAP server are present under `src/pop3` and `src/imap` and require Docker (see `specs/spec-02-testing.md`).

> [!note]
> Integration tests use containers (e.g., GreenMail) and should be run only when Docker is available on the host.

## Contribute code
1. Fork the repo and create a short-lived branch (feature/bugfix/your-name).
2. Keep changes focused and include tests or a short validation step when possible.
3. Follow the existing code style and naming patterns.
4. Commit messages: use the imperative present tense ("Add", "Fix", "Update").
5. Open a pull request against `main` referencing the issue (if any) and a short description of the change.

## Pull request checklist
- Runs, builds, and tests pass locally (see above for build instructions).
- Includes a short description and motivation.
- Adds or updates tests/examples where applicable.
- No unrelated changes or large formatting-only diffs.

## Where to look next

- Start reading the main logic: [src/main.go](../src/main.go#L1).
- Config loader: [src/config/config.go](../src/config/config.go#L1).
- POP3/IMAP client integration tests: [src/pop3/pop3_integration_test.go](../src/pop3/pop3_integration_test.go#L1) and [src/imap/imap_integration_test.go](../src/imap/imap_integration_test.go#L1).

## Questions
If you're unsure how to proceed, open an issue or contact a maintainer for guidance.

Thanks — we appreciate your help keeping this project healthy and useful.
