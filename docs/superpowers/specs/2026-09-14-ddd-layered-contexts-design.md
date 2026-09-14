# DDD layered bounded contexts for bonum-go

Date: 2026-09-14. Status: approved.

## Goal

Restructure the SDK so each bounded context (Gateway, Wallet) is a folder with explicit
`domain`, `ports`, `application` and `adapters` layers, while the public import surface stays
exactly what it is today: `bonum.New(...)`, `client.Invoices.Create(ctx, bonum.CreateInvoiceInput{...})`,
`bonum.ParseWebhook`, `errors.Is(err, bonum.ErrDeclined)`, `wallet.New(...)`, `psp.AwaitPayment(...)`.

Every existing test under `tests/gateway` and `tests/wallet` must keep passing without
modification. That is the compatibility contract.

## Non-goals (deliberately left out)

- No per-use-case handler structs, domain events, unit of work, outbox or repositories.
  An SDK has no persistence and no async consumers; every such seam would have no consumer.
- No separate DTO layer. ADR 0002 stands: domain structs carry the JSON tags Bonum documents.
- No change to `internal/rest`.
- No change to the vendor protocol, paths, headers or error mapping.

## Layout

```
bonum.go            facade (package bonum): Environment, Lang, Client, Option, New, Close,
                    Authenticate, Refresh
types.go            facade: type aliases for every exported Gateway domain and application type
errors.go           facade: sentinel error vars + error type aliases
webhook.go          facade: ChecksumHeader, Checksum, ParseWebhook wrappers

gateway/
  CONTEXT.md        moved from the repo root, unchanged
  domain/           package domain: ErrInvalidInput, ErrUnauthorized, ErrNotFound,
                    ErrRateLimited, APIError, ValidationError, Invalid(field, reason)
    access/         TokenPair
    checkout/       PaymentProvider, ExtraType, PaymentProviderStatus, Item, Extra,
                    CreateInvoiceInput (+Validate), Invoice
    card/           PurchaseStatus, TokenizePayment, TokenizeSubscription, TokenizeInput
                    (+Validate), Tokenization, PurchaseInput (+Validate), Purchase,
                    ErrDeclined, DeclinedError
    subscription/   RecurringType, PaymentPlan, SubscribeInput (+Validate), Subscription,
                    ChangeCardInput (+Validate)
    qr/             CreateQRInput (+Validate), Deeplink, QRInvoice, PayQRInput (+Validate)
    webhook/        ChecksumHeader, EventType, Outcome, Event, EventHeader, all *Event and
                    *Body types, Bank, Money, SubscriptionRef, ErrBadChecksum,
                    ErrUnknownEvent, Checksum, Parse
  ports/            package ports: AccessAPI, CheckoutAPI, CardAPI, SubscriptionAPI, QRAPI,
                    SandboxAPI
  application/      package application: Access, Invoices, Cards, Subscriptions, QR, Sandbox
  adapters/httpapi/ package httpapi: Client (implements all six ports), token lifecycle,
                    envelope unwrapping, request options, error decoding, decline detection

wallet/
  wallet.go         facade (package wallet): Environment, Client, Option, New, Close, the
                    six payment methods, aliases, error vars, webhook wrappers
  CONTEXT.md        unchanged
  domain/           package domain: ErrInvalidInput, ErrUnauthorized, ErrNotFound,
                    ErrRateLimited, APIError, ValidationError, Invalid
    payment/        Status, WalletType, Currency, BinCategory, Apple* types,
                    ProcessApplePayInput (+Validate), ProcessGooglePayInput (+Validate),
                    ProcessResponse, Payment, AwaitResult, MaxAwaitTimeout
    webhook/        SignatureHeader, TimestampHeader, ReplayTolerance, Event,
                    ErrMissingSignature, ErrTimestampExpired, ErrSignatureMismatch, Sign,
                    Parse, ParseAt
  ports/            package ports: PaymentAPI
  application/      package application: Payments
  adapters/httpapi/ package httpapi: Client (implements PaymentAPI), error decoding

internal/rest/      unchanged
tests/
  architecture/     deps_test.go: import-direction rules (see below)
  gateway/          existing facade suites, unchanged
  gateway/domain/   Validate tables per input type; webhook Parse edge cases
  gateway/application/  services against hand-written fake ports
  wallet/           existing facade suites, unchanged
  wallet/domain/    Validate tables; ParseAt replay window with a fixed clock
  wallet/application/   Payments against a fake port (await cap, id checks)
  smoke/            unchanged
```

Aggregate package names follow the CONTEXT.md sections: `checkout` (not `invoice`) because
the hosted-page value objects Item, Extra and PaymentProvider belong to checkout and are
reused by card and subscription inputs.

## Dependency rule

Enforced by `tests/architecture/deps_test.go`, which parses every non-test Go file with
`go/parser` in imports-only mode and checks module-internal imports:

| Package pattern | May import (inside the module) |
|---|---|
| `<ctx>/domain/**` | `<ctx>/domain`, sibling `<ctx>/domain/*` packages |
| `<ctx>/ports` | `<ctx>/domain/**` |
| `<ctx>/application` | `<ctx>/domain/**`, `<ctx>/ports` |
| `<ctx>/adapters/**` | `<ctx>/domain/**`, `<ctx>/ports`, `internal/rest` |
| facade (`bonum`, `wallet`) | anything in its own context, `internal/rest` |
| any `gateway/**` | never `wallet/**`, and vice versa |

Additionally `domain/**`, `ports` and `application` may import only the standard library
outside the module (no resty, no `internal/rest`). The test lists offending file and import.

## Layer contracts

### Domain

- Input structs gain an exported `Validate() error` that returns `*domain.ValidationError`
  (satisfying `domain.ErrInvalidInput`). Bodies are the current `validate()` bodies.
- `domain.Invalid(field, reason string) error` is the shared constructor.
- `card.DeclinedError` embeds `*domain.APIError` and carries `Purchase`; `Is` matches
  `card.ErrDeclined` and delegates to the API error.
- `webhook.Parse(body, checksumHeader, checksumKey) (Event, error)` is the current
  `ParseWebhook` body; `webhook.Checksum` is unchanged.
- Wallet `webhook.ParseAt(body, signature, timestamp, secret string, now time.Time)` holds
  the verification logic; `Parse` calls it with `time.Now()`. This is the one place time is
  passed in, so the replay window is testable without sleeping.
- Wallet `payment.MaxAwaitTimeout` stays a domain constant.
- Error message prefixes stay `bonum:` and `bonum wallet:`.

### Ports (interfaces the application layer depends on)

```go
// gateway/ports — method names are unique across all six interfaces because one adapter
// struct implements every port.
type AccessAPI interface {
    CreateToken(ctx) (*access.TokenPair, error)
    RefreshToken(ctx) (*access.TokenPair, error)
}
type CheckoutAPI interface {
    Providers(ctx) ([]checkout.PaymentProviderStatus, error)
    CreateInvoice(ctx, checkout.CreateInvoiceInput) (*checkout.Invoice, error)
}
type CardAPI interface {
    Tokenize(ctx, card.TokenizeInput) (*card.Tokenization, error)
    Purchase(ctx, cardToken string, card.PurchaseInput) (*card.Purchase, error) // *card.DeclinedError on decline
    Reverse(ctx, cardToken, transactionID string) error
}
type SubscriptionAPI interface {
    Plans(ctx) ([]subscription.PaymentPlan, error)
    Subscribe(ctx, cardToken string, subscription.SubscribeInput) (*subscription.Subscription, error)
    ListSubscriptions(ctx, cardToken string) ([]subscription.Subscription, error)
    ChangeCardByTokenizing(ctx, id int64, subscription.ChangeCardInput) (*card.Tokenization, error)
    ChangeCard(ctx, id int64, cardToken string) (*subscription.Subscription, error)
    Unsubscribe(ctx, id, planID int64) error
    DeleteSubscription(ctx, id, planID int64) error
}
type QRAPI interface {
    CreateQR(ctx, qr.CreateQRInput) (*qr.QRInvoice, error)
    LookupQR(ctx, qrCode string) (*qr.QRInvoice, error)
    PayQRWithCard(ctx, cardToken string, qr.PayQRInput) (*card.Purchase, error)
}
type SandboxAPI interface {
    InvoiceStatus(ctx, invoiceID string) (json.RawMessage, error)
    MarkInvoicePaid(ctx, invoiceID string) error
    RunSubscriptionBilling(ctx, id int64) error
}

// wallet/ports
type PaymentAPI interface {
    ProcessApplePay(ctx, payment.ProcessApplePayInput) (*payment.ProcessResponse, error)
    ProcessGooglePay(ctx, payment.ProcessGooglePayInput) (*payment.ProcessResponse, error)
    GetPayment(ctx, paymentID string) (*payment.Payment, error)
    LookupByOrderID(ctx, orderID string) (*payment.Payment, error)
    AwaitPayment(ctx, paymentID string, timeout time.Duration) (*payment.AwaitResult, error)
    AwaitURL(ctx, awaitURL string, timeout time.Duration) (*payment.AwaitResult, error)
}
```

Ports receive an already-validated input and an already-capped timeout. The adapter never
validates.

### Application

One struct per aggregate with a `New<Name>(port)` constructor and one method per use case,
keeping today's method names so the facade aliases them without renaming. A method body is:
validate (domain `Validate()` or an argument non-empty check) → call the port → return.
`Payments.Await` and `AwaitURL` additionally clamp `timeout` to `MaxAwaitTimeout`; the port
receives 0 for "server default".

`application.Access` wraps `AccessAPI` as `Authenticate` and `Refresh` so the facade never
calls a port directly.

### Adapters

`gateway/adapters/httpapi.Client` is one struct implementing all six gateway ports, with a
compile-time assertion per port. Constructor
`New(baseURL, appSecret, terminalID string) *Client` (30s timeout, language mn) plus
`SetBaseURL`, `SetTimeout`, `SetTransport`, `SetLanguage`, `Close`. Files split by aggregate mirror the
ports. The token lifecycle (`tokenSource`) stays inside the adapter unchanged; `Create` and
`Refresh` are its exported entry points. Error decoding builds `*domain.APIError`; a 400
whose envelope carries a FAILED Purchase becomes `*card.DeclinedError`.

`wallet/adapters/httpapi.Client` implements `PaymentAPI`; constructor
`New(baseURL, merchantKey string)` (35s timeout).

### Facades

`bonum.New` builds the adapter with the environment host, applies options (which forward to
adapter setters), then constructs the application services and exposes them as today:
`Invoices`, `Cards`, `Subscriptions`, `QR`, `Sandbox`. Service types are aliases
(`type InvoiceService = application.Invoices`). `Environment` and `Lang` stay facade types.

`types.go` carries one alias line per exported domain type, grouped by aggregate. Sentinel
errors are re-exported as `var ErrX = domain.ErrX` so `errors.Is` identity holds. Function
wrappers (`ParseWebhook`, `Checksum`, wallet `Sign`, `ParseWebhook`) keep their doc comments.

`wallet.Client` keeps its flat method set; each method delegates to `application.Payments`.

## Testing

- All existing tests stay byte-for-byte unchanged and green.
- Domain tests are table tests over `Validate()` and the webhook parsers, no HTTP.
- Application tests use small fake port structs that record the last call and return a
  configured result or error. They assert: invalid input never reaches the port, valid input
  is forwarded unchanged, port errors pass through unwrapped, await timeouts are clamped.
- The architecture test fails with a readable list of `file: forbidden import` lines.
- Verification command: `go build ./... && go vet ./... && go test ./...`.

## Documentation

- `docs/adr/0005-layered-contexts.md`: why an SDK gets layers (mockable ports, resty isolated
  in adapters, rules testable without HTTP) and the list of non-goals above.
- Amend ADR 0002: domain packages still carry JSON tags. Amend ADR 0004: tests mirror the
  layer folders; the wallet `ParseAt` clock parameter is a real API, not a test hook.
- `CONTEXT-MAP.md` links to `gateway/CONTEXT.md`; README layout section rewritten;
  `FLOW.md` needs no change.
- `docs/integration-report.md` is a dated snapshot pinned to a commit and is left as is.
