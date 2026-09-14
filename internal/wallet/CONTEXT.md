# Wallet

Bonum's V2 API for Apple Pay and Google Pay. The customer authorises in a native wallet
sheet, the app forwards the encrypted token, and the merchant backend submits it to Bonum.

## Language

**Merchant Key**:
The credential issued at onboarding that authenticates every Wallet request.
_Avoid_: API key, app secret (Gateway term)

**Wallet Token**:
The encrypted payload a wallet sheet hands to the app: Apple's PKPaymentToken object or Google's tokenization string. Forwarded untouched.
_Avoid_: card token (Gateway term), payment token

**Wallet Payment**:
One submission of a Wallet Token. Always PENDING when accepted; ends AUTHORIZED or FAILED.
_Avoid_: purchase, invoice, transaction

**Order ID**:
The merchant's own unique identifier for a Wallet Payment. Resubmitting the same Order ID returns the existing Wallet Payment.
_Avoid_: transaction id (Gateway term)

**Authorized**:
The bank approved and reserved the funds. Bonum's documented signal that it is safe to fulfil.
_Avoid_: paid, captured, settled

**Await**:
Holding one request open until a Wallet Payment reaches a final status, so the app can close the wallet sheet inside Apple's 30-second window.
_Avoid_: poll, long-poll

**Timed Out**:
An Await that ended before the bank answered. The Wallet Payment keeps processing; its outcome arrives on the webhook.

**Webhook Event**:
One signed delivery reporting that a Wallet Payment became AUTHORIZED or FAILED. Identified by a Webhook ID that is the idempotency key.
_Avoid_: notification, callback

**Signing Secret**:
The per-merchant secret behind the X-PSP-Signature header.
_Avoid_: checksum key (Gateway term)

**BIN Category**:
Whether the card's issuing bank is DOMESTIC or INTERNATIONAL, when Bonum could resolve it.
