# Changelog

## Unreleased

- `ParseWebhook` no longer refuses a genuine delivery over whitespace. Bonum signs
  `JSON.toJson(body, prettyPrint = false)` — a compact re-serialisation — not the bytes it
  sends, so a body that arrived pretty-printed or with a trailing newline failed with
  `ErrBadChecksum` even with the right key. The raw bytes are still tried first; on a
  mismatch the body is compacted (`json.Compact`: key order and number spelling kept) and
  checked again. The key is required either way.
- The `x-checksum-v2` value is trimmed and compared case-insensitively; an uppercase-hex
  spelling of the right MAC is accepted.

## v0.2.0

**Breaking change from v0.1.0.** The whole public API was redesigned: flat top-level
functions and a separate `types` package gave way to a `Client` grouped by aggregate, a
`context.Context` on every call, and a new `wallet` package for Apple Pay / Google Pay
(which did not exist in v0.1.0 at all). There is no compatibility shim; update call sites by
hand using the table below. Internally the SDK also moved to a layered, CQRS-lite
architecture under `internal/` — see `docs/adr/0005-layered-contexts.md` — but that part is
not observable from outside the module and needs no code changes.

### Migrating from v0.1.0

| v0.1.0 | v0.2.0 |
|---|---|
| `types.Environment`, `types.Lang` | `bonum.Environment`, `bonum.Lang` (same values) |
| `bonum.New(env, appSecret, terminalID, opts...)` (no context on any call) | `bonum.New(env, appSecret, terminalID, opts...)`; every method now takes a `context.Context` as its first argument |
| `client.GetPaymentProviders()` | `client.Invoices.Providers(ctx)` |
| `client.CreateInvoice(types.CreateInvoiceInput{...})` | `client.Invoices.Create(ctx, bonum.CreateInvoiceInput{...})` |
| `client.GetInvoiceStatusSandbox(id)` / `client.SetInvoicePaidSandbox(id)` | `client.Sandbox.InvoiceStatus(ctx, id)` / `client.Sandbox.MarkInvoicePaid(ctx, id)` |
| `client.CreateCardToken(types.CreateCardTokenInput{...})` | `client.Cards.Tokenize(ctx, bonum.TokenizeInput{...})` |
| `client.Purchase(cardToken, types.PurchaseInput{...})` | `client.Cards.Purchase(ctx, cardToken, bonum.PurchaseInput{...})` |
| `client.RollbackPurchase(cardToken, transactionID)` | `client.Cards.Reverse(ctx, cardToken, transactionID)` |
| Subscription plan/QR methods on `Client` directly | grouped under `client.Subscriptions.*` and `client.QR.*` |
| `bonum.Error` | `bonum.APIError` (same role: non-2xx response; field names also changed — check call sites that read fields directly) |
| `bonum.VerifyWebhook(body, header, key) bool` + `bonum.PeekWebhook` + four separate `bonum.Parse*Webhook` functions | one `bonum.ParseWebhook(body, header, key) (bonum.Event, error)`, verifying and decoding in one call; type-switch on the returned `Event` (`*PaymentEvent`, `*CardTokenEvent`, `*TokenPaymentEvent`, `*SubscriptionPaymentEvent`) |
| no wallet API | new `wallet` package: `wallet.New(env, merchantKey)` for Apple Pay / Google Pay — see the README |

### Also in this release

- Validation was missing on several gateway methods that take bare scalar arguments
  (`Cards.Reverse`, `Subscriptions.ChangeCard`/`Unsubscribe`/`Delete`/`List`, all three
  `Sandbox` methods) — they now return `ErrInvalidInput` for an empty string or a
  non-positive ID before making a network call, matching every other method.
- `wallet.AwaitURL` now rejects a URL whose host doesn't match the configured environment,
  instead of sending the merchant-key header to whatever host it's given. Always pass through
  the `awaitUrl` Bonum returns in `ProcessResponse` unmodified; that continues to work exactly
  as before.
- Added `wallet.ParseWebhookAt(body, signature, timestamp, secret, now)` for replaying a
  stored webhook delivery against its original time instead of the wall clock.

### Known inconsistency, not fixed here

The decoded webhook union type is named `bonum.Event` on the gateway side and
`wallet.WebhookEvent` on the wallet side. Renaming either to match the other is a breaking
change and out of scope for a patch release; noted here rather than silently left
unexplained.
