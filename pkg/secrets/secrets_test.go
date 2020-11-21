package secrets

import (
	"regexp"
	"testing"

	workflowsv1alpha1 "github.com/nubank/workflows/pkg/apis/workflows/v1alpha1"
)

func TestGenerateKeyPair(t *testing.T) {
	repo := &workflowsv1alpha1.Repository{
		Owner: "my-org",
		Name:  "my-repo",
	}
	keyPair, err := GenerateKeyPair(repo)

	if err != nil {
		t.Errorf("Key pair generation failed: %s", err)
	}

	if len(keyPair.PrivateKey) == 0 {
		t.Error("Want a valid private key, but got an empty one")
	}

	if len(keyPair.PublicKey) == 0 {
		t.Error("Want a valid public key, but got an empty one")
	}

	if keyPair.Repository != repo {
		t.Errorf("Want repository %+v, but got %+v", repo, keyPair.Repository)
	}
}

func TestGenerateRandomToken(t *testing.T) {
	tokens := make(map[string]bool)
	pattern := regexp.MustCompile("^[a-f0-9]{40}$")

	for i := 0; i < 25; i++ {
		token := GenerateRandomToken()
		if !pattern.MatchString(token) {
			t.Errorf("Failed after %d iterations: token %s doesn't match pattern %s", i+1, token, pattern)
			t.FailNow()
		}

		if exists := tokens[token]; exists {
			t.Errorf("Failed after %d iterations: collision found with token %s", i+1, token)
		} else {
			tokens[token] = true
		}
	}
}
