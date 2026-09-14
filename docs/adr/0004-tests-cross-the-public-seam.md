# Tests live under tests/ and use only the exported interface

All tests are external packages under `tests/<context>/`, never `package bonum` siblings.
Go's convention is colocated white-box tests, so this is a deliberate deviation: the
interface is the test surface, and tests that reach into unexported state (a clock, an expiry
field) have to change whenever the implementation does. Where a behaviour needs steering, the
fake server steers it (a 10-second token TTL to force a refresh) rather than an injection
point added for tests alone. `internal/rest` has no tests of its own; both contexts' suites
exercise it.

Amended 2026-09-14 (ADR 0005): tests now mirror the layers. `tests/<context>/` keeps the
fake-server suites that cross the facade; `tests/<context>/domain/` tests invariants and
parsers with no HTTP; `tests/<context>/application/` tests each aggregate through its facade
(`application.NewCards(fakePort)`, not through an individual CQRS Handler constructor) against
hand-written fake ports — a Handler has no behaviour the facade-level test cannot already see,
so there is no separate per-Handler test tier. The wallet webhook gained `ParseAt(..., now)`
so the replay window is deterministic; it is an exported API a merchant replaying stored
deliveries also needs, not a test-only hook.
