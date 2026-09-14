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
