package gateway_test

import (
	"strings"
	"testing"

	bonum "github.com/techpartners-asia/bonum-go"
)

func TestSubscriptionsPlans(t *testing.T) {
	_, c := newFakeGateway(t, 1800)
	plans, err := c.Subscriptions.Plans(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 1 || plans[0].PlanID != 30 || plans[0].RecurringType != bonum.RecurringMonthly {
		t.Fatalf("unexpected %+v", plans)
	}
}

func TestSubscriptionsSubscribeAndList(t *testing.T) {
	f, c := newFakeGateway(t, 1800)
	s, err := c.Subscriptions.Subscribe(ctx, "card-tok", bonum.SubscribeInput{PlanID: 30, CycleValue: 15})
	if err != nil {
		t.Fatal(err)
	}
	if s.SubscriptionID != 42 || s.Plan.PlanID != 30 {
		t.Fatalf("unexpected %+v", s)
	}
	if f.lastReq.Load().Header.Get("X-CARD-TOKEN") != "card-tok" {
		t.Fatal("card token header missing")
	}
	list, err := c.Subscriptions.List(ctx, "card-tok")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].SubscriptionID != 42 {
		t.Fatalf("unexpected %+v", list)
	}
}

func TestSubscriptionsUnsubscribeSendsPlanIDBody(t *testing.T) {
	f, c := newFakeGateway(t, 1800)
	if err := c.Subscriptions.Unsubscribe(ctx, 42, 30); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(f.body()); got != `{"planId":30}` {
		t.Fatalf("DELETE body = %q", got)
	}
	if err := c.Subscriptions.Delete(ctx, 42, 30); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(f.lastReq.Load().URL.Path, "/subscriptions/42/delete") {
		t.Fatalf("path = %s", f.lastReq.Load().URL.Path)
	}
}

func TestSubscriptionsChangeCard(t *testing.T) {
	_, c := newFakeGateway(t, 1800)
	tk, err := c.Subscriptions.ChangeCardByTokenizing(ctx, 7, bonum.ChangeCardInput{Callback: "https://m/cb", TransactionID: "T4"})
	if err != nil {
		t.Fatal(err)
	}
	if tk.ID != "tok-req-1" {
		t.Fatalf("id = %q", tk.ID)
	}
	s, err := c.Subscriptions.ChangeCard(ctx, 7, "card-tok")
	if err != nil {
		t.Fatal(err)
	}
	if s.SubscriptionID != 7 || s.CardMask != "4242" {
		t.Fatalf("unexpected %+v", s)
	}
}
