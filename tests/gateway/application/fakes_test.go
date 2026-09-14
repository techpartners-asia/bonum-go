package application_test

import (
	"context"
	"encoding/json"

	"github.com/techpartners-asia/bonum-go/internal/gateway/domain/access"
	"github.com/techpartners-asia/bonum-go/internal/gateway/domain/card"
	"github.com/techpartners-asia/bonum-go/internal/gateway/domain/checkout"
	"github.com/techpartners-asia/bonum-go/internal/gateway/domain/qr"
	"github.com/techpartners-asia/bonum-go/internal/gateway/domain/subscription"
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
