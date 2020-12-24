package scheduling

import (
	"testing"
)

var (
	sink = &EventSink{}
)

func TestDeniesRequestsIfSignatureIsMissing(t *testing.T) {
	wantedMessage := "Access denied: Github signature header X-Hub-Signature-256 is missing"
	valid, message := sink.verifySignature(&Request{}, []byte{})

	if valid {
		t.Error("Wanted an invalid signature result, but got a valid one")
	}

	if wantedMessage != message {
		t.Errorf("Wanted message %s, but got %s", wantedMessage, message)
	}
}

func TestAcceptsRequestsWhenSignaturesMatch(t *testing.T) {
	req := &Request{Body: []byte(`{
    "ref": "refs/heads/dev"
}`),
		HashSignature: []byte("sha256=4ae9df17f8cc696722c87f771f0c60fa7b03d44488ae3e0f712f570c4e7a3888"),
	}

	webhookSecret := []byte("secret")

	if valid, _ := sink.verifySignature(req, webhookSecret); !valid {
		t.Errorf("Wanted a valid result for the signature validation, but got an invalid one")
	}
}

func TestRejectsRequestsWhenSignaturesDoNotMatch(t *testing.T) {
	req := &Request{Body: []byte(`{
    "ref": "refs/heads/dev"
}`),
		// This digest was calculated with the key other-secret.
		HashSignature: []byte("d8a72707bd05f566becba60815c77f1e2adddddfceed668ca4844489d12ded07"),
	}

	webhookSecret := []byte("secret")

	if valid, _ := sink.verifySignature(req, webhookSecret); valid  {
		t.Errorf("Wanted a invalid result for the signature validation, but got a valid one")
	}
}
