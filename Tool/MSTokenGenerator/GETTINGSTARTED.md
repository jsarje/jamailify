1.  Go to the [Azure Portal / Entra ID Admin Center](https://entra.microsoft.com/).
2.  Go to **App registrations** -> **New registration**.
3.  Name it (e.g., "Family Gmailify").
4.  **Crucial Step:** Under "Supported account types", choose **Accounts in any organizational directory and personal Microsoft accounts (e.g. Skype, Xbox)**. If you don't choose this, personal Hotmail accounts will be blocked.
5.  Set the Redirect URI to `Mobile and desktop applications` with the value `http://localhost`.
6.  Go to **API Permissions**, click "Add a permission" -> "APIs my organization uses" -> search for `Office 365 Exchange Online`.
7.  Select **Delegated permissions**, check `IMAP.AccessAsUser.All`, and add it. (You do *not* need admin consent since it's just for you).
8.  Go to **Certificates & secrets** and create a New client secret. Save the **Value**.
9.  Go to the **Overview** page and save the **Application (client) ID**.

credentials.json format
{
  "client_id": "YOUR_AZURE_CLIENT_ID",
  "client_secret": "YOUR_AZURE_CLIENT_SECRET"
}