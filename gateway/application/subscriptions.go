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
