package wallet_test

import (
	"bytes"
	"compress/gzip"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/techpartners-asia/bonum-go/wallet"
)

func clientAnswering(t *testing.T, contentType, body string) *wallet.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	c := wallet.New(wallet.Sandbox, "mk_test_123", wallet.WithBaseURL(srv.URL))
	t.Cleanup(func() { c.Close() })
	return c
}

var googleIn = wallet.ProcessGooglePayInput{OrderID: "ORD-1", Token: "{}", CurrencyCode: wallet.MNT, Amount: 100}

// A JSON body is decoded whatever Content-Type Bonum labels it with. Live
// /process/google answered 200 and the client returned an empty paymentId.
func TestProcessGooglePay_decodesJSONWhateverTheContentType(t *testing.T) {
	for _, ct := range []string{"", "text/plain", "text/plain; charset=utf-8", "application/octet-stream", "application/json"} {
		t.Run(ct, func(t *testing.T) {
			c := clientAnswering(t, ct, `{"paymentId":"pay-1","orderId":"ORD-1","status":"PENDING"}`)
			got, err := c.ProcessGooglePay(ctx, googleIn)
			if err != nil {
				t.Fatalf("err = %v", err)
			}
			if got.PaymentID != "pay-1" || got.Status != wallet.StatusPending {
				t.Fatalf("got %+v, want paymentId pay-1 PENDING", got)
			}
		})
	}
}

// A 2xx body that is not the documented JSON must be an error that carries the
// body, never a zero-valued response.
func TestProcessGooglePay_undecodableSuccessIsAnError(t *testing.T) {
	c := clientAnswering(t, "text/html", "<html>gateway</html>")
	got, err := c.ProcessGooglePay(ctx, googleIn)
	if err == nil {
		t.Fatalf("got %+v, want an error", got)
	}
	var re *wallet.ResponseError
	if !errors.As(err, &re) || re.StatusCode != http.StatusOK || re.Body != "<html>gateway</html>" {
		t.Fatalf("err = %#v, want *wallet.ResponseError carrying status and body", err)
	}
}

// Live /process/google answered 200 "application/json; charset=utf-8" with a gzip body
// and no Content-Encoding, so neither resty nor net/http inflated it.
func TestProcessGooglePay_inflatesGzipWithoutContentEncoding(t *testing.T) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write([]byte(`{"paymentId":"pay-1","orderId":"ORD-1","status":"PENDING"}`))
	_ = zw.Close()
	c := clientAnswering(t, "application/json; charset=utf-8", buf.String())

	got, err := c.ProcessGooglePay(ctx, googleIn)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got.PaymentID != "pay-1" || got.Status != wallet.StatusPending {
		t.Fatalf("got %+v, want paymentId pay-1 PENDING", got)
	}
}

// A gzip error body must still yield Bonum's message.
func TestProcessGooglePay_inflatesGzipErrorBody(t *testing.T) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write([]byte(`{"statusCode":400,"message":"token is invalid","error":"Bad Request"}`))
	_ = zw.Close()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write(buf.Bytes())
	}))
	t.Cleanup(srv.Close)
	c := wallet.New(wallet.Sandbox, "mk_test_123", wallet.WithBaseURL(srv.URL))
	t.Cleanup(func() { c.Close() })

	_, err := c.ProcessGooglePay(ctx, googleIn)
	var api *wallet.APIError
	if !errors.As(err, &api) || api.Message != "token is invalid" {
		t.Fatalf("err = %#v, want APIError with Bonum's message", err)
	}
}
