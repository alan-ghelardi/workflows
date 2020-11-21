package secret

import (
	"encoding/base64"
	"fmt"

	"github.com/nubank/workflows/pkg/apis/workflows/v1alpha1"
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nubank/workflows/pkg/github"
)

// OfWebhook constructs a Kubernetes secret object to store Webhook secret tokens.
func OfWebhook(workflow *v1alpha1.Workflow, syncResult *github.SyncResult) *corev1.Secret {
	return newSecret(workflow, workflow.GetWebhookSecretName(), syncResult, true)
}

func newSecret(workflow *v1alpha1.Workflow, secretName string, syncResult *github.SyncResult, encodeToBase64 bool) *corev1.Secret {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: workflow.GetNamespace(),
		},
		Data: map[string][]byte{}}

	for _, entry := range syncResult.Entries {
		var secretValue []byte
		if encodeToBase64 {
			secretValue = make([]byte, base64.StdEncoding.EncodedLen(len(entry.Secret)))
			base64.StdEncoding.Encode(secretValue, entry.Secret)
		} else {
			secretValue = entry.Secret
		}

		secretKey := fmt.Sprintf("%s-%s", entry.Repository.Owner, entry.Repository.Name)
		secret.Data[secretKey] = secretValue
	}

	return secret
}

// OfDeployKeys constructs a Kubernetes secret object to store private SSH keys.
func OfDeployKeys(workflow *v1alpha1.Workflow, syncResult *github.SyncResult) *corev1.Secret {
	return newSecret(workflow, workflow.GetDeployKeysSecretName(), syncResult, false)
}
