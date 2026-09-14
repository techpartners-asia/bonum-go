package gateway_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	bonum "github.com/techpartners-asia/bonum-go"
)

const (
	ecommercePath = "/bonum-gateway/ecommerce"
	mpayPath      = "/mpay-service/merchant"
)

// fakeGateway is an httptest stand-in for the Bonum gateway. Tests cross the same seam
// callers do: the exported bonum.Client interface.
type fakeGateway struct {
	createCalls  atomic.Int32
	refreshCalls atomic.Int32
	accessTTL    int64
	lastReq      atomic.Pointer[http.Request]
	lastBody     atomic.Pointer[string]
}

func newFakeGateway(t *testing.T, accessTTL int64) (*fakeGateway, *bonum.Client) {
	t.Helper()
	f := &fakeGateway{accessTTL: accessTTL}
	srv := httptest.NewServer(http.HandlerFunc(f.handle))
	t.Cleanup(srv.Close)
	c := bonum.New(bonum.Sandbox, "secret-123", "17171119", bonum.WithBaseURL(srv.URL), bonum.WithLanguage(bonum.EN))
	t.Cleanup(func() { c.Close() })
	return f, c
}

func (f *fakeGateway) body() string { return *f.lastBody.Load() }

func (f *fakeGateway) sentJSON(t *testing.T) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(f.body()), &m); err != nil {
		t.Fatalf("body is not JSON: %v: %s", err, f.body())
	}
	return m
}

func (f *fakeGateway) handle(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	s := string(body)
	f.lastBody.Store(&s)
	f.lastReq.Store(r)
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.URL.Path == ecommercePath+"/auth/create":
		if r.Header.Get("Authorization") != "AppSecret secret-123" || r.Header.Get("X-TERMINAL-ID") != "17171119" {
			w.WriteHeader(401)
			w.Write([]byte(`{"traceId":"t-401","message":"bad secret","status":401}`))
			return
		}
		f.createCalls.Add(1)
		w.Write([]byte(`{"tokenType":"Bearer","accessToken":"access-1","expiresIn":` + itoa(f.accessTTL) + `,"refreshToken":"refresh-1","refreshExpiresIn":2000,"unit":"SECONDS"}`))
	case r.URL.Path == ecommercePath+"/auth/refresh":
		if r.Header.Get("Authorization") != "Bearer refresh-1" {
			w.WriteHeader(401)
			return
		}
		f.refreshCalls.Add(1)
		w.Write([]byte(`{"tokenType":"Bearer","accessToken":"access-2","expiresIn":1800}`))
	}
	if strings.HasPrefix(r.URL.Path, ecommercePath+"/auth/") {
		return
	}
	if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer access-") {
		w.WriteHeader(401)
		w.Write([]byte(`{"traceId":"t-401","message":"no token","status":401}`))
		return
	}

	switch {
	case r.URL.Path == ecommercePath+"/invoices/payment-providers":
		w.Write([]byte(`[{"provider":"QPAY","enabled":true},{"provider":"E_COMMERCE","enabled":false}]`))
	case r.URL.Path == ecommercePath+"/invoices" && r.Method == http.MethodPost:
		w.Write([]byte(`{"invoiceId":"inv-1","followUpLink":"https://ecommerce.bonum.mn/ecommerce?invoiceId=inv-1"}`))
	case r.URL.Path == ecommercePath+"/invoices/paid":
		w.Write([]byte(`{"ok":true,"invoiceId":"` + r.URL.Query().Get("invoiceId") + `"}`))
	case r.URL.Path == ecommercePath+"/invoices/inv-1":
		w.Write([]byte(`{"invoiceId":"inv-1","status":"PAID"}`))
	case r.URL.Path == mpayPath+"/cards/tokenize/request":
		w.Write([]byte(`{"id":"tok-req-1","followUpLink":"https://ecommerce.bonum.mn/tokenize?id=tok-req-1"}`))
	case r.URL.Path == mpayPath+"/transaction/purchase":
		switch r.Header.Get("X-CARD-TOKEN") {
		case "":
			w.WriteHeader(400)
			w.Write([]byte(`{"traceId":"t-1","message":"missing card token","status":400}`))
		case "declined-tok":
			w.WriteHeader(400)
			w.Write([]byte(`{"traceId":"t-d","errorCode":null,"message":"Insufficient funds","data":{"id":9,"status":"FAILED","cardStatus":"ACTIVE","respCode":"51"},"status":400}`))
		case "queued-tok":
			w.WriteHeader(201)
			w.Write([]byte(`{"traceId":"t-q","message":"queued","data":{"id":10,"status":"QUEUED","cardStatus":null},"status":201}`))
		default:
			w.Write([]byte(`{"traceId":"t-2","errorCode":null,"message":"ok","data":{"id":1,"completedAt":"2026-01-29 11:20:33","status":"SUCCESS","cardStatus":"ACTIVE"},"status":200}`))
		}
	case r.URL.Path == mpayPath+"/transaction/reverse/txn-9" && r.Method == http.MethodDelete:
		w.Write([]byte(`{"traceId":"t-3","message":"reversed","data":null,"status":200}`))
	case r.URL.Path == mpayPath+"/transaction/qr/create":
		w.Write([]byte(`{"traceId":"t-qr","message":"ok","data":{"invoiceId":"qr-1","qrCode":"0002010102...","qrImage":"iVBOR...","links":[{"name":"Khan Bank","link":"khanbank://q?qPay_QRcode=0002"}]},"status":200}`))
	case r.URL.Path == mpayPath+"/transaction/qr" && r.Method == http.MethodPost:
		w.Write([]byte(`{"traceId":"t-qr2","message":"ok","data":{"invoiceId":"qr-1","qrCode":"0002010102..."},"status":200}`))
	case r.URL.Path == mpayPath+"/transaction/qr/pay" && r.Method == http.MethodPut:
		w.Write([]byte(`{"traceId":"t-qr3","message":"ok","data":{"id":5,"status":"SUCCESS"},"status":200}`))
	case r.URL.Path == mpayPath+"/values/payment-plans":
		w.Write([]byte(`{"traceId":"t-p","message":"ok","data":[{"planId":30,"name":"Gold","recurringType":"MONTHLY","amount":5000,"status":"ACTIVE"}],"status":200}`))
	case r.URL.Path == mpayPath+"/subscriptions/subscribe":
		w.Write([]byte(`{"traceId":"t-s","message":"ok","data":{"subscriptionId":42,"cardMask":"5150 23** **** 4778","plan":{"planId":30,"name":"Gold"},"status":"ACTIVE"},"status":200}`))
	case r.URL.Path == mpayPath+"/subscriptions" && r.Method == http.MethodGet:
		w.Write([]byte(`{"traceId":"t-s2","message":"ok","data":[{"subscriptionId":42,"status":"ACTIVE"}],"status":200}`))
	case r.URL.Path == mpayPath+"/subscriptions/42" && r.Method == http.MethodDelete:
		w.Write([]byte(`{"traceId":"t-4","message":"unsubscribed","data":null,"status":200}`))
	case r.URL.Path == mpayPath+"/subscriptions/42/delete" && r.Method == http.MethodDelete:
		w.Write([]byte(`{"traceId":"t-5","message":"deleted","data":null,"status":200}`))
	case r.URL.Path == mpayPath+"/subscriptions/7/change/create-new-token" && r.Method == http.MethodPut:
		w.Write([]byte(`{"id":"tok-req-1","followUpLink":"https://ecommerce.bonum.mn/tokenize?id=tok-req-1"}`))
	case r.URL.Path == mpayPath+"/subscriptions/7/change" && r.Method == http.MethodPut:
		w.Write([]byte(`{"traceId":"t-6","message":"ok","data":{"subscriptionId":7,"cardMask":"4242","status":"ACTIVE"},"status":200}`))
	case r.URL.Path == mpayPath+"/subscriptions/7/execute" && r.Method == http.MethodPut:
		w.Write([]byte(`{"traceId":"t-7","message":"executed","data":null,"status":200}`))
	default:
		w.WriteHeader(404)
		w.Write([]byte(`{"traceId":"t-404","message":"not found","status":404}`))
	}
}

func itoa(n int64) string {
	return strings.TrimSpace(strings.Repeat(" ", 0) + json.Number(itoaRaw(n)).String())
}

func itoaRaw(n int64) string {
	b, _ := json.Marshal(n)
	return string(b)
}
