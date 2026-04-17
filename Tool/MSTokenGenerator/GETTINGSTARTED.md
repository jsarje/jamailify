Use this tool to generate a Microsoft `ms_refresh_token` for each Outlook or Hotmail account you want Jamailify to collect mail from.

## Set Up the Azure App Registration

1. Go to the [Azure Portal / Entra ID Admin Center](https://entra.microsoft.com/).
2. Go to **App registrations > New registration**.
3. Name it (e.g., "Family Jamailify").
4. Under **Supported account types**, choose **Accounts in any organizational directory and personal Microsoft accounts (e.g. Skype, Xbox)**.
   This is required — without it, personal Hotmail and Outlook.com accounts will be blocked.
5. Add a **Web** redirect URI with the value `http://localhost:8080/callback`.
6. Go to **Certificates & secrets** and create a **New client secret**. Save the **Value** immediately as it will not be shown again.
7. Go to **API permissions > Add a permission > Microsoft Graph**.
8. Select **Delegated permissions** and add both:
   - `IMAP.AccessAsUser.All`
   - `offline_access`
9. Go to the **Overview** page and save the **Application (client) ID**.
10. Create a `credentials.json` file in this directory with the following format, then run the tool: `go run .`

```json
{
  "client_id": "YOUR_AZURE_CLIENT_ID",
  "client_secret": "YOUR_AZURE_CLIENT_SECRET"
}
```

## Why This Works This Way

- The token generator redeems the auth code from the local Go process, not from a browser-only public client.
- A `Web` redirect plus a client secret matches how this repository exchanges and refreshes Microsoft tokens.
- Microsoft auth codes are short-lived, so the tool uses PKCE and redeems the code immediately after the OAuth callback.