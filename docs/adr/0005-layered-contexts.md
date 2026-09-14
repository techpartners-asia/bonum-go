# Each context is layered domain / ports / application / adapters, CQRS-lite

The gateway and wallet contexts were flat packages: aggregate types, validation, HTTP calls
and error decoding in one file per aggregate. That was compact but had three costs. A
merchant could not test code that uses the SDK without an HTTP fake, because the only seam
was resty. Domain rules (the await cap, cycle values, what a decline is) were tested only
through HTTP. And nothing stopped transport details from leaking into types callers hold.

Each context is now four packages with imports pointing inward: `domain/<aggregate>` holds
the types, `Validate` methods and domain errors; `ports` holds the interfaces the use cases
call; `application` implements those use cases CQRS-lite — one Command or Query type plus a
`*Handler` per use case, under `application/commands/<aggregate>` for anything that changes
state at Bonum and `application/queries/<aggregate>` for anything that only reads it — fronted
by one facade struct per aggregate at the package root (`Cards`, `Invoices`, `Payments`, ...)
that keeps the pre-existing method names and `New<Name>(port)` constructor so nothing calling
it needed to change; `adapters/httpapi` implements every port with resty against Bonum's
endpoints. Domain and ports do not care whether the layer above them is one service per
aggregate or one handler per use case, which is why the CQRS split could be added after
Tasks 1-3 had already landed without touching them.

The root `bonum` package and the `wallet` package are facades: they compose an adapter into
the aggregate facades and re-export the domain types as aliases, so the public API did not
change and a merchant still imports one package. That facade layer is itself split one file
per aggregate — `access.go`, `checkout.go`, `card.go`, `subscription.go`, `qr.go`,
`sandbox.go`, `webhook.go` (`payment.go` and `webhook.go` for wallet) — mirroring
`domain/<aggregate>` file for file, plus `bonum.go`/`wallet.go` for the `Client` itself and
`errors.go` for the handful of sentinel errors and error types every aggregate shares
(`ErrInvalidInput`, `ErrUnauthorized`, `ErrNotFound`, `ErrRateLimited`, `APIError`,
`ValidationError`). An aggregate-specific error stays with its aggregate: `ErrDeclined` and
`DeclinedError` live in `card.go`, `ErrBadChecksum` and `ErrUnknownEvent` in `webhook.go`,
`ErrMissingSignature`/`ErrTimestampExpired`/`ErrSignatureMismatch` in `wallet/webhook.go`. A
single `types.go` (and, on the wallet side, a `types.go` that also carried the errors and the
webhook wrapper) would have grown without bound as aggregates were added; a reader who wants
to know what `Purchase` looks like at the public API now opens `card.go`, not a file holding
every exported type in the package. `tests/architecture` fails the build if an import goes
the wrong way. `go test ./...` needed no changes for either split: the file layout moved, no
exported identifier did.

We deliberately left out the rest of the usual DDD toolkit. There is no generic command bus
or mediator (each aggregate facade dispatches to its own Handlers directly, so there is
nothing here for a bus to decouple), no domain events (nothing consumes them), no
repositories or unit of work (an SDK has no persistence), and no separate DTO layer (ADR
0002). Each of those would be a seam with no consumer.
