# Jamailify

A small, self-hosted "Gmailify" service written in Go: it polls POP3 and IMAP accounts and pushes new messages into Gmail inboxes using pre-provisioned OAuth2 refresh tokens.

> [!note]
> This repository implements a production-minded daemon that syncs one or more external accounts to Gmail. It expects pre-generated OAuth2 refresh tokens and local configuration (no interactive OAuth flow at runtime).

## ✨ Features

- **Multi-Protocol Support**: Poll multiple POP3 or IMAP accounts concurrently.
- **Modern Authentication**: Supports standard password auth or modern Microsoft OAuth2 (Outlook/Hotmail) for IMAP accounts.
- **Robust Deduplication**: Persists processed UIDs in SQLite and actively queries the Gmail `rfc822msgid:` API to avoid duplicate delivery, even if the local database is lost.
- **Load Reduction**: Uses header-first fetching strategies to determine message age, minimizing bandwidth usage for large mailboxes. 
- **Filter-Aware Import**: Imports raw RFC 2822 messages into Gmail using the `Users.Messages.Import` API, which applies your Gmail filters (labels, routing, etc.) just as if the message had arrived normally.
- **Container-Friendly**: Designed for easy deployment via Docker and simple `config.json` files.

## 🛠️ Configuration

The application reads configuration from `/app/config/config.json` (see [src/config/config.go](src/config/config.go)). The expected structure can handle both POP3 and IMAP accounts.

Here is an example configuration file:

```json
{
	"poll_interval_minutes": 10,
	"google_client_id": "YOUR_GMAIL_CLIENT_ID",
	"google_client_secret": "YOUR_GMAIL_CLIENT_SECRET",
	"gmail_fetch_metadata_after_import": true,
	"max_messages_to_check": 2000,
	"sync_window_days": 7,
	"accounts": [
		{
			"name": "Standard POP3 Account",
			"protocol": "pop3",
			"server": "pop.example.com:995",
			"user": "me@example.com",
			"pass": "supersecret",
			"gmail_refresh_token": "1//0eabc123..."
		},
		{
			"name": "Outlook IMAP Account",
			"protocol": "imap",
			"server": "outlook.office365.com:993",
			"user": "me@outlook.com",
			"auth_method": "oauth2",
			"ms_client_id": "YOUR_MS_CLIENT_ID",
			"ms_client_secret": "YOUR_MS_CLIENT_SECRET",
			"ms_refresh_token": "M.R3_BAY...",
			"gmail_refresh_token": "1//0eabc123..."
		}
	]
}
```

### Global Configuration Values

- `poll_interval_minutes` (int) - The pause between checking accounts.
- `google_client_id` & `google_client_secret` - The Google Cloud app credentials used to authenticate to Gmail.
- `gmail_fetch_metadata_after_import` (bool, optional) - Whether to fetch and log the created Gmail thread/message ID metadata after successfully pushing an email.
- `max_messages_to_check` (int, default: 2000) - Maximum number of newest messages to inspect per sync stream. This prevents scanning very large mailboxes on each poll.
- `sync_window_days` (int, default: 7) - Number of days to look back for messages to sync. Only messages with a `Date` header within this window are considered.

### Account Configuration Values

- `name` - Display name for logging.
- `protocol` - Must be `"pop3"` or `"imap"`.
- `server` - Server and port (e.g., `imap.example.com:993`).
- `user` - Account username.
- `gmail_refresh_token` - The offline OAuth2 token for the destination Gmail account.
- `no_tls` (bool, optional) - Set to true to disable TLS encryption (useful for local development or plain-text testing).

#### Password Authentication (Default)
When using standard authentication (like standard POP3/IMAP accounts or App Passwords):

- `auth_method` - Leave omitted or set to `"password"`.
- `pass` - The account password.

#### Microsoft OAuth2 Authentication (IMAP only)
When using `"auth_method": "oauth2"`, the following fields are required:

- `ms_client_id` - The Azure app registration client ID.
- `ms_client_secret` - The Azure app registration client secret.
- `ms_refresh_token` - The offline refresh token generated for the user's Microsoft account.

Runtime file locations used by the service:

- Config: `/app/config/config.json`
- SQLite DB: `/app/data/sync_state.db`

> [!warning]
> Do not commit real credentials to the repository. Provide `config.json` via secrets, mounts, or environment-specific tooling.

## 🔧 Generating OAuth Tokens

Because Jamailify is a background daemon lacking an interactive visual interface, it requires offline **refresh tokens** that last indefinitely. 

You can use the utilities included in this repository to easily perform the initial OAuth flow on your local machine and obtain these refresh tokens:

1. **Google Tokens**: See [Tool/GoogleTokenGenerator/GETTING-STARTED.md](Tool/GoogleTokenGenerator/GETTING-STARTED.md) to generate the needed `gmail_refresh_token`.
2. **Microsoft Tokens**: See [Tool/MSTokenGenerator/GETTINGSTARTED.md](Tool/MSTokenGenerator/GETTINGSTARTED.md) to generate the `ms_refresh_token`.

Copy the resulting refresh tokens directly into your `config.json` file.

## 🧑‍🤝‍🧑 Contributing

Information on building the project, running tests (including containerized integration tests), understanding the architecture (like the `Fetcher` interface), and contributing can be found in [CONTRIBUTING.md](CONTRIBUTING.md).
