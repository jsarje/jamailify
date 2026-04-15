package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
)

const (
	callbackPort = "8080"
	callbackPath = "/callback"
	redirectURL  = "http://localhost:" + callbackPort + callbackPath
)

// Credentials matches the structure of our JSON file.
type Credentials struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

func randomToken(size int) (string, error) {
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate random token: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
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
	if creds.ClientSecret == "" || creds.ClientSecret == "YOUR_AZURE_CLIENT_SECRET" {
		log.Fatal("Please update credentials.json with your actual Azure Client Secret.")
	}

	// 2. Configure the OAuth2 client
	config := &oauth2.Config{
		ClientID:     creds.ClientID,
		ClientSecret: creds.ClientSecret,
		Endpoint:     microsoft.AzureADEndpoint("common"),
		RedirectURL:  redirectURL,
		Scopes: []string{
			"offline_access", // required to get a refresh token
			"https://outlook.office.com/IMAP.AccessAsUser.All",
		},
	}

	stateToken, err := randomToken(32)
	if err != nil {
		log.Fatalf("Unable to generate OAuth state token: %v", err)
	}
	pckeVerifier := oauth2.GenerateVerifier()

	// 3. Start a local HTTP server to capture the callback automatically
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc(callbackPath, func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "failed to parse callback", http.StatusBadRequest)
			select {
			case errCh <- fmt.Errorf("parse callback form: %w", err):
			default:
			}
			return
		}

		if r.FormValue("state") != stateToken {
			http.Error(w, "state mismatch", http.StatusBadRequest)
			select {
			case errCh <- fmt.Errorf("state parameter mismatch"):
			default:
			}
			return
		}

		code := r.FormValue("code")
		if code == "" {
			msg := r.FormValue("error_description")
			if msg == "" {
				msg = r.FormValue("error")
			}
			http.Error(w, "no code received", http.StatusBadRequest)
			select {
			case errCh <- fmt.Errorf("no code in callback: %s", msg):
			default:
			}
			return
		}

		fmt.Fprintln(w, "Authorization successful! You can close this tab.")
		select {
		case codeCh <- code:
		default:
		}
	})

	srv := &http.Server{Addr: ":" + callbackPort, Handler: mux}
	go func() {
		if listenErr := srv.ListenAndServe(); listenErr != nil && listenErr != http.ErrServerClosed {
			errCh <- fmt.Errorf("HTTP server error: %w", listenErr)
		}
	}()

	// 4. Print the auth URL for the user to open
	authURL := config.AuthCodeURL(
		stateToken,
		oauth2.AccessTypeOffline,
		oauth2.S256ChallengeOption(pckeVerifier),
		oauth2.SetAuthURLParam("response_mode", "form_post"),
	)
	fmt.Println("=====================================================================")
	fmt.Printf("Open this URL in your browser:\n\n%s\n\n", authURL)
	fmt.Println("=====================================================================")
	fmt.Println("Log in and grant permissions. The token will be captured automatically.")
	fmt.Println("Waiting for callback on http://localhost:" + callbackPort + callbackPath + " ...")

	// 5. Wait for the auth code or an error
	var authCode string
	select {
	case authCode = <-codeCh:
	case err = <-errCh:
		log.Fatalf("Callback error: %v", err)
	}

	_ = srv.Shutdown(context.Background())

	// 6. Exchange the auth code for tokens
	tok, err := config.Exchange(context.Background(), authCode, oauth2.VerifierOption(pckeVerifier))
	if err != nil {
		log.Fatalf("Failed to exchange auth code for tokens: %v", err)
	}

	// 7. Print the refresh token
	fmt.Println("\nSUCCESS! Here is your token data:")
	fmt.Println("---------------------------------------------------------------------")
	fmt.Printf("Refresh Token: %s\n", tok.RefreshToken)
	fmt.Println("---------------------------------------------------------------------")
	fmt.Println("Copy this Refresh Token into your main app's config file.")
}
