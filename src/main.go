package main

import (
	"log"
	"sync"
	"time"

	"jamailify/src/config"
	"jamailify/src/database"
	"jamailify/src/gmail"
	"jamailify/src/pop3"
)

func main() {
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := database.NewDB("sync_state.db")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	var wg sync.WaitGroup
	for _, acc := range cfg.Accounts {
		wg.Add(1)
		go func(account config.Account) {
			defer wg.Done()
			log.Printf("[%s] Starting worker", account.Name)
			// Run first sync immediately
			syncAccount(account, cfg, db)
			// Then run on a ticker
			ticker := time.NewTicker(time.Duration(cfg.PollIntervalMinutes) * time.Minute)
			defer ticker.Stop()
			for range ticker.C {
				syncAccount(account, cfg, db)
			}
		}(acc)
	}

	wg.Wait()
}

func syncAccount(account config.Account, cfg *config.Config, db *database.DB) {
	log.Printf("[%s] Starting sync cycle", account.Name)

	// 1. Connect to POP3
	pop3Client, err := pop3.NewClient(account.Pop3Server, account.Pop3User, account.Pop3Pass)
	if err != nil {
		log.Printf("[%s] ERROR: Failed to connect to POP3 server: %v", account.Name, err)
		return
	}
	defer pop3Client.Close()

	// 2. Get all Messages
	messages, err := pop3Client.ListMessages()
	if err != nil {
		log.Printf("[%s] ERROR: Failed to list messages: %v", account.Name, err)
		return
	}
	log.Printf("[%s] Found %d messages on server", account.Name, len(messages))

	// 3. For each message, check IsSynced
	for _, msg := range messages {
		isSynced, err := db.IsSynced(account.Name, msg.UID)
		if err != nil {
			log.Printf("[%s] ERROR: Failed to check sync status for UID %s: %v", account.Name, msg.UID, err)
			continue
		}

		if isSynced {
			continue // Don't log for every synced message to avoid noise
		}

		// 4. If not synced: Download raw email -> Push via Gmail API -> MarkSynced in DB
		log.Printf("[%s] Found new email with UID: %s", account.Name, msg.UID)
		rawEmail, err := pop3Client.GetMessage(msg.SeqNum)
		if err != nil {
			log.Printf("[%s] ERROR: Failed to download email with UID %s: %v", account.Name, msg.UID, err)
			continue
		}

		gmailClient, err := gmail.NewClient(cfg.GoogleClientID, cfg.GoogleClientSecret, account.GmailRefreshToken)
		if err != nil {
			log.Printf("[%s] ERROR: Failed to create Gmail client: %v", account.Name, err)
			continue
		}

		if err := gmailClient.PushEmail(rawEmail); err != nil {
			log.Printf("[%s] ERROR: Failed to push email with UID %s to Gmail: %v", account.Name, err)
			continue
		}

		if err := db.MarkSynced(account.Name, msg.UID); err != nil {
			log.Printf("[%s] ERROR: Failed to mark email with UID %s as synced: %v", account.Name, msg.UID, err)
			continue
		}

		log.Printf("[%s] Successfully synced and pushed email with UID: %s", account.Name, msg.UID)
	}
	log.Printf("[%s] Sync cycle finished", account.Name)
}