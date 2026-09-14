# Tests live under tests/ and use only the exported interface

All tests are external packages under `tests/<context>/`, never `package bonum` siblings.
Go's convention is colocated white-box tests, so this is a deliberate deviation: the
interface is the test surface, and tests that reach into unexported state (a clock, an expiry
field) have to change whenever the implementation does. Where a behaviour needs steering, the
fake server steers it (a 10-second token TTL to force a refresh) rather than an injection
point added for tests alone. `internal/rest` has no tests of its own; both contexts' suites
exercise it.
