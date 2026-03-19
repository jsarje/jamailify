# Unit Test Specification: Self-Hosted Gmailify

## Overview
Generate a comprehensive unit test suite for the Golang Gmailify application. The goal is to achieve high test coverage (>80%) across configuration parsing, database state management, and the core synchronization logic without making live network calls to Google or external POP3 servers during test execution.

## Testing Philosophy & Tools
* **Framework:** Use Go's standard `testing` package.
* **Assertions:** Use `github.com/stretchr/testify/assert` and `github.com/stretchr/testify/require` for clean, readable assertions.
* **Pattern:** Use Table-Driven Tests for functions with multiple input/output permutations (like config parsing).
* **Mocking:** Use standard Go interfaces to mock external dependencies (POP3 and Gmail clients).

## Component Test Specifications

Please generate the following `_test.go` files alongside their respective implementations.

### 1. Configuration Tests (`config_test.go`)
* **Target:** The function responsible for loading and parsing `config.json`.
* **Test Cases:**
    * Valid configuration file parses correctly into the struct.
    * Missing file returns an expected error.
    * Malformed JSON returns a parsing error.
    * Missing required fields (e.g., no `pop3_server` or `gmail_refresh_token`) trigger validation errors.

### 2. Database Tests (`db/db_test.go`)
* **Target:** `IsSynced` and `MarkSynced` functions.
* **Setup:** Do NOT use a file on disk. Initialize the SQLite connection using the in-memory driver: `sql.Open("sqlite3", ":memory:")`.
* **Test Cases:**
    * Initialize DB verifies the table is created.
    * `IsSynced` returns `false` for a UID that hasn't been seen.
    * `MarkSynced` successfully saves a UID.
    * `IsSynced` returns `true` for a UID immediately after `MarkSynced` is called.
    * `MarkSynced` returns an error (or handles gracefully) if the exact same `account_name` and `pop3_uid` pair is inserted twice (testing the Primary Key constraint).
    * Cross-account isolation: Marking "UID-1" for "Account A" means `IsSynced` should still return `false` for "UID-1" on "Account B".

### 3. Gmail API Tests (`gmail_client/gmail_test.go`)
* **Target:** `PushEmail([]byte) error`.
* **Strategy:** Do not hit the real Google API. We need to verify that the payload is formatted correctly, specifically the Base64 encoding.
* **Implementation Requirement:** Abstract the Gmail API insertion behind an interface (e.g., `type MessageInserter interface { Insert(...) }`) so the test can inject a mock inserter.
* **Test Cases:**
    * Verify that a dummy raw RFC 2822 byte array is correctly encoded using `base64.URLEncoding` (without padding).
    * Verify the mock `Insert` method is called exactly once per `PushEmail` invocation.

### 4. POP3 Fetcher Tests (`pop3_client/pop3_test.go`)
* **Target:** UID fetching and email downloading logic.
* **Strategy:** Abstract the `go-pop3` client behind an interface (e.g., `type POP3Connection interface { Uidl(...) , Retr(...) }`). 
* **Test Cases:**
    * Inject a mock connection that returns a predefined list of UIDs and verify the fetcher parses them correctly.
    * Inject a mock connection that simulates a network error during `Uidl` and ensure the error bubbles up correctly.
    * Verify `Retr` is called with the correct sequence number when downloading an email.

### 5. Core Sync Logic Tests (`main_test.go` or `sync_test.go`)
* **Target:** The single iteration of the sync process (Step 4 of the main spec).
* **Refactoring Requirement:** Ensure the core sync logic (Connect -> Get UIDs -> Check DB -> Download -> Push -> Mark DB) is extracted into a standalone, testable function (e.g., `RunSingleSync(account Config, db DB, pop3 Pop3Client, gmail GmailClient) error`) rather than trapped inside the infinite `select {}` or `for` loop.
* **Test Cases:**
    * **The Happy Path:** Mock POP3 returns 2 UIDs. DB says both are new. Mock POP3 downloads both. Mock Gmail pushes both. DB marks both as synced.
    * **The Partial Sync:** Mock POP3 returns 3 UIDs. DB says 2 are already synced. Mock POP3 only downloads the 1 new email. Mock Gmail pushes 1 email. DB marks 1 as synced.
    * **The Error Path:** If pushing to Gmail fails, ensure `MarkSynced` is **NOT** called for that UID, so it can be retried on the next run.