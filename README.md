# bonum-go

Go SDK for the [Bonum Payment Gateway](https://psp.bonum.mn/): hosted checkout invoices,
card tokenization, token purchases, subscription plans, QR / deeplink payments and webhook
verification (package `bonum`), plus Bonum's separate V2 API for Apple Pay / Google Pay
(package `wallet`).

This library is **backend only**. The App Secret, Checksum Key, Merchant Key and bearer
tokens must never reach a browser or mobile app. See [FLOW.md](FLOW.md) for how backend,
web and mobile fit together, [CONTEXT-MAP.md](CONTEXT-MAP.md) for the two bounded contexts
and their vocabulary, and `docs/adr/` for the design decisions behind the shape of the API.

## Install

```sh
go get github.com/techpartners-asia/bonum-go
```

## Quick start

```go
import bonum "github.com/techpartners-asia/bonum-go"

client := bonum.New(bonum.Sandbox, appSecret, terminalID)   // bonum.Production for live
defer client.Close()

inv, err := client.Invoices.Create(ctx, bonum.CreateInvoiceInput{
    Amount:        15000,
    Callback:      "https://shop.example.mn/orders/123/return",
    TransactionID: "order-123",        // your unique id, echoed back on the webhook
    ExpiresIn:     900,                // seconds
    Providers:     []bonum.PaymentProvider{bonum.ProviderQPay, bonum.ProviderECommerce}, // optional
})
// redirect the customer to inv.FollowUpLink
```

Every call takes a `context.Context`. Inputs are validated before any network call and
return `ErrInvalidInput` if an invariant is broken. Tokens are handled for you: the first
call fetches an access token with your App Secret and Terminal ID, later calls reuse it, and
it is refreshed before it expires. `Authenticate` / `Refresh` exist to force either step.

Options: `bonum.WithLanguage(bonum.EN)`, `bonum.WithTimeout(d)`, `bonum.WithBaseURL(url)`,
`bonum.WithTransport(rt)`.

## Services

The client is grouped by aggregate. Sandbox-only helpers live on their own service so they
cannot leak into production code paths.

| Service | Methods |
|---|---|
| `Invoices` | `Providers`, `Create` |
| `Cards` | `Tokenize`, `Purchase`, `Reverse` |
| `Subscriptions` | `Plans`, `Subscribe`, `List`, `ChangeCard`, `ChangeCardByTokenizing`, `Unsubscribe`, `Delete` |
| `QR` | `Create`, `Lookup`, `PayWithCard` |
| `Sandbox` | `InvoiceStatus`, `MarkInvoicePaid`, `RunSubscriptionBilling` |

Methods that take a card token send it as the `X-CARD-TOKEN` header. mpay-service
responses are unwrapped: you get the `Purchase`, `Subscription` or `QRInvoice`, never the
`traceId / data / status` envelope.

### Errors

```go
p, err := client.Cards.Purchase(ctx, cardToken, bonum.PurchaseInput{Amount: 15000, Currency: "MNT", TransactionID: "order-124"})
switch {
case errors.Is(err, bonum.ErrDeclined):
    var d *bonum.DeclinedError          // d.Purchase.RespCode, d.TraceID
case errors.Is(err, bonum.ErrInvalidInput):
case errors.Is(err, bonum.ErrUnauthorized):
case errors.Is(err, bonum.ErrRateLimited):
case err != nil:
    var api *bonum.APIError             // StatusCode, TraceID (quote it to Bonum support), Message, Body
case p.Status == bonum.PurchaseQueued:
    // final result arrives as a TokenPaymentEvent
}
```

Sentinels: `ErrInvalidInput` (local validation or HTTP 400), `ErrUnauthorized` (401/403),
`ErrNotFound` (404), `ErrRateLimited` (429), `ErrDeclined` (bank refused a Purchase),
`ErrBadChecksum` and `ErrUnknownEvent` (webhooks).

## Webhooks

Bonum POSTs the outcome of every payment and tokenization to the URL registered in the
merchant portal, signed with an HMAC-SHA256 hex digest in `x-checksum-v2`. One call verifies
against the **raw** request bytes and returns a typed event:

```go
func handler(w http.ResponseWriter, r *http.Request) {
    body, _ := io.ReadAll(r.Body)
    ev, err := bonum.ParseWebhook(body, r.Header.Get(bonum.ChecksumHeader), checksumKey)
    if err != nil {
        w.WriteHeader(http.StatusUnauthorized) // ErrBadChecksum, ErrUnknownEvent, malformed body
        return
    }
    switch e := ev.(type) {
    case *bonum.PaymentEvent:             // invoice or QR invoice paid / failed / expired
        _ = e.Outcome == bonum.OutcomeSuccess
    case *bonum.CardTokenEvent:           // store e.Body.Token for the customer
    case *bonum.TokenPaymentEvent:        // result of a QUEUED Purchase
    case *bonum.SubscriptionPaymentEvent: // recurring charge ran
    }
    w.WriteHeader(http.StatusOK)
}
```

The webhook is the source of truth. The browser `Callback` redirect is UX only.

## Apple Pay / Google Pay (`wallet` package)

Bonum runs a second, independent API for wallet payments. It has its own hosts, is
authenticated with a Merchant Key issued at onboarding, and is asynchronous: every submission
returns `PENDING` and the final result comes from a blocking await call or the webhook. The
mobile app or web page only shows the wallet sheet and forwards the encrypted token to your
backend; the backend calls Bonum.

```go
import "github.com/techpartners-asia/bonum-go/wallet"

psp := wallet.New(wallet.Sandbox, merchantKey)   // wallet.Production for live
defer psp.Close()

// iOS: decode the PKPaymentToken JSON the app sent you into wallet.ApplePayToken.
res, err := psp.ProcessApplePay(ctx, wallet.ProcessApplePayInput{
    OrderID: "ORDER-2024-00123",
    Amount:  150.50,            // optional, overrides the token amount
    Token:   token,
})

// Android: forward paymentMethodData.tokenizationData.token as-is.
res, err = psp.ProcessGooglePay(ctx, wallet.ProcessGooglePayInput{
    OrderID:      "ORDER-2024-00124",
    Token:        googleToken,
    CurrencyCode: wallet.MNT,
    Amount:       150.50,
})

// Block until the bank answers so the app can close the sheet within Apple's 30 s window.
r, err := psp.AwaitPayment(ctx, res.PaymentID, 25*time.Second)   // or psp.AwaitURL(ctx, res.AwaitURL, ...)
switch {
case err != nil || r.TimedOut:
    // tell the app to complete with FAILURE; the webhook delivers the real outcome
case r.Status == wallet.StatusAuthorized:
    // tell the app to complete with SUCCESS
default:
    // r.FailureReason
}
```

`GetPayment` and `LookupByOrderID` return the full `wallet.Payment` record for polling and
reconciliation. `AwaitPayment` caps the wait at 28 s, Bonum's own limit. Errors follow the
same pattern as the gateway: `wallet.ErrInvalidInput`, `ErrUnauthorized`, `ErrNotFound`,
`ErrRateLimited`, and `*wallet.APIError`.

Wallet webhooks are signed differently: `X-PSP-Signature: v1=<hex>` is an HMAC-SHA256 over
`"<X-PSP-Timestamp>.<raw body>"`, and deliveries older than five minutes are rejected.
`WebhookID` is the idempotency key; return 200 for duplicates.

```go
func walletHandler(w http.ResponseWriter, r *http.Request) {
    body, _ := io.ReadAll(r.Body)
    ev, err := wallet.ParseWebhook(body, r.Header.Get(wallet.SignatureHeader), r.Header.Get(wallet.TimestampHeader), signingSecret)
    if err != nil {
        w.WriteHeader(http.StatusBadRequest) // ErrMissingSignature, ErrTimestampExpired, ErrSignatureMismatch
        return
    }
    // dedupe on ev.WebhookID, then fulfil when ev.EventType == wallet.StatusAuthorized
    w.WriteHeader(http.StatusOK)
}
```

Platform setup (Apple merchant ID and processing certificate, Google Pay Business Console
review) is described in `docs/integration-report.md` §11. No capture, void or refund
endpoint is documented for V2; confirm with Bonum how `AUTHORIZED` settles before go-live.

## Environments

| | Gateway (`bonum`) | Wallet (`wallet`) |
|---|---|---|
| Sandbox | `https://testapi.bonum.mn` | `https://testpsp.bonum.mn` |
| Production | `https://apis.bonum.mn` | `https://psp.bonum.mn` |

Gateway credentials (App Secret, Terminal ID, payment plans, webhook URL) are managed on the
[merchant portal](https://merchant.bonum.mn); the Checksum Key is handed out by Bonum. The
wallet Merchant Key and Signing Secret are issued by Bonum at onboarding.

## Layout

```
bonum.go, invoice.go, card.go, subscription.go, qr.go, sandbox.go, webhook.go, errors.go, auth.go
                       Gateway context: one file per aggregate, CONTEXT.md is its glossary
wallet/                Wallet context: Apple Pay / Google Pay, own CONTEXT.md
internal/rest/         HTTP execution adapter shared by both contexts
tests/gateway, tests/wallet
                       External test packages that cross only the exported interface
tests/smoke            Manual run against the sandbox
docs/adr/              Why the API is shaped this way
```

## Smoke test

```sh
BONUM_APP_SECRET=... BONUM_TERMINAL_ID=... go run ./tests/smoke
```
