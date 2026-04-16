package imap

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"reflect"
	"strings"
	"testing"

	"jamailify/src/config"

	imaplib "github.com/emersion/go-imap"
	imapclient "github.com/emersion/go-imap/client"
	"github.com/emersion/go-sasl"
)

func TestNormalizeExchangeResponseLine(t *testing.T) {
	input := []byte("A2 OK [Error=\"UserDisabled\" AuthResult=27 Proxy=LO2P123MB7360.GBRP123.PROD.OUTLOOK.COM:1993:SSL Service=Imap4] AUTHENTICATE completed.\r\n")
	got := string(normalizeExchangeResponseLine(input))
	want := "A2 OK [Error=UserDisabled AuthResult=27 Proxy=LO2P123MB7360.GBRP123.PROD.OUTLOOK.COM:1993:SSL Service=Imap4] AUTHENTICATE completed.\r\n"
	if got != want {
		t.Fatalf("normalizeExchangeResponseLine() = %q, want %q", got, want)
	}

	resp, err := imaplib.ReadResp(imaplib.NewReader(bufio.NewReader(strings.NewReader(got))))
	if err != nil {
		t.Fatalf("ReadResp() error = %v", err)
	}

	status, ok := resp.(*imaplib.StatusResp)
	if !ok {
		t.Fatalf("ReadResp() type = %T, want *imap.StatusResp", resp)
	}
	if status.Code != imaplib.StatusRespCode("ERROR=USERDISABLED") {
		t.Fatalf("status.Code = %q, want %q", status.Code, imaplib.StatusRespCode("ERROR=USERDISABLED"))
	}
}

func TestExchangeAuthStatusError(t *testing.T) {
	status := &imaplib.StatusResp{
		Type:      imaplib.StatusRespOk,
		Code:      imaplib.StatusRespCode("ERROR=USERDISABLED"),
		Arguments: []interface{}{"AuthResult=27", "Service=Imap4"},
	}

	err := exchangeAuthStatusError(status)
	if err == nil {
		t.Fatal("exchangeAuthStatusError() returned nil, want error")
	}

	want := "server reported authentication failure: ERROR=USERDISABLED AuthResult=27 Service=Imap4"
	if err.Error() != want {
		t.Fatalf("exchangeAuthStatusError() = %q, want %q", err.Error(), want)
	}
}

type xoauth2ServerScenario struct {
	name                   string
	greetingCapabilities   []string
	authenticateParts      []string
	expectContinuationData bool
}

func TestXOAUTH2ClientStart(t *testing.T) {
	client := newXOAUTH2Client("user@example.com", "access-token")

	mech, ir, err := client.Start()
	if err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}

	if mech != "XOAUTH2" {
		t.Fatalf("Start() mechanism = %q, want %q", mech, "XOAUTH2")
	}

	want := "user=user@example.com\x01auth=Bearer access-token\x01\x01"
	if got := string(ir); got != want {
		t.Fatalf("Start() initial response = %q, want %q", got, want)
	}
}

func TestXOAUTH2ClientNext(t *testing.T) {
	client := newXOAUTH2Client("user@example.com", "access-token")

	if _, err := client.Next([]byte("challenge")); err != sasl.ErrUnexpectedServerChallenge {
		t.Fatalf("Next() error = %v, want %v", err, sasl.ErrUnexpectedServerChallenge)
	}
}

func TestAuthenticateXOAUTH2(t *testing.T) {
	testCases := []xoauth2ServerScenario{
		{
			name:                   "uses initial response when SASL-IR is supported",
			greetingCapabilities:   []string{"IMAP4rev1", "AUTH=XOAUTH2", "SASL-IR"},
			authenticateParts:      []string{"AUTHENTICATE", "XOAUTH2", base64.StdEncoding.EncodeToString([]byte("user=user@example.com\x01auth=Bearer access-token\x01\x01"))},
			expectContinuationData: false,
		},
		{
			name:                   "falls back to continuation without SASL-IR",
			greetingCapabilities:   []string{"IMAP4rev1", "AUTH=XOAUTH2"},
			authenticateParts:      []string{"AUTHENTICATE", "XOAUTH2"},
			expectContinuationData: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runAuthenticateXOAUTH2Scenario(t, tc)
		})
	}
}

func runAuthenticateXOAUTH2Scenario(t *testing.T, tc xoauth2ServerScenario) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	serverErr := make(chan error, 1)
	go func() {
		rw := bufio.NewReadWriter(bufio.NewReader(serverConn), bufio.NewWriter(serverConn))
		capabilities := strings.Join(tc.greetingCapabilities, " ")

		if _, err := rw.WriteString("* OK [CAPABILITY " + capabilities + "] ready\r\n"); err != nil {
			serverErr <- err
			return
		}
		if err := rw.Flush(); err != nil {
			serverErr <- err
			return
		}

		authLine, err := rw.ReadString('\n')
		if err != nil {
			serverErr <- err
			return
		}
		authLine = strings.TrimSpace(authLine)
		parts := strings.Split(authLine, " ")
		if len(parts) != len(tc.authenticateParts)+1 {
			serverErr <- fmt.Errorf("unexpected AUTHENTICATE command: %q", authLine)
			return
		}
		if got, want := parts[1:], tc.authenticateParts; !reflect.DeepEqual(got, want) {
			serverErr <- fmt.Errorf("AUTHENTICATE command = %q, want %q", got, want)
			return
		}
		tag := parts[0]

		if tc.expectContinuationData {
			if _, err := rw.WriteString("+ \r\n"); err != nil {
				serverErr <- err
				return
			}
			if err := rw.Flush(); err != nil {
				serverErr <- err
				return
			}

			payloadLine, err := rw.ReadString('\n')
			if err != nil {
				serverErr <- err
				return
			}
			payloadLine = strings.TrimSpace(payloadLine)
			decoded, err := base64.StdEncoding.DecodeString(payloadLine)
			if err != nil {
				serverErr <- fmt.Errorf("decode XOAUTH2 payload: %w", err)
				return
			}
			wantPayload := "user=user@example.com\x01auth=Bearer access-token\x01\x01"
			if got := string(decoded); got != wantPayload {
				serverErr <- fmt.Errorf("XOAUTH2 payload = %q, want %q", got, wantPayload)
				return
			}
		}

		if _, err := rw.WriteString(tag + " OK AUTHENTICATE completed\r\n"); err != nil {
			serverErr <- err
			return
		}
		if err := rw.Flush(); err != nil {
			serverErr <- err
			return
		}
		serverErr <- rw.Flush()
	}()

	client, err := imapclient.New(clientConn)
	if err != nil {
		t.Fatalf("client.New() error = %v", err)
	}
	client.ErrorLog = log.New(io.Discard, "", 0)
	defer client.Terminate()

	imapClient := &IMAPClient{
		cfg:    &config.Account{User: "user@example.com"},
		client: client,
	}

	if err := imapClient.authenticateXOAUTH2("access-token"); err != nil {
		t.Fatalf("authenticateXOAUTH2() error = %v", err)
	}

	if err := <-serverErr; err != nil {
		t.Fatalf("server error = %v", err)
	}

	if state := client.State(); state != imaplib.AuthenticatedState {
		t.Fatalf("client.State() = %v, want %v", state, imaplib.AuthenticatedState)
	}
}
