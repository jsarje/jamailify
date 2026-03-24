# Project Overview
Write a complete, production-ready Golang application that acts as a self-hosted "Gmailify" clone for two users. The app runs as a background daemon, polling specific POP3 email accounts on a schedule, and pushing newly discovered emails directly into corresponding Gmail accounts via the Gmail API.

## Architecture & Constraints
* **Language:** Go (1.21+)
* **Concurrency:** Use one goroutine per configured account. Do not use complex worker pools; keep it simple.
* **State Management:** Use SQLite (`github.com/mattn/go-sqlite3`) to store a history of downloaded POP3 UIDs to prevent duplicate processing. 
* **Configuration:** Read account details from a local `config.json` file.
* **Authentication:** The app will use OAuth2 `refresh_token`s (provided in the config) to automatically generate short-lived access tokens for the Gmail API. No web server or manual OAuth flow should be in this codebase (tokens are pre-generated).

## Required Third-Party Libraries
* POP3 Client: `github.com/knadh/go-pop3`
* SQLite Driver: `github.com/mattn/go-sqlite3`
* Google OAuth: `golang.org/x/oauth2` and `golang.org/x/oauth2/google`
* Gmail API: `google.golang.org/api/gmail/v1`

## Data Models
The application must parse this exact `config.json` structure:

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

### Component Specifications
Please divide the logic into logical Go files/packages or clean functions within `main.go`. I need the following implementations:

**1. Database Manager (`db`)**
* Initialize a local `sync_state.db` file.
* Create table if it doesn't exist: `CREATE TABLE IF NOT EXISTS synced_emails (account_name TEXT, pop3_uid TEXT, PRIMARY KEY (account_name, pop3_uid));`
* Function: `IsSynced(accountName, uid string) (bool, error)`
* Function: `MarkSynced(accountName, uid string) error`

**2. Gmail API Client (`gmail_client`)**
* Function to initialize the Gmail service using the OAuth client configuration from `config.json` and the user's specific `refresh_token`.
* Function: `PushEmail(rawRFC2822 []byte) error`. This must use the `Users.Messages.Insert` endpoint to bypass Gmail filters and place the message directly in the inbox.
* **Crucial Detail:** The Gmail API requires the raw email bytes to be base64url encoded (without padding) inside the `gmail.Message{Raw: ...}` payload.

**3. POP3 Fetcher (`pop3_client`)**
* Connect to the POP3 server using TLS (port 995).
* Authenticate.
* Retrieve the list of messages and their unique UIDs using the `UIDL` command.
* Function to download the full raw email message (RFC 2822 format) by its message sequence number.

**4. The Main Sync Loop (`main`)**
* Load `config.json`.
* Initialize the SQLite database.
* Iterate over the `accounts` array. For each account, launch a goroutine running an infinite loop.
* Inside the goroutine:
    1.  Run the sync process immediately.
    2.  Block on a `time.NewTicker` based on `poll_interval_minutes`.
* The Sync Process steps:
  1. Connect to POP3.
  2. Inspect mailbox newest-first (use `STAT` to get message count and `TOP` to fetch headers) and consider only messages within the last 7 days.
  3. For each UID in that window, check `IsSynced` via the DB manager.
  4. If not synced and within the 7-day window: Download raw email -> Push via Gmail API -> `MarkSynced` in DB.
  *
  * Configuration:
    - `max_messages_to_check` (int): Maximum number of newest messages to inspect per sync run. Defaults to `2000` when omitted or zero.
    - `sync_window_days` (int): Number of days to look back for messages to sync. Defaults to `7` when omitted or zero.
  5. Log successes and errors cleanly (include account name in logs).
* Use a `select {}` or `sync.WaitGroup` at the end of `main()` to keep the daemon running forever.
