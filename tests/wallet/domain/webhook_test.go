package domain_test

import (
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/techpartners-asia/bonum-go/wallet/domain/payment"
	"github.com/techpartners-asia/bonum-go/wallet/domain/webhook"
)

const (
	secret  = "whsec_test"
	tsFixed = "1713174600" // 2024-04-15T10:30:00Z
	body    = `{"webhookId":"d290","paymentId":"550e","orderId":"ORDER-1","eventType":"AUTHORIZED","status":"AUTHORIZED","amount":"150.50","currency":"MNT","providerReference":"GBK1","failureReason":null,"occurredAt":"2024-04-15T10:30:04.123Z","walletType":"APPLE_PAY","binCategory":"DOMESTIC"}`
)

var sentAt = time.Unix(1713174600, 0)

func TestParseAtAcceptsInsideReplayWindow(t *testing.T) {
	sig := webhook.Sign([]byte(body), tsFixed, secret)
	ev, err := webhook.ParseAt([]byte(body), sig, tsFixed, secret, sentAt.Add(4*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if ev.WebhookID != "d290" || ev.EventType != payment.StatusAuthorized || *ev.BinCategory != payment.Domestic {
		t.Fatalf("unexpected %+v", ev)
	}
	if _, err := webhook.ParseAt([]byte(body), sig, tsFixed, secret, sentAt.Add(-4*time.Minute)); err != nil {
		t.Fatalf("clock skew inside tolerance must pass: %v", err)
	}
}

func TestParseAtRejectsOutsideReplayWindow(t *testing.T) {
	sig := webhook.Sign([]byte(body), tsFixed, secret)
	if _, err := webhook.ParseAt([]byte(body), sig, tsFixed, secret, sentAt.Add(webhook.ReplayTolerance+time.Second)); !errors.Is(err, webhook.ErrTimestampExpired) {
		t.Fatalf("stale: %v", err)
	}
	if _, err := webhook.ParseAt([]byte(body), sig, tsFixed, secret, sentAt.Add(-webhook.ReplayTolerance-time.Second)); !errors.Is(err, webhook.ErrTimestampExpired) {
		t.Fatalf("future: %v", err)
	}
}

func TestParseAtRejectsBadSignatures(t *testing.T) {
	good := webhook.Sign([]byte(body), tsFixed, secret)
	cases := map[string]struct {
		sig, ts string
		want    error
	}{
		"empty signature":       {"", tsFixed, webhook.ErrMissingSignature},
		"empty timestamp":       {good, "", webhook.ErrMissingSignature},
		"non-numeric timestamp": {good, "abc", webhook.ErrMissingSignature},
		"unknown prefix":        {"v2=" + good[3:], tsFixed, webhook.ErrSignatureMismatch},
		"wrong hmac":            {"v1=00", tsFixed, webhook.ErrSignatureMismatch},
	}
	for name, c := range cases {
		if _, err := webhook.ParseAt([]byte(body), c.sig, c.ts, secret, sentAt); !errors.Is(err, c.want) {
			t.Fatalf("%s: got %v, want %v", name, err, c.want)
		}
	}
}

func TestParseUsesWallClock(t *testing.T) {
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	if _, err := webhook.Parse([]byte(body), webhook.Sign([]byte(body), ts, secret), ts, secret); err != nil {
		t.Fatal(err)
	}
}
