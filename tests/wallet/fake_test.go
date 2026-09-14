package wallet_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/techpartners-asia/bonum-go/wallet"
)

var ctx = context.Background()

type fakePSP struct {
	lastReq  atomic.Pointer[http.Request]
	lastBody atomic.Pointer[string]
}

func newFakePSP(t *testing.T) (*fakePSP, *wallet.Client) {
	t.Helper()
	f := &fakePSP{}
	c := wallet.New(wallet.Sandbox, "mk_test_123", wallet.WithBaseURL(startPSP(t, f)))
	t.Cleanup(func() { c.Close() })
	return f, c
}

func startPSP(t *testing.T, f *fakePSP) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(f.handle))
	t.Cleanup(srv.Close)
	return srv.URL
}

func (f *fakePSP) body() string { return *f.lastBody.Load() }

func (f *fakePSP) sentJSON(t *testing.T) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(f.body()), &m); err != nil {
		t.Fatalf("body is not JSON: %v: %s", err, f.body())
	}
	return m
}

const paymentJSON = `{
  "paymentId": "550e8400-e29b-41d4-a716-446655440000",
  "orderId": "ORDER-2024-00123",
  "amount": "150.50",
  "currency": "496",
  "walletType": "APPLE_PAY",
  "status": "AUTHORIZED",
  "providerReference": "GBK20240415001234",
  "failureReason": null,
  "createdAt": "2024-04-15T10:30:00.000Z",
  "updatedAt": "2024-04-15T10:30:04.000Z"
}`

func (f *fakePSP) handle(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	s := string(body)
	f.lastBody.Store(&s)
	f.lastReq.Store(r)
	w.Header().Set("Content-Type", "application/json")

	if r.Header.Get("x-merchant-key") != "mk_test_123" {
		w.WriteHeader(401)
		w.Write([]byte(`{"statusCode":401,"message":"Invalid or inactive request key","error":"Unauthorized"}`))
		return
	}
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/api/v2/payment/process":
		if strings.Contains(s, `"order_id":"dup"`) {
			w.WriteHeader(429)
			w.Write([]byte(`{"statusCode":429,"message":"Too Many Requests","error":"Too Many Requests"}`))
			return
		}
		w.Write([]byte(`{"paymentId":"a1b2c3d4-e5f6-7890-abcd-ef1234567890","orderId":"ORD-20240521-001","status":"PENDING","acceptedAt":"2026-05-21T14:30:00.000+08:00","statusUrl":"https://psp.bonum.mn/api/v2/payments/a1b2c3d4-e5f6-7890-abcd-ef1234567890","awaitUrl":"https://psp.bonum.mn/api/v2/payments/a1b2c3d4-e5f6-7890-abcd-ef1234567890/await"}`))
	case r.Method == http.MethodPost && r.URL.Path == "/api/v2/payment/process/google":
		w.Write([]byte(`{"paymentId":"b2c3d4e5-f6a7-8901-bcde-f12345678901","orderId":"ORD-20240521-002","status":"PENDING","acceptedAt":"2026-05-21T14:31:00.000+08:00","statusUrl":"https://psp.bonum.mn/api/v2/payments/b2c3d4e5-f6a7-8901-bcde-f12345678901","awaitUrl":"https://psp.bonum.mn/api/v2/payments/b2c3d4e5-f6a7-8901-bcde-f12345678901/await"}`))
	case r.Method == http.MethodGet && r.URL.Path == "/api/v2/payments/550e8400-e29b-41d4-a716-446655440000/await":
		switch r.URL.Query().Get("timeoutMs") {
		case "1000":
			w.Write([]byte(`{"paymentId":"550e8400-e29b-41d4-a716-446655440000","status":"PENDING","failureReason":null,"timedOut":true}`))
		case "2000":
			w.Write([]byte(`{"paymentId":"550e8400-e29b-41d4-a716-446655440000","status":"FAILED","failureReason":"Insufficient funds"}`))
		default:
			w.Write([]byte(`{"paymentId":"550e8400-e29b-41d4-a716-446655440000","status":"AUTHORIZED","failureReason":null}`))
		}
	case r.Method == http.MethodGet && r.URL.Path == "/api/v2/payments/550e8400-e29b-41d4-a716-446655440000":
		w.Write([]byte(paymentJSON))
	case r.Method == http.MethodGet && r.URL.Path == "/api/v2/payments/lookup/by-order-id":
		if r.URL.Query().Get("orderId") != "ORDER-2024-00123" {
			w.WriteHeader(404)
			w.Write([]byte(`{"statusCode":404,"message":"Payment not found","error":"Not Found"}`))
			return
		}
		w.Write([]byte(paymentJSON))
	default:
		w.WriteHeader(404)
		w.Write([]byte(`{"statusCode":404,"message":"Not Found","error":"Not Found"}`))
	}
}
