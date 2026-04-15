1.  Go to the [Azure Portal / Entra ID Admin Center](https://entra.microsoft.com/).
2.  Go to **App registrations** -> **New registration**.
3.  Name it (e.g., "Family Gmailify").
4.  **Crucial Step:** Under "Supported account types", choose **Accounts in any organizational directory and personal Microsoft accounts (e.g. Skype, Xbox)**. If you don't choose this, personal Hotmail accounts will be blocked.
5.  Add a **Web** redirect URI with the value `http://localhost:8080/callback`.
6.  Go to **Certificates & secrets** and create a **New client secret**. Save the **Value**.
7.  Go to Microsoft Graph.
8.  Select **Delegated permissions**, check `IMAP.AccessAsUser.All`, and add it. (You do *not* need admin consent since it's just for you).
   Also select offline_access
9.  Go to the **Overview** page and save the **Application (client) ID**.

Why this matters

- The token generator redeems the auth code from the local Go process, not from a browser-only public client.
- A `Web` redirect plus a client secret matches how this repository exchanges and refreshes Microsoft tokens.
- Microsoft auth codes are short-lived, so the tool now uses PKCE and redeems the code immediately after the callback.

credentials.json format
{
  "client_id": "YOUR_AZURE_CLIENT_ID",
  "client_secret": "YOUR_AZURE_CLIENT_SECRET"
}