# Wire types are the domain types; the transport envelope is not

The structs callers pass and receive are the JSON shapes Bonum documents, with invariants
enforced by `validate()` before any network call. We do not keep a separate DTO layer mapped
onto "pure" domain structs: for an SDK the vendor's contract *is* the domain, and a 1:1
mapping would be ceremony without substance. The one transport detail we do hide is the
mpay-service envelope (`traceId / message / data / status`): services unwrap `data` and
return the aggregate. The trace id survives only on errors, where it is useful for support.

Amended 2026-09-14 (ADR 0005): the wire types now live in `gateway/domain/<aggregate>` and
`wallet/domain/<aggregate>` and still carry their JSON tags. The layered structure did not
introduce a DTO layer; the adapter serialises the domain struct directly. A CQRS-lite Command
or Query type is a type alias to the domain input where one already exists (e.g.
`type TokenizeCommand = card.TokenizeInput`), not a parallel struct — the same "no ceremony
without substance" reasoning this ADR gives for skipping a DTO layer.
