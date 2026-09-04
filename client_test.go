package bonum

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/techpartners-asia/bonum-go/types"
)

type fakeBonum struct {
	t            *testing.T
	createCalls  atomic.Int32
	refreshCalls atomic.Int32
	accessTTL    int64
	lastReq      atomic.Pointer[http.Request]
	lastBody     atomic.Pointer[string]
}

func newFakeBonum(t *testing.T, accessTTL int64) (*fakeBonum, *Client) {
	f := &fakeBonum{t: t, accessTTL: accessTTL}
	srv := httptest.NewServer(http.HandlerFunc(f.handle))
	t.Cleanup(srv.Close)
	c := New(types.Sandbox, "secret-123", "17171119", WithBaseURL(srv.URL), WithLanguage(types.EN))
	t.Cleanup(func() { c.Close() })
	return f, c
}

func (f *fakeBonum) handle(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	s := string(body)
	f.lastBody.Store(&s)
	f.lastReq.Store(r)
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.URL.Path == ecommercePath+"/auth/create":
		if r.Header.Get("Authorization") != "AppSecret secret-123" || r.Header.Get("X-TERMINAL-ID") != "17171119" {
			w.WriteHeader(401)
			return
		}
		f.createCalls.Add(1)
		json.NewEncoder(w).Encode(types.AuthResponse{
			TokenType: "Bearer", AccessToken: "access-1", ExpiresIn: f.accessTTL,
			RefreshToken: "refresh-1", RefreshExpiresIn: 2000, Unit: "SECONDS",
		})
	case r.URL.Path == ecommercePath+"/auth/refresh":
		if r.Header.Get("Authorization") != "Bearer refresh-1" {
			w.WriteHeader(401)
			return
		}
		f.refreshCalls.Add(1)
		json.NewEncoder(w).Encode(types.AuthResponse{TokenType: "Bearer", AccessToken: "access-2", ExpiresIn: 1800})
	case strings.HasSuffix(r.URL.Path, "/invoices/payment-providers"):
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer access-") {
			w.WriteHeader(401)
			return
		}
		w.Write([]byte(`[{"provider":"QPAY","enabled":true}]`))
	case r.URL.Path == ecommercePath+"/invoices" && r.Method == http.MethodPost:
		w.Write([]byte(`{"invoiceId":"inv-1","followUpLink":"https://ecommerce.bonum.mn/ecommerce?invoiceId=inv-1"}`))
	case r.URL.Path == mpayPath+"/transaction/purchase":
		if r.Header.Get("X-CARD-TOKEN") != "card-tok" {
			w.WriteHeader(400)
			w.Write([]byte(`{"traceId":"t-1","message":"missing card token","status":400}`))
			return
		}
		w.Write([]byte(`{"traceId":"t-2","errorCode":null,"message":"ok","data":{"id":1,"status":"SUCCESS","cardStatus":"ACTIVE"},"status":200}`))
	case r.URL.Path == mpayPath+"/transaction/reverse/txn-9" && r.Method == http.MethodDelete:
		w.Write([]byte(`{"traceId":"t-3","message":"reversed","data":null,"status":200}`))
	case r.URL.Path == mpayPath+"/subscriptions/42" && r.Method == http.MethodDelete:
		w.Write([]byte(`{"traceId":"t-4","message":"unsubscribed","data":null,"status":200}`))
	case r.URL.Path == mpayPath+"/subscriptions/7/change/create-new-token" && r.Method == http.MethodPut:
		w.Write([]byte(`{"id":"tok-req-1","followUpLink":"https://ecommerce.bonum.mn/tokenize?id=tok-req-1"}`))
	case r.URL.Path == ecommercePath+"/invoices/paid":
		w.Write([]byte(`{"ok":true,"invoiceId":"` + r.URL.Query().Get("invoiceId") + `"}`))
	default:
		w.WriteHeader(404)
		w.Write([]byte(`{"traceId":"t-404","message":"not found","status":404}`))
	}
}

func TestTokenIsFetchedOnceAndReused(t *testing.T) {
	f, c := newFakeBonum(t, 1800)

	for range 3 {
		if _, err := c.GetPaymentProviders(); err != nil {
			t.Fatal(err)
		}
	}
	if got := f.createCalls.Load(); got != 1 {
		t.Fatalf("auth/create called %d times, want 1", got)
	}
	if got := f.refreshCalls.Load(); got != 0 {
		t.Fatalf("auth/refresh called %d times, want 0", got)
	}
	req := f.lastReq.Load()
	if req.Header.Get("Accept-Language") != "en" {
		t.Fatalf("Accept-Language = %q", req.Header.Get("Accept-Language"))
	}
}

func TestExpiredAccessTokenIsRefreshed(t *testing.T) {
	f, c := newFakeBonum(t, 1800)
	if _, err := c.GetPaymentProviders(); err != nil {
		t.Fatal(err)
	}

	c.mu.Lock()
	c.accessExpiresAt = time.Now().Add(-time.Minute)
	c.mu.Unlock()

	if _, err := c.GetPaymentProviders(); err != nil {
		t.Fatal(err)
	}
	if got := f.refreshCalls.Load(); got != 1 {
		t.Fatalf("auth/refresh called %d times, want 1", got)
	}
	if got := f.createCalls.Load(); got != 1 {
		t.Fatalf("auth/create called %d times, want 1", got)
	}
	if f.lastReq.Load().Header.Get("Authorization") != "Bearer access-2" {
		t.Fatalf("refreshed token not used: %q", f.lastReq.Load().Header.Get("Authorization"))
	}
}

func TestCreateInvoiceSendsBody(t *testing.T) {
	f, c := newFakeBonum(t, 1800)
	res, err := c.CreateInvoice(types.CreateInvoiceInput{
		Amount: 100, Callback: "https://m/cb", TransactionID: "T1", ExpiresIn: 600,
		Providers: []types.PaymentProvider{types.ProviderQPay},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.InvoiceID != "inv-1" {
		t.Fatalf("invoiceId = %q", res.InvoiceID)
	}
	var sent map[string]any
	if err := json.Unmarshal([]byte(*f.lastBody.Load()), &sent); err != nil {
		t.Fatal(err)
	}
	if sent["transactionId"] != "T1" || sent["amount"].(float64) != 100 {
		t.Fatalf("unexpected body %v", sent)
	}
	if _, has := sent["items"]; has {
		t.Fatalf("empty optional items should be omitted: %v", sent)
	}
}

func TestPurchaseSendsCardTokenHeader(t *testing.T) {
	_, c := newFakeBonum(t, 1800)
	res, err := c.Purchase("card-tok", types.PurchaseInput{Amount: 15, Currency: "MNT", TransactionID: "T2"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Data.Status != types.TransactionSuccess || res.Data.CardStatus == nil || *res.Data.CardStatus != "ACTIVE" {
		t.Fatalf("unexpected data %+v", res.Data)
	}
}

func TestNon2xxBecomesError(t *testing.T) {
	_, c := newFakeBonum(t, 1800)
	_, err := c.Purchase("", types.PurchaseInput{Amount: 15, Currency: "MNT", TransactionID: "T3"})
	var be *Error
	if !errors.As(err, &be) {
		t.Fatalf("expected *Error, got %T %v", err, err)
	}
	if be.StatusCode != 400 || be.TraceID != "t-1" || be.Message != "missing card token" {
		t.Fatalf("unexpected error %+v", be)
	}
}

func TestDeleteWithPathParamAndBody(t *testing.T) {
	f, c := newFakeBonum(t, 1800)

	if _, err := c.RollbackPurchase("card-tok", "txn-9"); err != nil {
		t.Fatal(err)
	}
	if f.lastReq.Load().Header.Get("X-CARD-TOKEN") != "card-tok" {
		t.Fatal("X-CARD-TOKEN missing on rollback")
	}

	if _, err := c.Unsubscribe(42, 30); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(*f.lastBody.Load()); got != `{"planId":30}` {
		t.Fatalf("DELETE body = %q, want planId payload", got)
	}
}

func TestPutWithPathParam(t *testing.T) {
	_, c := newFakeBonum(t, 1800)
	res, err := c.ChangeSubscriptionTokenNew(7, types.ChangeSubscriptionTokenInput{Callback: "https://m/cb", TransactionID: "T4"})
	if err != nil {
		t.Fatal(err)
	}
	if res.ID != "tok-req-1" {
		t.Fatalf("id = %q", res.ID)
	}
}

func TestSandboxQueryParam(t *testing.T) {
	_, c := newFakeBonum(t, 1800)
	raw, err := c.SetInvoicePaidSandbox("inv-x")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"invoiceId":"inv-x"`) {
		t.Fatalf("query param not forwarded: %s", raw)
	}
}
