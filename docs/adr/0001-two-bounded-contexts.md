# Gateway and Wallet are separate bounded contexts

Bonum runs two unrelated APIs: the gateway (invoices, card tokens, subscriptions, QR) and the
V2 wallet API (Apple Pay, Google Pay). They differ in host, credential, error body, result
model and webhook signature, and their vocabularies collide (`transactionId` vs `order_id`,
`PAID` vs `AUTHORIZED`). We model them as two packages, `bonum` and `wallet`, that share only
the `internal/rest` HTTP adapter and never import each other's types. A single client with
both surfaces was rejected because it would force one credential model onto both and invite
callers to treat a Wallet Payment as an Invoice.
