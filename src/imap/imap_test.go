package imap

import (
	"testing"

	"github.com/emersion/go-sasl"
)

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
