package main

import (
	"bytes"
	"context"
	"log"
	"net"
	"net/mail"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"time"

	"jamailify/src/config"
	"jamailify/src/database"
	"jamailify/src/gmail"
	"jamailify/src/pop3"
)

type DBOperations interface {
	IsSynced(accountName, uid string) (bool, error)
	MarkSynced(accountName, uid string) error
}

type Pop3Operations interface {
	Stat() (int, error)
	UIDLForSeq(seqNum int) (string, error)
	TopMessage(seqNum int) ([]byte, error)
	GetMessage(seqNum int) ([]byte, error)
	Close() error
}

type GmailOperations interface {
	PushEmail(ctx context.Context, rawEmail []byte) error
}

func RunSingleSync(ctx context.Context, account config.Account, cfg *config.Config, db DBOperations, pop3Client Pop3Operations, gmailClient GmailOperations) {
	log.Printf("[%s] Starting sync cycle", account.Name)
	// We'll only sync messages from the last 7 days. Iterate newest->oldest
	// determine effective cutoff and maxToCheck from config (defaults if zero)
	windowDays := cfg.SyncWindowDays
	if windowDays <= 0 {
		windowDays = 7
	}
	cutoff := time.Now().Add(time.Duration(-windowDays) * 24 * time.Hour)

	maxToCheck := cfg.MaxMessagesToCheck
	if maxToCheck <= 0 {
		maxToCheck = 2000 // safety cap to avoid scanning huge mailboxes
	}

	count, err := pop3Client.Stat()
	if err != nil {
		log.Printf("[%s] ERROR: Failed to stat mailbox: %v", account.Name, err)
		return
	}
	log.Printf("[%s] Mailbox has %d messages; checking up to %d newest", account.Name, count, maxToCheck)

	startSeq := 1
	if count > maxToCheck {
		startSeq = count - maxToCheck + 1
	}

	for seq := count; seq >= startSeq; seq-- {
		uid, err := pop3Client.UIDLForSeq(seq)
		if err != nil {
			log.Printf("[%s] WARN: unable to get UID for seq %d: %v", account.Name, seq, err)
			continue
		}

		isSynced, err := db.IsSynced(account.Name, uid)
		if err != nil {
			log.Printf("[%s] ERROR: Failed to check sync status for UID %s: %v", account.Name, uid, err)
			continue
		}
		if isSynced {
			continue
		}

		// Fetch headers only. If TOP is unsupported by the server, fall back to RETR and parse headers.
		hdrs, err := pop3Client.TopMessage(seq)
		if err != nil {
			log.Printf("[%s] WARN: TOP failed for seq %d (UID %s): %v — falling back to RETR", account.Name, seq, uid, err)
			raw, err2 := pop3Client.GetMessage(seq)
			if err2 != nil {
				log.Printf("[%s] WARN: RETR fallback failed for seq %d (UID %s): %v", account.Name, seq, uid, err2)
				continue
			}
			hdrs = raw
		}
		mr, err := mail.ReadMessage(bytes.NewReader(hdrs))
		if err != nil {
			log.Printf("[%s] WARN: failed to read headers for seq %d: %v", account.Name, seq, err)
			continue
		}
		dateStr := mr.Header.Get("Date")
		if dateStr == "" {
			log.Printf("[%s] WARN: no Date header for seq %d (UID %s), skipping", account.Name, seq, uid)
			continue
		}
		t, err := mail.ParseDate(dateStr)
		if err != nil {
			log.Printf("[%s] WARN: cannot parse Date header %q for seq %d: %v", account.Name, dateStr, seq, err)
			continue
		}
		if t.Before(cutoff) {
			// since we're iterating newest->oldest, safe to stop
			break
		}

		// 4. If not synced and within cutoff: Download raw email -> Push via Gmail API -> MarkSynced in DB
		log.Printf("[%s] Found new email with UID: %s", account.Name, uid)
		rawEmail, err := pop3Client.GetMessage(seq)
		if err != nil {
			log.Printf("[%s] ERROR: Failed to download email seq %d UID %s: %v", account.Name, seq, uid, err)
			continue
		}

		if err := gmailClient.PushEmail(ctx, rawEmail); err != nil {
			log.Printf("[%s] ERROR: Failed to push email with UID %s to Gmail: %v", account.Name, uid, err)
			continue
		}

		if err := db.MarkSynced(account.Name, uid); err != nil {
			log.Printf("[%s] ERROR: Failed to mark email with UID %s as synced: %v", account.Name, uid, err)
			continue
		}

		log.Printf("[%s] Successfully synced and pushed email with UID: %s", account.Name, uid)
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
				host, portStr, err := net.SplitHostPort(account.Pop3Server)
				if err != nil {
					log.Printf("[%s] ERROR: Invalid POP3 server address: %v", account.Name, err)
					return
				}
				port, err := strconv.Atoi(portStr)
				if err != nil {
					log.Printf("[%s] ERROR: Invalid POP3 port: %v", account.Name, err)
					return
				}

				pop3Client, err := pop3.NewClient(host, port, account.Pop3User, account.Pop3Pass, true)
				if err != nil {
					log.Printf("[%s] ERROR: Failed to connect to POP3 server: %v", account.Name, err)
					return
				}
				defer pop3Client.Close()

				gmailClient, err := gmail.NewClient(ctx, cfg.GoogleClientID, cfg.GoogleClientSecret, account.GmailRefreshToken)
				if err != nil {
					log.Printf("[%s] ERROR: Failed to create Gmail client: %v", account.Name, err)
					return
				}

				RunSingleSync(ctx, account, cfg, db, pop3Client, gmailClient)
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
