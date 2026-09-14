# DDD layered bounded contexts for bonum-go

Date: 2026-09-14. Status: approved.

**Amended 2026-09-14 (mid-execution):** the user asked for CQRS after Tasks 1-3 (domain
layer) were already implemented on `ddd-layers`. This amendment changes only the
**Application** layer contract below (now CQRS-lite: one Command/Query + Handler per use
case) and the parts of Layout/Dependency rule/Documentation that describe it. Domain, ports,
adapters and facades — and everything already built before the amendment — are unchanged,
because the facade's public method names and `NewX(port)` constructor signatures stay
identical either way.

**Amended 2026-09-14 (second, after all ten code tasks landed):** the user asked for the
facade's own files (`types.go`, `errors.go`, `webhook.go`) to be "more architectural" too.
Resolved as: one facade file per aggregate (mirroring `domain/<aggregate>`) instead of one
file per kind of declaration; `types.go` is retired, its content redistributed into
`access.go`/`checkout.go`/`card.go`/`subscription.go`/`qr.go`/`sandbox.go` plus the existing
`errors.go` (trimmed to shared errors only) and `webhook.go` (gaining the webhook type
aliases). Wallet gets the same split via a new `errors.go`, a `payment.go` in place of
`types.go`, and a new `webhook.go`. This is a pure rename/regroup of type aliases — no
exported identifier changed — so it needed no test changes and no adapter/domain/ports
changes; see the Layout and Facades sections below for the resulting shape.

## Goal

Restructure the SDK so each bounded context (Gateway, Wallet) is a folder with explicit
`domain`, `ports`, `application` and `adapters` layers, while the public import surface stays
exactly what it is today: `bonum.New(...)`, `client.Invoices.Create(ctx, bonum.CreateInvoiceInput{...})`,
`bonum.ParseWebhook`, `errors.Is(err, bonum.ErrDeclined)`, `wallet.New(...)`, `psp.AwaitPayment(...)`.

Every existing test under `tests/gateway` and `tests/wallet` must keep passing without
modification. That is the compatibility contract.

## Non-goals (deliberately left out)

- No generic command bus, mediator, or reflection-based dispatch. Each aggregate facade
  calls its own Handlers directly; there is nothing here for a bus to decouple.
- No domain events, unit of work, outbox or repositories. An SDK has no persistence and no
  async consumers; every such seam would have no consumer.
- No separate DTO layer. ADR 0002 stands: domain structs carry the JSON tags Bonum documents;
  a Command/Query is a type alias to the domain input where one already exists.
- No change to `internal/rest`.
- No change to the vendor protocol, paths, headers or error mapping.

## Layout

```
bonum.go            facade (package bonum): Environment, Lang, Client, Option, New, Close,
                    Authenticate, Refresh
errors.go           facade: errors shared by every aggregate (ErrInvalidInput, ErrUnauthorized,
                    ErrNotFound, ErrRateLimited, APIError, ValidationError)
access.go, checkout.go, card.go, subscription.go, qr.go, sandbox.go
                    facade, one file per aggregate: that aggregate's type aliases, its
                    Service alias, and any error specific to it (ErrDeclined/DeclinedError
                    live in card.go, not errors.go)
webhook.go          facade: webhook type aliases, ErrBadChecksum/ErrUnknownEvent,
                    ChecksumHeader, Checksum, ParseWebhook wrapper

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
  application/      CQRS-lite. commands/<aggregate> and queries/<aggregate> hold one
                    Command/Query type + *Handler per use case; the package root holds one
                    facade struct per aggregate (Access, Invoices, Cards, Subscriptions, QR,
                    Sandbox) that keeps the old per-aggregate method names and NewX(port)
                    constructors, delegating to its Handlers
  adapters/httpapi/ package httpapi: Client (implements all six ports), token lifecycle,
                    envelope unwrapping, request options, error decoding, decline detection

wallet/
  wallet.go         facade (package wallet): Environment, Client, Option, New, Close, the
                    six payment methods
  errors.go         facade: errors shared by the Payment aggregate (ErrInvalidInput,
                    ErrUnauthorized, ErrNotFound, ErrRateLimited, APIError, ValidationError)
  payment.go        facade: Payment aggregate type aliases (Status, ProcessApplePayInput,
                    Payment, AwaitResult, MaxAwaitTimeout, ...)
  webhook.go        facade: webhook type alias, ErrMissingSignature/ErrTimestampExpired/
                    ErrSignatureMismatch, Sign, ParseWebhook wrapper
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
  application/      CQRS-lite, same shape as gateway: commands/payment and queries/payment
                    hold one Command/Query + *Handler per use case; the package root holds
                    one facade struct (Payments) with the old method names and NewPayments
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
| `<ctx>/application/**` | `<ctx>/domain/**`, `<ctx>/ports`, sibling `<ctx>/application/**` packages |
| `<ctx>/adapters/**` | `<ctx>/domain/**`, `<ctx>/ports`, `internal/rest` |
| facade (`bonum`, `wallet`) | anything in its own context, `internal/rest` |
| any `gateway/**` | never `wallet/**`, and vice versa |

`<ctx>/application/**` covers the package root (the aggregate facades) and its
`commands/<aggregate>` and `queries/<aggregate>` subpackages alike: all three are the same
layer, so a facade importing its own Handlers is an intra-layer dependency, not a violation.
Additionally `domain/**`, `ports` and `application/**` may import only the standard library
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

### Application (CQRS-lite)

Every use case is one of:

- a **Command**, under `application/commands/<aggregate>/`, for anything that changes state
  at Bonum (tokenize, purchase, reverse, create an invoice/QR, subscribe, change/unsubscribe/
  delete a subscription, mark an invoice paid, run a billing cycle, submit a wallet token);
- a **Query**, under `application/queries/<aggregate>/`, for anything that only reads state
  (list providers/plans/subscriptions, look up a QR/invoice/payment, await a payment's
  outcome — `Await*` blocks but never mutates, so it is a Query).

Each use case gets its own file: a Command/Query type (a type alias to the existing domain
input where one already exists, e.g. `type TokenizeCommand = card.TokenizeInput`; a small
struct when the port needs extra addressing like a card token or subscription ID; nothing at
all as a parameter when the use case is niladic — `Handle(ctx)` with no input, for
`Providers`, `Plans`, `Authenticate`, `Refresh`) plus a `*Handler` with a `New<UseCase>Handler(port)`
constructor and a `Handle(ctx, ...) (result, error)` method. A Handle body is: validate
(domain `Validate()`, or an argument non-empty check, when the use case takes one) → call the
port → return. `AwaitPayment`/`AwaitURL` additionally clamp `Timeout` to `MaxAwaitTimeout`
inside their Query's Handler (`clampAwait`, defined once in `queries/payment`); the port
receives 0 for "server default".

One facade struct per aggregate lives at the `application` package root (`Access`,
`Invoices`, `Cards`, `Subscriptions`, `QR`, `Sandbox`, `Payments`), each still built by
`New<Name>(port)` and still exposing exactly today's method names and signatures — the
facade holds its aggregate's Handlers and each method is a one-line `return
s.<handler>.Handle(ctx, ...)`. This is what keeps the amendment's blast radius inside
`application/`: ports, adapters and the root `bonum`/`wallet` facades never change, because
from their side an aggregate's public shape is identical to a non-CQRS build.

Where a `commands/<aggregate>` or `queries/<aggregate>` package would collide on import with
the domain package of the same name (both, e.g., named `card`), alias the two application-
layer imports as `<aggregate>cmd` / `<aggregate>qry` in the facade file and leave the domain
import unaliased — applied consistently, this needs no per-file special-casing.

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

The facade's own files are split one per aggregate, mirroring `domain/<aggregate>`:
`access.go`, `checkout.go`, `card.go`, `subscription.go`, `qr.go`, `sandbox.go`, `webhook.go`
(plus `bonum.go` for the `Client` itself). Each carries that aggregate's type aliases, its
`Service` alias, and any error specific to it — `ErrDeclined`/`DeclinedError` in `card.go`,
`ErrBadChecksum`/`ErrUnknownEvent` in `webhook.go`. `errors.go` keeps only what every
aggregate shares: `ErrInvalidInput`, `ErrUnauthorized`, `ErrNotFound`, `ErrRateLimited`,
`APIError`, `ValidationError`. All errors are re-exported as `var ErrX = domain.ErrX` (or the
aggregate package's own sentinel) so `errors.Is` identity holds. Function wrappers
(`ParseWebhook`, `Checksum`, wallet `Sign`, `ParseWebhook`) keep their doc comments. Wallet
mirrors this with `errors.go` (shared), `payment.go` (the one non-webhook aggregate), and
`webhook.go`.

`wallet.Client` keeps its flat method set; each method delegates to `application.Payments`.

## Testing

- All existing tests stay byte-for-byte unchanged and green.
- Domain tests are table tests over `Validate()` and the webhook parsers, no HTTP.
- Application tests use small fake port structs that record the last call and return a
  configured result or error, exercised through each aggregate's facade (`application.NewCards(fake)`,
  not through an individual Handler constructor). They assert: invalid input never reaches
  the port, valid input is forwarded unchanged, port errors pass through unwrapped, await
  timeouts are clamped. This is unchanged by the CQRS-lite split: the facade's public method
  is still the one test surface, and it now happens to delegate to a Handler instead of
  calling the port inline — no separate per-Handler test tier, since a Handler has no
  behaviour a facade-level test cannot already see.
- The architecture test fails with a readable list of `file: forbidden import` lines.
- Verification command: `go build ./... && go vet ./... && go test ./...`.

## Documentation

- `docs/adr/0005-layered-contexts.md`: why an SDK gets layers (mockable ports, resty isolated
  in adapters, rules testable without HTTP), why the application layer is CQRS-lite
  (Command/Query + Handler per use case, no bus), and the list of non-goals above.
- Amend ADR 0002: domain packages still carry JSON tags. Amend ADR 0004: tests mirror the
  layer folders; the wallet `ParseAt` clock parameter is a real API, not a test hook.
- `CONTEXT-MAP.md` links to `gateway/CONTEXT.md`; README layout section rewritten;
  `FLOW.md` needs no change.
- `docs/integration-report.md` is a dated snapshot pinned to a commit and is left as is.
