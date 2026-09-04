# bonum-go

Go SDK for the [Bonum Payment Gateway](https://psp.bonum.mn/) — invoices (checkout page), card
tokenization, token purchases, subscription plans, QR / deeplink payments, and webhook verification.

This library is **backend only**. `APP_SECRET`, `MERCHANT_CHECKSUM_KEY` and the bearer tokens must never
reach a browser or mobile app. See [FLOW.md](FLOW.md) for how backend, web and mobile fit together.

## Install

```sh
go get github.com/techpartners-asia/bonum-go
```

## Quick start

```go
import (
    bonum "github.com/techpartners-asia/bonum-go"
    "github.com/techpartners-asia/bonum-go/types"
)

client := bonum.New(types.Sandbox, appSecret, terminalID)   // types.Production for live
defer client.Close()

inv, err := client.CreateInvoice(types.CreateInvoiceInput{
    Amount:        15000,
    Callback:      "https://shop.example.mn/orders/123/return",
    TransactionID: "order-123",        // your unique id, echoed back in the webhook
    ExpiresIn:     900,                // seconds
    Providers:     []types.PaymentProvider{types.ProviderQPay, types.ProviderECommerce}, // optional
})
// redirect the customer to inv.FollowUpLink
```

Tokens are handled for you: the first call fetches an access token with your `APP_SECRET` +
`X-TERMINAL-ID`, later calls reuse it, and it is refreshed via `auth/refresh` before it expires.
`Authenticate()` / `Refresh()` exist if you want to force either step (e.g. at startup).

Options: `bonum.WithLanguage(types.EN)`, `bonum.WithTimeout(d)`, `bonum.WithBaseURL(url)`.

## Webhooks

Bonum POSTs the outcome of every payment / tokenization to the webhook URL registered in the
merchant portal, with an HMAC-SHA256 hex digest in the `x-checksum-v2` header. Verify against the
**raw** request bytes, then dispatch on `type`:

```go
func handler(w http.ResponseWriter, r *http.Request) {
    body, _ := io.ReadAll(r.Body)
    if !bonum.VerifyWebhook(body, r.Header.Get(bonum.ChecksumHeader), merchantChecksumKey) {
        w.WriteHeader(http.StatusUnauthorized)
        return
    }
    head, _ := bonum.PeekWebhook(body)
    switch head.Type {
    case types.WebhookPayment:
        m, _ := bonum.ParsePaymentWebhook(body)             // invoice paid / failed / expired
    case types.WebhookCardToken:
        m, _ := bonum.ParseCardTokenWebhook(body)           // store m.Body.Token for the customer
    case types.WebhookTokenPayment:
        m, _ := bonum.ParseTokenPaymentWebhook(body)        // result of a QUEUED Purchase
    case types.WebhookSubscriptionPayment:
        m, _ := bonum.ParseSubscriptionPaymentWebhook(body) // recurring charge ran
    }
    w.WriteHeader(http.StatusOK)
}
```

The webhook is the source of truth. The browser `callback` redirect is UX only.

## API surface

| Area | Methods |
|---|---|
| Auth | `Authenticate`, `Refresh` (automatic; rarely needed) |
| Invoices | `GetPaymentProviders`, `CreateInvoice`, `GetInvoiceStatusSandbox`\*, `SetInvoicePaidSandbox`\* |
| Card tokens | `CreateCardToken`, `Purchase`, `RollbackPurchase` |
| Subscriptions | `ListPaymentPlans`, `Subscribe`, `GetSubscriptions`, `ChangeSubscriptionTokenNew`, `ChangeSubscriptionTokenExisting`, `Unsubscribe`, `DeleteSubscription`, `ExecuteSubscriptionPaymentSandbox`\* |
| QR / deeplink | `CreateQrCode`, `InvoiceByQrCode`, `PayByCardToken` |
| Webhooks | `Checksum`, `VerifyWebhook`, `PeekWebhook`, `Parse*Webhook` |

\* Sandbox-only helpers. Bonum forbids polling invoice status in production.

Methods that take a `cardToken` send it as the `X-CARD-TOKEN` header.

### Errors

Any non-2xx reply is returned as `*bonum.Error` with `StatusCode`, `TraceID` (quote it to Bonum
support), `Message` and the raw `Body`. A declined card on `Purchase` is a 400 whose `Body` still
carries the `PurchaseResponse` envelope. `Purchase` may also succeed with `201` and
`Data.Status == QUEUED`; the final result then arrives as a `TOKEN-PAYMENT` webhook.

### Response types

`mpay-service` endpoints wrap their payload in `types.Envelope[T]`
(`traceId / errorCode / message / data / status`). Do not branch on `ErrorCode`; Bonum marks it
internal. A few endpoints have no example response in Bonum's collection — their `Data` is typed
as `json.RawMessage` (or the shape is noted as inferred in `types/`) until confirmed against the
sandbox.

## Environments

| | Base URL |
|---|---|
| `types.Sandbox` | `https://testapi.bonum.mn` |
| `types.Production` | `https://apis.bonum.mn` |

Credentials (`APP_SECRET`, terminal id, payment plans, webhook URL) are managed on the
[merchant portal](https://merchant.bonum.mn). `MERCHANT_CHECKSUM_KEY` is handed out by Bonum.

## Smoke test

```sh
BONUM_APP_SECRET=... BONUM_TERMINAL_ID=... go run ./tests
```

Unit tests (no network): `go test ./...`

Not covered: the "Neo App" chat endpoints in Bonum's collection (separate, undocumented hosts).
