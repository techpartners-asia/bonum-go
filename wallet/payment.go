package wallet

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"resty.dev/v3"
)

// MaxAwaitTimeout is the server-side cap on the await window, chosen by Bonum to leave
// a buffer before Apple Pay's 30-second hard limit.
const MaxAwaitTimeout = 28 * time.Second

// GetPayment returns the current state of a Wallet Payment by Bonum's paymentId.
func (c *Client) GetPayment(ctx context.Context, paymentID string) (*Payment, error) {
	if paymentID == "" {
		return nil, invalid("paymentID", "required")
	}
	return call[Payment](ctx, c, http.MethodGet, paymentsPath+"/{id}", pathParam("id", paymentID))
}

// LookupByOrderID returns the Wallet Payment submitted with your Order ID.
func (c *Client) LookupByOrderID(ctx context.Context, orderID string) (*Payment, error) {
	if orderID == "" {
		return nil, invalid("orderID", "required")
	}
	return call[Payment](ctx, c, http.MethodGet, paymentsPath+"/lookup/by-order-id", query("orderId", orderID))
}

// AwaitPayment blocks until the payment reaches AUTHORIZED or FAILED, or timeout elapses.
// timeout 0 uses Bonum's default (25s); anything above MaxAwaitTimeout is capped at 28s.
// On TimedOut the payment is still processing and the outcome arrives via the webhook.
func (c *Client) AwaitPayment(ctx context.Context, paymentID string, timeout time.Duration) (*AwaitResult, error) {
	if paymentID == "" {
		return nil, invalid("paymentID", "required")
	}
	return call[AwaitResult](ctx, c, http.MethodGet, paymentsPath+"/{id}/await", pathParam("id", paymentID), awaitTimeout(timeout))
}

// AwaitURL is AwaitPayment for the absolute awaitUrl returned by ProcessApplePay / ProcessGooglePay.
func (c *Client) AwaitURL(ctx context.Context, awaitURL string, timeout time.Duration) (*AwaitResult, error) {
	if awaitURL == "" {
		return nil, invalid("awaitURL", "required")
	}
	return call[AwaitResult](ctx, c, http.MethodGet, awaitURL, awaitTimeout(timeout))
}

func awaitTimeout(d time.Duration) reqOpt {
	if d <= 0 {
		return func(*resty.Request) {}
	}
	if d > MaxAwaitTimeout {
		d = MaxAwaitTimeout
	}
	return query("timeoutMs", strconv.FormatInt(d.Milliseconds(), 10))
}
