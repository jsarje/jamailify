# Feature Specification: IMAP Integration & Dual-Protocol Support

## Overview
Expand the self-hosted jamailify application to support both IMAP and POP3 protocols for fetching source emails. The user must be able to specify either protocol on a per-account basis in the configuration file. The core synchronization loop will be refactored to use a common interface so it functions identically regardless of the underlying protocol.

## 1. Required Third-Party Libraries
* Add IMAP Client: `github.com/emersion/go-imap`
* (Keep existing `pop3.go`)

## 2. Configuration Schema Updates
Update the configuration parser to handle a new `protocol` flag. 

The application must parse this updated config.json structure:

    {
      "poll_interval_minutes": 10,
      "accounts": [
        {
          "name": "My Email",
          "protocol": "imap",
          "server": "imap.example.com:993",
          "user": "me@example.com",
          "pass": "supersecret",
          "gmail_refresh_token": "1//0eabc123..."
        },
        {
          "name": "Wife's Email",
          "protocol": "pop3",
          "server": "pop.wife-example.com:995",
          "user": "wife@wife-example.com",
          "pass": "alsosecret",
          "gmail_refresh_token": "1//0exyz789..."
        }
      ]
    }

## 3. Database Abstraction (`db`)
The database should no longer reference POP3 specifically. 
* **Schema Change:** Ensure the table uses a generic `message_uid` column. 
  `CREATE TABLE IF NOT EXISTS synced_emails (account_name TEXT, message_uid TEXT, PRIMARY KEY (account_name, message_uid));`
* *Note to AI:* IMAP UIDs are typically integers (uint32). Convert them to strings before storing them in SQLite to maintain compatibility with POP3 UIDs (which are strings).

## 4. The Fetcher Interface (`fetcher`)
Create a new Go interface that both the POP3 and IMAP clients will implement. This isolates the protocol logic from the main sync loop.

    type EmailFetcher interface {
        // Connects to the server and authenticates
        Connect() error
        // Returns a slice of all unique IDs currently in the INBOX
        GetUIDs() ([]string, error)
        // Downloads only the headers for a specific UID
        DownloadEmailHeaders(uid string) ([]byte, error)
        // Downloads the raw RFC 2822 bytes for a specific UID
        DownloadEmail(uid string) ([]byte, error)
        // Closes the connection cleanly
        Close() error
    }

## 5. IMAP Client Implementation (`imap_client`)
Implement the `EmailFetcher` interface using `github.com/emersion/go-imap`.
* **Connect:** Connect using TLS (port 993 is standard). Authenticate and `Select` the `"INBOX"`.
* **GetUIDs:** Use a `Search` command for all messages (or specifically `UNSEEN` if preferred, but fetching all UIDs and checking the local SQLite DB is safer for parity with the POP3 logic).
* **DownloadEmailHeaders:** Fetch just the headers for a message to inspect metadata (like the Date) before downloading the full body. This is a critical load reduction step. Use the `BODY.PEEK[HEADER]` fetch item.
* **DownloadEmail:** Fetch the specific message using its IMAP UID. 
  * **Crucial Detail:** You MUST use the `BODY.PEEK[]` fetch item so that the application downloads the raw bytes *without* accidentally marking the email as "Read" on the source server.
* **Extraction:** Extract the raw bytes from the returned `imap.Message` literal and return them.

## 6. Sync Loop Refactoring (`main`)
* Update the setup logic inside the goroutine for each account:
    * Check `account.Protocol`.
    * If `"imap"`, instantiate the `IMAPClient`.
    * If `"pop3"`, instantiate the `POP3Client`.
    * Assign the instantiated client to an `EmailFetcher` variable.
* The rest of the sync loop logic remains largely the same, but with the addition of the header-first check: Get UIDs -> Check DB -> Download Headers -> Check Sync Window -> Download Full Email -> Push via Gmail API -> Mark DB. This interacts only with the `EmailFetcher` interface.

## 7. Safeguards
To prevent duplicate emails and reduce load, the application will incorporate two primary safeguards that apply regardless of the chosen protocol (POP3 or IMAP).

### 7.1. Header-Only Fetching for Load Reduction
Both the POP3 and IMAP fetcher implementations will support a method for downloading only the email headers (`DownloadEmailHeaders`). The main sync loop will use this method first to inspect the email's date. If the email is outside the user-configured `sync_window_days`, the application will not download its full body, saving significant bandwidth and processing time.

### 7.2. Gmail Message-ID Based Deduplication
This is a critical, secondary layer of defense against duplicate emails in the destination Gmail account.
* **Mechanism:** Before pushing any email (retrieved via either POP3 or IMAP) to Gmail, the application will first parse the `Message-ID` header from the raw email content.
* **Process:**
    1. Extract the `Message-ID`.
    2. Construct a Gmail API search query in the format `rfc822msgid:YOUR_MESSAGE_ID`.
    3. Query the user's Gmail inbox to see if a message with that `Message-ID` already exists.
    4. If the search returns one or more results, the application will skip the upload and consider the email successfully synced.
* **Benefit:** This provides a strong guarantee against duplicates, even if the local SQLite database were to be reset or corrupted. It leverages a standard, globally unique email header, making the sync process robust and reliable.