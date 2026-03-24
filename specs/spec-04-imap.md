# Feature Specification: IMAP Integration & Dual-Protocol Support

## Overview
Expand the self-hosted jamailify application to support both IMAP and POP3 protocols for fetching source emails. The user must be able to specify either protocol on a per-account basis in the configuration file. The core synchronization loop will be refactored to use a common interface so it functions identically regardless of the underlying protocol.

## 1. Required Third-Party Libraries
* Add IMAP Client: `github.com/emersion/go-imap`
* (Keep existing `github.com/knadh/go-pop3`)

## 2. Configuration Schema Updates
Update the configuration parser to handle a new `protocol` flag and distinct fields for IMAP and POP3. 

The application must parse this updated `config.json` structure:
```json
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