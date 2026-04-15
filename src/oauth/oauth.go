// Package oauth provides utilities for obtaining OAuth2 access tokens.
package oauth

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
)

// microsoftIMAPScopes are the scopes required for Outlook IMAP access.
var microsoftIMAPScopes = []string{
	"offline_access",
	"https://outlook.office.com/IMAP.AccessAsUser.All",
}

// GetMSAccessToken exchanges a Microsoft refresh token for a fresh access token.
// It must be called on every sync cycle because access tokens expire after ~1 hour.
func GetMSAccessToken(clientID, clientSecret, refreshToken string) (string, error) {
	cfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     microsoft.AzureADEndpoint("common"),
		Scopes:       microsoftIMAPScopes,
	}

	existing := &oauth2.Token{
		RefreshToken: refreshToken,
	}

	ts := cfg.TokenSource(context.Background(), existing)
	token, err := ts.Token()
	if err != nil {
		return "", fmt.Errorf("refresh Microsoft OAuth2 token: %w", err)
	}

	return token.AccessToken, nil
}
