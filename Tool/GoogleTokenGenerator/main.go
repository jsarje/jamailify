package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
)

func main() {
	// 1. Read the credentials.json file downloaded from Google Cloud
	b, err := os.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// 2. Parse the credentials into an OAuth2 config
	// We are requesting the 'gmail.insert' scope to bypass filters, or 'gmail.modify'
	config, err := google.ConfigFromJSON(b, gmail.GmailInsertScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}

	// 3. Generate the Auth URL
	// AccessTypeOffline is REQUIRED to get a refresh token
	// ApprovalForce forces the consent screen so a refresh token is always returned
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline, oauth2.ApprovalForce)

	fmt.Println("=====================================================================")
	fmt.Printf("Go to the following link in your browser:\n\n%v\n\n", authURL)
	fmt.Println("=====================================================================")
	fmt.Println("Log in with the Gmail account you want to push emails TO.")
	fmt.Println("Ignore the 'Google hasn't verified this app' warning (click Advanced -> Continue).")
	fmt.Println("Type the authorization code here (will be in the query string of URL) and press Enter:")

	// 4. Get the auth code from the user
	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	// 5. Exchange the auth code for access and refresh tokens
	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}

	// 6. Print the prize!
	fmt.Println("\n🎉 SUCCESS! Here is your token data:")
	fmt.Println("---------------------------------------------------------------------")
	fmt.Printf("Refresh Token: %s\n", tok.RefreshToken)
	fmt.Println("---------------------------------------------------------------------")
	fmt.Println("Copy that Refresh Token into your config.json.")
}
