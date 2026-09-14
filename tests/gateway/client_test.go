package gateway_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	bonum "github.com/techpartners-asia/bonum-go"
)

var ctx = context.Background()

func TestTokenIsFetchedOnceAndReused(t *testing.T) {
	f, c := newFakeGateway(t, 1800)
	for range 3 {
		if _, err := c.Invoices.Providers(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if f.createCalls.Load() != 1 || f.refreshCalls.Load() != 0 {
		t.Fatalf("create=%d refresh=%d, want 1/0", f.createCalls.Load(), f.refreshCalls.Load())
	}
	if got := f.lastReq.Load().Header.Get("Accept-Language"); got != "en" {
		t.Fatalf("Accept-Language = %q", got)
	}
}

func TestShortLivedAccessTokenIsRefreshedNotRecreated(t *testing.T) {
	// A 10s TTL is inside the 30s early-refresh skew, so the second call must refresh.
	f, c := newFakeGateway(t, 10)
	for range 2 {
		if _, err := c.Invoices.Providers(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if f.createCalls.Load() != 1 || f.refreshCalls.Load() != 1 {
		t.Fatalf("create=%d refresh=%d, want 1/1", f.createCalls.Load(), f.refreshCalls.Load())
	}
	if got := f.lastReq.Load().Header.Get("Authorization"); got != "Bearer access-2" {
		t.Fatalf("refreshed token not used: %q", got)
	}
}

func TestBadCredentialsSurfaceAsUnauthorized(t *testing.T) {
	f := &fakeGateway{accessTTL: 1800}
	c := bonum.New(bonum.Sandbox, "wrong", "17171119", bonum.WithBaseURL(startServer(t, f)))
	_, err := c.Invoices.Providers(ctx)
	if !errors.Is(err, bonum.ErrUnauthorized) {
		t.Fatalf("want ErrUnauthorized, got %v", err)
	}
	var api *bonum.APIError
	if !errors.As(err, &api) || api.TraceID != "t-401" {
		t.Fatalf("want *APIError with trace, got %#v", err)
	}
}

func TestContextCancellationAbortsCall(t *testing.T) {
	_, c := newFakeGateway(t, 1800)
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := c.Invoices.Providers(cancelled); !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
}

func TestEnvironmentsAndOptions(t *testing.T) {
	if bonum.Sandbox != "https://testapi.bonum.mn" || bonum.Production != "https://apis.bonum.mn" {
		t.Fatalf("hosts: %s %s", bonum.Sandbox, bonum.Production)
	}
	f, c := newFakeGateway(t, 1800)
	if _, err := c.Invoices.Providers(ctx); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(f.lastReq.Load().Header.Get("Authorization"), "Bearer ") {
		t.Fatal("bearer missing")
	}
}
