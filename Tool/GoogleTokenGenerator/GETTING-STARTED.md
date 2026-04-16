# Get Your Credentials
- Go to the Google Cloud Console.
- Create a new project (e.g., "Family Gmailify").
- Go to APIs & Services > Library and enable the Gmail API.
- Go to OAuth consent screen. Choose External, fill in the required fields (just put your email for support), and click through. Do not submit it for verification, but DO hit the "Publish App" button on the summary screen to push it to Production.
- Go to Credentials > Create Credentials > OAuth client ID.
- Select Desktop App as the application type (this is crucial for CLI scripts). Name it something like "Token Generator".
- Click Download JSON and save the file as credentials.json in the same folder where you will write your Go script.

## Required Scopes
- jamailify needs both Gmail insert access and Gmail readonly access.
- `gmail.insert` is the OAuth scope used by the Gmail `users.messages.import` endpoint.
- `gmail.readonly` is used to search Gmail by `Message-ID` before import to avoid duplicates.
- You cannot add scopes to an existing refresh token after it has been issued. If the token was created with the wrong scopes, generate a new refresh token.

## Important Notes
- The refresh token must be created from the same Google OAuth client ID and client secret that jamailify uses in [config/config.json](c:/GiT/jamailify/config/config.json).
- If Google keeps returning `invalid_grant`, the refresh token is usually revoked, expired, tied to a different OAuth client, or was minted before the app was in production mode.
- If needed, remove the app from the Google account first: Google Account > Security > Third-party apps with account access, then re-run the token generator and consent again.