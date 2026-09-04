package types

// Environment selects which Bonum gateway host the client talks to.
type Environment string

const (
	Sandbox    Environment = "https://testapi.bonum.mn"
	Production Environment = "https://apis.bonum.mn"
)

// Lang is sent as the Accept-Language header so Bonum localizes response messages.
type Lang string

const (
	MN Lang = "mn"
	EN Lang = "en"
)

// Envelope is the standard response wrapper returned by every mpay-service endpoint
// (card tokenization, subscriptions, QR). Data holds the endpoint-specific payload.
type Envelope[T any] struct {
	TraceID   string  `json:"traceId"`   // Bonum-side trace id, useful when contacting support
	ErrorCode *string `json:"errorCode"` // Internal use only per Bonum docs - do not branch on it
	Error     *string `json:"error"`
	Message   string  `json:"message"` // Localized message, may change at any time
	Data      T       `json:"data"`
	Detail    *string `json:"detail"`
	Duration  int64   `json:"duration"` // Server-side processing time in ms
	Status    int     `json:"status"`   // Mirrors the HTTP status code
}

// Item is an optional line item rendered on the Bonum checkout / tokenization page.
type Item struct {
	Image  string  `json:"image,omitempty"`
	Title  string  `json:"title"`
	Remark string  `json:"remark,omitempty"`
	Amount float64 `json:"amount"`
	Count  int     `json:"count"`
}
