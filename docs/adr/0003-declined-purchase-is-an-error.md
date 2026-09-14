# A declined Purchase is returned as an error, not a FAILED result

Bonum answers a bank decline with HTTP 400 whose body still carries the Purchase record. We
surface it as `*DeclinedError`, which satisfies `errors.Is(err, ErrDeclined)` and still
carries the Purchase and trace id. The alternative, returning a Purchase with Status FAILED
and a nil error, was rejected because it makes the common `if err != nil` guard silently
treat a refused charge as success. A QUEUED Purchase, by contrast, is a nil-error result
because the charge is still in flight.
