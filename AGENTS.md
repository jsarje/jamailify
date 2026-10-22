## Project overview

Jamailify is a Go service that imports mail from POP3 and IMAP accounts into
Gmail. The process entry point and sync loop are in [src/main.go](src/main.go).

- `src/fetcher/` defines the interface shared by the POP3 and IMAP clients.
- `src/config/` loads and validates account configuration.
- `src/database/` stores processed message UIDs in SQLite.
- `src/pop3/` and `src/imap/` implement the source mail clients.
- `src/gmail/` handles Gmail API calls and message classification.
- `src/oauth/` contains Microsoft OAuth2 support for IMAP.
- `Tool/GoogleTokenGenerator/` and `Tool/MSTokenGenerator/` generate OAuth
  refresh tokens.

See the [README](README.md) for configuration and deployment instructions.
See [CONTRIBUTING.md](CONTRIBUTING.md) for the sync flow and contribution
guidance. The [specs/](specs/) directory contains design and testing notes.

## Build and test

Build the service with:

```sh
go build -o jamailify ./src
```

Run the full Go test suite with:

```sh
go test ./...
```

The POP3 and IMAP integration tests use testcontainers and require Docker.
Use a focused package test, such as `go test ./src/config`, when Docker is not
available.

## Runtime and credentials

The service reads configuration from `/app/config/config.json` and stores its
SQLite database at `/app/data/sync_state.db`. These are the container paths in
the [Dockerfile](Dockerfile). Local runs need those paths or a deliberate code
change.

Do not add real credentials or refresh tokens to the repository. Supply them
through local configuration, ignored files, or deployment secrets.

## Change guidance

Keep the sync cycle testable through `RunSingleSync` in
[src/main.go](src/main.go).
Use the `fetcher.EmailFetcher` interface when working on protocol clients, and
keep unit tests independent of live mail servers and Google API calls. Add or
update integration tests when a change depends on real POP3 or IMAP behavior.

Check the README and relevant specs before changing configuration, runtime
paths, or container behavior. Have a maintainer review changes to the Dockerfile
or Docker Compose configuration before merging.
