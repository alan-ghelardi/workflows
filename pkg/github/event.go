package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

const (
	// githubSignatureHeader defines the header name for hash signatures sent by Github.
	githubSignatureHeader = "X-Hub-Signature-256"
)

// Event represents a Github Webhook event.
type Event struct {
	Body          []byte
	Branch        string
	DeliveryID    string
	HeadCommitSHA string
	HMACSignature []byte
	Name          string
	ModifiedFiles []string
	Repository    string
}

// VerifySignature validates the payload sent by Github Webhooks by calculating
// a hash signature using the provided key and comparing it with the signature
// sent along with the request.
// For further details about the algorithm, please see:
// https://docs.github.com/en/free-pro-team@latest/developers/webhooks-and-events/securing-your-webhooks.
func (e *Event) VerifySignature(webhookSecret []byte) (bool, string) {
	if e.HMACSignature == nil || len(e.HMACSignature) == 0 {
		return false, fmt.Sprintf("Access denied: Github signature header %s is missing", githubSignatureHeader)
	}

	hash := hmac.New(sha256.New, webhookSecret)
	hash.Write(e.Body)
	digest := hash.Sum(nil)

	signature := make([]byte, hex.EncodedLen(len(digest)))
	hex.Encode(signature, digest)
	signature = append([]byte("sha256="), signature...)

	if !hmac.Equal(e.HMACSignature, signature) {
		return false, "Access denied: HMAC signatures don't match. The request signature we calculated does not match the provided signature."
	}

	return true, "Access permitted: the signature we calculated match the provided signature."
}
