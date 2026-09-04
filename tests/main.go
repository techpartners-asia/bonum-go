// Manual smoke test against the Bonum sandbox.
//
//	BONUM_APP_SECRET=... BONUM_TERMINAL_ID=... go run ./tests
package main

import (
	"fmt"
	"log"
	"os"

	bonum "github.com/techpartners-asia/bonum-go"
	"github.com/techpartners-asia/bonum-go/types"
)

func main() {
	appSecret := os.Getenv("BONUM_APP_SECRET")
	terminalID := os.Getenv("BONUM_TERMINAL_ID")
	if appSecret == "" || terminalID == "" {
		log.Fatal("set BONUM_APP_SECRET and BONUM_TERMINAL_ID")
	}

	client := bonum.New(types.Sandbox, appSecret, terminalID)
	defer client.Close()

	providers, err := client.GetPaymentProviders()
	if err != nil {
		log.Fatalf("GetPaymentProviders: %v", err)
	}
	fmt.Printf("providers: %+v\n", providers)

	invoice, err := client.CreateInvoice(types.CreateInvoiceInput{
		Amount:        1,
		Callback:      "https://example.com/bonum/callback",
		TransactionID: fmt.Sprintf("smoke-%d", os.Getpid()),
		ExpiresIn:     600,
	})
	if err != nil {
		log.Fatalf("CreateInvoice: %v", err)
	}
	fmt.Printf("invoice: %+v\n", invoice)

	plans, err := client.ListPaymentPlans()
	if err != nil {
		log.Fatalf("ListPaymentPlans: %v", err)
	}
	fmt.Printf("plans: %+v\n", plans.Data)
}
