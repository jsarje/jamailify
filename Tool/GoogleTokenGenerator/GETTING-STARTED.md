Use this tool to generate a Google `gmail_refresh_token` for each Gmail account you want Jamailify to deliver mail into.

## Set Up the Google Cloud Project

1. Go to the [Google Cloud Console](https://console.cloud.google.com/).
2. Create a new project (e.g., "Family Jamailify").
3. Go to **APIs & Services > Library** and enable the **Gmail API**.
4. Go to **OAuth consent screen**. Choose **External**, fill in the required fields, and click through.
   Do not submit it for verification, but DO hit **Publish App** on the summary screen to push it to Production.
5. Go to **Credentials > Create Credentials > OAuth client ID**.
6. Select **Desktop App** as the application type. Name it something like "Token Generator".
7. Click **Download JSON** and save the file as `credentials.json` in this directory.
8. Run the tool: `go run .`
9. Follow the browser prompt to consent. The resulting `gmail_refresh_token` will be printed to the terminal.

## Required OAuth Scopes

Jamailify requests two scopes during the OAuth consent flow:

- `gmail.insert` — used by the Gmail `users.messages.import` endpoint to insert messages.
- `gmail.readonly` — used to search Gmail by `Message-ID` before import to avoid duplicates.

> [!note]
> You cannot add scopes to an existing refresh token after it has been issued. If the token was created with the wrong scopes, generate a new one.

## Important Notes

- The refresh token must be created from the same Google OAuth client ID and client secret that Jamailify uses in `config/config.json`.
- If Google returns `invalid_grant`, the refresh token is usually revoked, expired, tied to a different OAuth client, or was minted before the app was in Production mode.
- To reset consent, remove the app from the Google account first: **Google Account > Security > Third-party apps with account access**, then re-run the token generator and consent again.