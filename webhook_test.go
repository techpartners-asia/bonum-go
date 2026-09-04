package bonum

import (
	"testing"

	"github.com/techpartners-asia/bonum-go/types"
)

const (
	testChecksumKey = "test-checksum-key"
	testPaymentBody = `{"type":"PAYMENT","status":"SUCCESS","message":"","body":{"transactionId":"N998921","invoiceId":"8eff7d69001c03f486f64410f9daa82c"}}`
	// Computed independently with: printf '%s' "$body" | openssl dgst -sha256 -hmac "$key"
	testPaymentChecksum = "bd77c3999d3cca578932fe9f1fd5b8e625885f9190b2490bc12f7610c39927be"
)

func TestChecksumMatchesOpenSSL(t *testing.T) {
	if got := Checksum([]byte(testPaymentBody), testChecksumKey); got != testPaymentChecksum {
		t.Fatalf("Checksum() = %s, want %s", got, testPaymentChecksum)
	}
}

func TestVerifyWebhook(t *testing.T) {
	body := []byte(testPaymentBody)
	if !VerifyWebhook(body, testPaymentChecksum, testChecksumKey) {
		t.Fatal("valid checksum rejected")
	}
	if VerifyWebhook(body, testPaymentChecksum, "wrong-key") {
		t.Fatal("wrong key accepted")
	}
	if VerifyWebhook([]byte(testPaymentBody+" "), testPaymentChecksum, testChecksumKey) {
		t.Fatal("tampered body accepted")
	}
	if VerifyWebhook(body, "", testChecksumKey) {
		t.Fatal("empty header accepted")
	}
}

func TestPeekWebhook(t *testing.T) {
	h, err := PeekWebhook([]byte(testPaymentBody))
	if err != nil {
		t.Fatal(err)
	}
	if h.Type != types.WebhookPayment || h.Status != types.WebhookSuccess {
		t.Fatalf("got type=%s status=%s", h.Type, h.Status)
	}
}

func TestParsePaymentWebhookSuccess(t *testing.T) {
	body := `{
		"type": "PAYMENT", "status": "SUCCESS", "message": "",
		"body": {
			"amount": 10000.00, "currency": "MNT", "completedAt": "2026-01-29 11:20:33",
			"terminalId": "17171994", "invoiceId": "8eff7d69001c03f486f64410f9daa82c",
			"paymentVendor": "QPAY", "initType": "ECOMMERCE", "status": "PAID", "respCode": "",
			"transactionId": "N998921", "extras-inputs": [], "extras": []
		}
	}`
	m, err := ParsePaymentWebhook([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if m.Body.TransactionID != "N998921" || m.Body.Amount != 10000 {
		t.Fatalf("unexpected body %+v", m.Body)
	}
	if m.Body.InvoiceID == nil || *m.Body.InvoiceID != "8eff7d69001c03f486f64410f9daa82c" {
		t.Fatalf("invoiceId not decoded: %+v", m.Body.InvoiceID)
	}
	if m.Body.PaymentVendor == nil || *m.Body.PaymentVendor != types.ProviderQPay {
		t.Fatalf("paymentVendor not decoded: %+v", m.Body.PaymentVendor)
	}
	if m.Body.UpdatedAt != nil || m.Body.InvoiceStatus != nil {
		t.Fatalf("failed-only fields should be nil on success: %+v", m.Body)
	}
}

func TestParsePaymentWebhookFailed(t *testing.T) {
	body := `{
		"type": "PAYMENT", "status": "FAILED", "message": "",
		"body": {
			"transactionId": "B347699", "amount": 15000.00, "currency": "MNT",
			"updatedAt": 1769657291559, "terminalId": "17171994", "invoiceStatus": "EXPIRED"
		}
	}`
	m, err := ParsePaymentWebhook([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if m.Status != types.WebhookFailed {
		t.Fatalf("status = %s", m.Status)
	}
	if m.Body.InvoiceStatus == nil || *m.Body.InvoiceStatus != "EXPIRED" {
		t.Fatalf("invoiceStatus not decoded: %+v", m.Body)
	}
	if m.Body.UpdatedAt == nil || *m.Body.UpdatedAt != 1769657291559 {
		t.Fatalf("updatedAt not decoded: %+v", m.Body)
	}
}

func TestParseCardTokenWebhook(t *testing.T) {
	body := `{
		"type":"CARD-TOKEN","status":"SUCCESS","message":"",
		"body":{
			"token":"tok_123","mask":"5150 23** **** 4778","expiry":"2026/11",
			"bank":{"id":19,"code":"150000","name":"Голомт банк","icon":"","iBanCode":"0015","transferCode":"GMT"},
			"transactionId":"6ab20250512180511006","completedAt":"2026-01-26 12:58:03",
			"amounts":[{"amount":5.00,"currency":"MNT"}],
			"subscriptions":[{"subscriptionId":1,"planId":1,"nextBillingDate":"2026-02-02 00:00:00"}]
		}
	}`
	m, err := ParseCardTokenWebhook([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if m.Body.Token != "tok_123" || m.Body.Bank == nil || m.Body.Bank.TransferCode != "GMT" {
		t.Fatalf("unexpected body %+v", m.Body)
	}
	if len(m.Body.Amounts) != 1 || m.Body.Amounts[0].Amount != 5 {
		t.Fatalf("amounts not decoded: %+v", m.Body.Amounts)
	}
	if len(m.Body.Subscriptions) != 1 || m.Body.Subscriptions[0].SubscriptionID != 1 {
		t.Fatalf("subscriptions not decoded: %+v", m.Body.Subscriptions)
	}
}

func TestParseSubscriptionPaymentWebhook(t *testing.T) {
	body := `{
		"type":"SUBSCRIPTION-PAYMENT","status":"SUCCESS","message":"",
		"body":{"subscriptionId":41,"invoiceId":786,"planId":4,"transactionId":"20000007",
		        "completedAt":"2026-01-27 02:00:08","amount":3,"currency":"MNT"}
	}`
	m, err := ParseSubscriptionPaymentWebhook([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if m.Body.SubscriptionID != 41 || m.Body.InvoiceID != 786 || m.Body.Amount != 3 {
		t.Fatalf("unexpected body %+v", m.Body)
	}
}

func TestParseWebhookRejectsWrongType(t *testing.T) {
	if _, err := ParseCardTokenWebhook([]byte(testPaymentBody)); err == nil {
		t.Fatal("PAYMENT body accepted as CARD-TOKEN")
	}
}
