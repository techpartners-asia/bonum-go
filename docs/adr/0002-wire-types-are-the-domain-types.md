# Wire types are the domain types; the transport envelope is not

The structs callers pass and receive are the JSON shapes Bonum documents, with invariants
enforced by `validate()` before any network call. We do not keep a separate DTO layer mapped
onto "pure" domain structs: for an SDK the vendor's contract *is* the domain, and a 1:1
mapping would be ceremony without substance. The one transport detail we do hide is the
mpay-service envelope (`traceId / message / data / status`): services unwrap `data` and
return the aggregate. The trace id survives only on errors, where it is useful for support.
