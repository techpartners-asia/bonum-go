# Each context is layered domain / ports / application / adapters, CQRS-lite, under internal/

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

Both contexts' `domain`, `ports`, `application` and `adapters` now live under `internal/`:
`internal/gateway/...` and `internal/wallet/...`, alongside the pre-existing `internal/rest`.
`tests/architecture` already enforced that only `bonum` and `wallet` are the intended public
surface, but that was a convention a merchant could still route around — before this move,
nothing stopped an external module from importing `.../bonum-go/gateway/domain/card` directly.
Moving the layers under `internal/` makes that a compiler error instead: Go refuses to build
an import of an `internal/` package from outside the module tree it lives under.
`tests/architecture` still does its job — it catches an inward-import violation *inside* the
module, which `internal/` visibility does not — so the two mechanisms are complementary, not
redundant. The move only touched import paths: every exported identifier, every test, and
the two facades' public shape are unchanged.
