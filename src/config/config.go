package config

import (
	"encoding/json"
	"os"
)

type Account struct {
	Name              string `json:"name"`
	Pop3Server        string `json:"pop3_server"`
	Pop3User          string `json:"pop3_user"`
	Pop3Pass          string `json:"pop3_pass"`
	GmailRefreshToken string `json:"gmail_refresh_token"`
}

type Config struct {
	PollIntervalMinutes int       `json:"poll_interval_minutes"`
	GoogleClientID      string    `json:"google_client_id"`
	GoogleClientSecret  string    `json:"google_client_secret"`
	Accounts            []Account `json:"accounts"`
}

func LoadConfig(path string) (*Config, error) {
	configFile, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	err = json.Unmarshal(configFile, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}