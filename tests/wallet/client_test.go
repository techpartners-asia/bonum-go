package wallet_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/techpartners-asia/bonum-go/wallet"
)

func appleToken() wallet.ApplePayToken {
	return wallet.ApplePayToken{
		PaymentData: wallet.ApplePaymentData{
			Data: "aGVsbG8gd29ybGQ...", Signature: "MIAGCSqGSIb3DQ...",
			Header:  wallet.ApplePaymentHeader{PublicKeyHash: "Hmng9JBG...", EphemeralPublicKey: "MFkwEwYH...", TransactionID: "9AFDA47D..."},
			Version: "EC_v1",
		},
		PaymentMethod:         wallet.ApplePaymentMethod{DisplayName: "Visa 4242", Network: "Visa", Type: "debit"},
		TransactionIdentifier: "9AFDA47D...",
	}
}

func TestProcessApplePaySendsTokenAndReturnsPending(t *testing.T) {
	f, c := newFakePSP(t)
	res, err := c.ProcessApplePay(ctx, wallet.ProcessApplePayInput{OrderID: "ORDER-2024-00123", Amount: 150.50, BranchID: "BRANCH-001", Token: appleToken()})
	if err != nil {
		t.Fatal(err)
	}
	if res.PaymentID != "a1b2c3d4-e5f6-7890-abcd-ef1234567890" || res.Status != wallet.StatusPending || !strings.HasSuffix(res.AwaitURL, "/await") {
		t.Fatalf("unexpected response %+v", res)
	}
	req := f.lastReq.Load()
	if req.Header.Get("Content-Type") != "application/json" || req.Header.Get("x-merchant-key") != "mk_test_123" {
		t.Fatalf("headers wrong: %v", req.Header)
	}
	sent := f.sentJSON(t)
	if sent["order_id"] != "ORDER-2024-00123" || sent["amount"] != 150.50 || sent["branch_id"] != "BRANCH-001" {
		t.Fatalf("top-level fields wrong: %v", sent)
	}
	tok := sent["token"].(map[string]any)
	pd := tok["paymentData"].(map[string]any)
	if pd["version"] != "EC_v1" || pd["header"].(map[string]any)["transactionId"] != "9AFDA47D..." || tok["transactionIdentifier"] != "9AFDA47D..." {
		t.Fatalf("token not serialised as PKPaymentToken: %v", tok)
	}
}

func TestProcessApplePayOmitsOptionalFields(t *testing.T) {
	f, c := newFakePSP(t)
	if _, err := c.ProcessApplePay(ctx, wallet.ProcessApplePayInput{OrderID: "ORDER-1", Token: appleToken()}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(f.body(), `"amount"`) || strings.Contains(f.body(), `"branch_id"`) {
		t.Fatalf("optional fields must be omitted when zero: %s", f.body())
	}
}

func TestProcessApplePayValidatesBeforeCalling(t *testing.T) {
	f, c := newFakePSP(t)
	cases := []wallet.ProcessApplePayInput{
		{Token: appleToken()}, // no order id
		{OrderID: strings.Repeat("x", 129), Token: appleToken()},               // too long
		{OrderID: "o", Amount: 0.001, Token: appleToken()},                     // below minimum
		{OrderID: "o", Amount: 1.234, Token: appleToken()},                     // 3 decimals
		{OrderID: "o", BranchID: strings.Repeat("b", 65), Token: appleToken()}, // branch too long
		{OrderID: "o"}, // empty token
	}
	for i, in := range cases {
		if _, err := c.ProcessApplePay(ctx, in); !errors.Is(err, wallet.ErrInvalidInput) {
			t.Fatalf("case %d: want ErrInvalidInput, got %v", i, err)
		}
	}
	if f.lastReq.Load() != nil {
		t.Fatal("invalid input must not reach the network")
	}
}

func TestProcessGooglePaySendsStringTokenAndCurrency(t *testing.T) {
	f, c := newFakePSP(t)
	res, err := c.ProcessGooglePay(ctx, wallet.ProcessGooglePayInput{OrderID: "ORDER-2024-00124", Token: "eyJzaWduYXR1cmUiOiJNRUlDSVFDa...", CurrencyCode: wallet.MNT, Amount: 150.50, BranchID: "BRANCH-001"})
	if err != nil {
		t.Fatal(err)
	}
	if res.PaymentID != "b2c3d4e5-f6a7-8901-bcde-f12345678901" || res.OrderID != "ORD-20240521-002" {
		t.Fatalf("unexpected response %+v", res)
	}
	sent := f.sentJSON(t)
	if sent["token"] != "eyJzaWduYXR1cmUiOiJNRUlDSVFDa..." || sent["currency_code"] != "MNT" || sent["amount"] != 150.50 {
		t.Fatalf("body wrong: %v", sent)
	}
}

func TestProcessGooglePayValidatesCurrency(t *testing.T) {
	_, c := newFakePSP(t)
	_, err := c.ProcessGooglePay(ctx, wallet.ProcessGooglePayInput{OrderID: "o", Token: "t", CurrencyCode: "GBP"})
	if !errors.Is(err, wallet.ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
	_, err = c.ProcessGooglePay(ctx, wallet.ProcessGooglePayInput{OrderID: "o", CurrencyCode: wallet.MNT})
	if !errors.Is(err, wallet.ErrInvalidInput) {
		t.Fatalf("empty token: want ErrInvalidInput, got %v", err)
	}
}

func TestErrorsMapToSentinels(t *testing.T) {
	f := &fakePSP{}
	bad := wallet.New(wallet.Sandbox, "wrong", wallet.WithBaseURL(startPSP(t, f)))
	_, err := bad.GetPayment(ctx, "550e8400-e29b-41d4-a716-446655440000")
	if !errors.Is(err, wallet.ErrUnauthorized) {
		t.Fatalf("want ErrUnauthorized, got %v", err)
	}
	var api *wallet.APIError
	if !errors.As(err, &api) || api.StatusCode != 401 || api.Message != "Invalid or inactive request key" || api.ErrorText != "Unauthorized" {
		t.Fatalf("error not decoded: %#v", err)
	}

	_, c := newFakePSP(t)
	if _, err := c.LookupByOrderID(ctx, "nope"); !errors.Is(err, wallet.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
	if _, err := c.ProcessApplePay(ctx, wallet.ProcessApplePayInput{OrderID: "dup", Token: appleToken()}); !errors.Is(err, wallet.ErrRateLimited) {
		t.Fatalf("want ErrRateLimited, got %v", err)
	}
}

func TestContextCancellationAbortsCall(t *testing.T) {
	_, c := newFakePSP(t)
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := c.GetPayment(cancelled, "550e8400-e29b-41d4-a716-446655440000"); !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
}

func TestGetPaymentAndLookup(t *testing.T) {
	f, c := newFakePSP(t)
	p, err := c.GetPayment(ctx, "550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatal(err)
	}
	if p.Status != wallet.StatusAuthorized || p.WalletType != wallet.ApplePay || p.Amount != "150.50" || p.Currency != "496" {
		t.Fatalf("unexpected payment %+v", p)
	}
	if p.ProviderReference == nil || *p.ProviderReference != "GBK20240415001234" || p.FailureReason != nil {
		t.Fatalf("reference/failure wrong: %+v", p)
	}
	if _, err := c.LookupByOrderID(ctx, "ORDER-2024-00123"); err != nil {
		t.Fatal(err)
	}
	if got := f.lastReq.Load().URL.Query().Get("orderId"); got != "ORDER-2024-00123" {
		t.Fatalf("orderId query = %q", got)
	}
}

func TestAwaitPaymentOutcomes(t *testing.T) {
	f, c := newFakePSP(t)
	const id = "550e8400-e29b-41d4-a716-446655440000"

	r, err := c.AwaitPayment(ctx, id, 0)
	if err != nil || r.Status != wallet.StatusAuthorized || r.TimedOut {
		t.Fatalf("authorized: %+v %v", r, err)
	}
	if got := f.lastReq.Load().URL.Query().Get("timeoutMs"); got != "" {
		t.Fatalf("zero timeout must omit timeoutMs, got %q", got)
	}

	r, err = c.AwaitPayment(ctx, id, 2*time.Second)
	if err != nil || r.Status != wallet.StatusFailed || r.FailureReason == nil || *r.FailureReason != "Insufficient funds" {
		t.Fatalf("failed: %+v %v", r, err)
	}

	r, err = c.AwaitPayment(ctx, id, time.Second)
	if err != nil || !r.TimedOut || r.Status != wallet.StatusPending {
		t.Fatalf("timed out: %+v %v", r, err)
	}

	if _, err := c.AwaitPayment(ctx, id, time.Minute); err != nil {
		t.Fatal(err)
	}
	if got := f.lastReq.Load().URL.Query().Get("timeoutMs"); got != "28000" {
		t.Fatalf("timeoutMs = %q, want 28000 cap", got)
	}
}

func TestAwaitURLUsesReturnedAbsoluteURL(t *testing.T) {
	f := &fakePSP{}
	base := startPSP(t, f)
	c := wallet.New(wallet.Sandbox, "mk_test_123", wallet.WithBaseURL(base))
	r, err := c.AwaitURL(ctx, base+"/api/v2/payments/550e8400-e29b-41d4-a716-446655440000/await", 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != wallet.StatusFailed || f.lastReq.Load().Header.Get("x-merchant-key") != "mk_test_123" {
		t.Fatalf("unexpected %+v", r)
	}
}

func TestAwaitURLRejectsAnUnrelatedHost(t *testing.T) {
	f, c := newFakePSP(t)
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	t.Cleanup(other.Close)

	if _, err := c.AwaitURL(ctx, other.URL+"/api/v2/payments/x/await", time.Second); !errors.Is(err, wallet.ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput for a mismatched host, got %v", err)
	}
	if f.lastReq.Load() != nil {
		t.Fatal("a rejected host must never receive the merchant-key header")
	}

	if _, err := c.AwaitURL(ctx, "not-a-url", time.Second); !errors.Is(err, wallet.ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput for a non-absolute URL, got %v", err)
	}
}

func TestEnvironmentHosts(t *testing.T) {
	if wallet.Sandbox != "https://testpsp.bonum.mn" || wallet.Production != "https://psp.bonum.mn" {
		t.Fatalf("hosts wrong: %s %s", wallet.Sandbox, wallet.Production)
	}
}
