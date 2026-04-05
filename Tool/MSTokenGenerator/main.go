package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
)

// Credentials matches the structure of our JSON file
type Credentials struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

func main() {
	// 1. Read and parse the credentials.json file
	b, err := os.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read credentials.json: %v\nMake sure the file exists in the same directory.", err)
	}

	var creds Credentials
	if err := json.Unmarshal(b, &creds); err != nil {
		log.Fatalf("Unable to parse credentials.json: %v\nEnsure it is valid JSON.", err)
	}

	if creds.ClientID == "" || creds.ClientID == "YOUR_AZURE_CLIENT_ID" {
		log.Fatal("Please update credentials.json with your actual Azure Client ID and Secret.")
	}

	// 2. Configure the OAuth2 client using the loaded credentials
	// "common" allows both personal (Hotmail/Outlook) and organizational/school accounts.
	config := &oauth2.Config{
		ClientID:     creds.ClientID,
		ClientSecret: creds.ClientSecret,
		Endpoint:     microsoft.AzureADEndpoint("common"),
		RedirectURL:  "http://localhost",
		Scopes: []string{
			"offline_access", // REQUIRED to get a refresh token
			"https://outlook.office.com/IMAP.AccessAsUser.All",
		},
	}

	// 3. Generate the Auth URL
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)

	fmt.Println("=====================================================================")
	fmt.Printf("Go to the following link in your browser:\n\n%v\n\n", authURL)
	fmt.Println("=====================================================================")
	fmt.Println("Log in with your Microsoft/Outlook account and grant permissions.")
	fmt.Println("Your browser will redirect you to a broken 'localhost' page. That is normal!")
	fmt.Println("Copy the ENTIRE URL from your browser's address bar and paste it below:")
	fmt.Print("\nPaste URL here: ")

	// 4. Read the full redirected URL from the user
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	redirectedURL := strings.TrimSpace(scanner.Text())

	// 5. Parse the URL to extract the "code" parameter
	u, err := url.Parse(redirectedURL)
	if err != nil {
		log.Fatalf("Failed to parse the URL you pasted: %v", err)
	}

	authCode := u.Query().Get("code")
	if authCode == "" {
		log.Fatal("Could not find an authorization 'code' in the URL. Did you paste the whole thing?")
	}

	// 6. Exchange the auth code for access and refresh tokens
	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Failed to exchange auth code for tokens: %v", err)
	}

	// 7. Print the final Refresh Token
	fmt.Println("\n🎉 SUCCESS! Here is your token data:")
	fmt.Println("---------------------------------------------------------------------")
	fmt.Printf("Refresh Token: %s\n", tok.RefreshToken)
	fmt.Println("---------------------------------------------------------------------")
	fmt.Println("Copy this Refresh Token into your main app's config file.")
}
