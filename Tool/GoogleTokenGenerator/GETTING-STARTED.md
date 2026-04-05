# Get Your Credentials
- Go to the Google Cloud Console.
- Create a new project (e.g., "Family Gmailify").
- Go to APIs & Services > Library and enable the Gmail API.
- Go to OAuth consent screen. Choose External, fill in the required fields (just put your email for support), and click through. Do not submit it for verification, but DO hit the "Publish App" button on the summary screen to push it to Production.
- Go to Credentials > Create Credentials > OAuth client ID.
- Select Desktop App as the application type (this is crucial for CLI scripts). Name it something like "Token Generator".
- Click Download JSON and save the file as credentials.json in the same folder where you will write your Go script.