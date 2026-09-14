# Gateway

The Bonum payment gateway as seen by a merchant backend: opening hosted checkouts, storing
cards, charging them on demand or on a schedule, and receiving outcomes on a webhook.

## Language

### Access

**Terminal**:
The merchant's unit of integration. Identified by a Terminal ID and authenticated with an App Secret.
_Avoid_: merchant account, shop

**Checksum Key**:
The per-merchant secret Bonum uses to sign every gateway webhook delivery.
_Avoid_: webhook secret, signing key (that is the Wallet context's term)

### Checkout

**Invoice**:
A request for payment shown to the customer on Bonum's hosted checkout page. Settled through any enabled Payment Provider.
_Avoid_: order, checkout session, payment request

**Follow-up Link**:
The Bonum-hosted URL the customer's browser is sent to in order to pay an Invoice or complete a Tokenization.
_Avoid_: redirect URL, checkout URL

**Callback**:
The merchant URL the customer's browser returns to after the hosted page. It is user experience only and carries no trusted result.
_Avoid_: webhook, return URL

**Payment Provider**:
A way an Invoice can be settled on the hosted page: QPay, e-commerce card, WeChat, SonoShop.
_Avoid_: payment method, channel

**Transaction ID**:
The merchant's own unique identifier for an Invoice, Purchase, Tokenization or QR Invoice. Echoed back on the webhook.
_Avoid_: order id (that is the Wallet context's term), reference

### Cards

**Card Token**:
Bonum's opaque credential for a customer's stored card. The only thing the merchant keeps.
_Avoid_: card, PAN, token (ambiguous with bearer and wallet tokens)

**Tokenization**:
The hosted flow in which the customer enters a card and Bonum issues a Card Token.
_Avoid_: card registration, vaulting

**Purchase**:
A charge against a Card Token. Ends SUCCESS, FAILED, or QUEUED when Bonum defers the bank call.
_Avoid_: payment, charge, transaction

**Declined**:
A Purchase the bank refused. Bonum reports it as an HTTP error whose body still carries the Purchase record.
_Avoid_: failed (which also covers processing errors)

**Reversal**:
Rolling back a Purchase by its Transaction ID.
_Avoid_: refund, rollback, void

### Subscriptions

**Payment Plan**:
A recurring billing template, managed on the merchant portal, with a recurring type and amount.
_Avoid_: plan (alone), product

**Subscription**:
A Card Token enrolled in a Payment Plan.
_Avoid_: recurring payment, membership

**Cycle Value**:
The day within the recurring period on which a Subscription bills: 1-7 weekly, 1-31 monthly, 1-366 yearly.
_Avoid_: billing day

**Unsubscribe**:
Ending a Subscription after its already scheduled next billing runs.

**Delete**:
Ending a Subscription immediately with no further billing.
_Avoid_: cancel (ambiguous between the two)

### QR

**QR Invoice**:
A QPay-compatible invoice that a customer pays by scanning or by opening a Deeplink.
_Avoid_: QR code (that is the string inside it)

**Deeplink**:
A per-bank app link that opens a QR Invoice directly in that bank's app.
_Avoid_: bank link

### Webhook

**Event**:
One signed webhook delivery reporting the outcome of an Invoice, Tokenization, queued Purchase or Subscription billing.
_Avoid_: notification, message, callback

**Outcome**:
The SUCCESS or FAILED flag on every Event.
_Avoid_: status (used for Invoice and Purchase states)
