package main

import (
	"context"
	"log"
	"net"
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
	ListMessages() ([]pop3.MessageInfo, error)
	GetMessage(seqNum int) ([]byte, error)
	Close() error
}

type GmailOperations interface {
	PushEmail(ctx context.Context, rawEmail []byte) error
}

func RunSingleSync(ctx context.Context, account config.Account, cfg *config.Config, db DBOperations, pop3Client Pop3Operations, gmailClient GmailOperations) {
	log.Printf("[%s] Starting sync cycle", account.Name)

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

		if err := gmailClient.PushEmail(ctx, rawEmail); err != nil {
			log.Printf("[%s] ERROR: Failed to push email with UID %s to Gmail: %v", account.Name, msg.UID, err)
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
