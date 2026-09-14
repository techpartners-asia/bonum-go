package bonum

import (
	"github.com/techpartners-asia/bonum-go/gateway/application"
	"github.com/techpartners-asia/bonum-go/gateway/domain/subscription"
)

// SubscriptionService is the Subscription aggregate's use cases: recurring charges on a
// Card Token.
type SubscriptionService = application.Subscriptions

type RecurringType = subscription.RecurringType

type (
	// PaymentPlan is a recurring billing template managed on the merchant portal.
	PaymentPlan    = subscription.PaymentPlan
	SubscribeInput = subscription.SubscribeInput
	// Subscription is a Card Token enrolled in a Payment Plan.
	Subscription    = subscription.Subscription
	ChangeCardInput = subscription.ChangeCardInput
)

const (
	RecurringWeekly  = subscription.RecurringWeekly
	RecurringMonthly = subscription.RecurringMonthly
	RecurringYearly  = subscription.RecurringYearly
)
