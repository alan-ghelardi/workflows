package github

// Event represents a Github Webhook event.
type Event struct {
	Body          []byte
	Branch        string
	DeliveryID    string
	EventData     interface{}
	HeadCommitSHA string
	HMACSignature []byte
	Name          string
	ModifiedFiles []string
	Repository    string
}
