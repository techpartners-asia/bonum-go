package wallet_test

import (
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/techpartners-asia/bonum-go/wallet"
)

const (
	secret   = "whsec_test"
	tsFixed  = "1713174600" // 2024-04-15T10:30:00Z
	bodyAuth = `{"webhookId":"d290f1ee-6c54-4b01-90e6-d701748f0851","paymentId":"550e8400-e29b-41d4-a716-446655440000","orderId":"ORDER-2024-00123","eventType":"AUTHORIZED","status":"AUTHORIZED","amount":"150.50","currency":"MNT","providerReference":"GBK20240415001234","failureReason":null,"occurredAt":"2024-04-15T10:30:04.123Z","walletType":"APPLE_PAY","binCategory":"DOMESTIC"}`
	// Computed independently with: printf '%s' "$ts.$body" | openssl dgst -sha256 -hmac "$secret"
	sigFixed = "v1=c21ef6590a53e56633fb3a4db3e19282b2a6a988ca26584ee915ef6c85ec5193"
)

func nowTS() string { return strconv.FormatInt(time.Now().Unix(), 10) }

func TestSignMatchesOpenSSL(t *testing.T) {
	if got := wallet.Sign([]byte(bodyAuth), tsFixed, secret); got != sigFixed {
		t.Fatalf("Sign() = %s, want %s", got, sigFixed)
	}
}

func TestParseWebhookAuthorized(t *testing.T) {
	ts := nowTS()
	ev, err := wallet.ParseWebhook([]byte(bodyAuth), wallet.Sign([]byte(bodyAuth), ts, secret), ts, secret)
	if err != nil {
		t.Fatal(err)
	}
	if ev.WebhookID != "d290f1ee-6c54-4b01-90e6-d701748f0851" || ev.EventType != wallet.StatusAuthorized || ev.WalletType != wallet.ApplePay {
		t.Fatalf("unexpected event %+v", ev)
	}
	if ev.ProviderReference == nil || *ev.ProviderReference != "GBK20240415001234" || ev.BinCategory == nil || *ev.BinCategory != wallet.Domestic {
		t.Fatalf("reference/bin wrong: %+v", ev)
	}
}

func TestParseWebhookFailed(t *testing.T) {
	body := `{"webhookId":"a1b2c3d4-e5f6-7890-abcd-ef1234567890","paymentId":"7b12c830-f9d2-4a3e-b101-885544220011","orderId":"ORDER-2024-00124","eventType":"FAILED","status":"FAILED","amount":"150.50","currency":"MNT","providerReference":null,"failureReason":"Insufficient funds","occurredAt":"2024-04-15T10:31:09.456Z","walletType":"GOOGLE_PAY","binCategory":null}`
	ts := nowTS()
	ev, err := wallet.ParseWebhook([]byte(body), wallet.Sign([]byte(body), ts, secret), ts, secret)
	if err != nil {
		t.Fatal(err)
	}
	if ev.EventType != wallet.StatusFailed || ev.WalletType != wallet.GooglePay || ev.FailureReason == nil || *ev.FailureReason != "Insufficient funds" || ev.BinCategory != nil {
		t.Fatalf("unexpected %+v", ev)
	}
}

func TestParseWebhookRejects(t *testing.T) {
	ts := nowTS()
	good := wallet.Sign([]byte(bodyAuth), ts, secret)

	if _, err := wallet.ParseWebhook([]byte(bodyAuth), good, ts, "other"); !errors.Is(err, wallet.ErrSignatureMismatch) {
		t.Fatalf("wrong secret: %v", err)
	}
	if _, err := wallet.ParseWebhook([]byte(bodyAuth+" "), good, ts, secret); !errors.Is(err, wallet.ErrSignatureMismatch) {
		t.Fatalf("tampered body: %v", err)
	}
	if _, err := wallet.ParseWebhook([]byte(bodyAuth), "v2="+good[3:], ts, secret); !errors.Is(err, wallet.ErrSignatureMismatch) {
		t.Fatalf("unknown prefix: %v", err)
	}
	if _, err := wallet.ParseWebhook([]byte(bodyAuth), "", ts, secret); !errors.Is(err, wallet.ErrMissingSignature) {
		t.Fatalf("empty signature: %v", err)
	}
	if _, err := wallet.ParseWebhook([]byte(bodyAuth), good, "", secret); !errors.Is(err, wallet.ErrMissingSignature) {
		t.Fatalf("empty timestamp: %v", err)
	}
	if _, err := wallet.ParseWebhook([]byte(bodyAuth), good, "abc", secret); !errors.Is(err, wallet.ErrMissingSignature) {
		t.Fatalf("non-numeric timestamp: %v", err)
	}
	// A 2024 timestamp is far outside the 5 minute window today.
	if _, err := wallet.ParseWebhook([]byte(bodyAuth), sigFixed, tsFixed, secret); !errors.Is(err, wallet.ErrTimestampExpired) {
		t.Fatalf("stale: %v", err)
	}
	future := strconv.FormatInt(time.Now().Add(6*time.Minute).Unix(), 10)
	if _, err := wallet.ParseWebhook([]byte(bodyAuth), wallet.Sign([]byte(bodyAuth), future, secret), future, secret); !errors.Is(err, wallet.ErrTimestampExpired) {
		t.Fatalf("future: %v", err)
	}
	if _, err := wallet.ParseWebhook([]byte("not json"), wallet.Sign([]byte("not json"), ts, secret), ts, secret); err == nil {
		t.Fatal("garbage body must fail")
	}
}
