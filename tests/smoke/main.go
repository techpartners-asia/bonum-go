// Manual smoke test against the Bonum sandbox.
//
//	BONUM_APP_SECRET=... BONUM_TERMINAL_ID=... go run ./tests/smoke
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	bonum "github.com/techpartners-asia/bonum-go"
)

func main() {
	appSecret := os.Getenv("BONUM_APP_SECRET")
	terminalID := os.Getenv("BONUM_TERMINAL_ID")
	if appSecret == "" || terminalID == "" {
		log.Fatal("set BONUM_APP_SECRET and BONUM_TERMINAL_ID")
	}
	ctx := context.Background()

	client := bonum.New(bonum.Sandbox, appSecret, terminalID)
	defer client.Close()

	providers, err := client.Invoices.Providers(ctx)
	if err != nil {
		log.Fatalf("Invoices.Providers: %v", err)
	}
	fmt.Printf("providers: %+v\n", providers)

	invoice, err := client.Invoices.Create(ctx, bonum.CreateInvoiceInput{
		Amount:        1,
		Callback:      "https://example.com/bonum/callback",
		TransactionID: fmt.Sprintf("smoke-%d", os.Getpid()),
		ExpiresIn:     600,
	})
	if err != nil {
		log.Fatalf("Invoices.Create: %v", err)
	}
	fmt.Printf("invoice: %+v\n", invoice)

	plans, err := client.Subscriptions.Plans(ctx)
	if err != nil {
		log.Fatalf("Subscriptions.Plans: %v", err)
	}
	fmt.Printf("plans: %+v\n", plans)
}
