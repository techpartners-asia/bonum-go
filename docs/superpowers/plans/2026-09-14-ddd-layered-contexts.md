# DDD Layered Contexts Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restructure the two bounded contexts (Gateway, Wallet) into `domain / ports / application / adapters` layers while keeping the public `bonum` and `wallet` APIs byte-for-byte compatible.

**Architecture:** New layer packages are added first, next to the existing flat code, so every task builds green on its own. The facade switch (Tasks 6 and 9) deletes the old flat files once the layers exist. An architecture test enforces the import direction.

**Tech Stack:** Go 1.24, resty v3 (adapters only), stdlib `go/parser` for the architecture test.

**Spec:** `docs/superpowers/specs/2026-09-14-ddd-layered-contexts-design.md`

## Global Constraints

- Module path `github.com/techpartners-asia/bonum-go`; go 1.24.
- Every existing test under `tests/gateway`, `tests/wallet` must pass unmodified.
- Error string prefixes stay `bonum:` (gateway) and `bonum wallet:` (wallet).
- `internal/rest` is not modified.
- Domain, ports, application packages import only stdlib and their own context's domain/ports.
- All tests live under `tests/`, mirroring the source path; none beside source.
- Verification after every task: `go build ./... && go vet ./... && go test ./...`.
- Commit messages end with the attribution lines from the session reminder.

---

### Task 1: Gateway domain root, access and checkout

**Files:**
- Create: `gateway/domain/errors.go`, `gateway/domain/access/access.go`, `gateway/domain/checkout/checkout.go`
- Test: `tests/gateway/domain/checkout_test.go`, `tests/gateway/domain/errors_test.go`

**Interfaces:**
- Produces: `domain.ErrInvalidInput/ErrUnauthorized/ErrNotFound/ErrRateLimited`, `domain.APIError{StatusCode,TraceID,Message,Body}`, `domain.ValidationError{Field,Reason}`, `domain.Invalid(field, reason string) error`, `access.TokenPair`, `checkout.CreateInvoiceInput.Validate() error`, `checkout.{PaymentProvider,ExtraType,PaymentProviderStatus,Item,Extra,Invoice}`.

- [ ] **Step 1: Write the failing tests**

`tests/gateway/domain/errors_test.go`:
```go
package domain_test

import (
	"errors"
	"testing"

	"github.com/techpartners-asia/bonum-go/gateway/domain"
)

func TestAPIErrorMapsStatusToSentinels(t *testing.T) {
	cases := map[int]error{400: domain.ErrInvalidInput, 401: domain.ErrUnauthorized, 403: domain.ErrUnauthorized, 404: domain.ErrNotFound, 429: domain.ErrRateLimited}
	for status, want := range cases {
		err := &domain.APIError{StatusCode: status, TraceID: "t", Message: "m"}
		if !errors.Is(err, want) {
			t.Fatalf("%d: want %v", status, want)
		}
	}
	if errors.Is(&domain.APIError{StatusCode: 500}, domain.ErrInvalidInput) {
		t.Fatal("500 must not match ErrInvalidInput")
	}
	if got := (&domain.APIError{StatusCode: 404, TraceID: "t-1", Message: "gone"}).Error(); got != "bonum: 404 gone (trace t-1)" {
		t.Fatalf("Error() = %q", got)
	}
}

func TestInvalidIsValidationError(t *testing.T) {
	err := domain.Invalid("Amount", "must be positive")
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatal("want ErrInvalidInput")
	}
	var v *domain.ValidationError
	if !errors.As(err, &v) || v.Field != "Amount" || err.Error() != "bonum: invalid Amount: must be positive" {
		t.Fatalf("unexpected %#v", err)
	}
}
```

`tests/gateway/domain/checkout_test.go`:
```go
package domain_test

import (
	"errors"
	"testing"

	"github.com/techpartners-asia/bonum-go/gateway/domain"
	"github.com/techpartners-asia/bonum-go/gateway/domain/checkout"
)

func TestCreateInvoiceInputValidate(t *testing.T) {
	valid := checkout.CreateInvoiceInput{Amount: 100, Callback: "https://m/cb", TransactionID: "T1", ExpiresIn: 600}
	if err := valid.Validate(); err != nil {
		t.Fatal(err)
	}
	cases := map[string]checkout.CreateInvoiceInput{
		"Amount":        {Callback: "https://m/cb", TransactionID: "T1", ExpiresIn: 600},
		"Callback":      {Amount: 100, TransactionID: "T1", ExpiresIn: 600},
		"TransactionID": {Amount: 100, Callback: "https://m/cb", ExpiresIn: 600},
		"ExpiresIn":     {Amount: 100, Callback: "https://m/cb", TransactionID: "T1"},
	}
	for field, in := range cases {
		err := in.Validate()
		var v *domain.ValidationError
		if !errors.As(err, &v) || v.Field != field || !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("%s: got %v", field, err)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./tests/gateway/domain/`
Expected: FAIL, package `gateway/domain` not found.

- [ ] **Step 3: Write the domain packages**

`gateway/domain/errors.go`:
```go
// Package domain holds the errors every Gateway aggregate shares. The aggregates themselves
// live in the sub-packages access, checkout, card, subscription, qr and webhook.
package domain

import (
	"errors"
	"fmt"
)

// Sentinel errors. Match them with errors.Is; the concrete *APIError / *ValidationError is
// still available through errors.As.
var (
	ErrInvalidInput = errors.New("bonum: invalid input") // local validation failed, or Bonum answered 400
	ErrUnauthorized = errors.New("bonum: unauthorized")  // 401 / 403: bad AppSecret, terminal or token
	ErrNotFound     = errors.New("bonum: not found")     // 404
	ErrRateLimited  = errors.New("bonum: rate limited")  // 429
)

// APIError is returned when Bonum answers with a non-2xx status.
type APIError struct {
	StatusCode int
	TraceID    string // Quote it to Bonum support
	Message    string
	Body       string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("bonum: %d %s (trace %s)", e.StatusCode, e.Message, e.TraceID)
	}
	return fmt.Sprintf("bonum: %d %s", e.StatusCode, e.Body)
}

// Is maps HTTP status classes onto the sentinel errors.
func (e *APIError) Is(target error) bool {
	switch target {
	case ErrInvalidInput:
		return e.StatusCode == 400
	case ErrUnauthorized:
		return e.StatusCode == 401 || e.StatusCode == 403
	case ErrNotFound:
		return e.StatusCode == 404
	case ErrRateLimited:
		return e.StatusCode == 429
	}
	return false
}

// ValidationError is returned before any network call when an input violates an invariant.
type ValidationError struct {
	Field  string
	Reason string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("bonum: invalid %s: %s", e.Field, e.Reason)
}
func (e *ValidationError) Is(target error) bool { return target == ErrInvalidInput }

// Invalid builds the ValidationError every aggregate's Validate method returns.
func Invalid(field, reason string) error { return &ValidationError{Field: field, Reason: reason} }
```

`gateway/domain/access/access.go`:
```go
// Package access is the Access aggregate: the Terminal's bearer credentials.
package access

// TokenPair is returned by auth/create and auth/refresh.
type TokenPair struct {
	TokenType        string `json:"tokenType"` // "Bearer"
	AccessToken      string `json:"accessToken"`
	ExpiresIn        int64  `json:"expiresIn"` // access token ttl, in Unit
	RefreshToken     string `json:"refreshToken"`
	RefreshExpiresIn int64  `json:"refreshExpiresIn"` // refresh token ttl, in Unit
	Unit             string `json:"unit"`             // "SECONDS"
}
```

`gateway/domain/checkout/checkout.go`:
```go
// Package checkout is the Invoice aggregate plus the value objects rendered on Bonum's
// hosted page (Item, Extra, PaymentProvider), which card and subscription flows reuse.
package checkout

import "github.com/techpartners-asia/bonum-go/gateway/domain"

// PaymentProvider is a payment option that can be shown on the Bonum checkout page.
type PaymentProvider string

const (
	ProviderQPay      PaymentProvider = "QPAY"       // QR based, supported by every bank and fintech app
	ProviderECommerce PaymentProvider = "E_COMMERCE" // Online card payment
	ProviderWeChat    PaymentProvider = "WE_CHAT"    // WeChat wallet
	ProviderSonoShop  PaymentProvider = "SONO_SHOP"  // Buy now, pay later
)

// ExtraType constrains what the customer may type into an Extra on the checkout page.
type ExtraType string

const (
	ExtraText   ExtraType = "TEXT"
	ExtraNumber ExtraType = "NUMBER"
	ExtraPhone  ExtraType = "PHONE"
	ExtraEmail  ExtraType = "EMAIL"
	ExtraAll    ExtraType = "ALL"
)

type (
	PaymentProviderStatus struct {
		Provider PaymentProvider `json:"provider"`
		Enabled  bool            `json:"enabled"`
	}

	// Item is an optional line item rendered on the Bonum checkout / tokenization page.
	Item struct {
		Image  string  `json:"image,omitempty"`
		Title  string  `json:"title"`
		Remark string  `json:"remark,omitempty"`
		Amount float64 `json:"amount"`
		Count  int     `json:"count"`
	}

	// Extra is an additional input field the customer fills in on the checkout page.
	Extra struct {
		Placeholder string    `json:"placeholder"`
		Type        ExtraType `json:"type"`
		Required    bool      `json:"required"`
	}

	CreateInvoiceInput struct {
		Amount        float64           `json:"amount"`
		Callback      string            `json:"callback"`      // Browser is redirected here once payment is done/cancelled/expired
		TransactionID string            `json:"transactionId"` // Merchant's unique id, echoed back in the webhook
		ExpiresIn     int64             `json:"expiresIn"`     // Invoice ttl in seconds
		Providers     []PaymentProvider `json:"providers,omitempty"`
		Items         []Item            `json:"items,omitempty"`
		Extras        []Extra           `json:"extras,omitempty"`
	}

	// Invoice is a checkout request. The outcome arrives as a PaymentEvent on the webhook.
	Invoice struct {
		ID           string `json:"invoiceId"`
		FollowUpLink string `json:"followUpLink"` // Redirect the customer's browser here to pay
	}
)

// Validate enforces the Invoice invariants before any network call.
func (in CreateInvoiceInput) Validate() error {
	switch {
	case in.Amount <= 0:
		return domain.Invalid("Amount", "must be positive")
	case in.Callback == "":
		return domain.Invalid("Callback", "required")
	case in.TransactionID == "":
		return domain.Invalid("TransactionID", "required")
	case in.ExpiresIn <= 0:
		return domain.Invalid("ExpiresIn", "must be positive seconds")
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go build ./... && go vet ./... && go test ./tests/gateway/domain/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add gateway/domain tests/gateway/domain
git commit -m "Add gateway domain root, access and checkout aggregates"
```

---

### Task 2: Gateway card, subscription and QR aggregates

**Files:**
- Create: `gateway/domain/card/card.go`, `gateway/domain/subscription/subscription.go`, `gateway/domain/qr/qr.go`
- Test: `tests/gateway/domain/card_test.go`, `tests/gateway/domain/subscription_test.go`, `tests/gateway/domain/qr_test.go`

**Interfaces:**
- Consumes: `domain.Invalid`, `domain.APIError`, `checkout.Item`.
- Produces: `card.{PurchaseStatus,TokenizePayment,TokenizeSubscription,TokenizeInput,Tokenization,PurchaseInput,Purchase,ErrDeclined,DeclinedError}`, `subscription.{RecurringType,PaymentPlan,SubscribeInput,Subscription,ChangeCardInput}`, `qr.{CreateQRInput,Deeplink,QRInvoice,PayQRInput}`; each input has `Validate() error`.

- [ ] **Step 1: Write the failing tests**

`tests/gateway/domain/card_test.go`:
```go
package domain_test

import (
	"errors"
	"testing"

	"github.com/techpartners-asia/bonum-go/gateway/domain"
	"github.com/techpartners-asia/bonum-go/gateway/domain/card"
)

func TestTokenizeInputValidate(t *testing.T) {
	if err := (card.TokenizeInput{Callback: "https://m/cb", TransactionID: "T1"}).Validate(); err != nil {
		t.Fatal(err)
	}
	for field, in := range map[string]card.TokenizeInput{
		"Callback":      {TransactionID: "T1"},
		"TransactionID": {Callback: "https://m/cb"},
	} {
		var v *domain.ValidationError
		if err := in.Validate(); !errors.As(err, &v) || v.Field != field {
			t.Fatalf("%s: got %v", field, err)
		}
	}
}

func TestPurchaseInputValidate(t *testing.T) {
	if err := (card.PurchaseInput{Amount: 1, Currency: "MNT", TransactionID: "T1"}).Validate(); err != nil {
		t.Fatal(err)
	}
	for field, in := range map[string]card.PurchaseInput{
		"Amount":        {Currency: "MNT", TransactionID: "T1"},
		"Currency":      {Amount: 1, TransactionID: "T1"},
		"TransactionID": {Amount: 1, Currency: "MNT"},
	} {
		var v *domain.ValidationError
		if err := in.Validate(); !errors.As(err, &v) || v.Field != field {
			t.Fatalf("%s: got %v", field, err)
		}
	}
}

func TestDeclinedErrorMatchesDeclinedAndInvalidInput(t *testing.T) {
	err := &card.DeclinedError{
		APIError: &domain.APIError{StatusCode: 400, TraceID: "t-d", Message: "Insufficient funds"},
		Purchase: card.Purchase{ID: 9, Status: card.PurchaseFailed},
	}
	if !errors.Is(err, card.ErrDeclined) || !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatal("decline must satisfy both ErrDeclined and ErrInvalidInput")
	}
	var api *domain.APIError
	if !errors.As(err, &api) || api.TraceID != "t-d" {
		t.Fatal("APIError must be reachable through errors.As")
	}
}
```

`tests/gateway/domain/subscription_test.go`:
```go
package domain_test

import (
	"errors"
	"testing"

	"github.com/techpartners-asia/bonum-go/gateway/domain"
	"github.com/techpartners-asia/bonum-go/gateway/domain/subscription"
)

func TestSubscribeInputValidate(t *testing.T) {
	ok := []subscription.SubscribeInput{
		{PlanID: 30, CycleValue: 1},
		{PlanID: 30, CycleValue: 366},
		{PlanID: 30, PayNow: true}, // CycleValue ignored when PayNow
	}
	for i, in := range ok {
		if err := in.Validate(); err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
	}
	for field, in := range map[string]subscription.SubscribeInput{
		"PlanID":     {CycleValue: 5},
		"CycleValue": {PlanID: 30, CycleValue: 0},
	} {
		var v *domain.ValidationError
		if err := in.Validate(); !errors.As(err, &v) || v.Field != field {
			t.Fatalf("%s: got %v", field, err)
		}
	}
	var v *domain.ValidationError
	if err := (subscription.SubscribeInput{PlanID: 30, CycleValue: 367}).Validate(); !errors.As(err, &v) || v.Field != "CycleValue" {
		t.Fatalf("367: got %v", err)
	}
}

func TestChangeCardInputValidate(t *testing.T) {
	if err := (subscription.ChangeCardInput{Callback: "https://m/cb", TransactionID: "T1"}).Validate(); err != nil {
		t.Fatal(err)
	}
	for field, in := range map[string]subscription.ChangeCardInput{
		"Callback":      {TransactionID: "T1"},
		"TransactionID": {Callback: "https://m/cb"},
	} {
		var v *domain.ValidationError
		if err := in.Validate(); !errors.As(err, &v) || v.Field != field {
			t.Fatalf("%s: got %v", field, err)
		}
	}
}
```

`tests/gateway/domain/qr_test.go`:
```go
package domain_test

import (
	"errors"
	"testing"

	"github.com/techpartners-asia/bonum-go/gateway/domain"
	"github.com/techpartners-asia/bonum-go/gateway/domain/qr"
)

func TestCreateQRInputValidate(t *testing.T) {
	if err := (qr.CreateQRInput{Amount: 1, TransactionID: "T1", ExpiresIn: 60}).Validate(); err != nil {
		t.Fatal(err)
	}
	for field, in := range map[string]qr.CreateQRInput{
		"Amount":        {TransactionID: "T1", ExpiresIn: 60},
		"TransactionID": {Amount: 1, ExpiresIn: 60},
		"ExpiresIn":     {Amount: 1, TransactionID: "T1"},
	} {
		var v *domain.ValidationError
		if err := in.Validate(); !errors.As(err, &v) || v.Field != field {
			t.Fatalf("%s: got %v", field, err)
		}
	}
}

func TestPayQRInputValidate(t *testing.T) {
	if err := (qr.PayQRInput{QrCode: "0002", TransactionID: "T1"}).Validate(); err != nil {
		t.Fatal(err)
	}
	for field, in := range map[string]qr.PayQRInput{
		"QrCode":        {TransactionID: "T1"},
		"TransactionID": {QrCode: "0002"},
	} {
		var v *domain.ValidationError
		if err := in.Validate(); !errors.As(err, &v) || v.Field != field {
			t.Fatalf("%s: got %v", field, err)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./tests/gateway/domain/`
Expected: FAIL, packages `card`, `subscription`, `qr` not found.

- [ ] **Step 3: Write the aggregates**

`gateway/domain/card/card.go`:
```go
// Package card is the Card Token aggregate: tokenization and purchases against stored cards.
package card

import (
	"errors"

	"github.com/techpartners-asia/bonum-go/gateway/domain"
	"github.com/techpartners-asia/bonum-go/gateway/domain/checkout"
)

// ErrDeclined marks a Purchase the bank refused. Match with errors.Is; the *DeclinedError
// carrying Bonum's Purchase record is available through errors.As.
var ErrDeclined = errors.New("bonum: purchase declined")

// PurchaseStatus is the outcome of a Purchase.
type PurchaseStatus string

const (
	PurchaseSuccess PurchaseStatus = "SUCCESS"
	PurchaseFailed  PurchaseStatus = "FAILED"
	PurchaseQueued  PurchaseStatus = "QUEUED" // Result arrives later as a TokenPaymentEvent
)

type (
	// TokenizePayment charges the card while it is being tokenized. Defaults to 0.01 MNT if omitted.
	TokenizePayment struct {
		Amount float64 `json:"amount"`
	}

	// TokenizeSubscription enrols the new Card Token in a Payment Plan in the same flow.
	TokenizeSubscription struct {
		PlanID     int64  `json:"planId"`
		CycleValue string `json:"cycleValue"` // 1-7 weekly, 1-31 monthly, 1-366 yearly
		Cycles     *int64 `json:"cycles,omitempty"`
		PayNow     bool   `json:"payNow"`
		CustEmail  string `json:"custEmail,omitempty"`
	}

	TokenizeInput struct {
		Callback      string                `json:"callback"`
		TransactionID string                `json:"transactionId"`
		Payment       *TokenizePayment      `json:"payment,omitempty"`
		Subscription  *TokenizeSubscription `json:"subscription,omitempty"`
		Items         []checkout.Item       `json:"items,omitempty"`
	}

	// Tokenization is a hosted card-entry session. The Card Token arrives as a CardTokenEvent.
	Tokenization struct {
		ID           string `json:"id"`
		FollowUpLink string `json:"followUpLink"` // Redirect the customer's browser here to enter card details
	}

	PurchaseInput struct {
		Amount        float64 `json:"amount"`
		Currency      string  `json:"currency"` // e.g. "MNT"
		TransactionID string  `json:"transactionId"`
	}

	// Purchase is a charge against a Card Token.
	Purchase struct {
		ID          int64          `json:"id"`
		CompletedAt string         `json:"completedAt"`
		Status      PurchaseStatus `json:"status"`
		Description string         `json:"description"`
		CardStatus  *string        `json:"cardStatus"` // ACTIVE | INACTIVE, nil when queued
		RespCode    *string        `json:"respCode"`
	}
)

// Validate enforces the Tokenization invariants before any network call.
func (in TokenizeInput) Validate() error {
	switch {
	case in.Callback == "":
		return domain.Invalid("Callback", "required")
	case in.TransactionID == "":
		return domain.Invalid("TransactionID", "required")
	}
	return nil
}

// Validate enforces the Purchase invariants before any network call.
func (in PurchaseInput) Validate() error {
	switch {
	case in.Amount <= 0:
		return domain.Invalid("Amount", "must be positive")
	case in.Currency == "":
		return domain.Invalid("Currency", "required")
	case in.TransactionID == "":
		return domain.Invalid("TransactionID", "required")
	}
	return nil
}

// DeclinedError is an APIError whose body carried a FAILED Purchase: the bank refused the
// card. Purchase holds Bonum's record of the attempt (id, respCode, cardStatus).
type DeclinedError struct {
	*domain.APIError
	Purchase Purchase
}

func (e *DeclinedError) Is(target error) bool { return target == ErrDeclined || e.APIError.Is(target) }
func (e *DeclinedError) Unwrap() error        { return e.APIError }
```

`gateway/domain/subscription/subscription.go`:
```go
// Package subscription is the Subscription aggregate: a Card Token enrolled in a Payment Plan.
package subscription

import (
	"github.com/techpartners-asia/bonum-go/gateway/domain"
	"github.com/techpartners-asia/bonum-go/gateway/domain/checkout"
)

type RecurringType string

const (
	RecurringWeekly  RecurringType = "WEEKLY"
	RecurringMonthly RecurringType = "MONTHLY"
	RecurringYearly  RecurringType = "YEARLY"
)

type (
	// PaymentPlan is a recurring billing template managed on the merchant portal.
	PaymentPlan struct {
		PlanID        int64         `json:"planId"`
		Name          string        `json:"name"`
		Remark        string        `json:"remark"`
		CreatedAt     string        `json:"createdAt"`
		RecurringType RecurringType `json:"recurringType"`
		Amount        float64       `json:"amount"`
		Status        string        `json:"status"`
		CardCount     int64         `json:"cardCount"`
		RetryCount    int64         `json:"retryCount"`
	}

	SubscribeInput struct {
		PlanID     int64  `json:"planId"`
		CycleValue int64  `json:"cycleValue"` // 1-7 weekly, 1-31 monthly, 1-366 yearly; ignored when PayNow
		Cycles     *int64 `json:"cycles,omitempty"`
		PayNow     bool   `json:"payNow"`
		CustEmail  string `json:"custEmail,omitempty"`
	}

	// Subscription is a Card Token enrolled in a Payment Plan.
	Subscription struct {
		SubscriptionID int64       `json:"subscriptionId"`
		SubscribedAt   string      `json:"subscribedAt"`
		CardMask       string      `json:"cardMask"`
		Plan           PaymentPlan `json:"plan"`
		NextBillAt     string      `json:"nextBillAt"`
		LastBilledAt   string      `json:"lastBilledAt"`
		Status         string      `json:"status"`
	}

	ChangeCardInput struct {
		Callback      string          `json:"callback"`
		TransactionID string          `json:"transactionId"`
		Items         []checkout.Item `json:"items,omitempty"`
	}
)

// Validate enforces the Subscription invariants before any network call.
func (in SubscribeInput) Validate() error {
	switch {
	case in.PlanID <= 0:
		return domain.Invalid("PlanID", "required")
	case !in.PayNow && (in.CycleValue < 1 || in.CycleValue > 366):
		return domain.Invalid("CycleValue", "must be 1-7 weekly, 1-31 monthly or 1-366 yearly")
	}
	return nil
}

// Validate enforces the card-change invariants before any network call.
func (in ChangeCardInput) Validate() error {
	switch {
	case in.Callback == "":
		return domain.Invalid("Callback", "required")
	case in.TransactionID == "":
		return domain.Invalid("TransactionID", "required")
	}
	return nil
}
```

`gateway/domain/qr/qr.go`:
```go
// Package qr is the QR Invoice aggregate: QPay-compatible QR and deeplink payments.
package qr

import "github.com/techpartners-asia/bonum-go/gateway/domain"

type (
	CreateQRInput struct {
		Amount        float64 `json:"amount"`
		TransactionID string  `json:"transactionId"`
		ExpiresIn     int64   `json:"expiresIn"` // seconds
	}

	// Deeplink opens the QR Invoice directly in one bank's app.
	Deeplink struct {
		Name               string `json:"name"`
		Description        string `json:"description"`
		Logo               string `json:"logo"`
		Link               string `json:"link"`
		AppStoreID         string `json:"appStoreId"`
		AndroidPackageName string `json:"androidPackageName"`
	}

	// QRInvoice is a QPay-compatible invoice payable by scanning or via a Deeplink.
	QRInvoice struct {
		InvoiceID string     `json:"invoiceId"`
		QrCode    string     `json:"qrCode"`
		QrImage   string     `json:"qrImage"` // base64 png
		Links     []Deeplink `json:"links"`
	}

	PayQRInput struct {
		QrCode        string `json:"qrCode"`
		TransactionID string `json:"transactionId"`
	}
)

// Validate enforces the QR Invoice invariants before any network call.
func (in CreateQRInput) Validate() error {
	switch {
	case in.Amount <= 0:
		return domain.Invalid("Amount", "must be positive")
	case in.TransactionID == "":
		return domain.Invalid("TransactionID", "required")
	case in.ExpiresIn <= 0:
		return domain.Invalid("ExpiresIn", "must be positive seconds")
	}
	return nil
}

// Validate enforces the QR payment invariants before any network call.
func (in PayQRInput) Validate() error {
	switch {
	case in.QrCode == "":
		return domain.Invalid("QrCode", "required")
	case in.TransactionID == "":
		return domain.Invalid("TransactionID", "required")
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go build ./... && go vet ./... && go test ./tests/gateway/domain/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add gateway/domain tests/gateway/domain
git commit -m "Add gateway card, subscription and qr aggregates"
```

---

### Task 3: Gateway webhook domain

**Files:**
- Create: `gateway/domain/webhook/webhook.go`
- Test: `tests/gateway/domain/webhook_test.go`

**Interfaces:**
- Consumes: `checkout.PaymentProvider`.
- Produces: `webhook.ChecksumHeader`, `webhook.{EventType,Outcome,Event,EventHeader,PaymentBody,PaymentEvent,Bank,Money,SubscriptionRef,CardTokenBody,CardTokenEvent,TokenPaymentBody,TokenPaymentEvent,SubscriptionPaymentBody,SubscriptionPaymentEvent}`, `webhook.ErrBadChecksum`, `webhook.ErrUnknownEvent`, `webhook.Checksum(body []byte, key string) string`, `webhook.Parse(body []byte, checksumHeader, key string) (Event, error)`.

- [ ] **Step 1: Write the failing test**

`tests/gateway/domain/webhook_test.go`:
```go
package domain_test

import (
	"errors"
	"testing"

	"github.com/techpartners-asia/bonum-go/gateway/domain/checkout"
	"github.com/techpartners-asia/bonum-go/gateway/domain/webhook"
)

const key = "checksum-key"

func TestParseDispatchesOnType(t *testing.T) {
	body := []byte(`{"type":"PAYMENT","status":"SUCCESS","message":"ok","body":{"transactionId":"T1","amount":100,"currency":"MNT","terminalId":"17","invoiceId":"inv-1","paymentVendor":"QPAY","status":"PAID"}}`)
	ev, err := webhook.Parse(body, webhook.Checksum(body, key), key)
	if err != nil {
		t.Fatal(err)
	}
	p, ok := ev.(*webhook.PaymentEvent)
	if !ok || p.Outcome != webhook.OutcomeSuccess || p.Body.TransactionID != "T1" || *p.Body.PaymentVendor != checkout.ProviderQPay {
		t.Fatalf("unexpected %#v", ev)
	}
	if ev.Header().Type != webhook.EventPayment {
		t.Fatal("Header() must expose the envelope")
	}
}

func TestParseRejects(t *testing.T) {
	body := []byte(`{"type":"PAYMENT","status":"SUCCESS","body":{}}`)
	if _, err := webhook.Parse(body, "deadbeef", key); !errors.Is(err, webhook.ErrBadChecksum) {
		t.Fatalf("bad checksum: %v", err)
	}
	unknown := []byte(`{"type":"REFUND","status":"SUCCESS"}`)
	if _, err := webhook.Parse(unknown, webhook.Checksum(unknown, key), key); !errors.Is(err, webhook.ErrUnknownEvent) {
		t.Fatalf("unknown type: %v", err)
	}
	garbage := []byte(`nope`)
	if _, err := webhook.Parse(garbage, webhook.Checksum(garbage, key), key); err == nil {
		t.Fatal("garbage body must fail")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./tests/gateway/domain/`
Expected: FAIL, package `webhook` not found.

- [ ] **Step 3: Write the webhook package**

`gateway/domain/webhook/webhook.go`: copy the whole of the current root `webhook.go` with these changes: `package webhook`; add `"errors"` and the checkout import; `PaymentVendor *checkout.PaymentProvider`; rename `ParseWebhook` to `Parse`; define the two sentinels locally:

```go
// Package webhook is the Event aggregate: verified gateway webhook deliveries.
package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/techpartners-asia/bonum-go/gateway/domain/checkout"
)

var (
	ErrBadChecksum  = errors.New("bonum: webhook checksum mismatch")
	ErrUnknownEvent = errors.New("bonum: unknown webhook event type")
)

// ChecksumHeader carries the HMAC Bonum attaches to every gateway webhook delivery.
const ChecksumHeader = "x-checksum-v2"

// ... EventType, Outcome, Event, EventHeader and every Body/Event struct exactly as in the
// current root webhook.go, with PaymentBody.PaymentVendor typed *checkout.PaymentProvider ...

// Checksum computes hex(HMAC-SHA256(key, body)) exactly as Bonum does for gateway webhooks.
func Checksum(body []byte, checksumKey string) string {
	mac := hmac.New(sha256.New, []byte(checksumKey))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// Parse verifies a delivery against the x-checksum-v2 header and decodes it into the Event
// for its type. Pass the body bytes exactly as received; re-serialising changes the hash.
func Parse(body []byte, checksumHeader, checksumKey string) (Event, error) {
	if !hmac.Equal([]byte(Checksum(body, checksumKey)), []byte(checksumHeader)) {
		return nil, ErrBadChecksum
	}
	var h EventHeader
	if err := json.Unmarshal(body, &h); err != nil {
		return nil, fmt.Errorf("bonum: webhook body: %w", err)
	}
	switch h.Type {
	case EventPayment:
		return decodeEvent[PaymentEvent](body)
	case EventCardToken:
		return decodeEvent[CardTokenEvent](body)
	case EventTokenPayment:
		return decodeEvent[TokenPaymentEvent](body)
	case EventSubscriptionPayment:
		return decodeEvent[SubscriptionPaymentEvent](body)
	}
	return nil, fmt.Errorf("%w: %q", ErrUnknownEvent, h.Type)
}

func decodeEvent[T any, PT interface {
	*T
	Event
}](body []byte) (Event, error) {
	var ev T
	if err := json.Unmarshal(body, &ev); err != nil {
		return nil, fmt.Errorf("bonum: webhook body: %w", err)
	}
	return PT(&ev), nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go build ./... && go vet ./... && go test ./tests/gateway/domain/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add gateway/domain/webhook tests/gateway/domain/webhook_test.go
git commit -m "Add gateway webhook aggregate"
```

---

### Task 4: Gateway ports and application services

**Files:**
- Create: `gateway/ports/ports.go`
- Create (commands, one write use case per file): `gateway/application/commands/access/authenticate.go`, `gateway/application/commands/access/refresh.go`, `gateway/application/commands/checkout/create_invoice.go`, `gateway/application/commands/card/tokenize.go`, `gateway/application/commands/card/purchase.go`, `gateway/application/commands/card/reverse.go`, `gateway/application/commands/subscription/subscribe.go`, `gateway/application/commands/subscription/change_card_by_tokenizing.go`, `gateway/application/commands/subscription/change_card.go`, `gateway/application/commands/subscription/unsubscribe.go`, `gateway/application/commands/subscription/delete.go`, `gateway/application/commands/qr/create_qr.go`, `gateway/application/commands/qr/pay_qr.go`, `gateway/application/commands/sandbox/mark_invoice_paid.go`, `gateway/application/commands/sandbox/run_subscription_billing.go`
- Create (queries, one read use case per file): `gateway/application/queries/checkout/providers.go`, `gateway/application/queries/subscription/plans.go`, `gateway/application/queries/subscription/list_subscriptions.go`, `gateway/application/queries/qr/lookup_qr.go`, `gateway/application/queries/sandbox/invoice_status.go`
- Create (facades, one per aggregate, unchanged public shape): `gateway/application/access.go`, `gateway/application/invoices.go`, `gateway/application/cards.go`, `gateway/application/subscriptions.go`, `gateway/application/qr.go`, `gateway/application/sandbox.go`
- Test: `tests/gateway/application/fakes_test.go`, `tests/gateway/application/services_test.go`

**CQRS-lite:** every write use case is a Command type + a `*Handler` with `Handle(ctx, cmd) (result, error)` under `gateway/application/commands/<aggregate>/`; every read use case is a Query (or a niladic `Handle(ctx)`) + `*Handler` under `gateway/application/queries/<aggregate>/`. A Command that already matches an existing domain input is a type alias (`type TokenizeCommand = card.TokenizeInput`) — no parallel struct, per ADR 0002. A facade struct per aggregate (`Cards`, `Invoices`, ...) holds the aggregate's handlers and keeps the exact public method names and signatures the rest of the plan (Task 6) already wires up, so nothing outside `gateway/application/` needs to change. Where a commands/<aggregate> or queries/<aggregate> package would collide on import with the domain package of the same name, alias the application-layer import as `<aggregate>cmd` / `<aggregate>qry` and leave the domain import unaliased — do this consistently in every facade file that imports both.

**Interfaces:**
- Consumes: all Task 1–3 domain types.
- Produces: the six port interfaces exactly as in the spec (unchanged from a non-CQRS build); `application.NewAccess(ports.AccessAPI) *Access` with `Authenticate/Refresh`; `NewInvoices(ports.CheckoutAPI) *Invoices` with `Providers/Create`; `NewCards(ports.CardAPI) *Cards` with `Tokenize/Purchase/Reverse`; `NewSubscriptions(ports.SubscriptionAPI) *Subscriptions` with `Plans/Subscribe/List/ChangeCardByTokenizing/ChangeCard/Unsubscribe/Delete`; `NewQR(ports.QRAPI) *QR` with `Create/Lookup/PayWithCard`; `NewSandbox(ports.SandboxAPI) *Sandbox` with `InvoiceStatus/MarkInvoicePaid/RunSubscriptionBilling`. Internally, each of those also exposes the constituent `*Handler` types listed below, for anyone who wants a single use case without the aggregate facade.

- [ ] **Step 1: Write the failing tests**

`tests/gateway/application/fakes_test.go`:
```go
package application_test

import (
	"context"
	"encoding/json"

	"github.com/techpartners-asia/bonum-go/gateway/domain/access"
	"github.com/techpartners-asia/bonum-go/gateway/domain/card"
	"github.com/techpartners-asia/bonum-go/gateway/domain/checkout"
	"github.com/techpartners-asia/bonum-go/gateway/domain/qr"
	"github.com/techpartners-asia/bonum-go/gateway/domain/subscription"
)

var ctx = context.Background()

// fakeAPI implements every gateway port. It records the last method called and its
// arguments and returns a configured error, so tests can assert what reached the port.
type fakeAPI struct {
	calls []string
	args  []any
	err   error
}

func (f *fakeAPI) record(name string, args ...any) { f.calls = append(f.calls, name); f.args = args }

func (f *fakeAPI) CreateToken(context.Context) (*access.TokenPair, error) {
	f.record("CreateToken")
	return &access.TokenPair{AccessToken: "a"}, f.err
}
func (f *fakeAPI) RefreshToken(context.Context) (*access.TokenPair, error) {
	f.record("RefreshToken")
	return &access.TokenPair{AccessToken: "b"}, f.err
}
func (f *fakeAPI) Providers(context.Context) ([]checkout.PaymentProviderStatus, error) {
	f.record("Providers")
	return []checkout.PaymentProviderStatus{{Provider: checkout.ProviderQPay, Enabled: true}}, f.err
}
func (f *fakeAPI) CreateInvoice(_ context.Context, in checkout.CreateInvoiceInput) (*checkout.Invoice, error) {
	f.record("CreateInvoice", in)
	return &checkout.Invoice{ID: "inv-1"}, f.err
}
func (f *fakeAPI) Tokenize(_ context.Context, in card.TokenizeInput) (*card.Tokenization, error) {
	f.record("Tokenize", in)
	return &card.Tokenization{ID: "tok-1"}, f.err
}
func (f *fakeAPI) Purchase(_ context.Context, tok string, in card.PurchaseInput) (*card.Purchase, error) {
	f.record("Purchase", tok, in)
	return &card.Purchase{ID: 1, Status: card.PurchaseSuccess}, f.err
}
func (f *fakeAPI) Reverse(_ context.Context, tok, txn string) error {
	f.record("Reverse", tok, txn)
	return f.err
}
func (f *fakeAPI) Plans(context.Context) ([]subscription.PaymentPlan, error) {
	f.record("Plans")
	return []subscription.PaymentPlan{{PlanID: 30}}, f.err
}
func (f *fakeAPI) Subscribe(_ context.Context, tok string, in subscription.SubscribeInput) (*subscription.Subscription, error) {
	f.record("Subscribe", tok, in)
	return &subscription.Subscription{SubscriptionID: 42}, f.err
}
func (f *fakeAPI) ListSubscriptions(_ context.Context, tok string) ([]subscription.Subscription, error) {
	f.record("ListSubscriptions", tok)
	return []subscription.Subscription{{SubscriptionID: 42}}, f.err
}
func (f *fakeAPI) ChangeCardByTokenizing(_ context.Context, id int64, in subscription.ChangeCardInput) (*card.Tokenization, error) {
	f.record("ChangeCardByTokenizing", id, in)
	return &card.Tokenization{ID: "tok-2"}, f.err
}
func (f *fakeAPI) ChangeCard(_ context.Context, id int64, tok string) (*subscription.Subscription, error) {
	f.record("ChangeCard", id, tok)
	return &subscription.Subscription{SubscriptionID: id}, f.err
}
func (f *fakeAPI) Unsubscribe(_ context.Context, id, planID int64) error {
	f.record("Unsubscribe", id, planID)
	return f.err
}
func (f *fakeAPI) DeleteSubscription(_ context.Context, id, planID int64) error {
	f.record("DeleteSubscription", id, planID)
	return f.err
}
func (f *fakeAPI) CreateQR(_ context.Context, in qr.CreateQRInput) (*qr.QRInvoice, error) {
	f.record("CreateQR", in)
	return &qr.QRInvoice{InvoiceID: "qr-1"}, f.err
}
func (f *fakeAPI) LookupQR(_ context.Context, code string) (*qr.QRInvoice, error) {
	f.record("LookupQR", code)
	return &qr.QRInvoice{InvoiceID: "qr-1", QrCode: code}, f.err
}
func (f *fakeAPI) PayQRWithCard(_ context.Context, tok string, in qr.PayQRInput) (*card.Purchase, error) {
	f.record("PayQRWithCard", tok, in)
	return &card.Purchase{ID: 5}, f.err
}
func (f *fakeAPI) InvoiceStatus(_ context.Context, id string) (json.RawMessage, error) {
	f.record("InvoiceStatus", id)
	return json.RawMessage(`{"status":"PAID"}`), f.err
}
func (f *fakeAPI) MarkInvoicePaid(_ context.Context, id string) error {
	f.record("MarkInvoicePaid", id)
	return f.err
}
func (f *fakeAPI) RunSubscriptionBilling(_ context.Context, id int64) error {
	f.record("RunSubscriptionBilling", id)
	return f.err
}
```

`tests/gateway/application/services_test.go`:
```go
package application_test

import (
	"errors"
	"testing"

	"github.com/techpartners-asia/bonum-go/gateway/application"
	"github.com/techpartners-asia/bonum-go/gateway/domain"
	"github.com/techpartners-asia/bonum-go/gateway/domain/card"
	"github.com/techpartners-asia/bonum-go/gateway/domain/checkout"
	"github.com/techpartners-asia/bonum-go/gateway/domain/qr"
	"github.com/techpartners-asia/bonum-go/gateway/domain/subscription"
)

var errPort = errors.New("port failed")

func TestInvoicesValidateBeforePort(t *testing.T) {
	f := &fakeAPI{}
	s := application.NewInvoices(f)
	if _, err := s.Create(ctx, checkout.CreateInvoiceInput{}); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
	if len(f.calls) != 0 {
		t.Fatal("invalid input must not reach the port")
	}
	in := checkout.CreateInvoiceInput{Amount: 1, Callback: "https://m/cb", TransactionID: "T1", ExpiresIn: 60}
	inv, err := s.Create(ctx, in)
	if err != nil || inv.ID != "inv-1" || f.calls[0] != "CreateInvoice" || f.args[0] != in {
		t.Fatalf("forwarding: %v %v %v", inv, err, f.calls)
	}
	if p, err := s.Providers(ctx); err != nil || len(p) != 1 {
		t.Fatalf("providers: %v %v", p, err)
	}
}

func TestPortErrorsPassThroughUnwrapped(t *testing.T) {
	f := &fakeAPI{err: errPort}
	if _, err := application.NewInvoices(f).Providers(ctx); err != errPort {
		t.Fatalf("want the port error itself, got %v", err)
	}
	if err := application.NewCards(f).Reverse(ctx, "tok", "T1"); err != errPort {
		t.Fatalf("want the port error itself, got %v", err)
	}
}

func TestCards(t *testing.T) {
	f := &fakeAPI{}
	s := application.NewCards(f)
	if _, err := s.Tokenize(ctx, card.TokenizeInput{}); !errors.Is(err, domain.ErrInvalidInput) || len(f.calls) != 0 {
		t.Fatalf("tokenize validation: %v %v", err, f.calls)
	}
	if _, err := s.Purchase(ctx, "tok", card.PurchaseInput{}); !errors.Is(err, domain.ErrInvalidInput) || len(f.calls) != 0 {
		t.Fatalf("purchase validation: %v %v", err, f.calls)
	}
	in := card.PurchaseInput{Amount: 1, Currency: "MNT", TransactionID: "T1"}
	if p, err := s.Purchase(ctx, "tok", in); err != nil || p.ID != 1 || f.args[0] != "tok" || f.args[1] != in {
		t.Fatalf("purchase forwarding: %v %v %v", p, err, f.args)
	}
	if err := s.Reverse(ctx, "tok", "T1"); err != nil || f.calls[len(f.calls)-1] != "Reverse" {
		t.Fatalf("reverse: %v %v", err, f.calls)
	}
}

func TestSubscriptions(t *testing.T) {
	f := &fakeAPI{}
	s := application.NewSubscriptions(f)
	if _, err := s.Subscribe(ctx, "tok", subscription.SubscribeInput{}); !errors.Is(err, domain.ErrInvalidInput) || len(f.calls) != 0 {
		t.Fatalf("subscribe validation: %v", err)
	}
	if _, err := s.ChangeCardByTokenizing(ctx, 7, subscription.ChangeCardInput{}); !errors.Is(err, domain.ErrInvalidInput) || len(f.calls) != 0 {
		t.Fatalf("change card validation: %v", err)
	}
	if _, err := s.Subscribe(ctx, "tok", subscription.SubscribeInput{PlanID: 30, PayNow: true}); err != nil || f.calls[0] != "Subscribe" {
		t.Fatal(err)
	}
	if _, err := s.List(ctx, "tok"); err != nil || f.calls[1] != "ListSubscriptions" || f.args[0] != "tok" {
		t.Fatal(err)
	}
	if _, err := s.ChangeCard(ctx, 7, "tok"); err != nil || f.calls[2] != "ChangeCard" || f.args[0] != int64(7) {
		t.Fatal(err)
	}
	if err := s.Unsubscribe(ctx, 42, 30); err != nil || f.calls[3] != "Unsubscribe" || f.args[1] != int64(30) {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, 42, 30); err != nil || f.calls[4] != "DeleteSubscription" {
		t.Fatal(err)
	}
	if p, err := s.Plans(ctx); err != nil || len(p) != 1 {
		t.Fatal(err)
	}
}

func TestQR(t *testing.T) {
	f := &fakeAPI{}
	s := application.NewQR(f)
	if _, err := s.Create(ctx, qr.CreateQRInput{}); !errors.Is(err, domain.ErrInvalidInput) || len(f.calls) != 0 {
		t.Fatalf("create validation: %v", err)
	}
	if _, err := s.Lookup(ctx, ""); !errors.Is(err, domain.ErrInvalidInput) || len(f.calls) != 0 {
		t.Fatalf("lookup validation: %v", err)
	}
	if _, err := s.PayWithCard(ctx, "tok", qr.PayQRInput{}); !errors.Is(err, domain.ErrInvalidInput) || len(f.calls) != 0 {
		t.Fatalf("pay validation: %v", err)
	}
	if q, err := s.Lookup(ctx, "0002"); err != nil || q.QrCode != "0002" || f.calls[0] != "LookupQR" {
		t.Fatal(err)
	}
	if _, err := s.Create(ctx, qr.CreateQRInput{Amount: 1, TransactionID: "T1", ExpiresIn: 60}); err != nil || f.calls[1] != "CreateQR" {
		t.Fatal(err)
	}
	if p, err := s.PayWithCard(ctx, "tok", qr.PayQRInput{QrCode: "0002", TransactionID: "T1"}); err != nil || p.ID != 5 || f.args[0] != "tok" {
		t.Fatal(err)
	}
}

func TestSandboxAndAccessDelegate(t *testing.T) {
	f := &fakeAPI{}
	sb := application.NewSandbox(f)
	if raw, err := sb.InvoiceStatus(ctx, "inv-1"); err != nil || string(raw) != `{"status":"PAID"}` || f.args[0] != "inv-1" {
		t.Fatal(err)
	}
	if err := sb.MarkInvoicePaid(ctx, "inv-1"); err != nil || f.calls[1] != "MarkInvoicePaid" {
		t.Fatal(err)
	}
	if err := sb.RunSubscriptionBilling(ctx, 7); err != nil || f.calls[2] != "RunSubscriptionBilling" || f.args[0] != int64(7) {
		t.Fatal(err)
	}
	a := application.NewAccess(f)
	if tp, err := a.Authenticate(ctx); err != nil || tp.AccessToken != "a" || f.calls[3] != "CreateToken" {
		t.Fatal(err)
	}
	if tp, err := a.Refresh(ctx); err != nil || tp.AccessToken != "b" || f.calls[4] != "RefreshToken" {
		t.Fatal(err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./tests/gateway/application/`
Expected: FAIL, package `gateway/application` not found.

- [ ] **Step 3: Write ports and application**

`gateway/ports/ports.go`:
```go
// Package ports declares the outbound capabilities the Gateway application layer depends on:
// one interface per aggregate, all implemented by adapters/httpapi. Method names are unique
// across the interfaces so a single adapter can satisfy every port.
package ports

import (
	"context"
	"encoding/json"

	"github.com/techpartners-asia/bonum-go/gateway/domain/access"
	"github.com/techpartners-asia/bonum-go/gateway/domain/card"
	"github.com/techpartners-asia/bonum-go/gateway/domain/checkout"
	"github.com/techpartners-asia/bonum-go/gateway/domain/qr"
	"github.com/techpartners-asia/bonum-go/gateway/domain/subscription"
)

// AccessAPI exchanges the Terminal credential for bearer tokens.
type AccessAPI interface {
	CreateToken(ctx context.Context) (*access.TokenPair, error)
	RefreshToken(ctx context.Context) (*access.TokenPair, error)
}

// CheckoutAPI opens hosted checkout Invoices.
type CheckoutAPI interface {
	Providers(ctx context.Context) ([]checkout.PaymentProviderStatus, error)
	CreateInvoice(ctx context.Context, in checkout.CreateInvoiceInput) (*checkout.Invoice, error)
}

// CardAPI tokenizes cards and charges Card Tokens. Purchase returns *card.DeclinedError
// when the bank refuses.
type CardAPI interface {
	Tokenize(ctx context.Context, in card.TokenizeInput) (*card.Tokenization, error)
	Purchase(ctx context.Context, cardToken string, in card.PurchaseInput) (*card.Purchase, error)
	Reverse(ctx context.Context, cardToken, transactionID string) error
}

// SubscriptionAPI manages Card Tokens enrolled in Payment Plans.
type SubscriptionAPI interface {
	Plans(ctx context.Context) ([]subscription.PaymentPlan, error)
	Subscribe(ctx context.Context, cardToken string, in subscription.SubscribeInput) (*subscription.Subscription, error)
	ListSubscriptions(ctx context.Context, cardToken string) ([]subscription.Subscription, error)
	ChangeCardByTokenizing(ctx context.Context, id int64, in subscription.ChangeCardInput) (*card.Tokenization, error)
	ChangeCard(ctx context.Context, id int64, cardToken string) (*subscription.Subscription, error)
	Unsubscribe(ctx context.Context, id, planID int64) error
	DeleteSubscription(ctx context.Context, id, planID int64) error
}

// QRAPI creates and settles QR Invoices.
type QRAPI interface {
	CreateQR(ctx context.Context, in qr.CreateQRInput) (*qr.QRInvoice, error)
	LookupQR(ctx context.Context, qrCode string) (*qr.QRInvoice, error)
	PayQRWithCard(ctx context.Context, cardToken string, in qr.PayQRInput) (*card.Purchase, error)
}

// SandboxAPI groups helpers Bonum only permits outside production.
type SandboxAPI interface {
	InvoiceStatus(ctx context.Context, invoiceID string) (json.RawMessage, error)
	MarkInvoicePaid(ctx context.Context, invoiceID string) error
	RunSubscriptionBilling(ctx context.Context, id int64) error
}
```

`gateway/application/commands/access/authenticate.go`:
```go
// Package access holds the Access aggregate's commands: forcing a fresh token or refresh.
package access

import (
	"context"

	"github.com/techpartners-asia/bonum-go/gateway/domain/access"
	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

// AuthenticateHandler forces a fresh TokenPair via auth/create. The endpoint is rate
// limited; do not call it in a loop.
type AuthenticateHandler struct{ api ports.AccessAPI }

func NewAuthenticateHandler(api ports.AccessAPI) *AuthenticateHandler {
	return &AuthenticateHandler{api: api}
}

// Handle takes no input: Authenticate always re-derives credentials from the Terminal's
// AppSecret held by the adapter.
func (h *AuthenticateHandler) Handle(ctx context.Context) (*access.TokenPair, error) {
	return h.api.CreateToken(ctx)
}
```

`gateway/application/commands/access/refresh.go`:
```go
package access

import (
	"context"

	"github.com/techpartners-asia/bonum-go/gateway/domain/access"
	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

// RefreshHandler exchanges the cached refresh token for a new access token via auth/refresh.
type RefreshHandler struct{ api ports.AccessAPI }

func NewRefreshHandler(api ports.AccessAPI) *RefreshHandler { return &RefreshHandler{api: api} }

func (h *RefreshHandler) Handle(ctx context.Context) (*access.TokenPair, error) {
	return h.api.RefreshToken(ctx)
}
```

`gateway/application/commands/checkout/create_invoice.go`:
```go
// Package checkout holds the Checkout aggregate's write use case: opening an Invoice.
package checkout

import (
	"context"

	"github.com/techpartners-asia/bonum-go/gateway/domain/checkout"
	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

// CreateInvoiceCommand is the Invoice aggregate's create input.
type CreateInvoiceCommand = checkout.CreateInvoiceInput

// CreateInvoiceHandler opens an Invoice. Redirect the customer to FollowUpLink.
type CreateInvoiceHandler struct{ api ports.CheckoutAPI }

func NewCreateInvoiceHandler(api ports.CheckoutAPI) *CreateInvoiceHandler {
	return &CreateInvoiceHandler{api: api}
}

func (h *CreateInvoiceHandler) Handle(ctx context.Context, cmd CreateInvoiceCommand) (*checkout.Invoice, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}
	return h.api.CreateInvoice(ctx, cmd)
}
```

`gateway/application/queries/checkout/providers.go`:
```go
// Package checkout holds the Checkout aggregate's read use case: listing enabled providers.
package checkout

import (
	"context"

	"github.com/techpartners-asia/bonum-go/gateway/domain/checkout"
	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

// ProvidersHandler lists the payment options currently enabled for the Terminal.
type ProvidersHandler struct{ api ports.CheckoutAPI }

func NewProvidersHandler(api ports.CheckoutAPI) *ProvidersHandler {
	return &ProvidersHandler{api: api}
}

// Handle takes no input: Providers always reads the calling Terminal's configuration.
func (h *ProvidersHandler) Handle(ctx context.Context) ([]checkout.PaymentProviderStatus, error) {
	return h.api.Providers(ctx)
}
```

`gateway/application/commands/card/tokenize.go`:
```go
// Package card holds the Card Token aggregate's write use cases: tokenization, purchases
// and reversals.
package card

import (
	"context"

	"github.com/techpartners-asia/bonum-go/gateway/domain/card"
	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

// TokenizeCommand starts a card tokenization flow.
type TokenizeCommand = card.TokenizeInput

// TokenizeHandler starts a card tokenization flow. Redirect the customer to FollowUpLink.
type TokenizeHandler struct{ api ports.CardAPI }

func NewTokenizeHandler(api ports.CardAPI) *TokenizeHandler { return &TokenizeHandler{api: api} }

func (h *TokenizeHandler) Handle(ctx context.Context, cmd TokenizeCommand) (*card.Tokenization, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}
	return h.api.Tokenize(ctx, cmd)
}
```

`gateway/application/commands/card/purchase.go`:
```go
package card

import (
	"context"

	"github.com/techpartners-asia/bonum-go/gateway/domain/card"
	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

// PurchaseCommand charges CardToken for the amount and details in the embedded PurchaseInput.
type PurchaseCommand struct {
	CardToken string
	card.PurchaseInput
}

// PurchaseHandler charges a Card Token. Under load Bonum may answer with Status QUEUED; the
// final result then arrives as a TokenPaymentEvent. A bank refusal is returned as
// *card.DeclinedError.
type PurchaseHandler struct{ api ports.CardAPI }

func NewPurchaseHandler(api ports.CardAPI) *PurchaseHandler { return &PurchaseHandler{api: api} }

func (h *PurchaseHandler) Handle(ctx context.Context, cmd PurchaseCommand) (*card.Purchase, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}
	return h.api.Purchase(ctx, cmd.CardToken, cmd.PurchaseInput)
}
```

`gateway/application/commands/card/reverse.go`:
```go
package card

import (
	"context"

	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

// ReverseCommand rolls back a Purchase identified by the merchant TransactionID.
type ReverseCommand struct {
	CardToken     string
	TransactionID string
}

type ReverseHandler struct{ api ports.CardAPI }

func NewReverseHandler(api ports.CardAPI) *ReverseHandler { return &ReverseHandler{api: api} }

func (h *ReverseHandler) Handle(ctx context.Context, cmd ReverseCommand) error {
	return h.api.Reverse(ctx, cmd.CardToken, cmd.TransactionID)
}
```

`gateway/application/commands/subscription/subscribe.go`:
```go
// Package subscription holds the Subscription aggregate's write use cases.
package subscription

import (
	"context"

	"github.com/techpartners-asia/bonum-go/gateway/domain/subscription"
	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

// SubscribeCommand enrols CardToken in a Payment Plan.
type SubscribeCommand struct {
	CardToken string
	subscription.SubscribeInput
}

// SubscribeHandler enrols a Card Token in a Payment Plan. If today matches CycleValue (or
// PayNow is set) the first charge happens immediately.
type SubscribeHandler struct{ api ports.SubscriptionAPI }

func NewSubscribeHandler(api ports.SubscriptionAPI) *SubscribeHandler {
	return &SubscribeHandler{api: api}
}

func (h *SubscribeHandler) Handle(ctx context.Context, cmd SubscribeCommand) (*subscription.Subscription, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}
	return h.api.Subscribe(ctx, cmd.CardToken, cmd.SubscribeInput)
}
```

`gateway/application/commands/subscription/change_card_by_tokenizing.go`:
```go
package subscription

import (
	"context"

	"github.com/techpartners-asia/bonum-go/gateway/domain/card"
	"github.com/techpartners-asia/bonum-go/gateway/domain/subscription"
	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

// ChangeCardByTokenizingCommand moves Subscription ID onto a brand-new card.
type ChangeCardByTokenizingCommand struct {
	ID int64
	subscription.ChangeCardInput
}

// ChangeCardByTokenizingHandler starts a Tokenization for the Subscription's new card.
// Redirect the customer to FollowUpLink.
type ChangeCardByTokenizingHandler struct{ api ports.SubscriptionAPI }

func NewChangeCardByTokenizingHandler(api ports.SubscriptionAPI) *ChangeCardByTokenizingHandler {
	return &ChangeCardByTokenizingHandler{api: api}
}

func (h *ChangeCardByTokenizingHandler) Handle(ctx context.Context, cmd ChangeCardByTokenizingCommand) (*card.Tokenization, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}
	return h.api.ChangeCardByTokenizing(ctx, cmd.ID, cmd.ChangeCardInput)
}
```

`gateway/application/commands/subscription/change_card.go`:
```go
package subscription

import (
	"context"

	"github.com/techpartners-asia/bonum-go/gateway/domain/subscription"
	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

// ChangeCardCommand moves Subscription ID onto an already stored Card Token.
type ChangeCardCommand struct {
	ID        int64
	CardToken string
}

type ChangeCardHandler struct{ api ports.SubscriptionAPI }

func NewChangeCardHandler(api ports.SubscriptionAPI) *ChangeCardHandler {
	return &ChangeCardHandler{api: api}
}

func (h *ChangeCardHandler) Handle(ctx context.Context, cmd ChangeCardCommand) (*subscription.Subscription, error) {
	return h.api.ChangeCard(ctx, cmd.ID, cmd.CardToken)
}
```

`gateway/application/commands/subscription/unsubscribe.go`:
```go
package subscription

import (
	"context"

	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

// UnsubscribeCommand cancels Subscription ID; the already scheduled next billing still runs.
type UnsubscribeCommand struct {
	ID     int64
	PlanID int64
}

type UnsubscribeHandler struct{ api ports.SubscriptionAPI }

func NewUnsubscribeHandler(api ports.SubscriptionAPI) *UnsubscribeHandler {
	return &UnsubscribeHandler{api: api}
}

func (h *UnsubscribeHandler) Handle(ctx context.Context, cmd UnsubscribeCommand) error {
	return h.api.Unsubscribe(ctx, cmd.ID, cmd.PlanID)
}
```

`gateway/application/commands/subscription/delete.go`:
```go
package subscription

import (
	"context"

	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

// DeleteCommand cancels Subscription ID immediately; no further billing is created.
type DeleteCommand struct {
	ID     int64
	PlanID int64
}

type DeleteHandler struct{ api ports.SubscriptionAPI }

func NewDeleteHandler(api ports.SubscriptionAPI) *DeleteHandler { return &DeleteHandler{api: api} }

func (h *DeleteHandler) Handle(ctx context.Context, cmd DeleteCommand) error {
	return h.api.DeleteSubscription(ctx, cmd.ID, cmd.PlanID)
}
```

`gateway/application/queries/subscription/plans.go`:
```go
// Package subscription holds the Subscription aggregate's read use cases.
package subscription

import (
	"context"

	"github.com/techpartners-asia/bonum-go/gateway/domain/subscription"
	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

// PlansHandler returns the Terminal's Payment Plans.
type PlansHandler struct{ api ports.SubscriptionAPI }

func NewPlansHandler(api ports.SubscriptionAPI) *PlansHandler { return &PlansHandler{api: api} }

func (h *PlansHandler) Handle(ctx context.Context) ([]subscription.PaymentPlan, error) {
	return h.api.Plans(ctx)
}
```

`gateway/application/queries/subscription/list_subscriptions.go`:
```go
package subscription

import (
	"context"

	"github.com/techpartners-asia/bonum-go/gateway/domain/subscription"
	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

// ListSubscriptionsQuery returns the Subscriptions attached to CardToken.
type ListSubscriptionsQuery struct{ CardToken string }

type ListSubscriptionsHandler struct{ api ports.SubscriptionAPI }

func NewListSubscriptionsHandler(api ports.SubscriptionAPI) *ListSubscriptionsHandler {
	return &ListSubscriptionsHandler{api: api}
}

func (h *ListSubscriptionsHandler) Handle(ctx context.Context, q ListSubscriptionsQuery) ([]subscription.Subscription, error) {
	return h.api.ListSubscriptions(ctx, q.CardToken)
}
```

`gateway/application/commands/qr/create_qr.go`:
```go
// Package qr holds the QR Invoice aggregate's write use cases.
package qr

import (
	"context"

	"github.com/techpartners-asia/bonum-go/gateway/domain/qr"
	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

// CreateQRCommand opens a QR Invoice.
type CreateQRCommand = qr.CreateQRInput

// CreateQRHandler opens a QR Invoice. Render QrImage or offer Links; the outcome arrives as
// a PaymentEvent.
type CreateQRHandler struct{ api ports.QRAPI }

func NewCreateQRHandler(api ports.QRAPI) *CreateQRHandler { return &CreateQRHandler{api: api} }

func (h *CreateQRHandler) Handle(ctx context.Context, cmd CreateQRCommand) (*qr.QRInvoice, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}
	return h.api.CreateQR(ctx, cmd)
}
```

`gateway/application/commands/qr/pay_qr.go`:
```go
package qr

import (
	"context"

	"github.com/techpartners-asia/bonum-go/gateway/domain/card"
	"github.com/techpartners-asia/bonum-go/gateway/domain/qr"
	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

// PayQRCommand settles a QR Invoice with a stored Card Token.
type PayQRCommand struct {
	CardToken string
	qr.PayQRInput
}

type PayQRHandler struct{ api ports.QRAPI }

func NewPayQRHandler(api ports.QRAPI) *PayQRHandler { return &PayQRHandler{api: api} }

func (h *PayQRHandler) Handle(ctx context.Context, cmd PayQRCommand) (*card.Purchase, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}
	return h.api.PayQRWithCard(ctx, cmd.CardToken, cmd.PayQRInput)
}
```

`gateway/application/queries/qr/lookup_qr.go`:
```go
// Package qr holds the QR Invoice aggregate's read use case.
package qr

import (
	"context"

	"github.com/techpartners-asia/bonum-go/gateway/domain"
	"github.com/techpartners-asia/bonum-go/gateway/domain/qr"
	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

// LookupQRQuery returns the QR Invoice behind a scanned QPay QR string.
type LookupQRQuery struct{ QrCode string }

type LookupQRHandler struct{ api ports.QRAPI }

func NewLookupQRHandler(api ports.QRAPI) *LookupQRHandler { return &LookupQRHandler{api: api} }

func (h *LookupQRHandler) Handle(ctx context.Context, q LookupQRQuery) (*qr.QRInvoice, error) {
	if q.QrCode == "" {
		return nil, domain.Invalid("qrCode", "required")
	}
	return h.api.LookupQR(ctx, q.QrCode)
}
```

`gateway/application/commands/sandbox/mark_invoice_paid.go`:
```go
// Package sandbox holds the Sandbox aggregate's write use cases: helpers Bonum only permits
// outside production.
package sandbox

import (
	"context"

	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

// MarkInvoicePaidCommand marks InvoiceID as paid so the webhook fires.
type MarkInvoicePaidCommand struct{ InvoiceID string }

type MarkInvoicePaidHandler struct{ api ports.SandboxAPI }

func NewMarkInvoicePaidHandler(api ports.SandboxAPI) *MarkInvoicePaidHandler {
	return &MarkInvoicePaidHandler{api: api}
}

func (h *MarkInvoicePaidHandler) Handle(ctx context.Context, cmd MarkInvoicePaidCommand) error {
	return h.api.MarkInvoicePaid(ctx, cmd.InvoiceID)
}
```

`gateway/application/commands/sandbox/run_subscription_billing.go`:
```go
package sandbox

import (
	"context"

	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

// RunSubscriptionBillingCommand triggers a billing run for Subscription ID on demand.
type RunSubscriptionBillingCommand struct{ ID int64 }

type RunSubscriptionBillingHandler struct{ api ports.SandboxAPI }

func NewRunSubscriptionBillingHandler(api ports.SandboxAPI) *RunSubscriptionBillingHandler {
	return &RunSubscriptionBillingHandler{api: api}
}

func (h *RunSubscriptionBillingHandler) Handle(ctx context.Context, cmd RunSubscriptionBillingCommand) error {
	return h.api.RunSubscriptionBilling(ctx, cmd.ID)
}
```

`gateway/application/queries/sandbox/invoice_status.go`:
```go
// Package sandbox holds the Sandbox aggregate's read use case.
package sandbox

import (
	"context"
	"encoding/json"

	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

// InvoiceStatusQuery returns the raw invoice record. Bonum forbids polling in production:
// rely on your own invoice table plus the webhook instead.
type InvoiceStatusQuery struct{ InvoiceID string }

type InvoiceStatusHandler struct{ api ports.SandboxAPI }

func NewInvoiceStatusHandler(api ports.SandboxAPI) *InvoiceStatusHandler {
	return &InvoiceStatusHandler{api: api}
}

func (h *InvoiceStatusHandler) Handle(ctx context.Context, q InvoiceStatusQuery) (json.RawMessage, error) {
	return h.api.InvoiceStatus(ctx, q.InvoiceID)
}
```

`gateway/application/access.go`:
```go
// Package application holds the Gateway facades: one struct per aggregate exposing the same
// public methods as before, each delegating to a Command/Query Handler in
// application/commands/<aggregate> or application/queries/<aggregate>. This is the CQRS-lite
// split: one Handler per use case, kept behind a facade so callers (ultimately the bonum
// package) see one object per aggregate rather than one per use case.
package application

import (
	"context"

	accesscmd "github.com/techpartners-asia/bonum-go/gateway/application/commands/access"
	"github.com/techpartners-asia/bonum-go/gateway/domain/access"
	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

// Access forces token creation or refresh. Normally unnecessary: every call obtains a token
// on demand inside the adapter.
type Access struct {
	authenticate *accesscmd.AuthenticateHandler
	refresh      *accesscmd.RefreshHandler
}

func NewAccess(api ports.AccessAPI) *Access {
	return &Access{
		authenticate: accesscmd.NewAuthenticateHandler(api),
		refresh:      accesscmd.NewRefreshHandler(api),
	}
}

// Authenticate forces a fresh TokenPair via auth/create. The endpoint is rate limited.
func (s *Access) Authenticate(ctx context.Context) (*access.TokenPair, error) {
	return s.authenticate.Handle(ctx)
}

// Refresh exchanges the cached refresh token for a new access token via auth/refresh.
func (s *Access) Refresh(ctx context.Context) (*access.TokenPair, error) {
	return s.refresh.Handle(ctx)
}
```

`gateway/application/invoices.go`:
```go
package application

import (
	"context"

	checkoutcmd "github.com/techpartners-asia/bonum-go/gateway/application/commands/checkout"
	checkoutqry "github.com/techpartners-asia/bonum-go/gateway/application/queries/checkout"
	"github.com/techpartners-asia/bonum-go/gateway/domain/checkout"
	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

// Invoices is the Invoice aggregate's use cases: hosted checkout.
type Invoices struct {
	providers *checkoutqry.ProvidersHandler
	create    *checkoutcmd.CreateInvoiceHandler
}

func NewInvoices(api ports.CheckoutAPI) *Invoices {
	return &Invoices{
		providers: checkoutqry.NewProvidersHandler(api),
		create:    checkoutcmd.NewCreateInvoiceHandler(api),
	}
}

// Providers lists the payment options currently enabled for this Terminal.
func (s *Invoices) Providers(ctx context.Context) ([]checkout.PaymentProviderStatus, error) {
	return s.providers.Handle(ctx)
}

// Create opens an Invoice. Redirect the customer to FollowUpLink.
func (s *Invoices) Create(ctx context.Context, in checkout.CreateInvoiceInput) (*checkout.Invoice, error) {
	return s.create.Handle(ctx, in)
}
```

`gateway/application/cards.go`:
```go
package application

import (
	"context"

	cardcmd "github.com/techpartners-asia/bonum-go/gateway/application/commands/card"
	"github.com/techpartners-asia/bonum-go/gateway/domain/card"
	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

// Cards is the Card Token aggregate's use cases: tokenization and charges against stored cards.
type Cards struct {
	tokenize *cardcmd.TokenizeHandler
	purchase *cardcmd.PurchaseHandler
	reverse  *cardcmd.ReverseHandler
}

func NewCards(api ports.CardAPI) *Cards {
	return &Cards{
		tokenize: cardcmd.NewTokenizeHandler(api),
		purchase: cardcmd.NewPurchaseHandler(api),
		reverse:  cardcmd.NewReverseHandler(api),
	}
}

// Tokenize starts a card tokenization flow. Redirect the customer to FollowUpLink.
func (s *Cards) Tokenize(ctx context.Context, in card.TokenizeInput) (*card.Tokenization, error) {
	return s.tokenize.Handle(ctx, in)
}

// Purchase charges a Card Token. Under load Bonum may answer with Status QUEUED; the final
// result then arrives as a TokenPaymentEvent. A bank refusal is returned as *card.DeclinedError.
func (s *Cards) Purchase(ctx context.Context, cardToken string, in card.PurchaseInput) (*card.Purchase, error) {
	return s.purchase.Handle(ctx, cardcmd.PurchaseCommand{CardToken: cardToken, PurchaseInput: in})
}

// Reverse rolls back a Purchase identified by the merchant TransactionID.
func (s *Cards) Reverse(ctx context.Context, cardToken, transactionID string) error {
	return s.reverse.Handle(ctx, cardcmd.ReverseCommand{CardToken: cardToken, TransactionID: transactionID})
}
```

`gateway/application/subscriptions.go`:
```go
package application

import (
	"context"

	subscriptioncmd "github.com/techpartners-asia/bonum-go/gateway/application/commands/subscription"
	subscriptionqry "github.com/techpartners-asia/bonum-go/gateway/application/queries/subscription"
	"github.com/techpartners-asia/bonum-go/gateway/domain/card"
	"github.com/techpartners-asia/bonum-go/gateway/domain/subscription"
	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

// Subscriptions is the Subscription aggregate's use cases: recurring charges on a Card Token.
type Subscriptions struct {
	plans                  *subscriptionqry.PlansHandler
	subscribe              *subscriptioncmd.SubscribeHandler
	list                   *subscriptionqry.ListSubscriptionsHandler
	changeCardByTokenizing *subscriptioncmd.ChangeCardByTokenizingHandler
	changeCard             *subscriptioncmd.ChangeCardHandler
	unsubscribe            *subscriptioncmd.UnsubscribeHandler
	delete                 *subscriptioncmd.DeleteHandler
}

func NewSubscriptions(api ports.SubscriptionAPI) *Subscriptions {
	return &Subscriptions{
		plans:                  subscriptionqry.NewPlansHandler(api),
		subscribe:              subscriptioncmd.NewSubscribeHandler(api),
		list:                   subscriptionqry.NewListSubscriptionsHandler(api),
		changeCardByTokenizing: subscriptioncmd.NewChangeCardByTokenizingHandler(api),
		changeCard:             subscriptioncmd.NewChangeCardHandler(api),
		unsubscribe:            subscriptioncmd.NewUnsubscribeHandler(api),
		delete:                 subscriptioncmd.NewDeleteHandler(api),
	}
}

// Plans returns the Terminal's Payment Plans.
func (s *Subscriptions) Plans(ctx context.Context) ([]subscription.PaymentPlan, error) {
	return s.plans.Handle(ctx)
}

// Subscribe enrols a Card Token in a Payment Plan. If today matches CycleValue (or PayNow
// is set) the first charge happens immediately.
func (s *Subscriptions) Subscribe(ctx context.Context, cardToken string, in subscription.SubscribeInput) (*subscription.Subscription, error) {
	return s.subscribe.Handle(ctx, subscriptioncmd.SubscribeCommand{CardToken: cardToken, SubscribeInput: in})
}

// List returns the Subscriptions attached to a Card Token.
func (s *Subscriptions) List(ctx context.Context, cardToken string) ([]subscription.Subscription, error) {
	return s.list.Handle(ctx, subscriptionqry.ListSubscriptionsQuery{CardToken: cardToken})
}

// ChangeCardByTokenizing moves a Subscription onto a brand-new card by starting a
// Tokenization. Redirect the customer to FollowUpLink.
func (s *Subscriptions) ChangeCardByTokenizing(ctx context.Context, id int64, in subscription.ChangeCardInput) (*card.Tokenization, error) {
	return s.changeCardByTokenizing.Handle(ctx, subscriptioncmd.ChangeCardByTokenizingCommand{ID: id, ChangeCardInput: in})
}

// ChangeCard moves a Subscription onto an already stored Card Token.
func (s *Subscriptions) ChangeCard(ctx context.Context, id int64, cardToken string) (*subscription.Subscription, error) {
	return s.changeCard.Handle(ctx, subscriptioncmd.ChangeCardCommand{ID: id, CardToken: cardToken})
}

// Unsubscribe cancels a Subscription; the already scheduled next billing still runs.
func (s *Subscriptions) Unsubscribe(ctx context.Context, id, planID int64) error {
	return s.unsubscribe.Handle(ctx, subscriptioncmd.UnsubscribeCommand{ID: id, PlanID: planID})
}

// Delete cancels a Subscription immediately; no further billing is created.
func (s *Subscriptions) Delete(ctx context.Context, id, planID int64) error {
	return s.delete.Handle(ctx, subscriptioncmd.DeleteCommand{ID: id, PlanID: planID})
}
```

`gateway/application/qr.go`:
```go
package application

import (
	"context"

	qrcmd "github.com/techpartners-asia/bonum-go/gateway/application/commands/qr"
	qrqry "github.com/techpartners-asia/bonum-go/gateway/application/queries/qr"
	"github.com/techpartners-asia/bonum-go/gateway/domain/card"
	"github.com/techpartners-asia/bonum-go/gateway/domain/qr"
	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

// QR is the QR Invoice aggregate's use cases: QPay-compatible QR and deeplink payments.
type QR struct {
	create      *qrcmd.CreateQRHandler
	lookup      *qrqry.LookupQRHandler
	payWithCard *qrcmd.PayQRHandler
}

func NewQR(api ports.QRAPI) *QR {
	return &QR{
		create:      qrcmd.NewCreateQRHandler(api),
		lookup:      qrqry.NewLookupQRHandler(api),
		payWithCard: qrcmd.NewPayQRHandler(api),
	}
}

// Create opens a QR Invoice. Render QrImage or offer Links; the outcome arrives as a PaymentEvent.
func (s *QR) Create(ctx context.Context, in qr.CreateQRInput) (*qr.QRInvoice, error) {
	return s.create.Handle(ctx, in)
}

// Lookup returns the QR Invoice behind a scanned QPay QR string.
func (s *QR) Lookup(ctx context.Context, qrCode string) (*qr.QRInvoice, error) {
	return s.lookup.Handle(ctx, qrqry.LookupQRQuery{QrCode: qrCode})
}

// PayWithCard settles a QR Invoice with a stored Card Token.
func (s *QR) PayWithCard(ctx context.Context, cardToken string, in qr.PayQRInput) (*card.Purchase, error) {
	return s.payWithCard.Handle(ctx, qrcmd.PayQRCommand{CardToken: cardToken, PayQRInput: in})
}
```

`gateway/application/sandbox.go`:
```go
package application

import (
	"context"
	"encoding/json"

	sandboxcmd "github.com/techpartners-asia/bonum-go/gateway/application/commands/sandbox"
	sandboxqry "github.com/techpartners-asia/bonum-go/gateway/application/queries/sandbox"
	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

// Sandbox groups helpers Bonum only permits outside production. Keeping them off the
// aggregate services stops them leaking into production code paths.
type Sandbox struct {
	invoiceStatus          *sandboxqry.InvoiceStatusHandler
	markInvoicePaid        *sandboxcmd.MarkInvoicePaidHandler
	runSubscriptionBilling *sandboxcmd.RunSubscriptionBillingHandler
}

func NewSandbox(api ports.SandboxAPI) *Sandbox {
	return &Sandbox{
		invoiceStatus:          sandboxqry.NewInvoiceStatusHandler(api),
		markInvoicePaid:        sandboxcmd.NewMarkInvoicePaidHandler(api),
		runSubscriptionBilling: sandboxcmd.NewRunSubscriptionBillingHandler(api),
	}
}

// InvoiceStatus returns the raw invoice record. Bonum forbids polling in production:
// rely on your own invoice table plus the webhook instead.
func (s *Sandbox) InvoiceStatus(ctx context.Context, invoiceID string) (json.RawMessage, error) {
	return s.invoiceStatus.Handle(ctx, sandboxqry.InvoiceStatusQuery{InvoiceID: invoiceID})
}

// MarkInvoicePaid marks an Invoice as paid so the webhook fires.
func (s *Sandbox) MarkInvoicePaid(ctx context.Context, invoiceID string) error {
	return s.markInvoicePaid.Handle(ctx, sandboxcmd.MarkInvoicePaidCommand{InvoiceID: invoiceID})
}

// RunSubscriptionBilling triggers a billing run for a Subscription on demand.
func (s *Sandbox) RunSubscriptionBilling(ctx context.Context, id int64) error {
	return s.runSubscriptionBilling.Handle(ctx, sandboxcmd.RunSubscriptionBillingCommand{ID: id})
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go build ./... && go vet ./... && go test ./tests/gateway/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add gateway/ports gateway/application tests/gateway/application
git commit -m "Add gateway ports and application services"
```

---

### Task 5: Gateway HTTP adapter

**Files:**
- Create: `gateway/adapters/httpapi/client.go`, `gateway/adapters/httpapi/errors.go`, `gateway/adapters/httpapi/auth.go`, `gateway/adapters/httpapi/checkout.go`, `gateway/adapters/httpapi/card.go`, `gateway/adapters/httpapi/subscription.go`, `gateway/adapters/httpapi/qr.go`, `gateway/adapters/httpapi/sandbox.go`

**Interfaces:**
- Consumes: `internal/rest`, all six ports, domain types.
- Produces: `httpapi.New(baseURL, appSecret, terminalID string) *Client`; `(*Client).SetBaseURL(string)`, `SetTimeout(time.Duration)`, `SetTransport(http.RoundTripper)`, `SetLanguage(string)`, `Close() error`; `*Client` satisfies every port.

No new test file: the adapter is exercised end to end by the existing `tests/gateway` fake-server suite once Task 6 wires it. Compile-time port assertions are the check for this task.

- [ ] **Step 1: Write the adapter**

`gateway/adapters/httpapi/client.go`:
```go
// Package httpapi is the Gateway's outbound HTTP adapter. It implements every interface in
// gateway/ports against Bonum's gateway endpoints and owns everything transport-specific:
// the bearer token lifecycle, common headers, the mpay-service envelope and error decoding.
package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/techpartners-asia/bonum-go/internal/rest"
	"resty.dev/v3"
)

const (
	ecommercePath   = "/bonum-gateway/ecommerce"
	mpayPath        = "/mpay-service/merchant"
	cardTokenHeader = "X-CARD-TOKEN"
	defaultTimeout  = 30 * time.Second
	defaultLanguage = "mn"
)

// Client talks to the Bonum gateway on behalf of one merchant Terminal. It is safe for
// concurrent use; access tokens are fetched lazily and refreshed automatically.
type Client struct {
	rest   *rest.Client
	tokens *tokenSource
	lang   string
}

// New creates an adapter for baseURL authenticated with the Terminal's AppSecret.
func New(baseURL, appSecret, terminalID string) *Client {
	r := rest.New(baseURL, defaultTimeout, decodeError)
	return &Client{rest: r, tokens: newTokenSource(r, appSecret, terminalID), lang: defaultLanguage}
}

func (c *Client) SetBaseURL(u string)               { c.rest.SetBaseURL(u) }
func (c *Client) SetTimeout(d time.Duration)        { c.rest.SetTimeout(d) }
func (c *Client) SetTransport(rt http.RoundTripper) { c.rest.SetTransport(rt) }
func (c *Client) SetLanguage(lang string)           { c.lang = lang }
func (c *Client) Close() error                      { return c.rest.Close() }

// envelope is the wrapper every mpay-service endpoint returns. It is a transport detail:
// callers get Data and never see it.
type envelope[T any] struct {
	TraceID string `json:"traceId"`
	Message string `json:"message"`
	Data    T      `json:"data"`
	Status  int    `json:"status"`
}

type reqOpt func(*resty.Request)

func body(v any) reqOpt { return func(r *resty.Request) { r.SetBody(v) } }
func cardToken(tok string) reqOpt {
	return func(r *resty.Request) { r.SetHeader(cardTokenHeader, tok) }
}
func pathParam(k, v string) reqOpt { return func(r *resty.Request) { r.SetPathParam(k, v) } }
func query(k, v string) reqOpt     { return func(r *resty.Request) { r.SetQueryParam(k, v) } }

// call is the single path every gateway endpoint goes through: obtain a bearer token, add
// the common headers, apply the endpoint's options, execute, decode into T.
func call[T any](ctx context.Context, c *Client, method, path string, opts ...reqOpt) (*T, error) {
	token, err := c.tokens.Token(ctx)
	if err != nil {
		return nil, err
	}
	req := c.rest.R().
		SetContext(ctx).
		SetHeader("Accept-Language", c.lang).
		SetAuthToken(token)
	for _, opt := range opts {
		opt(req)
	}
	var out T
	if err := c.rest.Do(req, method, path, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// callEnveloped is call for mpay-service endpoints; it returns the unwrapped Data.
func callEnveloped[T any](ctx context.Context, c *Client, method, path string, opts ...reqOpt) (*T, error) {
	env, err := call[envelope[T]](ctx, c, method, path, opts...)
	if err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// callAction is callEnveloped for endpoints whose Data carries nothing useful.
func callAction(ctx context.Context, c *Client, method, path string, opts ...reqOpt) error {
	_, err := call[envelope[any]](ctx, c, method, path, opts...)
	return err
}
```

`gateway/adapters/httpapi/errors.go`:
```go
package httpapi

import (
	"encoding/json"
	"errors"

	"github.com/techpartners-asia/bonum-go/gateway/domain"
	"github.com/techpartners-asia/bonum-go/gateway/domain/card"
)

type errorBody struct {
	TraceID string `json:"traceId"`
	Message string `json:"message"`
}

// decodeError turns a non-2xx response into *domain.APIError.
func decodeError(status int, body string) error {
	e := &domain.APIError{StatusCode: status, Body: body}
	var eb errorBody
	if json.Unmarshal([]byte(body), &eb) == nil {
		e.TraceID = eb.TraceID
		e.Message = eb.Message
	}
	return e
}

// asDeclined upgrades a 400 whose body is a FAILED Purchase envelope into *card.DeclinedError.
func asDeclined(err error) error {
	var api *domain.APIError
	if !errors.As(err, &api) || api.StatusCode != 400 {
		return err
	}
	var env envelope[card.Purchase]
	if json.Unmarshal([]byte(api.Body), &env) != nil || env.Data.Status != card.PurchaseFailed {
		return err
	}
	return &card.DeclinedError{APIError: api, Purchase: env.Data}
}
```

`gateway/adapters/httpapi/auth.go`: the current root `auth.go` `tokenSource` verbatim (package `httpapi`, `TokenPair` → `access.TokenPair`), plus the port methods:
```go
package httpapi

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/techpartners-asia/bonum-go/gateway/domain/access"
	"github.com/techpartners-asia/bonum-go/gateway/ports"
	"github.com/techpartners-asia/bonum-go/internal/rest"
	"resty.dev/v3"
)

var _ ports.AccessAPI = (*Client)(nil)

// Refresh a little early so an in-flight request never races the server-side expiry.
const tokenExpirySkew = 30 * time.Second

// tokenSource owns the AppSecret -> bearer token lifecycle. Its interface is Token():
// callers get a valid access token and never see create/refresh/expiry.
type tokenSource struct {
	rest       *rest.Client
	appSecret  string
	terminalID string
	now        func() time.Time

	mu               sync.Mutex
	accessToken      string
	refreshToken     string
	accessExpiresAt  time.Time
	refreshExpiresAt time.Time
}

func newTokenSource(r *rest.Client, appSecret, terminalID string) *tokenSource {
	return &tokenSource{rest: r, appSecret: appSecret, terminalID: terminalID, now: time.Now}
}

// Token returns a usable access token, refreshing or re-authenticating as needed.
func (t *tokenSource) Token(ctx context.Context) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := t.now()
	if t.accessToken != "" && now.Before(t.accessExpiresAt.Add(-tokenExpirySkew)) {
		return t.accessToken, nil
	}
	if t.refreshToken != "" && now.Before(t.refreshExpiresAt.Add(-tokenExpirySkew)) {
		if _, err := t.refreshLocked(ctx); err == nil {
			return t.accessToken, nil
		}
	}
	if _, err := t.createLocked(ctx); err != nil {
		return "", err
	}
	return t.accessToken, nil
}

func (t *tokenSource) create(ctx context.Context) (*access.TokenPair, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.createLocked(ctx)
}

func (t *tokenSource) refresh(ctx context.Context) (*access.TokenPair, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.refreshLocked(ctx)
}

func (t *tokenSource) createLocked(ctx context.Context) (*access.TokenPair, error) {
	req := t.rest.R().SetContext(ctx).
		SetHeader("Authorization", "AppSecret "+t.appSecret).
		SetHeader("X-TERMINAL-ID", t.terminalID)
	return t.exchange(req, ecommercePath+"/auth/create")
}

func (t *tokenSource) refreshLocked(ctx context.Context) (*access.TokenPair, error) {
	req := t.rest.R().SetContext(ctx).SetAuthToken(t.refreshToken)
	return t.exchange(req, ecommercePath+"/auth/refresh")
}

func (t *tokenSource) exchange(req *resty.Request, path string) (*access.TokenPair, error) {
	var out access.TokenPair
	if err := t.rest.Do(req, http.MethodGet, path, &out); err != nil {
		return nil, err
	}
	now := t.now()
	t.accessToken = out.AccessToken
	t.accessExpiresAt = now.Add(time.Duration(out.ExpiresIn) * time.Second)
	// auth/refresh may omit the refresh token; keep the one we already have in that case.
	if out.RefreshToken != "" {
		t.refreshToken = out.RefreshToken
		t.refreshExpiresAt = now.Add(time.Duration(out.RefreshExpiresIn) * time.Second)
	}
	return &out, nil
}

// CreateToken forces a fresh TokenPair via auth/create.
func (c *Client) CreateToken(ctx context.Context) (*access.TokenPair, error) { return c.tokens.create(ctx) }

// RefreshToken exchanges the cached refresh token via auth/refresh.
func (c *Client) RefreshToken(ctx context.Context) (*access.TokenPair, error) { return c.tokens.refresh(ctx) }
```

`gateway/adapters/httpapi/checkout.go`:
```go
package httpapi

import (
	"context"
	"net/http"

	"github.com/techpartners-asia/bonum-go/gateway/domain/checkout"
	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

var _ ports.CheckoutAPI = (*Client)(nil)

func (c *Client) Providers(ctx context.Context) ([]checkout.PaymentProviderStatus, error) {
	out, err := call[[]checkout.PaymentProviderStatus](ctx, c, http.MethodGet, ecommercePath+"/invoices/payment-providers")
	if err != nil {
		return nil, err
	}
	return *out, nil
}

func (c *Client) CreateInvoice(ctx context.Context, in checkout.CreateInvoiceInput) (*checkout.Invoice, error) {
	return call[checkout.Invoice](ctx, c, http.MethodPost, ecommercePath+"/invoices", body(in))
}
```

`gateway/adapters/httpapi/card.go`:
```go
package httpapi

import (
	"context"
	"net/http"

	"github.com/techpartners-asia/bonum-go/gateway/domain/card"
	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

var _ ports.CardAPI = (*Client)(nil)

func (c *Client) Tokenize(ctx context.Context, in card.TokenizeInput) (*card.Tokenization, error) {
	return call[card.Tokenization](ctx, c, http.MethodPost, mpayPath+"/cards/tokenize/request", body(in))
}

func (c *Client) Purchase(ctx context.Context, cardTok string, in card.PurchaseInput) (*card.Purchase, error) {
	p, err := callEnveloped[card.Purchase](ctx, c, http.MethodPost, mpayPath+"/transaction/purchase", cardToken(cardTok), body(in))
	if err != nil {
		return nil, asDeclined(err)
	}
	return p, nil
}

func (c *Client) Reverse(ctx context.Context, cardTok, transactionID string) error {
	return callAction(ctx, c, http.MethodDelete, mpayPath+"/transaction/reverse/{transactionId}",
		cardToken(cardTok), pathParam("transactionId", transactionID))
}
```

`gateway/adapters/httpapi/subscription.go`:
```go
package httpapi

import (
	"context"
	"net/http"
	"strconv"

	"github.com/techpartners-asia/bonum-go/gateway/domain/card"
	"github.com/techpartners-asia/bonum-go/gateway/domain/subscription"
	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

var _ ports.SubscriptionAPI = (*Client)(nil)

func subscriptionID(id int64) reqOpt { return pathParam("id", strconv.FormatInt(id, 10)) }

func (c *Client) Plans(ctx context.Context) ([]subscription.PaymentPlan, error) {
	out, err := callEnveloped[[]subscription.PaymentPlan](ctx, c, http.MethodGet, mpayPath+"/values/payment-plans")
	if err != nil {
		return nil, err
	}
	return *out, nil
}

func (c *Client) Subscribe(ctx context.Context, cardTok string, in subscription.SubscribeInput) (*subscription.Subscription, error) {
	return callEnveloped[subscription.Subscription](ctx, c, http.MethodPost, mpayPath+"/subscriptions/subscribe", cardToken(cardTok), body(in))
}

func (c *Client) ListSubscriptions(ctx context.Context, cardTok string) ([]subscription.Subscription, error) {
	out, err := callEnveloped[[]subscription.Subscription](ctx, c, http.MethodGet, mpayPath+"/subscriptions", cardToken(cardTok))
	if err != nil {
		return nil, err
	}
	return *out, nil
}

func (c *Client) ChangeCardByTokenizing(ctx context.Context, id int64, in subscription.ChangeCardInput) (*card.Tokenization, error) {
	return call[card.Tokenization](ctx, c, http.MethodPut, mpayPath+"/subscriptions/{id}/change/create-new-token", subscriptionID(id), body(in))
}

func (c *Client) ChangeCard(ctx context.Context, id int64, cardTok string) (*subscription.Subscription, error) {
	return callEnveloped[subscription.Subscription](ctx, c, http.MethodPut, mpayPath+"/subscriptions/{id}/change", subscriptionID(id), cardToken(cardTok))
}

func (c *Client) Unsubscribe(ctx context.Context, id, planID int64) error {
	return callAction(ctx, c, http.MethodDelete, mpayPath+"/subscriptions/{id}", subscriptionID(id), body(map[string]int64{"planId": planID}))
}

func (c *Client) DeleteSubscription(ctx context.Context, id, planID int64) error {
	return callAction(ctx, c, http.MethodDelete, mpayPath+"/subscriptions/{id}/delete", subscriptionID(id), body(map[string]int64{"planId": planID}))
}
```

`gateway/adapters/httpapi/qr.go`:
```go
package httpapi

import (
	"context"
	"net/http"

	"github.com/techpartners-asia/bonum-go/gateway/domain/card"
	"github.com/techpartners-asia/bonum-go/gateway/domain/qr"
	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

var _ ports.QRAPI = (*Client)(nil)

func (c *Client) CreateQR(ctx context.Context, in qr.CreateQRInput) (*qr.QRInvoice, error) {
	return callEnveloped[qr.QRInvoice](ctx, c, http.MethodPost, mpayPath+"/transaction/qr/create", body(in))
}

func (c *Client) LookupQR(ctx context.Context, qrCode string) (*qr.QRInvoice, error) {
	return callEnveloped[qr.QRInvoice](ctx, c, http.MethodPost, mpayPath+"/transaction/qr", body(map[string]string{"qrCode": qrCode}))
}

func (c *Client) PayQRWithCard(ctx context.Context, cardTok string, in qr.PayQRInput) (*card.Purchase, error) {
	return callEnveloped[card.Purchase](ctx, c, http.MethodPut, mpayPath+"/transaction/qr/pay", cardToken(cardTok), body(in))
}
```

`gateway/adapters/httpapi/sandbox.go`:
```go
package httpapi

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/techpartners-asia/bonum-go/gateway/ports"
)

var _ ports.SandboxAPI = (*Client)(nil)

func (c *Client) InvoiceStatus(ctx context.Context, invoiceID string) (json.RawMessage, error) {
	out, err := call[json.RawMessage](ctx, c, http.MethodGet, ecommercePath+"/invoices/{id}", pathParam("id", invoiceID))
	if err != nil {
		return nil, err
	}
	return *out, nil
}

func (c *Client) MarkInvoicePaid(ctx context.Context, invoiceID string) error {
	_, err := call[json.RawMessage](ctx, c, http.MethodGet, ecommercePath+"/invoices/paid", query("invoiceId", invoiceID))
	return err
}

func (c *Client) RunSubscriptionBilling(ctx context.Context, id int64) error {
	return callAction(ctx, c, http.MethodPut, mpayPath+"/subscriptions/{id}/execute", subscriptionID(id))
}
```

- [ ] **Step 2: Verify it compiles and every port assertion holds**

Run: `go build ./... && go vet ./...`
Expected: no output. A missing or misnamed method fails here with "does not implement ports.X".

- [ ] **Step 3: Commit**

```bash
git add gateway/adapters
git commit -m "Add gateway HTTP adapter implementing every port"
```

---

### Task 6: Switch the bonum facade onto the layers

**Files:**
- Rewrite: `bonum.go`, `errors.go`, `webhook.go`
- Create: `types.go`
- Delete: `auth.go`, `card.go`, `invoice.go`, `qr.go`, `sandbox.go`, `subscription.go`
- Test: existing `tests/gateway/*` (must pass unmodified), `tests/smoke/main.go` (must compile)

**Interfaces:**
- Consumes: `httpapi.New/Set*/Close`, `application.New*`, every domain type.
- Produces: the unchanged public API of package `bonum`.

- [ ] **Step 1: Run the existing suite as the baseline**

Run: `go test ./tests/gateway/`
Expected: PASS (against the old flat code).

- [ ] **Step 2: Rewrite the facade**

`bonum.go`:
```go
// Package bonum is the Gateway bounded context of the Bonum payment SDK: invoices, card
// tokens, purchases, subscriptions, QR invoices and the gateway webhook.
//
// It is backend only. The AppSecret, checksum key and bearer tokens must never reach a
// browser or mobile app. Apple Pay / Google Pay live in the separate wallet package.
//
// This package is a facade: it composes gateway/adapters/httpapi into the use cases in
// gateway/application and re-exports the domain types so callers import only bonum.
package bonum

import (
	"context"
	"net/http"
	"time"

	"github.com/techpartners-asia/bonum-go/gateway/adapters/httpapi"
	"github.com/techpartners-asia/bonum-go/gateway/application"
)

// Environment selects which Bonum gateway host the client talks to.
type Environment string

const (
	Sandbox    Environment = "https://testapi.bonum.mn"
	Production Environment = "https://apis.bonum.mn"
)

// Lang is sent as Accept-Language so Bonum localises response messages.
type Lang string

const (
	MN Lang = "mn"
	EN Lang = "en"
)

// Client talks to the Bonum gateway on behalf of one merchant Terminal. It is safe for
// concurrent use; access tokens are fetched lazily and refreshed automatically.
//
// Operations are grouped by aggregate: Invoices, Cards, Subscriptions, QR. Sandbox holds
// helpers Bonum only allows outside production.
type Client struct {
	Invoices      *InvoiceService
	Cards         *CardService
	Subscriptions *SubscriptionService
	QR            *QRService
	Sandbox       *SandboxService

	access *application.Access
	api    *httpapi.Client
}

type Option func(*Client)

// WithLanguage sets the Accept-Language header (default MN).
func WithLanguage(lang Lang) Option { return func(c *Client) { c.api.SetLanguage(string(lang)) } }

// WithBaseURL overrides the host derived from the Environment, e.g. to go through a proxy.
func WithBaseURL(baseURL string) Option { return func(c *Client) { c.api.SetBaseURL(baseURL) } }

// WithTimeout sets the per-request HTTP timeout (default 30s).
func WithTimeout(d time.Duration) Option { return func(c *Client) { c.api.SetTimeout(d) } }

// WithTransport replaces the HTTP transport, e.g. to record outbound calls.
func WithTransport(rt http.RoundTripper) Option { return func(c *Client) { c.api.SetTransport(rt) } }

// New creates a Client. appSecret and terminalID come from the Bonum merchant portal.
func New(env Environment, appSecret, terminalID string, opts ...Option) *Client {
	api := httpapi.New(string(env), appSecret, terminalID)
	c := &Client{api: api}
	for _, opt := range opts {
		opt(c)
	}
	c.access = application.NewAccess(api)
	c.Invoices = application.NewInvoices(api)
	c.Cards = application.NewCards(api)
	c.Subscriptions = application.NewSubscriptions(api)
	c.QR = application.NewQR(api)
	c.Sandbox = application.NewSandbox(api)
	return c
}

// Close releases the underlying HTTP resources.
func (c *Client) Close() error { return c.api.Close() }

// Authenticate forces a fresh TokenPair via auth/create. Normally unnecessary: every call
// obtains a token on demand. The endpoint is rate limited; do not call it in a loop.
func (c *Client) Authenticate(ctx context.Context) (*TokenPair, error) { return c.access.Authenticate(ctx) }

// Refresh exchanges the cached refresh token for a new access token via auth/refresh.
func (c *Client) Refresh(ctx context.Context) (*TokenPair, error) { return c.access.Refresh(ctx) }
```

`types.go`:
```go
package bonum

import (
	"github.com/techpartners-asia/bonum-go/gateway/application"
	"github.com/techpartners-asia/bonum-go/gateway/domain/access"
	"github.com/techpartners-asia/bonum-go/gateway/domain/card"
	"github.com/techpartners-asia/bonum-go/gateway/domain/checkout"
	"github.com/techpartners-asia/bonum-go/gateway/domain/qr"
	"github.com/techpartners-asia/bonum-go/gateway/domain/subscription"
	"github.com/techpartners-asia/bonum-go/gateway/domain/webhook"
)

// Services (gateway/application).
type (
	InvoiceService      = application.Invoices
	CardService         = application.Cards
	SubscriptionService = application.Subscriptions
	QRService           = application.QR
	SandboxService      = application.Sandbox
)

// Access (gateway/domain/access).
type TokenPair = access.TokenPair

// Checkout (gateway/domain/checkout).
type (
	PaymentProvider       = checkout.PaymentProvider
	ExtraType             = checkout.ExtraType
	PaymentProviderStatus = checkout.PaymentProviderStatus
	Item                  = checkout.Item
	Extra                 = checkout.Extra
	CreateInvoiceInput    = checkout.CreateInvoiceInput
	Invoice               = checkout.Invoice
)

const (
	ProviderQPay      = checkout.ProviderQPay
	ProviderECommerce = checkout.ProviderECommerce
	ProviderWeChat    = checkout.ProviderWeChat
	ProviderSonoShop  = checkout.ProviderSonoShop

	ExtraText   = checkout.ExtraText
	ExtraNumber = checkout.ExtraNumber
	ExtraPhone  = checkout.ExtraPhone
	ExtraEmail  = checkout.ExtraEmail
	ExtraAll    = checkout.ExtraAll
)

// Cards (gateway/domain/card).
type (
	PurchaseStatus       = card.PurchaseStatus
	TokenizePayment      = card.TokenizePayment
	TokenizeSubscription = card.TokenizeSubscription
	TokenizeInput        = card.TokenizeInput
	Tokenization         = card.Tokenization
	PurchaseInput        = card.PurchaseInput
	Purchase             = card.Purchase
)

const (
	PurchaseSuccess = card.PurchaseSuccess
	PurchaseFailed  = card.PurchaseFailed
	PurchaseQueued  = card.PurchaseQueued
)

// Subscriptions (gateway/domain/subscription).
type (
	RecurringType   = subscription.RecurringType
	PaymentPlan     = subscription.PaymentPlan
	SubscribeInput  = subscription.SubscribeInput
	Subscription    = subscription.Subscription
	ChangeCardInput = subscription.ChangeCardInput
)

const (
	RecurringWeekly  = subscription.RecurringWeekly
	RecurringMonthly = subscription.RecurringMonthly
	RecurringYearly  = subscription.RecurringYearly
)

// QR (gateway/domain/qr).
type (
	CreateQRInput = qr.CreateQRInput
	Deeplink      = qr.Deeplink
	QRInvoice     = qr.QRInvoice
	PayQRInput    = qr.PayQRInput
)

// Webhook (gateway/domain/webhook).
type (
	EventType                = webhook.EventType
	Outcome                  = webhook.Outcome
	Event                    = webhook.Event
	EventHeader              = webhook.EventHeader
	PaymentBody              = webhook.PaymentBody
	PaymentEvent             = webhook.PaymentEvent
	Bank                     = webhook.Bank
	Money                    = webhook.Money
	SubscriptionRef          = webhook.SubscriptionRef
	CardTokenBody            = webhook.CardTokenBody
	CardTokenEvent           = webhook.CardTokenEvent
	TokenPaymentBody         = webhook.TokenPaymentBody
	TokenPaymentEvent        = webhook.TokenPaymentEvent
	SubscriptionPaymentBody  = webhook.SubscriptionPaymentBody
	SubscriptionPaymentEvent = webhook.SubscriptionPaymentEvent
)

const (
	EventPayment             = webhook.EventPayment
	EventCardToken           = webhook.EventCardToken
	EventTokenPayment        = webhook.EventTokenPayment
	EventSubscriptionPayment = webhook.EventSubscriptionPayment

	OutcomeSuccess = webhook.OutcomeSuccess
	OutcomeFailed  = webhook.OutcomeFailed
)
```

`errors.go`:
```go
package bonum

import (
	"github.com/techpartners-asia/bonum-go/gateway/domain"
	"github.com/techpartners-asia/bonum-go/gateway/domain/card"
	"github.com/techpartners-asia/bonum-go/gateway/domain/webhook"
)

// Sentinel errors. Match them with errors.Is; the concrete *APIError / *DeclinedError /
// *ValidationError is still available through errors.As.
var (
	ErrInvalidInput = domain.ErrInvalidInput  // local validation failed, or Bonum answered 400
	ErrUnauthorized = domain.ErrUnauthorized  // 401 / 403: bad AppSecret, terminal or token
	ErrNotFound     = domain.ErrNotFound      // 404
	ErrRateLimited  = domain.ErrRateLimited   // 429
	ErrDeclined     = card.ErrDeclined        // card purchase refused by the bank
	ErrBadChecksum  = webhook.ErrBadChecksum  // webhook checksum mismatch
	ErrUnknownEvent = webhook.ErrUnknownEvent // unknown webhook event type
)

type (
	// APIError is returned when Bonum answers with a non-2xx status.
	APIError = domain.APIError
	// DeclinedError is an APIError whose body carried a FAILED Purchase.
	DeclinedError = card.DeclinedError
	// ValidationError is returned before any network call when an input violates an invariant.
	ValidationError = domain.ValidationError
)
```

`webhook.go`:
```go
package bonum

import "github.com/techpartners-asia/bonum-go/gateway/domain/webhook"

// ChecksumHeader carries the HMAC Bonum attaches to every gateway webhook delivery.
const ChecksumHeader = webhook.ChecksumHeader

// Checksum computes hex(HMAC-SHA256(key, body)) exactly as Bonum does for gateway webhooks.
func Checksum(body []byte, checksumKey string) string { return webhook.Checksum(body, checksumKey) }

// ParseWebhook verifies a delivery against the x-checksum-v2 header and decodes it into the
// Event for its type. Pass the body bytes exactly as received; re-serialising changes the hash.
// Returns ErrBadChecksum or ErrUnknownEvent (wrapped) on the two rejection paths.
func ParseWebhook(body []byte, checksumHeader, checksumKey string) (Event, error) {
	return webhook.Parse(body, checksumHeader, checksumKey)
}
```

Delete the old flat files:
```bash
git rm -q auth.go card.go invoice.go qr.go sandbox.go subscription.go
```

- [ ] **Step 3: Run the full suite**

Run: `go build ./... && go vet ./... && go test ./...`
Expected: every package PASS, including the unmodified `tests/gateway` suite; `tests/smoke` compiles.

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "Switch bonum facade onto the gateway layers"
```

---

### Task 7: Wallet domain

**Files:**
- Create: `wallet/domain/errors.go`, `wallet/domain/payment/payment.go`, `wallet/domain/webhook/webhook.go`
- Test: `tests/wallet/domain/payment_test.go`, `tests/wallet/domain/webhook_test.go`

**Interfaces:**
- Produces: `domain.{ErrInvalidInput,ErrUnauthorized,ErrNotFound,ErrRateLimited,APIError{StatusCode,Message,ErrorText,Body},ValidationError,Invalid}`; `payment.{Status,WalletType,Currency,BinCategory,ApplePaymentHeader,ApplePaymentData,ApplePaymentMethod,ApplePayToken,ProcessApplePayInput,ProcessGooglePayInput,ProcessResponse,Payment,AwaitResult}`, `payment.MaxAwaitTimeout`, both inputs' `Validate() error`; `webhook.{SignatureHeader,TimestampHeader,ReplayTolerance,Event,ErrMissingSignature,ErrTimestampExpired,ErrSignatureMismatch}`, `webhook.Sign(body []byte, timestamp, secret string) string`, `webhook.Parse(body []byte, signature, timestamp, secret string) (*Event, error)`, `webhook.ParseAt(body []byte, signature, timestamp, secret string, now time.Time) (*Event, error)`.

- [ ] **Step 1: Write the failing tests**

`tests/wallet/domain/payment_test.go`:
```go
package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/techpartners-asia/bonum-go/wallet/domain"
	"github.com/techpartners-asia/bonum-go/wallet/domain/payment"
)

func appleToken() payment.ApplePayToken {
	return payment.ApplePayToken{
		PaymentData:           payment.ApplePaymentData{Data: "aGVsbG8=", Signature: "MIAG", Version: "EC_v1"},
		TransactionIdentifier: "9AFDA47D",
	}
}

func TestProcessApplePayInputValidate(t *testing.T) {
	if err := (payment.ProcessApplePayInput{OrderID: "o", Amount: 150.50, BranchID: "b", Token: appleToken()}).Validate(); err != nil {
		t.Fatal(err)
	}
	for field, in := range map[string]payment.ProcessApplePayInput{
		"OrderID":  {Token: appleToken()},
		"Amount":   {OrderID: "o", Amount: 0.001, Token: appleToken()},
		"BranchID": {OrderID: "o", BranchID: strings.Repeat("b", 65), Token: appleToken()},
		"Token":    {OrderID: "o"},
	} {
		var v *domain.ValidationError
		if err := in.Validate(); !errors.As(err, &v) || v.Field != field || !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("%s: got %v", field, err)
		}
	}
	long := payment.ProcessApplePayInput{OrderID: strings.Repeat("x", 129), Token: appleToken()}
	if err := long.Validate(); err == nil {
		t.Fatal("129-char order id must fail")
	}
	dec := payment.ProcessApplePayInput{OrderID: "o", Amount: 1.234, Token: appleToken()}
	if err := dec.Validate(); err == nil {
		t.Fatal("3 decimals must fail")
	}
}

func TestProcessGooglePayInputValidate(t *testing.T) {
	if err := (payment.ProcessGooglePayInput{OrderID: "o", Token: "t", CurrencyCode: payment.MNT}).Validate(); err != nil {
		t.Fatal(err)
	}
	for field, in := range map[string]payment.ProcessGooglePayInput{
		"Token":        {OrderID: "o", CurrencyCode: payment.MNT},
		"CurrencyCode": {OrderID: "o", Token: "t", CurrencyCode: "GBP"},
	} {
		var v *domain.ValidationError
		if err := in.Validate(); !errors.As(err, &v) || v.Field != field {
			t.Fatalf("%s: got %v", field, err)
		}
	}
}

func TestAPIErrorMapsStatus(t *testing.T) {
	err := &domain.APIError{StatusCode: 401, Message: "Invalid key", ErrorText: "Unauthorized"}
	if !errors.Is(err, domain.ErrUnauthorized) || err.Error() != "bonum wallet: 401 Invalid key" {
		t.Fatalf("unexpected %v", err)
	}
	if !errors.Is(&domain.APIError{StatusCode: 429}, domain.ErrRateLimited) {
		t.Fatal("429")
	}
}
```

`tests/wallet/domain/webhook_test.go`:
```go
package domain_test

import (
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/techpartners-asia/bonum-go/wallet/domain/payment"
	"github.com/techpartners-asia/bonum-go/wallet/domain/webhook"
)

const (
	secret  = "whsec_test"
	tsFixed = "1713174600" // 2024-04-15T10:30:00Z
	body    = `{"webhookId":"d290","paymentId":"550e","orderId":"ORDER-1","eventType":"AUTHORIZED","status":"AUTHORIZED","amount":"150.50","currency":"MNT","providerReference":"GBK1","failureReason":null,"occurredAt":"2024-04-15T10:30:04.123Z","walletType":"APPLE_PAY","binCategory":"DOMESTIC"}`
)

var sentAt = time.Unix(1713174600, 0)

func TestParseAtAcceptsInsideReplayWindow(t *testing.T) {
	sig := webhook.Sign([]byte(body), tsFixed, secret)
	ev, err := webhook.ParseAt([]byte(body), sig, tsFixed, secret, sentAt.Add(4*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if ev.WebhookID != "d290" || ev.EventType != payment.StatusAuthorized || *ev.BinCategory != payment.Domestic {
		t.Fatalf("unexpected %+v", ev)
	}
	if _, err := webhook.ParseAt([]byte(body), sig, tsFixed, secret, sentAt.Add(-4*time.Minute)); err != nil {
		t.Fatalf("clock skew inside tolerance must pass: %v", err)
	}
}

func TestParseAtRejectsOutsideReplayWindow(t *testing.T) {
	sig := webhook.Sign([]byte(body), tsFixed, secret)
	if _, err := webhook.ParseAt([]byte(body), sig, tsFixed, secret, sentAt.Add(webhook.ReplayTolerance+time.Second)); !errors.Is(err, webhook.ErrTimestampExpired) {
		t.Fatalf("stale: %v", err)
	}
	if _, err := webhook.ParseAt([]byte(body), sig, tsFixed, secret, sentAt.Add(-webhook.ReplayTolerance-time.Second)); !errors.Is(err, webhook.ErrTimestampExpired) {
		t.Fatalf("future: %v", err)
	}
}

func TestParseAtRejectsBadSignatures(t *testing.T) {
	good := webhook.Sign([]byte(body), tsFixed, secret)
	cases := map[string]struct {
		sig, ts string
		want    error
	}{
		"empty signature":       {"", tsFixed, webhook.ErrMissingSignature},
		"empty timestamp":       {good, "", webhook.ErrMissingSignature},
		"non-numeric timestamp": {good, "abc", webhook.ErrMissingSignature},
		"unknown prefix":        {"v2=" + good[3:], tsFixed, webhook.ErrSignatureMismatch},
		"wrong hmac":            {"v1=00", tsFixed, webhook.ErrSignatureMismatch},
	}
	for name, c := range cases {
		if _, err := webhook.ParseAt([]byte(body), c.sig, c.ts, secret, sentAt); !errors.Is(err, c.want) {
			t.Fatalf("%s: got %v, want %v", name, err, c.want)
		}
	}
}

func TestParseUsesWallClock(t *testing.T) {
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	if _, err := webhook.Parse([]byte(body), webhook.Sign([]byte(body), ts, secret), ts, secret); err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./tests/wallet/domain/`
Expected: FAIL, packages not found.

- [ ] **Step 3: Write the wallet domain**

`wallet/domain/errors.go`:
```go
// Package domain holds the errors every Wallet aggregate shares. The aggregates live in the
// sub-packages payment and webhook.
package domain

import (
	"errors"
	"fmt"
)

// Sentinel errors. Match them with errors.Is; the concrete *APIError / *ValidationError is
// still available through errors.As.
var (
	ErrInvalidInput = errors.New("bonum wallet: invalid input") // local validation failed, or Bonum answered 400
	ErrUnauthorized = errors.New("bonum wallet: unauthorized")  // 401: missing or inactive merchant key
	ErrNotFound     = errors.New("bonum wallet: not found")     // 404: unknown payment or order
	ErrRateLimited  = errors.New("bonum wallet: rate limited")  // 429
)

// APIError is returned when Bonum answers with a non-2xx status.
type APIError struct {
	StatusCode int
	Message    string // e.g. "order_id is required"
	ErrorText  string // e.g. "Bad Request"
	Body       string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("bonum wallet: %d %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("bonum wallet: %d %s", e.StatusCode, e.Body)
}

// Is maps HTTP status classes onto the sentinel errors.
func (e *APIError) Is(target error) bool {
	switch target {
	case ErrInvalidInput:
		return e.StatusCode == 400
	case ErrUnauthorized:
		return e.StatusCode == 401 || e.StatusCode == 403
	case ErrNotFound:
		return e.StatusCode == 404
	case ErrRateLimited:
		return e.StatusCode == 429
	}
	return false
}

// ValidationError is returned before any network call when an input violates an invariant.
type ValidationError struct {
	Field  string
	Reason string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("bonum wallet: invalid %s: %s", e.Field, e.Reason)
}
func (e *ValidationError) Is(target error) bool { return target == ErrInvalidInput }

// Invalid builds the ValidationError every aggregate's Validate method returns.
func Invalid(field, reason string) error { return &ValidationError{Field: field, Reason: reason} }
```

`wallet/domain/payment/payment.go`: the current `wallet/types.go` minus `Environment`, as `package payment`, with `validate()` renamed `Validate()`, `invalid(` replaced by `domain.Invalid(`, and this constant added:
```go
// MaxAwaitTimeout is the server-side cap on the await window, chosen by Bonum to leave
// a buffer before Apple Pay's 30-second hard limit.
const MaxAwaitTimeout = 28 * time.Second
```
Package doc:
```go
// Package payment is the Wallet Payment aggregate: one submission of an Apple Pay or
// Google Pay token and its lifecycle from PENDING to AUTHORIZED or FAILED.
package payment
```

`wallet/domain/webhook/webhook.go`: the current `wallet/webhook.go` as `package webhook`, `WebhookEvent` renamed `Event` (fields typed `payment.Status`, `payment.WalletType`, `*payment.BinCategory`), the three sentinels defined locally, `ParseWebhook` split into `Parse` and `ParseAt`:
```go
// Package webhook is the Wallet Webhook Event aggregate: signed deliveries that report a
// Wallet Payment reaching AUTHORIZED or FAILED.
package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/techpartners-asia/bonum-go/wallet/domain/payment"
)

var (
	ErrMissingSignature  = errors.New("bonum wallet: missing or malformed signature headers")
	ErrTimestampExpired  = errors.New("bonum wallet: webhook timestamp outside replay tolerance")
	ErrSignatureMismatch = errors.New("bonum wallet: webhook signature mismatch")
)

const (
	// SignatureHeader carries "v1=<hex HMAC-SHA256>" on every V2 webhook delivery.
	SignatureHeader = "X-PSP-Signature"
	// TimestampHeader carries the unix seconds at which Bonum sent the delivery.
	TimestampHeader = "X-PSP-Timestamp"

	// ReplayTolerance is how far the timestamp may drift from now before the delivery is rejected.
	ReplayTolerance = 5 * time.Minute

	signaturePrefix = "v1="
)

// Event is the JSON body Bonum POSTs when a payment reaches a final status.
type Event struct {
	WebhookID         string                `json:"webhookId"` // Unique per delivery; use as the idempotency key
	PaymentID         string                `json:"paymentId"`
	OrderID           string                `json:"orderId"`
	EventType         payment.Status        `json:"eventType"` // AUTHORIZED or FAILED
	Status            payment.Status        `json:"status"`    // Mirrors EventType
	Amount            string                `json:"amount"`    // Decimal string in major units
	Currency          string                `json:"currency"`  // e.g. "MNT"
	ProviderReference *string               `json:"providerReference"`
	FailureReason     *string               `json:"failureReason"`
	OccurredAt        string                `json:"occurredAt"` // ISO 8601
	WalletType        payment.WalletType    `json:"walletType"`
	BinCategory       *payment.BinCategory  `json:"binCategory"` // nil when the BIN could not be resolved
}

// Sign computes "v1=" + hex(HMAC-SHA256(secret, timestamp + "." + body)) exactly as Bonum does.
func Sign(body []byte, timestamp, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(body)
	return signaturePrefix + hex.EncodeToString(mac.Sum(nil))
}

// Parse is ParseAt against the wall clock.
func Parse(body []byte, signature, timestamp, secret string) (*Event, error) {
	return ParseAt(body, signature, timestamp, secret, time.Now())
}

// ParseAt verifies a delivery against the X-PSP-Signature and X-PSP-Timestamp header values
// as of now, then decodes it. Pass the body bytes exactly as received; re-serialising changes
// the hash. Rejections are ErrMissingSignature, ErrTimestampExpired or ErrSignatureMismatch.
func ParseAt(body []byte, signature, timestamp, secret string, now time.Time) (*Event, error) {
	if err := verify(body, signature, timestamp, secret, now); err != nil {
		return nil, err
	}
	var ev Event
	if err := json.Unmarshal(body, &ev); err != nil {
		return nil, fmt.Errorf("bonum wallet: webhook body: %w", err)
	}
	return &ev, nil
}

func verify(body []byte, signature, timestamp, secret string, now time.Time) error {
	if signature == "" || timestamp == "" {
		return ErrMissingSignature
	}
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return ErrMissingSignature
	}
	drift := now.Sub(time.Unix(ts, 0))
	if drift < 0 {
		drift = -drift
	}
	if drift > ReplayTolerance {
		return ErrTimestampExpired
	}
	if !strings.HasPrefix(signature, signaturePrefix) {
		return ErrSignatureMismatch
	}
	expected := Sign(body, timestamp, secret)
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return ErrSignatureMismatch
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go build ./... && go vet ./... && go test ./tests/wallet/domain/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add wallet/domain tests/wallet/domain
git commit -m "Add wallet domain: payment and webhook aggregates"
```

---

### Task 8: Wallet ports and application

**Files:**
- Create: `wallet/ports/ports.go`
- Create (commands): `wallet/application/commands/payment/process_apple_pay.go`, `wallet/application/commands/payment/process_google_pay.go`
- Create (queries): `wallet/application/queries/payment/get_payment.go`, `wallet/application/queries/payment/lookup_by_order_id.go`, `wallet/application/queries/payment/await_payment.go`, `wallet/application/queries/payment/await_url.go`
- Create (facade, unchanged public shape): `wallet/application/payments.go`
- Test: `tests/wallet/application/payments_test.go`

**CQRS-lite:** same split as Task 4 — `ProcessApplePay`/`ProcessGooglePay` are Commands (they submit a token and change state at Bonum); `GetPayment`/`LookupByOrderID`/`AwaitPayment`/`AwaitURL` are Queries (all four only read a Wallet Payment's current status, `Await*` just does it with a blocking wait). `AwaitPaymentQuery`/`AwaitURLQuery` carry the un-clamped `Timeout`; `clampAwait` (the same domain rule from the original plan) moves into `wallet/application/queries/payment` and is defined once, used by both. The `payment` package name repeats across `wallet/domain/payment`, `wallet/application/commands/payment` and `wallet/application/queries/payment` — in `payments.go`, which imports all three, alias the two application-layer imports as `paymentcmd` / `paymentqry` and leave the domain import unaliased.

**Interfaces:**
- Consumes: Task 7 domain.
- Produces: `ports.PaymentAPI` as in the spec (unchanged); `application.NewPayments(ports.PaymentAPI) *Payments` with `ProcessApplePay/ProcessGooglePay/GetPayment/LookupByOrderID/AwaitPayment/AwaitURL`, each now delegating to a `*Handler` in `commands/payment` or `queries/payment`.

- [ ] **Step 1: Write the failing test**

`tests/wallet/application/payments_test.go`:
```go
package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/techpartners-asia/bonum-go/wallet/application"
	"github.com/techpartners-asia/bonum-go/wallet/domain"
	"github.com/techpartners-asia/bonum-go/wallet/domain/payment"
)

var ctx = context.Background()

type fakeAPI struct {
	calls   []string
	args    []any
	timeout time.Duration
	err     error
}

func (f *fakeAPI) record(name string, args ...any) { f.calls = append(f.calls, name); f.args = args }

func (f *fakeAPI) ProcessApplePay(_ context.Context, in payment.ProcessApplePayInput) (*payment.ProcessResponse, error) {
	f.record("ProcessApplePay", in)
	return &payment.ProcessResponse{PaymentID: "p1", Status: payment.StatusPending}, f.err
}
func (f *fakeAPI) ProcessGooglePay(_ context.Context, in payment.ProcessGooglePayInput) (*payment.ProcessResponse, error) {
	f.record("ProcessGooglePay", in)
	return &payment.ProcessResponse{PaymentID: "p2", Status: payment.StatusPending}, f.err
}
func (f *fakeAPI) GetPayment(_ context.Context, id string) (*payment.Payment, error) {
	f.record("GetPayment", id)
	return &payment.Payment{PaymentID: id}, f.err
}
func (f *fakeAPI) LookupByOrderID(_ context.Context, orderID string) (*payment.Payment, error) {
	f.record("LookupByOrderID", orderID)
	return &payment.Payment{OrderID: orderID}, f.err
}
func (f *fakeAPI) AwaitPayment(_ context.Context, id string, timeout time.Duration) (*payment.AwaitResult, error) {
	f.record("AwaitPayment", id)
	f.timeout = timeout
	return &payment.AwaitResult{PaymentID: id, Status: payment.StatusAuthorized}, f.err
}
func (f *fakeAPI) AwaitURL(_ context.Context, url string, timeout time.Duration) (*payment.AwaitResult, error) {
	f.record("AwaitURL", url)
	f.timeout = timeout
	return &payment.AwaitResult{Status: payment.StatusAuthorized}, f.err
}

func appleToken() payment.ApplePayToken {
	return payment.ApplePayToken{PaymentData: payment.ApplePaymentData{Data: "d"}, TransactionIdentifier: "t"}
}

func TestProcessValidatesBeforePort(t *testing.T) {
	f := &fakeAPI{}
	s := application.NewPayments(f)
	if _, err := s.ProcessApplePay(ctx, payment.ProcessApplePayInput{}); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("apple: %v", err)
	}
	if _, err := s.ProcessGooglePay(ctx, payment.ProcessGooglePayInput{OrderID: "o"}); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("google: %v", err)
	}
	if len(f.calls) != 0 {
		t.Fatal("invalid input must not reach the port")
	}
	in := payment.ProcessApplePayInput{OrderID: "o", Token: appleToken()}
	if res, err := s.ProcessApplePay(ctx, in); err != nil || res.PaymentID != "p1" || f.args[0] != in {
		t.Fatalf("forwarding: %v %v", res, err)
	}
	g := payment.ProcessGooglePayInput{OrderID: "o", Token: "t", CurrencyCode: payment.USD}
	if res, err := s.ProcessGooglePay(ctx, g); err != nil || res.PaymentID != "p2" || f.args[0] != g {
		t.Fatalf("forwarding: %v %v", res, err)
	}
}

func TestIDsAreRequired(t *testing.T) {
	f := &fakeAPI{}
	s := application.NewPayments(f)
	if _, err := s.GetPayment(ctx, ""); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatal(err)
	}
	if _, err := s.LookupByOrderID(ctx, ""); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatal(err)
	}
	if _, err := s.AwaitPayment(ctx, "", time.Second); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatal(err)
	}
	if _, err := s.AwaitURL(ctx, "", time.Second); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatal(err)
	}
	if len(f.calls) != 0 {
		t.Fatal("empty ids must not reach the port")
	}
	if p, err := s.GetPayment(ctx, "p1"); err != nil || p.PaymentID != "p1" {
		t.Fatal(err)
	}
	if p, err := s.LookupByOrderID(ctx, "o1"); err != nil || p.OrderID != "o1" {
		t.Fatal(err)
	}
}

func TestAwaitTimeoutIsClamped(t *testing.T) {
	f := &fakeAPI{}
	s := application.NewPayments(f)
	cases := map[time.Duration]time.Duration{
		0:                time.Duration(0),
		-time.Second:     time.Duration(0),
		2 * time.Second:  2 * time.Second,
		time.Minute:      payment.MaxAwaitTimeout,
		28 * time.Second: payment.MaxAwaitTimeout,
	}
	for in, want := range cases {
		if _, err := s.AwaitPayment(ctx, "p1", in); err != nil || f.timeout != want {
			t.Fatalf("AwaitPayment(%v): timeout %v, want %v (err %v)", in, f.timeout, want, err)
		}
		if _, err := s.AwaitURL(ctx, "https://x/await", in); err != nil || f.timeout != want {
			t.Fatalf("AwaitURL(%v): timeout %v, want %v (err %v)", in, f.timeout, want, err)
		}
	}
}

func TestPortErrorPassesThrough(t *testing.T) {
	want := errors.New("port failed")
	s := application.NewPayments(&fakeAPI{err: want})
	if _, err := s.GetPayment(ctx, "p1"); err != want {
		t.Fatalf("got %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./tests/wallet/application/`
Expected: FAIL, package not found.

- [ ] **Step 3: Write ports and application**

`wallet/ports/ports.go`:
```go
// Package ports declares the outbound capability the Wallet application layer depends on,
// implemented by adapters/httpapi.
package ports

import (
	"context"
	"time"

	"github.com/techpartners-asia/bonum-go/wallet/domain/payment"
)

// PaymentAPI submits wallet tokens and reads Wallet Payments. Inputs arrive validated and
// timeouts arrive clamped; 0 means Bonum's server-side default.
type PaymentAPI interface {
	ProcessApplePay(ctx context.Context, in payment.ProcessApplePayInput) (*payment.ProcessResponse, error)
	ProcessGooglePay(ctx context.Context, in payment.ProcessGooglePayInput) (*payment.ProcessResponse, error)
	GetPayment(ctx context.Context, paymentID string) (*payment.Payment, error)
	LookupByOrderID(ctx context.Context, orderID string) (*payment.Payment, error)
	AwaitPayment(ctx context.Context, paymentID string, timeout time.Duration) (*payment.AwaitResult, error)
	AwaitURL(ctx context.Context, awaitURL string, timeout time.Duration) (*payment.AwaitResult, error)
}
```

`wallet/application/commands/payment/process_apple_pay.go`:
```go
// Package payment holds the Wallet Payment aggregate's write use cases: submitting a wallet
// token.
package payment

import (
	"context"

	"github.com/techpartners-asia/bonum-go/wallet/domain/payment"
	"github.com/techpartners-asia/bonum-go/wallet/ports"
)

// ProcessApplePayCommand submits an Apple Pay token.
type ProcessApplePayCommand = payment.ProcessApplePayInput

// ProcessApplePayHandler submits an Apple Pay token. The response is always PENDING; call
// AwaitPayment (or AwaitURL) to learn the outcome in time to close the wallet sheet.
type ProcessApplePayHandler struct{ api ports.PaymentAPI }

func NewProcessApplePayHandler(api ports.PaymentAPI) *ProcessApplePayHandler {
	return &ProcessApplePayHandler{api: api}
}

func (h *ProcessApplePayHandler) Handle(ctx context.Context, cmd ProcessApplePayCommand) (*payment.ProcessResponse, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}
	return h.api.ProcessApplePay(ctx, cmd)
}
```

`wallet/application/commands/payment/process_google_pay.go`:
```go
package payment

import (
	"context"

	"github.com/techpartners-asia/bonum-go/wallet/domain/payment"
	"github.com/techpartners-asia/bonum-go/wallet/ports"
)

// ProcessGooglePayCommand submits a Google Pay token string.
type ProcessGooglePayCommand = payment.ProcessGooglePayInput

// ProcessGooglePayHandler submits a Google Pay token string. See ProcessApplePayHandler for
// the result model.
type ProcessGooglePayHandler struct{ api ports.PaymentAPI }

func NewProcessGooglePayHandler(api ports.PaymentAPI) *ProcessGooglePayHandler {
	return &ProcessGooglePayHandler{api: api}
}

func (h *ProcessGooglePayHandler) Handle(ctx context.Context, cmd ProcessGooglePayCommand) (*payment.ProcessResponse, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}
	return h.api.ProcessGooglePay(ctx, cmd)
}
```

`wallet/application/queries/payment/get_payment.go`:
```go
// Package payment holds the Wallet Payment aggregate's read use cases.
package payment

import (
	"context"

	"github.com/techpartners-asia/bonum-go/wallet/domain"
	"github.com/techpartners-asia/bonum-go/wallet/domain/payment"
	"github.com/techpartners-asia/bonum-go/wallet/ports"
)

// GetPaymentQuery returns the current state of a Wallet Payment by Bonum's paymentId.
type GetPaymentQuery struct{ PaymentID string }

type GetPaymentHandler struct{ api ports.PaymentAPI }

func NewGetPaymentHandler(api ports.PaymentAPI) *GetPaymentHandler {
	return &GetPaymentHandler{api: api}
}

func (h *GetPaymentHandler) Handle(ctx context.Context, q GetPaymentQuery) (*payment.Payment, error) {
	if q.PaymentID == "" {
		return nil, domain.Invalid("paymentID", "required")
	}
	return h.api.GetPayment(ctx, q.PaymentID)
}
```

`wallet/application/queries/payment/lookup_by_order_id.go`:
```go
package payment

import (
	"context"

	"github.com/techpartners-asia/bonum-go/wallet/domain"
	"github.com/techpartners-asia/bonum-go/wallet/domain/payment"
	"github.com/techpartners-asia/bonum-go/wallet/ports"
)

// LookupByOrderIDQuery returns the Wallet Payment submitted with your Order ID.
type LookupByOrderIDQuery struct{ OrderID string }

type LookupByOrderIDHandler struct{ api ports.PaymentAPI }

func NewLookupByOrderIDHandler(api ports.PaymentAPI) *LookupByOrderIDHandler {
	return &LookupByOrderIDHandler{api: api}
}

func (h *LookupByOrderIDHandler) Handle(ctx context.Context, q LookupByOrderIDQuery) (*payment.Payment, error) {
	if q.OrderID == "" {
		return nil, domain.Invalid("orderID", "required")
	}
	return h.api.LookupByOrderID(ctx, q.OrderID)
}
```

`wallet/application/queries/payment/await_payment.go`:
```go
package payment

import (
	"context"
	"time"

	"github.com/techpartners-asia/bonum-go/wallet/domain"
	"github.com/techpartners-asia/bonum-go/wallet/domain/payment"
	"github.com/techpartners-asia/bonum-go/wallet/ports"
)

// AwaitPaymentQuery blocks until the payment reaches AUTHORIZED or FAILED, or Timeout
// elapses. Timeout 0 uses Bonum's default (25s); anything above payment.MaxAwaitTimeout is
// capped at 28s.
type AwaitPaymentQuery struct {
	PaymentID string
	Timeout   time.Duration
}

type AwaitPaymentHandler struct{ api ports.PaymentAPI }

func NewAwaitPaymentHandler(api ports.PaymentAPI) *AwaitPaymentHandler {
	return &AwaitPaymentHandler{api: api}
}

func (h *AwaitPaymentHandler) Handle(ctx context.Context, q AwaitPaymentQuery) (*payment.AwaitResult, error) {
	if q.PaymentID == "" {
		return nil, domain.Invalid("paymentID", "required")
	}
	return h.api.AwaitPayment(ctx, q.PaymentID, clampAwait(q.Timeout))
}

// clampAwait applies the domain rule: non-positive means server default, never above the cap.
// Defined once here; await_url.go in this same package reuses it.
func clampAwait(d time.Duration) time.Duration {
	switch {
	case d <= 0:
		return 0
	case d > payment.MaxAwaitTimeout:
		return payment.MaxAwaitTimeout
	}
	return d
}
```

`wallet/application/queries/payment/await_url.go`:
```go
package payment

import (
	"context"
	"time"

	"github.com/techpartners-asia/bonum-go/wallet/domain"
	"github.com/techpartners-asia/bonum-go/wallet/domain/payment"
	"github.com/techpartners-asia/bonum-go/wallet/ports"
)

// AwaitURLQuery is AwaitPaymentQuery for the absolute awaitUrl returned by
// ProcessApplePay / ProcessGooglePay.
type AwaitURLQuery struct {
	AwaitURL string
	Timeout  time.Duration
}

type AwaitURLHandler struct{ api ports.PaymentAPI }

func NewAwaitURLHandler(api ports.PaymentAPI) *AwaitURLHandler { return &AwaitURLHandler{api: api} }

func (h *AwaitURLHandler) Handle(ctx context.Context, q AwaitURLQuery) (*payment.AwaitResult, error) {
	if q.AwaitURL == "" {
		return nil, domain.Invalid("awaitURL", "required")
	}
	return h.api.AwaitURL(ctx, q.AwaitURL, clampAwait(q.Timeout))
}
```

`wallet/application/payments.go`:
```go
// Package application holds the Wallet facade: one struct exposing the same public methods
// as before, delegating to Command/Query Handlers in application/commands/payment and
// application/queries/payment.
package application

import (
	"context"
	"time"

	paymentcmd "github.com/techpartners-asia/bonum-go/wallet/application/commands/payment"
	paymentqry "github.com/techpartners-asia/bonum-go/wallet/application/queries/payment"
	"github.com/techpartners-asia/bonum-go/wallet/domain/payment"
	"github.com/techpartners-asia/bonum-go/wallet/ports"
)

// Payments is the Wallet Payment aggregate's use cases.
type Payments struct {
	processApplePay  *paymentcmd.ProcessApplePayHandler
	processGooglePay *paymentcmd.ProcessGooglePayHandler
	getPayment       *paymentqry.GetPaymentHandler
	lookupByOrderID  *paymentqry.LookupByOrderIDHandler
	awaitPayment     *paymentqry.AwaitPaymentHandler
	awaitURL         *paymentqry.AwaitURLHandler
}

func NewPayments(api ports.PaymentAPI) *Payments {
	return &Payments{
		processApplePay:  paymentcmd.NewProcessApplePayHandler(api),
		processGooglePay: paymentcmd.NewProcessGooglePayHandler(api),
		getPayment:       paymentqry.NewGetPaymentHandler(api),
		lookupByOrderID:  paymentqry.NewLookupByOrderIDHandler(api),
		awaitPayment:     paymentqry.NewAwaitPaymentHandler(api),
		awaitURL:         paymentqry.NewAwaitURLHandler(api),
	}
}

// ProcessApplePay submits an Apple Pay token. The response is always PENDING; call
// AwaitPayment (or AwaitURL) to learn the outcome in time to close the wallet sheet.
func (s *Payments) ProcessApplePay(ctx context.Context, in payment.ProcessApplePayInput) (*payment.ProcessResponse, error) {
	return s.processApplePay.Handle(ctx, in)
}

// ProcessGooglePay submits a Google Pay token string. See ProcessApplePay for the result model.
func (s *Payments) ProcessGooglePay(ctx context.Context, in payment.ProcessGooglePayInput) (*payment.ProcessResponse, error) {
	return s.processGooglePay.Handle(ctx, in)
}

// GetPayment returns the current state of a Wallet Payment by Bonum's paymentId.
func (s *Payments) GetPayment(ctx context.Context, paymentID string) (*payment.Payment, error) {
	return s.getPayment.Handle(ctx, paymentqry.GetPaymentQuery{PaymentID: paymentID})
}

// LookupByOrderID returns the Wallet Payment submitted with your Order ID.
func (s *Payments) LookupByOrderID(ctx context.Context, orderID string) (*payment.Payment, error) {
	return s.lookupByOrderID.Handle(ctx, paymentqry.LookupByOrderIDQuery{OrderID: orderID})
}

// AwaitPayment blocks until the payment reaches AUTHORIZED or FAILED, or timeout elapses.
// timeout 0 uses Bonum's default (25s); anything above MaxAwaitTimeout is capped at 28s.
// On TimedOut the payment is still processing and the outcome arrives via the webhook.
func (s *Payments) AwaitPayment(ctx context.Context, paymentID string, timeout time.Duration) (*payment.AwaitResult, error) {
	return s.awaitPayment.Handle(ctx, paymentqry.AwaitPaymentQuery{PaymentID: paymentID, Timeout: timeout})
}

// AwaitURL is AwaitPayment for the absolute awaitUrl returned by ProcessApplePay / ProcessGooglePay.
func (s *Payments) AwaitURL(ctx context.Context, awaitURL string, timeout time.Duration) (*payment.AwaitResult, error) {
	return s.awaitURL.Handle(ctx, paymentqry.AwaitURLQuery{AwaitURL: awaitURL, Timeout: timeout})
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go build ./... && go vet ./... && go test ./tests/wallet/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add wallet/ports wallet/application tests/wallet/application
git commit -m "Add wallet ports and Payments application service"
```

---

### Task 9: Wallet adapter and facade switch

**Files:**
- Create: `wallet/adapters/httpapi/client.go`, `wallet/adapters/httpapi/errors.go`, `wallet/wallet.go`, `wallet/types.go`
- Delete: `wallet/client.go`, `wallet/payment.go`, `wallet/errors.go`, `wallet/types.go` (old), `wallet/webhook.go`
- Test: existing `tests/wallet/*` unmodified.

**Interfaces:**
- Produces: `httpapi.New(baseURL, merchantKey string) *Client` with `SetBaseURL/SetTimeout/SetTransport/Close`, satisfying `ports.PaymentAPI`; the unchanged public API of package `wallet`.

- [ ] **Step 1: Baseline**

Run: `go test ./tests/wallet/`
Expected: PASS against the old flat code.

- [ ] **Step 2: Write the adapter**

`wallet/adapters/httpapi/client.go`:
```go
// Package httpapi is the Wallet's outbound HTTP adapter: it implements wallet/ports against
// Bonum's V2 endpoints and owns the merchant-key header, timeouts and error decoding.
package httpapi

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/techpartners-asia/bonum-go/internal/rest"
	"github.com/techpartners-asia/bonum-go/wallet/domain/payment"
	"github.com/techpartners-asia/bonum-go/wallet/ports"
	"resty.dev/v3"
)

var _ ports.PaymentAPI = (*Client)(nil)

const (
	processApplePath  = "/api/v2/payment/process"
	processGooglePath = "/api/v2/payment/process/google"
	paymentsPath      = "/api/v2/payments"

	// MerchantKeyHeader authenticates every request.
	MerchantKeyHeader = "x-merchant-key"

	// The await endpoint blocks up to 28s; leave room for network latency on top.
	defaultTimeout = 35 * time.Second
)

// Client talks to the Bonum PSP V2 API on behalf of one merchant. It is safe for concurrent use.
type Client struct {
	rest        *rest.Client
	merchantKey string
}

// New creates an adapter for baseURL authenticated with the Merchant Key.
func New(baseURL, merchantKey string) *Client {
	return &Client{rest: rest.New(baseURL, defaultTimeout, decodeError), merchantKey: merchantKey}
}

func (c *Client) SetBaseURL(u string)               { c.rest.SetBaseURL(u) }
func (c *Client) SetTimeout(d time.Duration)        { c.rest.SetTimeout(d) }
func (c *Client) SetTransport(rt http.RoundTripper) { c.rest.SetTransport(rt) }
func (c *Client) Close() error                      { return c.rest.Close() }

func (c *Client) ProcessApplePay(ctx context.Context, in payment.ProcessApplePayInput) (*payment.ProcessResponse, error) {
	return call[payment.ProcessResponse](ctx, c, http.MethodPost, processApplePath, body(in))
}

func (c *Client) ProcessGooglePay(ctx context.Context, in payment.ProcessGooglePayInput) (*payment.ProcessResponse, error) {
	return call[payment.ProcessResponse](ctx, c, http.MethodPost, processGooglePath, body(in))
}

func (c *Client) GetPayment(ctx context.Context, paymentID string) (*payment.Payment, error) {
	return call[payment.Payment](ctx, c, http.MethodGet, paymentsPath+"/{id}", pathParam("id", paymentID))
}

func (c *Client) LookupByOrderID(ctx context.Context, orderID string) (*payment.Payment, error) {
	return call[payment.Payment](ctx, c, http.MethodGet, paymentsPath+"/lookup/by-order-id", query("orderId", orderID))
}

func (c *Client) AwaitPayment(ctx context.Context, paymentID string, timeout time.Duration) (*payment.AwaitResult, error) {
	return call[payment.AwaitResult](ctx, c, http.MethodGet, paymentsPath+"/{id}/await", pathParam("id", paymentID), awaitTimeout(timeout))
}

func (c *Client) AwaitURL(ctx context.Context, awaitURL string, timeout time.Duration) (*payment.AwaitResult, error) {
	return call[payment.AwaitResult](ctx, c, http.MethodGet, awaitURL, awaitTimeout(timeout))
}

type reqOpt func(*resty.Request)

func body(v any) reqOpt            { return func(r *resty.Request) { r.SetBody(v) } }
func query(k, v string) reqOpt     { return func(r *resty.Request) { r.SetQueryParam(k, v) } }
func pathParam(k, v string) reqOpt { return func(r *resty.Request) { r.SetPathParam(k, v) } }

// awaitTimeout encodes an already-clamped timeout; 0 omits the parameter so Bonum applies its default.
func awaitTimeout(d time.Duration) reqOpt {
	if d <= 0 {
		return func(*resty.Request) {}
	}
	return query("timeoutMs", strconv.FormatInt(d.Milliseconds(), 10))
}

// call is the single path every V2 endpoint goes through: merchant key, JSON headers,
// endpoint options, execute against a path or absolute URL, decode into T.
func call[T any](ctx context.Context, c *Client, method, path string, opts ...reqOpt) (*T, error) {
	req := c.rest.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetHeader(MerchantKeyHeader, c.merchantKey)
	for _, opt := range opts {
		opt(req)
	}
	var out T
	if err := c.rest.Do(req, method, path, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
```

`wallet/adapters/httpapi/errors.go`:
```go
package httpapi

import (
	"encoding/json"

	"github.com/techpartners-asia/bonum-go/wallet/domain"
)

type errorBody struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
	Error      string `json:"error"`
}

// decodeError turns a non-2xx response into *domain.APIError.
func decodeError(status int, body string) error {
	e := &domain.APIError{StatusCode: status, Body: body}
	var eb errorBody
	if json.Unmarshal([]byte(body), &eb) == nil {
		e.Message = eb.Message
		e.ErrorText = eb.Error
	}
	return e
}
```

- [ ] **Step 3: Rewrite the facade and delete the flat files**

```bash
git rm -q wallet/client.go wallet/payment.go wallet/errors.go wallet/types.go wallet/webhook.go
```

`wallet/wallet.go`:
```go
// Package wallet wraps the Bonum PSP V2 API for Apple Pay and Google Pay.
//
// The V2 API is separate from the gateway API wrapped by the root bonum package:
// different hosts, an x-merchant-key credential instead of AppSecret/terminal,
// and an asynchronous result model. Your mobile app or web page collects the
// encrypted wallet token; this package is for the backend that forwards it.
//
// This package is a facade: it composes wallet/adapters/httpapi into the use cases in
// wallet/application and re-exports the domain types so callers import only wallet.
package wallet

import (
	"context"
	"net/http"
	"time"

	"github.com/techpartners-asia/bonum-go/wallet/adapters/httpapi"
	"github.com/techpartners-asia/bonum-go/wallet/application"
)

// Environment selects which Bonum PSP host the client talks to.
type Environment string

const (
	Sandbox    Environment = "https://testpsp.bonum.mn"
	Production Environment = "https://psp.bonum.mn"
)

// MerchantKeyHeader authenticates every request.
const MerchantKeyHeader = httpapi.MerchantKeyHeader

// Client talks to the Bonum PSP V2 API on behalf of one merchant. It is safe for concurrent use.
type Client struct {
	payments *application.Payments
	api      *httpapi.Client
}

type Option func(*Client)

// WithBaseURL overrides the host derived from the Environment, e.g. to go through a proxy.
func WithBaseURL(baseURL string) Option { return func(c *Client) { c.api.SetBaseURL(baseURL) } }

// WithTimeout sets the per-request HTTP timeout (default 35s, which covers the 28s await cap).
func WithTimeout(d time.Duration) Option { return func(c *Client) { c.api.SetTimeout(d) } }

// WithTransport replaces the HTTP transport, e.g. to record outbound calls.
func WithTransport(rt http.RoundTripper) Option { return func(c *Client) { c.api.SetTransport(rt) } }

// New creates a Client. merchantKey is the Merchant Key issued by Bonum at onboarding.
func New(env Environment, merchantKey string, opts ...Option) *Client {
	api := httpapi.New(string(env), merchantKey)
	c := &Client{api: api}
	for _, opt := range opts {
		opt(c)
	}
	c.payments = application.NewPayments(api)
	return c
}

// Close releases the underlying HTTP resources.
func (c *Client) Close() error { return c.api.Close() }

// ProcessApplePay submits an Apple Pay token. The response is always PENDING; call
// AwaitPayment (or AwaitURL) to learn the outcome in time to close the wallet sheet.
func (c *Client) ProcessApplePay(ctx context.Context, in ProcessApplePayInput) (*ProcessResponse, error) {
	return c.payments.ProcessApplePay(ctx, in)
}

// ProcessGooglePay submits a Google Pay token string. See ProcessApplePay for the result model.
func (c *Client) ProcessGooglePay(ctx context.Context, in ProcessGooglePayInput) (*ProcessResponse, error) {
	return c.payments.ProcessGooglePay(ctx, in)
}

// GetPayment returns the current state of a Wallet Payment by Bonum's paymentId.
func (c *Client) GetPayment(ctx context.Context, paymentID string) (*Payment, error) {
	return c.payments.GetPayment(ctx, paymentID)
}

// LookupByOrderID returns the Wallet Payment submitted with your Order ID.
func (c *Client) LookupByOrderID(ctx context.Context, orderID string) (*Payment, error) {
	return c.payments.LookupByOrderID(ctx, orderID)
}

// AwaitPayment blocks until the payment reaches AUTHORIZED or FAILED, or timeout elapses.
// timeout 0 uses Bonum's default (25s); anything above MaxAwaitTimeout is capped at 28s.
// On TimedOut the payment is still processing and the outcome arrives via the webhook.
func (c *Client) AwaitPayment(ctx context.Context, paymentID string, timeout time.Duration) (*AwaitResult, error) {
	return c.payments.AwaitPayment(ctx, paymentID, timeout)
}

// AwaitURL is AwaitPayment for the absolute awaitUrl returned by ProcessApplePay / ProcessGooglePay.
func (c *Client) AwaitURL(ctx context.Context, awaitURL string, timeout time.Duration) (*AwaitResult, error) {
	return c.payments.AwaitURL(ctx, awaitURL, timeout)
}
```

`wallet/types.go`:
```go
package wallet

import (
	"github.com/techpartners-asia/bonum-go/wallet/domain"
	"github.com/techpartners-asia/bonum-go/wallet/domain/payment"
	"github.com/techpartners-asia/bonum-go/wallet/domain/webhook"
)

// Payment aggregate (wallet/domain/payment).
type (
	Status                = payment.Status
	WalletType            = payment.WalletType
	Currency              = payment.Currency
	BinCategory           = payment.BinCategory
	ApplePaymentHeader    = payment.ApplePaymentHeader
	ApplePaymentData      = payment.ApplePaymentData
	ApplePaymentMethod    = payment.ApplePaymentMethod
	ApplePayToken         = payment.ApplePayToken
	ProcessApplePayInput  = payment.ProcessApplePayInput
	ProcessGooglePayInput = payment.ProcessGooglePayInput
	ProcessResponse       = payment.ProcessResponse
	Payment               = payment.Payment
	AwaitResult           = payment.AwaitResult
)

const (
	StatusPending    = payment.StatusPending
	StatusAuthorized = payment.StatusAuthorized
	StatusFailed     = payment.StatusFailed

	ApplePay  = payment.ApplePay
	GooglePay = payment.GooglePay

	MNT = payment.MNT
	USD = payment.USD
	EUR = payment.EUR
	JPY = payment.JPY

	Domestic      = payment.Domestic
	International = payment.International

	// MaxAwaitTimeout is the server-side cap on the await window.
	MaxAwaitTimeout = payment.MaxAwaitTimeout
)

// Errors (wallet/domain, wallet/domain/webhook).
var (
	ErrInvalidInput      = domain.ErrInvalidInput
	ErrUnauthorized      = domain.ErrUnauthorized
	ErrNotFound          = domain.ErrNotFound
	ErrRateLimited       = domain.ErrRateLimited
	ErrMissingSignature  = webhook.ErrMissingSignature
	ErrTimestampExpired  = webhook.ErrTimestampExpired
	ErrSignatureMismatch = webhook.ErrSignatureMismatch
)

type (
	// APIError is returned when Bonum answers with a non-2xx status.
	APIError = domain.APIError
	// ValidationError is returned before any network call when an input violates an invariant.
	ValidationError = domain.ValidationError
)

// Webhook aggregate (wallet/domain/webhook).
type WebhookEvent = webhook.Event

const (
	SignatureHeader = webhook.SignatureHeader
	TimestampHeader = webhook.TimestampHeader
	ReplayTolerance = webhook.ReplayTolerance
)

// Sign computes "v1=" + hex(HMAC-SHA256(secret, timestamp + "." + body)) exactly as Bonum does.
func Sign(body []byte, timestamp, secret string) string { return webhook.Sign(body, timestamp, secret) }

// ParseWebhook verifies a delivery against the X-PSP-Signature and X-PSP-Timestamp header
// values and decodes it. Pass the body bytes exactly as received; re-serialising changes the
// hash. Rejections are ErrMissingSignature, ErrTimestampExpired or ErrSignatureMismatch.
func ParseWebhook(body []byte, signature, timestamp, secret string) (*WebhookEvent, error) {
	return webhook.Parse(body, signature, timestamp, secret)
}
```

- [ ] **Step 4: Run the full suite**

Run: `go build ./... && go vet ./... && go test ./...`
Expected: all PASS, including the unmodified `tests/wallet` suite.

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "Add wallet HTTP adapter and switch the wallet facade onto the layers"
```

---

### Task 10: Architecture test enforcing import direction

**Files:**
- Test: `tests/architecture/deps_test.go`

**Interfaces:** none produced. Consumes the on-disk layout of Tasks 1–9.

- [ ] **Step 1: Write the test**

```go
// Package architecture_test enforces the dependency rule from
// docs/superpowers/specs/2026-09-14-ddd-layered-contexts-design.md: inside each bounded
// context, imports point inward (adapters -> ports/domain, application -> ports/domain,
// ports -> domain, domain -> nothing), contexts never import each other, and only adapters
// and facades may reach third-party code or internal/rest.
package architecture_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const module = "github.com/techpartners-asia/bonum-go"

var contexts = []string{"gateway", "wallet"}

// classify maps a directory (module-relative, slash separated) to its context and layer.
// It returns ok=false for directories the rule does not cover (tests, internal).
func classify(dir string) (ctx, layer string, ok bool) {
	if dir == "." {
		return "gateway", "facade", true
	}
	parts := strings.Split(dir, "/")
	for _, c := range contexts {
		if parts[0] != c {
			continue
		}
		if len(parts) == 1 {
			return c, "facade", true
		}
		return c, parts[1], true
	}
	return "", "", false
}

func isStdlib(imp string) bool {
	first, _, _ := strings.Cut(imp, "/")
	return !strings.Contains(first, ".")
}

// allowed reports whether a file in (ctx, layer) may import imp.
func allowed(ctx, layer, imp string) bool {
	if isStdlib(imp) {
		return true
	}
	rel, internal := strings.CutPrefix(imp, module+"/")
	if !internal {
		return layer == "adapters" || layer == "facade"
	}
	for _, other := range contexts {
		if other != ctx && (rel == other || strings.HasPrefix(rel, other+"/")) {
			return false
		}
	}
	domain := strings.HasPrefix(rel, ctx+"/domain")
	ports := rel == ctx+"/ports"
	application := strings.HasPrefix(rel, ctx+"/application/")
	switch layer {
	case "domain":
		return domain
	case "ports":
		return domain
	case "application":
		// application also covers its own commands/<aggregate> and queries/<aggregate>
		// subpackages (CQRS-lite): a facade file imports those, and each of those
		// imports only domain + ports, same as any other application-layer file.
		return domain || ports || application
	case "adapters":
		return domain || ports || rel == "internal/rest"
	case "facade":
		return strings.HasPrefix(rel, ctx+"/") || rel == "internal/rest"
	}
	return false
}

func TestImportsPointInward(t *testing.T) {
	root := filepath.Join("..", "..")
	var violations []string
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if name := d.Name(); name == ".git" || name == "tests" || name == "docs" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		dir := filepath.ToSlash(filepath.Dir(rel))
		ctx, layer, ok := classify(dir)
		if !ok {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range f.Imports {
			p := strings.Trim(imp.Path.Value, `"`)
			if !allowed(ctx, layer, p) {
				violations = append(violations, rel+": "+layer+" layer may not import "+p)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) > 0 {
		t.Fatalf("dependency rule violated:\n  %s", strings.Join(violations, "\n  "))
	}
}

func TestRuleCatchesViolations(t *testing.T) {
	bad := []struct{ ctx, layer, imp string }{
		{"gateway", "domain", module + "/internal/rest"},
		{"gateway", "domain", "resty.dev/v3"},
		{"gateway", "application", module + "/gateway/adapters/httpapi"},
		{"gateway", "ports", module + "/gateway/application"},
		{"gateway", "adapters", module + "/wallet/domain"},
		{"wallet", "facade", module + "/gateway/domain"},
		{"gateway", "application", module + "/wallet/application/commands/payment"},
	}
	for _, b := range bad {
		if allowed(b.ctx, b.layer, b.imp) {
			t.Errorf("%s/%s importing %s must be rejected", b.ctx, b.layer, b.imp)
		}
	}
	good := []struct{ ctx, layer, imp string }{
		{"gateway", "domain", "encoding/json"},
		{"gateway", "domain", module + "/gateway/domain/checkout"},
		{"gateway", "application", module + "/gateway/ports"},
		{"gateway", "application", module + "/gateway/application/commands/card"},
		{"wallet", "application", module + "/wallet/application/queries/payment"},
		{"gateway", "adapters", "resty.dev/v3"},
		{"wallet", "facade", module + "/wallet/application"},
	}
	for _, g := range good {
		if !allowed(g.ctx, g.layer, g.imp) {
			t.Errorf("%s/%s importing %s must be allowed", g.ctx, g.layer, g.imp)
		}
	}
}
```

- [ ] **Step 2: Run it**

Run: `go test ./tests/architecture/ -v`
Expected: both tests PASS. If `TestImportsPointInward` lists a violation, the offending file from Tasks 1–9 is wrong; fix the source, not the rule.

- [ ] **Step 3: Commit**

```bash
git add tests/architecture
git commit -m "Add architecture test enforcing the dependency rule"
```

---

### Task 11: Documentation

**Files:**
- Move: `CONTEXT.md` → `gateway/CONTEXT.md`
- Create: `docs/adr/0005-layered-contexts.md`
- Modify: `docs/adr/0002-wire-types-are-the-domain-types.md`, `docs/adr/0004-tests-cross-the-public-seam.md`, `CONTEXT-MAP.md`, `README.md` (the Layout section and the sentence in the intro that points at CONTEXT-MAP)

- [ ] **Step 1: Move the gateway glossary and fix links**

```bash
git mv CONTEXT.md gateway/CONTEXT.md
```

In `CONTEXT-MAP.md` replace `[Gateway](./CONTEXT.md)` with `[Gateway](./gateway/CONTEXT.md)` and append to the Relationships section:

```markdown
- **Inside each context**: `domain` (aggregates, invariants, errors) ← `ports` (interfaces
  the use cases need) ← `application` (one service per aggregate) ← `adapters/httpapi`
  (Bonum's HTTP endpoints). The root `bonum` package and the `wallet` package are facades
  that compose these and re-export the domain types. `tests/architecture` enforces the
  direction.
```

- [ ] **Step 2: Write ADR 0005**

`docs/adr/0005-layered-contexts.md`:
```markdown
# Each context is layered domain / ports / application / adapters

The gateway and wallet contexts were flat packages: aggregate types, validation, HTTP calls
and error decoding in one file per aggregate. That was compact but had three costs. A
merchant could not test code that uses the SDK without an HTTP fake, because the only seam
was resty. Domain rules (the await cap, cycle values, what a decline is) were tested only
through HTTP. And nothing stopped transport details from leaking into types callers hold.

Each context is now four packages with imports pointing inward: `domain/<aggregate>` holds
the types, `Validate` methods and domain errors; `ports` holds the interfaces the use cases
call; `application` implements those use cases CQRS-lite, one Command or Query type plus a
`*Handler` per use case under `application/commands/<aggregate>` or
`application/queries/<aggregate>`, fronted by one facade struct per aggregate that keeps the
pre-existing method names; `adapters/httpapi` implements every port with resty against
Bonum's endpoints. The root `bonum` package and the `wallet` package are facades: they
compose an adapter into the aggregate facades and re-export the domain types as aliases, so
the public API did not change and a merchant still imports one package. `tests/architecture`
fails the build if an import goes the wrong way.

The CQRS split was added after Tasks 1-3 (domain-only, unaffected) had already landed: the
domain and ports layers do not care whether the application layer above them is one service
per aggregate or one handler per use case, so widening the application layer's shape cost
nothing outside `application/`.

We deliberately left out the rest of the usual DDD toolkit. There is no generic
command bus or mediator (the aggregate facade dispatches directly, so there is nothing a
bus would decouple here), no domain events (nothing consumes them), no repositories or unit
of work (an SDK has no persistence), and no separate DTO layer (ADR 0002). Each of those
would be a seam with no consumer.
```

- [ ] **Step 3: Amend ADR 0002 and ADR 0004**

Append to `docs/adr/0002-wire-types-are-the-domain-types.md`:
```markdown

Amended 2026-09-14 (ADR 0005): the wire types now live in `gateway/domain/<aggregate>` and
`wallet/domain/<aggregate>` and still carry their JSON tags. The layered structure did not
introduce a DTO layer; the adapter serialises the domain struct directly.
```

Append to `docs/adr/0004-tests-cross-the-public-seam.md`:
```markdown

Amended 2026-09-14 (ADR 0005): tests now mirror the layers. `tests/<context>/` keeps the
fake-server suites that cross the facade; `tests/<context>/domain/` tests invariants and
parsers with no HTTP; `tests/<context>/application/` tests services against hand-written
fake ports. The wallet webhook gained `ParseAt(..., now)` so the replay window is
deterministic; it is an exported API a merchant replaying stored deliveries also needs, not
a test-only hook.
```

- [ ] **Step 4: Rewrite the README Layout section**

Replace the whole `## Layout` block at the end of `README.md` with:

````markdown
## Layout

Each bounded context is layered, with imports pointing inward (`tests/architecture`
enforces it). Callers only ever import `bonum` and `wallet`; the layers are how the SDK is
built, not how it is used.

```
bonum.go, types.go, errors.go, webhook.go
                       Gateway facade: composes the layers, re-exports every type
gateway/CONTEXT.md     Gateway glossary
gateway/domain/        errors shared by the context, then one package per aggregate:
                       access, checkout, card, subscription, qr, webhook
gateway/ports/         interfaces the use cases call (one per aggregate)
gateway/application/   CQRS-lite: commands/<aggregate> and queries/<aggregate> hold one
                       Command/Query + Handler per use case; one facade struct per
                       aggregate in the package root keeps the old per-aggregate method names
gateway/adapters/httpapi/
                       resty implementation of every port: token lifecycle, envelope,
                       error decoding, decline detection
wallet/wallet.go, wallet/types.go
                       Wallet facade; wallet/CONTEXT.md is its glossary
wallet/domain/, ports/, application/, adapters/httpapi/
                       same layering for Apple Pay / Google Pay
internal/rest/         HTTP execution adapter shared by both contexts
tests/gateway, tests/wallet
                       fake-server suites that cross only the facade
tests/*/domain, tests/*/application
                       invariant and fake-port tests, no HTTP
tests/architecture     import-direction rule
tests/smoke            manual run against the sandbox
docs/adr/              why the API is shaped this way
```
````

In the README intro, change `[CONTEXT-MAP.md](CONTEXT-MAP.md) for the two bounded contexts` to keep the link as is (the map now points at `gateway/CONTEXT.md`); no other README change.

- [ ] **Step 5: Verify and commit**

Run: `go build ./... && go vet ./... && go test ./... && grep -rn "\./CONTEXT.md" --include=*.md . ; git status --short`
Expected: tests PASS; the grep prints nothing (no stale link to the root glossary).

```bash
git add -A
git commit -m "Document the layered contexts: ADR 0005, amended ADRs, README layout"
```

---

## Self-review

- **Spec coverage:** Layout → Tasks 1–9, 11. Dependency rule → Task 10. Domain contracts (Validate, Invalid, DeclinedError, Parse/ParseAt, MaxAwaitTimeout, prefixes) → Tasks 1, 2, 3, 7. Ports → Tasks 4, 8. Application (CQRS-lite: Command/Query + Handler per use case, await clamp, Access) → Tasks 4, 8. Adapters incl. constructors and setters → Tasks 5, 9. Facades with aliases and error vars → Tasks 6, 9. Testing split → Tasks 1–4, 7, 8, 10. Docs → Task 11.
- **Placeholders:** Tasks 3 and 7 say "copy the current file with these changes" for the large webhook/payment type blocks and list every change; the executor has the source file in the repo. No TBDs.
- **Type consistency:** port method names in Task 4 tests, Task 4 ports, Task 5 adapter and the spec all match (`CreateToken/RefreshToken`, `Providers/CreateInvoice`, `Tokenize/Purchase/Reverse`, `Plans/Subscribe/ListSubscriptions/ChangeCardByTokenizing/ChangeCard/Unsubscribe/DeleteSubscription`, `CreateQR/LookupQR/PayQRWithCard`, `InvoiceStatus/MarkInvoicePaid/RunSubscriptionBilling`). Wallet: `ProcessApplePay/ProcessGooglePay/GetPayment/LookupByOrderID/AwaitPayment/AwaitURL` in Task 8 tests, ports, application and Task 9 adapter/facade. The Task 4/8 facade structs (`Invoices`, `Cards`, `Subscriptions`, `QR`, `Sandbox`, `Access`, `Payments`) keep the exact old method names and `NewX(port) *X` constructor signatures, unchanged by the CQRS split inside them — that identity is what keeps `tests/gateway`, `tests/wallet`, and Tasks 6/9 (facade wiring) unchanged.
- **Amendment 1 (2026-09-14, mid-execution):** the user asked for CQRS after Tasks 1–3 were already implemented and reviewed on the `ddd-layers` branch by a concurrently running session. Tasks 4 and 8 (application layer) were rewritten in place to the CQRS-lite Command/Query-per-use-case shape described above; Task 10's `allowed()` gained one clause so `application/commands/**` and `application/queries/**` are recognised as the same "application" layer as their facade; Task 11's ADR 0005 draft and README layout paragraph were reworded to describe the split. Tasks 1, 2, 3, 5, 6, 7, 9 needed no change — ports, adapters, and facade public signatures are identical either way, which is why the change could land without touching or re-reviewing already-completed work.
- **Amendment 2 (2026-09-14, after all ten tasks landed):** the user asked for the facade's own files (`types.go`, `errors.go`, `webhook.go`) to be split per aggregate too, mirroring `domain/<aggregate>` file for file. `types.go` (both `bonum` and `wallet`) is retired; its content moved into one file per aggregate (`access.go`/`checkout.go`/`card.go`/`subscription.go`/`qr.go`/`sandbox.go` for gateway, `payment.go` for wallet) plus a trimmed shared `errors.go` and an expanded `webhook.go` per context. Applied to a build that already had committed code (Tasks 1–10 done), so this landed as a direct edit rather than through the task-implementer/reviewer loop — justified because it is a pure rename/regroup of type aliases with no exported identifier change, verified by `go build`/`go vet`/`go test` (including `tests/architecture`) passing unmodified.
- **Amendment 3 (2026-09-14):** the user asked to separate the public package from everything internal. Each context's `domain`, `ports`, `application` and `adapters` moved under `internal/<context>/` (`internal/gateway/...`, `internal/wallet/...`), joining the pre-existing `internal/rest`; the two facade packages (`bonum` at the module root, `wallet` at `wallet/`) kept their location and import path. This upgrades "callers only import `bonum`/`wallet`" from a `tests/architecture`-enforced convention to a Go compiler guarantee — `internal/` packages cannot be imported from outside the module tree they live under. Only import paths changed (plus `tests/architecture`'s `classify`/`allowed` functions, which needed to resolve the new `internal/` prefix, and two package-doc comments that named the old paths in prose) — every exported identifier, every test's behavior, and both facades' public shape are unchanged. Landed as a direct edit for the same reason as Amendment 2: mechanical, no exported-identifier change, verified green end to end.
