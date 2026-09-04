# Bonum integration flow — backend, web, mobile

Bonum (`psp.bonum.mn`) is a **server-to-server** gateway. Only your backend talks to Bonum.
Web and mobile clients talk to *your* backend and are handed a Bonum-hosted link, a QR image,
or bank deeplinks. Secrets (`APP_SECRET`, `MERCHANT_CHECKSUM_KEY`, bearer tokens) never leave the
backend. This SDK is the backend piece.

```
 web / mobile ──── your API ────► your backend ──── bonum-go ────► Bonum
      ▲                              ▲                                   │
      │  followUpLink / qrImage      │  webhook (x-checksum-v2)          │
      └──────────────────────────────┴───────────────────────────────────┘
```

Two rules underpin everything below:

1. **The webhook is the source of truth.** The browser `callback` redirect only tells you the
   customer came back; it does not prove they paid.
2. **Verify every webhook** with `bonum.VerifyWebhook(rawBody, header, MERCHANT_CHECKSUM_KEY)`
   before acting on it. Then look up the order by `transactionId` (your id) and update it
   idempotently — Bonum may retry deliveries.

---

## 0. Authentication (automatic)

```mermaid
sequenceDiagram
    participant B as Backend (SDK)
    participant G as Bonum
    B->>G: GET /auth/create  (Authorization: AppSecret …, X-TERMINAL-ID)
    G-->>B: accessToken (30 min), refreshToken
    Note over B: cached in memory
    B->>G: any API call  (Authorization: Bearer accessToken)
    Note over B,G: near expiry → GET /auth/refresh (Bearer refreshToken)
```

`auth/create` is rate limited (429 if hammered). The SDK fetches once, reuses, and refreshes
before expiry; you never pass tokens around. Create one `bonum.Client` per process/terminal.

---

## 1. Checkout payment (web or mobile)

Use this for one-off purchases. The customer pays on Bonum's page with QPay, card, WeChat,
SonoShop… whatever the terminal has enabled (`GetPaymentProviders`).

```mermaid
sequenceDiagram
    participant C as Web / Mobile
    participant B as Backend
    participant G as Bonum
    C->>B: POST /orders/123/pay
    B->>G: CreateInvoice{amount, transactionId:"order-123", callback, expiresIn, providers?}
    G-->>B: {invoiceId, followUpLink}
    B-->>C: followUpLink
    C->>G: open followUpLink (redirect / in-app browser)
    Note over C,G: customer picks a method and pays
    G-->>C: redirect to callback URL (UX only)
    G->>B: POST webhook {type:PAYMENT, status:SUCCESS|FAILED, body{transactionId,…}}
    B->>B: VerifyWebhook → mark order-123 paid / failed
    C->>B: GET /orders/123 (poll or push) → show result
```

**Web:** `window.location = followUpLink` (or a new tab). Make `callback` a page in your app
that shows "checking your payment…" and reads the order status from your backend.

**Mobile:** open `followUpLink` in an in-app browser (SFSafariViewController / Chrome Custom Tabs).
Set `callback` to a URL your app intercepts (universal link / app link) so it returns to the app
and then fetches the order status. For a fully native feel use the QR flow (§4).

Webhook payloads:

- `SUCCESS` body: `amount, currency, completedAt, terminalId, invoiceId, paymentVendor,
  initType, status:"PAID", respCode, transactionId, extras`
- `FAILED` body: `transactionId, amount, currency, updatedAt, terminalId, invoiceStatus:"EXPIRED"…`

Do **not** poll `GetInvoiceStatusSandbox` in production — Bonum blocks it. Keep an invoice table
on your side and let the webhook update it.

---

## 2. Save a card (tokenization)

Needed for one-click repeat purchases and subscriptions. The card details are entered on
Bonum's page; you only ever receive an opaque token.

```mermaid
sequenceDiagram
    participant C as Web / Mobile
    participant B as Backend
    participant G as Bonum
    C->>B: POST /me/cards
    B->>G: CreateCardToken{callback, transactionId:"cardreq-77", payment?{amount}, subscription?}
    G-->>B: {id, followUpLink}
    B-->>C: followUpLink
    C->>G: open followUpLink → customer enters card, 3-D Secure
    G-->>C: redirect to callback
    G->>B: POST webhook {type:CARD-TOKEN, body{token, mask, expiry, bank, transactionId, amounts, subscriptions}}
    B->>B: VerifyWebhook → store token against the customer (show mask "5150 23** **** 4778")
```

- `payment.amount` charges the card during tokenization (default 0.01 MNT verification charge).
- `subscription{planId, cycleValue, cycles?, payNow, custEmail?}` enrols the new card into a plan
  in the same step; the webhook's `subscriptions[]` then carries the `subscriptionId`.

---

## 3. Charge a saved card

### One-off (`Purchase`)

```mermaid
sequenceDiagram
    participant C as Web / Mobile
    participant B as Backend
    participant G as Bonum
    C->>B: POST /orders/124/pay-with-card {cardId}
    B->>G: Purchase(cardToken, {amount, currency:"MNT", transactionId:"order-124"})
    alt processed now
        G-->>B: 200 {data.status:SUCCESS}  or  400 *Error (declined)
        B-->>C: result
    else high traffic
        G-->>B: 201 {data.status:QUEUED}
        B-->>C: "pending"
        G->>B: POST webhook {type:TOKEN-PAYMENT, status, body{transactionId, completedAt}}
        B->>B: VerifyWebhook → finalise order-124
    end
```

`RollbackPurchase(cardToken, transactionId)` reverses a purchase. The card token goes in the
`X-CARD-TOKEN` header; the SDK does that for you.

### Recurring (`Subscribe`)

```mermaid
sequenceDiagram
    participant B as Backend
    participant G as Bonum
    B->>G: ListPaymentPlans()
    G-->>B: [{planId, recurringType:WEEKLY|MONTHLY|YEARLY, amount, …}]
    B->>G: Subscribe(cardToken, {planId, cycleValue, cycles?, payNow, custEmail?})
    G-->>B: {subscriptionId, nextBillAt, status:ACTIVE}
    loop every billing date
        G->>G: charge the card automatically
        G->>B: POST webhook {type:SUBSCRIPTION-PAYMENT, status, body{subscriptionId, planId, transactionId, amount, completedAt}}
        B->>B: VerifyWebhook → extend the customer's service period
    end
```

- `cycleValue`: 1–7 (Mon–Sun) weekly, 1–31 monthly, 1–366 yearly. Ignored when `payNow`.
- If the subscription day equals today's `cycleValue`, the first charge runs immediately.
- Change card: `ChangeSubscriptionTokenExisting(id, otherToken)` or
  `ChangeSubscriptionTokenNew(id, …)` (returns a `followUpLink`, same as §2).
- Stop: `Unsubscribe` (already scheduled charge still runs) vs `DeleteSubscription` (stops now).
- Sandbox: `ExecuteSubscriptionPaymentSandbox(id)` fires a billing run so you can test the webhook.

Plans are created on the merchant portal, not through the API.

---

## 4. QR / deeplink payment (mobile-native, kiosk, POS)

Use when you want to show a QR the customer scans with any bank app, or offer
"Pay with Khan Bank / SocialPay…" buttons that deep-link straight into that app.

```mermaid
sequenceDiagram
    participant C as Mobile / Web
    participant B as Backend
    participant G as Bonum
    participant A as Bank app
    C->>B: POST /orders/125/qr
    B->>G: CreateQrCode{amount, transactionId:"order-125", expiresIn}
    G-->>B: {invoiceId, qrCode, qrImage(base64 png), links[{name, logo, link, appStoreId, androidPackageName}]}
    B-->>C: qrImage + links
    alt same device (mobile)
        C->>A: open links[i].link (deeplink)
    else another device (web / kiosk)
        A->>A: customer scans qrImage
    end
    A->>G: pays
    G->>B: POST webhook {type:PAYMENT, …}
    B->>B: VerifyWebhook → mark order-125 paid
    C->>B: poll / push → show success
```

- On mobile, render `links[]` as buttons; `androidPackageName` / `appStoreId` let you hide apps
  that are not installed or fall back to the store.
- `InvoiceByQrCode(qrCode)` resolves a scanned QPay string back to its invoice.
- `PayByCardToken(cardToken, {qrCode, transactionId})` settles a QR invoice with a saved card —
  useful when *your* app is the one scanning a merchant's QR.

---

## 5. Webhook endpoint checklist

```mermaid
flowchart LR
    R[POST from Bonum] --> V{VerifyWebhook<br/>x-checksum-v2}
    V -- no --> U[401, stop]
    V -- yes --> P[PeekWebhook.type]
    P --> PAY[PAYMENT] & TOK[CARD-TOKEN] & TP[TOKEN-PAYMENT] & SUB[SUBSCRIPTION-PAYMENT]
    PAY & TOK & TP & SUB --> F[find by transactionId,<br/>apply idempotently] --> OK[200]
```

- Register the URL with Bonum (merchant portal / support) — it is not set per request.
- Reply `200` quickly; do heavy work asynchronously.
- Ignore `message` — Bonum says its text may change at any time.
- Store `traceId`s from API errors; Bonum support asks for them.

---

## Endpoint ↔ SDK map

| Step | Bonum endpoint | SDK |
|---|---|---|
| token | `GET /bonum-gateway/ecommerce/auth/create`, `/auth/refresh` | automatic |
| providers | `GET …/invoices/payment-providers` | `GetPaymentProviders` |
| checkout | `POST …/invoices` | `CreateInvoice` |
| save card | `POST /mpay-service/merchant/cards/tokenize/request` | `CreateCardToken` |
| charge token | `POST …/transaction/purchase` | `Purchase` |
| reverse | `DELETE …/transaction/reverse/{transactionId}` | `RollbackPurchase` |
| plans | `GET …/values/payment-plans` | `ListPaymentPlans` |
| subscribe | `POST …/subscriptions/subscribe` | `Subscribe` |
| list subs | `GET …/subscriptions` | `GetSubscriptions` |
| change card | `PUT …/subscriptions/{id}/change[/create-new-token]` | `ChangeSubscriptionToken*` |
| cancel | `DELETE …/subscriptions/{id}[/delete]` | `Unsubscribe`, `DeleteSubscription` |
| QR | `POST …/transaction/qr/create`, `/qr`, `PUT …/qr/pay` | `CreateQrCode`, `InvoiceByQrCode`, `PayByCardToken` |
| webhook | merchant URL, header `x-checksum-v2` | `VerifyWebhook`, `Parse*Webhook` |
