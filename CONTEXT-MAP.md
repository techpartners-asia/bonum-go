# Context Map

Bonum exposes two independent payment APIs. This SDK mirrors them as two bounded contexts
that share nothing but an internal HTTP adapter.

## Contexts

- [Gateway](./internal/gateway/CONTEXT.md) (package `bonum`): hosted checkout invoices, card
  tokens and purchases, subscriptions, QR invoices, and the gateway webhook.
- [Wallet](./internal/wallet/CONTEXT.md) (package `wallet`): Apple Pay and Google Pay
  payments submitted with an encrypted wallet token, and the wallet webhook.

## Relationships

- **Gateway ↔ Wallet**: no shared types. The same merchant may use both, but a Wallet
  Payment is never an Invoice and a Purchase is never a Wallet Payment. Each context has its
  own credential, host, error body and webhook signature.
- **Both → internal/rest**: a shared HTTP execution adapter. It knows nothing about either
  domain; each context hands it an error decoder.
- **Merchant backend → both**: the merchant's server is the only caller. Browsers and
  mobile apps never hold either credential; they redirect to a Follow-up Link (Gateway) or
  forward a Wallet Token (Wallet).
- **Inside each context**: `internal/<context>/domain` (aggregates, invariants, errors) ←
  `ports` (interfaces the use cases need) ← `application` (CQRS-lite: one Command/Query +
  Handler per use case, fronted by one facade struct per aggregate) ← `adapters/httpapi`
  (Bonum's HTTP endpoints). Living under `internal/` means none of this is importable outside
  this module at all, not just by convention. The root `bonum` package and the `wallet`
  package are the only public surface: facades, one file per aggregate, that compose the
  internal packages and re-export the domain types. `tests/architecture` enforces the
  direction.
