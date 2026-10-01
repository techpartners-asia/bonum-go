package wallet_test

import (
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
