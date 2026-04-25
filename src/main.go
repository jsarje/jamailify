package main

import (
	"bytes"
	"context"
	"log"
	"net/mail"
	"os"
	"os/signal"
	"sync"
	"time"

	"jamailify/src/config"
	"jamailify/src/database"
	"jamailify/src/fetcher"
	"jamailify/src/gmail"
	"jamailify/src/imap"
	"jamailify/src/pop3"
)

type DBOperations interface {
	IsSynced(accountName, uid string) (bool, error)
	MarkSynced(accountName, uid string) error
}

type GmailOperations interface {
	PushEmail(ctx context.Context, rawEmail []byte) (*gmail.PushResult, error)
	MessageIdExists(ctx context.Context, messageId string) (bool, error)
}

func RunSingleSync(ctx context.Context, account config.Account, cfg *config.Config, db DBOperations, emailFetcher fetcher.EmailFetcher, gmailClient GmailOperations) {
	log.Printf("[%s] Starting sync cycle", account.Name)
	log.Printf("[%s] preserve_original_timestamps=%t", account.Name, cfg.PreserveOriginalTimestamps)
	// We'll only sync messages from the last 7 days. Iterate newest->oldest
	// determine effective cutoff and maxToCheck from config (defaults if zero)
	windowDays := cfg.SyncWindowDays
	if windowDays <= 0 {
		windowDays = 7
	}
	cutoff := time.Now().Add(time.Duration(-windowDays) * 24 * time.Hour)

	uids, err := emailFetcher.GetUIDs()
	if err != nil {
		log.Printf("[%s] ERROR: Failed to get UIDs: %v", account.Name, err)
		return
	}

	maxToCheck := cfg.MaxMessagesToCheck
	if maxToCheck <= 0 {
		maxToCheck = 2000 // safety cap to avoid scanning huge mailboxes
	}
	if len(uids) > maxToCheck {
		uids = uids[len(uids)-maxToCheck:]
	}

	log.Printf("[%s] Found %d UIDs; checking up to %d newest", account.Name, len(uids), maxToCheck)

	for i := len(uids) - 1; i >= 0; i-- {
		uid := uids[i]

		isSynced, err := db.IsSynced(account.Name, uid)
		if err != nil {
			log.Printf("[%s] ERROR: Failed to check sync status for UID %s: %v", account.Name, uid, err)
			continue
		}
		if isSynced {
			continue
		}

		// Fetch headers only to check the date.
		hdrs, err := emailFetcher.DownloadEmailHeaders(uid)
		if err != nil {
			log.Printf("[%s] WARN: Failed to download headers for UID %s: %v", account.Name, uid, err)
			continue
		}
		mr, err := mail.ReadMessage(bytes.NewReader(hdrs))
		if err != nil {
			log.Printf("[%s] WARN: failed to read headers for UID %s: %v", account.Name, uid, err)
			continue
		}
		dateStr := mr.Header.Get("Date")
		if dateStr == "" {
			log.Printf("[%s] WARN: no Date header for UID %s, skipping", account.Name, uid)
			continue
		}
		t, err := mail.ParseDate(dateStr)
		if err != nil {
			log.Printf("[%s] WARN: cannot parse Date header %q for UID %s: %v", account.Name, dateStr, uid, err)
			continue
		}
		if t.Before(cutoff) {
			// since we're iterating newest->oldest, safe to stop
			break
		}

		// Check for duplicates in Gmail using Message-ID
		messageID := mr.Header.Get("Message-ID")
		if messageID != "" {
			exists, err := gmailClient.MessageIdExists(ctx, messageID)
			if err != nil {
				log.Printf("[%s] ERROR: Failed to check for existing Message-ID %s: %v", account.Name, messageID, err)
				// a failure here isn't critical, we can proceed with the sync
			}
			if exists {
				log.Printf("[%s] INFO: Message with Message-ID %s already exists in Gmail, marking as synced", account.Name, messageID)
				if err := db.MarkSynced(account.Name, uid); err != nil {
					log.Printf("[%s] ERROR: Failed to mark email with UID %s as synced: %v", account.Name, uid, err)
				}
				continue
			}
		}

		// If not synced and within cutoff: Download raw email -> Push via Gmail API -> MarkSynced in DB
		log.Printf("[%s] Found new email with UID: %s", account.Name, uid)
		rawEmail, err := emailFetcher.DownloadEmail(uid)
		if err != nil {
			log.Printf("[%s] ERROR: Failed to download email UID %s: %v", account.Name, uid, err)
			continue
		}

		pushResult, err := gmailClient.PushEmail(ctx, rawEmail)
		if err != nil {
			log.Printf("[%s] ERROR: Failed to push email with UID %s to Gmail: %v", account.Name, uid, err)
			continue
		}

		if err := db.MarkSynced(account.Name, uid); err != nil {
			log.Printf("[%s] ERROR: Failed to mark email with UID %s as synced: %v", account.Name, uid, err)
			continue
		}

		log.Printf("[%s] Successfully synced and pushed email with UID: %s (gmail_message_id=%s labels=%v internal_date=%d)", account.Name, uid, pushResult.MessageID, pushResult.LabelIDs, pushResult.InternalDate)
	}
	log.Printf("[%s] Sync cycle finished", account.Name)
}

func main() {
	cfg, err := config.LoadConfig("/app/config/config.json")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := database.NewDB("/app/data/sync_state.db")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create a cancellable root context and handle OS signals for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	go func() {
		<-sigs
		log.Printf("Received interrupt, shutting down")
		cancel()
	}()

	var wg sync.WaitGroup
	for _, acc := range cfg.Accounts {
		wg.Add(1)
		go func(account config.Account) {
			defer wg.Done()
			log.Printf("[%s] Starting worker", account.Name)

			ticker := time.NewTicker(time.Duration(cfg.PollIntervalMinutes) * time.Minute)
			defer ticker.Stop()

			// helper to run one sync cycle, returns when ctx is done
			runOnce := func() {
				var emailFetcher fetcher.EmailFetcher
				var err error
				switch account.Protocol {
				case "pop3":
					emailFetcher, err = pop3.NewPOP3Client(&account)
				case "imap":
					emailFetcher, err = imap.NewIMAPClient(&account)
				default:
					log.Printf("[%s] ERROR: Unknown protocol: %s", account.Name, account.Protocol)
					return
				}
				if err != nil {
					log.Printf("[%s] ERROR: Failed to create email client: %v", account.Name, err)
					return
				}

				if err := emailFetcher.Connect(); err != nil {
					log.Printf("[%s] ERROR: Failed to connect to email server: %v", account.Name, err)
					return
				}
				defer emailFetcher.Close()

				gmailClient, err := gmail.NewClient(ctx, cfg.GoogleClientID, cfg.GoogleClientSecret, account.GmailRefreshToken, cfg.GmailFetchMetadataAfterImport, cfg.PreserveOriginalTimestamps)
				if err != nil {
					log.Printf("[%s] ERROR: Failed to create Gmail client: %v", account.Name, err)
					return
				}

				RunSingleSync(ctx, account, cfg, db, emailFetcher, gmailClient)
			}

			// First immediate run
			runOnce()

			for {
				select {
				case <-ctx.Done():
					log.Printf("[%s] Worker received shutdown", account.Name)
					return
				case <-ticker.C:
					runOnce()
				}
			}
		}(acc)
	}

	wg.Wait()
}
