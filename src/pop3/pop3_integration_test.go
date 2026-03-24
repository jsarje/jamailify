package pop3

import (
	"context"
	"fmt"
	"jamailify/src/config"
	"log"
	"net"
	"net/smtp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestPop3Integration(t *testing.T) {
	ctx := context.Background()

	// 1. Setup
	req := testcontainers.ContainerRequest{
		Image:        "greenmail/standalone:latest",
		ExposedPorts: []string{"3110/tcp", "3025/tcp"},
		Env: map[string]string{
			"GREENMAIL_OPTS": "-Dgreenmail.setup.test.all -Dgreenmail.users=testuser:testpass@example.com -Dgreenmail.bindAddress=0.0.0.0 -Dgreenmail.hostname=0.0.0.0",
		},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort("3110/tcp"),
			wait.ForListeningPort("3025/tcp"),
		).WithDeadline(60 * time.Second),
	}
	greenmailContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	defer func() {
		if err := greenmailContainer.Terminate(ctx); err != nil {
			log.Fatalf("failed to terminate container: %s", err.Error())
		}
	}()

	pop3Port, err := greenmailContainer.MappedPort(ctx, "3110/tcp")
	require.NoError(t, err)
	smtpPort, err := greenmailContainer.MappedPort(ctx, "3025/tcp")
	require.NoError(t, err)
	host, err := greenmailContainer.Host(ctx)
	require.NoError(t, err)

	// 2. Seed Data (SMTP)
	if err := sendMail(t, fmt.Sprintf("%s:%d", host, smtpPort.Int()), "testuser@example.com", "Test Email 1", "This is the first email."); err != nil {
		require.NoError(t, err)
	}
	if err := sendMail(t, fmt.Sprintf("%s:%d", host, smtpPort.Int()), "testuser@example.com", "Test Email 2", "This is the second email."); err != nil {
		require.NoError(t, err)
	}

	// Allow some time for mail to be processed
	time.Sleep(2 * time.Second)

	// 3. Execution (POP3)
	pop3Client, err := NewPOP3Client(&config.Account{
		Server: fmt.Sprintf("%s:%d", host, pop3Port.Int()),
		User:   "testuser",
		Pass:   "testpass",
		NoTls:  true,
	})
	require.NoError(t, err)

	err = pop3Client.Connect()
	require.NoError(t, err)
	defer pop3Client.Close()

	uids, err := pop3Client.GetUIDs()
	require.NoError(t, err)

	// 4. Assertions
	assert.Len(t, uids, 2, "Should have fetched 2 UIDs")

	if len(uids) == 2 {
		var email1Found, email2Found bool
		for _, uid := range uids {
			retrievedEmail, err := pop3Client.DownloadEmail(uid)
			require.NoErrorf(t, err, "failed to RETR uid=%s", uid)
			if strings.Contains(string(retrievedEmail), "Subject: Test Email 1") {
				assert.Contains(t, string(retrievedEmail), "This is the first email.")
				email1Found = true
			}
			if strings.Contains(string(retrievedEmail), "Subject: Test Email 2") {
				assert.Contains(t, string(retrievedEmail), "This is the second email.")
				email2Found = true
			}
		}
		assert.True(t, email1Found, "Test Email 1 not found")
		assert.True(t, email2Found, "Test Email 2 not found")
	}
}

func sendMail(t *testing.T, addr, to, subject, body string) error {
	// use smtp client methods (connect, MAIL, RCPT, DATA) for more robust interaction
	c, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("smtp dial failed: %w", err)
	}
	defer c.Close()

	// Use EHLO/HELO and authenticate with PlainAuth using the test credentials
	if err := c.Hello("localhost"); err != nil {
		return fmt.Errorf("smtp hello failed: %w", err)
	}

	hostOnly, _, err := net.SplitHostPort(addr)
	if err != nil {
		// fallback: use addr as host if SplitHostPort fails
		hostOnly = addr
	}

	auth := smtp.PlainAuth("", "testuser", "testpass", hostOnly)
	if err := c.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth failed: %w", err)
	}

	if err := c.Mail("sender@example.com"); err != nil {
		return fmt.Errorf("smtp mail from failed: %w", err)
	}

	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt to failed: %w", err)
	}

	wc, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp data failed: %w", err)
	}

	msg := fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s\r\n", to, subject, body)
	if _, err := wc.Write([]byte(msg)); err != nil {
		return fmt.Errorf("writing smtp data failed: %w", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("closing smtp data failed: %w", err)
	}

	if err := c.Quit(); err != nil {
		return fmt.Errorf("smtp quit failed: %w", err)
	}
	return nil
}
