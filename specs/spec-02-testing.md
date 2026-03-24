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

## 4. POP3 Integration Testing (`pop3_client/pop3_integration_test.go`)

### Overview
Skip unit testing with mocks for the POP3 client. Instead, implement an integration test using Docker to verify the POP3 fetching logic against a real, running mail server.

### Tools & Libraries
* **Container Management:** Use `github.com/testcontainers/testcontainers-go` to programmatically manage the Docker container lifecycle within the Go test.
* **Test Mail Server Image:** Use the `greenmail/standalone:latest` Docker image. This is a lightweight, open-source test mail server that provides both POP3 and SMTP out of the box without complex configuration.

### Integration Test Lifecycle & Setup
1. **Setup:** * Use `testcontainers.GenericContainer` to spin up `greenmail/standalone`.
   * Map the container's default POP3 port (3110) and SMTP port (3025) to random available ports on the host to avoid CI/CD port conflicts.
   * Configure GreenMail with a test user via environment variables (e.g., `GREENMAIL_OPTS=-Dgreenmail.setup.test.all -Dgreenmail.users=testuser:testpass@example.com`).
2. **Seed Data (SMTP):**
   * Write a helper function in the test that connects to the mapped SMTP port using standard Go `net/smtp`.
   * Send 2-3 raw RFC 2822 dummy emails to the `testuser@example.com` inbox.
3. **Execution (POP3):**
   * Instantiate the `pop3_client` using the mapped POP3 port and the test credentials.
   * Call the function to fetch UIDs.
   * Call the function to download the raw emails.
4. **Assertions:**
   * Assert that the correct number of UIDs were returned.
   * Assert that the downloaded raw email bytes match the payloads that were injected via SMTP.
5. **Teardown:**
   * Use `defer container.Terminate(ctx)` to ensure the Docker container is destroyed immediately after the test finishes, even if it fails.

### Refactoring Note for AI
Ensure the `pop3_client` is built to accept dynamic hosts and ports rather than hardcoding port 995 or forcing TLS, so that it can connect to the unencrypted local GreenMail container during testing, but use TLS in production based on the `config.json`.

### 5. Core Sync Logic Tests (`main_test.go` or `sync_test.go`)
* **Target:** The single iteration of the sync process (Step 4 of the main spec).
* **Refactoring Requirement:** Ensure the core sync logic (Connect -> Get UIDs -> Check DB -> Download -> Push -> Mark DB) is extracted into a standalone, testable function (e.g., `RunSingleSync(account Config, db DB, pop3 Pop3Client, gmail GmailClient) error`) rather than trapped inside the infinite `select {}` or `for` loop.
* **Test Cases:**
    * **The Happy Path:** Mock POP3 returns 2 UIDs. DB says both are new. Mock POP3 downloads both. Mock Gmail pushes both. DB marks both as synced.
    * **The Partial Sync:** Mock POP3 returns 3 UIDs. DB says 2 are already synced. Mock POP3 only downloads the 1 new email. Mock Gmail pushes 1 email. DB marks 1 as synced.
        * **The Error Path:** If pushing to Gmail fails, ensure `MarkSynced` is **NOT** called for that UID, so it can be retried on the next run.
    
    ## 6. IMAP Integration Testing (`imap_client/imap_integration_test.go`)
    
    ### Overview
    Skip unit testing with mocks for the IMAP client. Instead, implement an integration test using Docker to verify the IMAP fetching logic against a real, running mail server.
    
    ### Tools & Libraries
    * **Container Management:** Use `github.com/testcontainers/testcontainers-go` to programmatically manage the Docker container lifecycle within the Go test.
    * **Test Mail Server Image:** Use the `greenmail/standalone:latest` Docker image. This is a lightweight, open-source test mail server that provides both IMAP and SMTP out of the box without complex configuration.
    
    ### Integration Test Lifecycle & Setup
    1. **Setup:**
       * Use `testcontainers.GenericContainer` to spin up `greenmail/standalone`.
       * Map the container's default IMAP port (3143) and SMTP port (3025) to random available ports on the host to avoid CI/CD port conflicts.
       * Configure GreenMail with a test user via environment variables (e.g., `GREENMAIL_OPTS=-Dgreenmail.setup.test.all -Dgreenmail.users=testuser:testpass@example.com`).
    2. **Seed Data (SMTP):**
       * Write a helper function in the test that connects to the mapped SMTP port using standard Go `net/smtp`.
       * Send 2-3 raw RFC 2822 dummy emails to the `testuser@example.com` inbox.
    3. **Execution (IMAP):**
       * Instantiate the `imap_client` using the mapped IMAP port and the test credentials.
       * Call the function to fetch UIDs.
       * Call the function to download the raw emails.
    4. **Assertions:**
       * Assert that the correct number of UIDs were returned.
       * Assert that the downloaded raw email bytes match the payloads that were injected via SMTP.
    5. **Teardown:**
       * Use `defer container.Terminate(ctx)` to ensure the Docker container is destroyed immediately after the test finishes, even if it fails.
    
    ### Refactoring Note for AI
    Ensure the `imap_client` is built to accept dynamic hosts and ports rather than hardcoding port 993 or forcing TLS, so that it can connect to the unencrypted local GreenMail container during testing, but use TLS in production based on the `config.json`.
    