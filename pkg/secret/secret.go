package secret

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"

	workflowsv1alpha1 "github.com/nubank/workflows/pkg/apis/workflows/v1alpha1"
	"golang.org/x/crypto/ssh"
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (

	// Key that stores Webhook secrets.
	secretTokenKey = "secret-token"

	// Size of private keys.
	keySize = 4096
)

// KeyPair represents a pair of SSH keys.
type KeyPair struct {
	PrivateKey []byte
	PublicKey  []byte
	Repository *workflowsv1alpha1.Repository
}

// GenerateKeyPair returns a new pair of SSH keys to be used by workflows to
// interact with Github repositories.
// repo is the Github repository to which the public key is associated.
func GenerateKeyPair(repo *workflowsv1alpha1.Repository) (*KeyPair, error) {
	privateKey, err := generateRSAPrivateKey()
	if err != nil {
		return nil, err
	}

	publicKey, err := generateRSAPublicKey(privateKey)
	if err != nil {
		return nil, err
	}

	return &KeyPair{
		PrivateKey: encodePrivateKeyToPEM(privateKey),
		PublicKey:  publicKey,
		Repository: repo,
	}, nil
}

// generateRSAPrivateKey returns a new RSA private key.
func generateRSAPrivateKey() (*rsa.PrivateKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		return nil, fmt.Errorf("Error generating private RSA key: %w", err)
	}
	return privateKey, nil
}

// encodePrivateKeyToPEM encodes the supplied RSA private key to PEM format.
func encodePrivateKeyToPEM(privateKey *rsa.PrivateKey) []byte {
	keyContent := x509.MarshalPKCS1PrivateKey(privateKey)
	block := pem.Block{Type: "RSA PRIVATE KEY",
		Bytes: keyContent,
	}
	return pem.EncodeToMemory(&block)
}

// generateRSAPublicKey returns the public key part of the supplied RSA private key.
func generateRSAPublicKey(privateKey *rsa.PrivateKey) ([]byte, error) {
	publicKey, err := ssh.NewPublicKey(privateKey.Public())
	if err != nil {
		return nil, fmt.Errorf("Error generating public RSA key: %w", err)
	}

	return ssh.MarshalAuthorizedKey(publicKey), nil
}

// GenerateRandomToken returns a secure random token in the hexadecimal format.
func GenerateRandomToken() string {
	token := make([]byte, 20)
	_, _ = rand.Read(token)
	return hex.EncodeToString(token)
}

// OfWebhook constructs a Kubernetes secret object to store Webhook secret tokens.
func OfWebhook(workflow *workflowsv1alpha1.Workflow, secretToken []byte) *corev1.Secret {
	webhookSecret := newSecret(workflow.GetWebhookSecretName(), workflow)
	SetSecretToken(webhookSecret, secretToken)
	return webhookSecret
}

// newSecret returns a corev1.Secret object with basic definitions.
func newSecret(name string, workflow *workflowsv1alpha1.Workflow) *corev1.Secret {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: workflow.Namespace,
			Labels: map[string]string{
				"workflows.dev/workflow": workflow.GetName(),
			},
		},
		Data: make(map[string][]byte),
	}

	setOwnerReference(secret, workflow)

	return secret
}

// setOwnerReference makes the provided Secret object dependent on the
// workflow. Thus, if the workflow is deleted, the Secret will be garbage
// collected automatically.
func setOwnerReference(secret *corev1.Secret, workflow *workflowsv1alpha1.Workflow) {
	ownerRef := metav1.NewControllerRef(workflow, workflow.GetGroupVersionKind())
	references := []metav1.OwnerReference{*ownerRef}
	secret.SetOwnerReferences(references)
}

// SetSecretToken assigns the supplied secret token to the Secret object in question.
func SetSecretToken(webhookSecret *corev1.Secret, secretToken []byte) {
	webhookSecret.Data[secretTokenKey] = secretToken
}

// GetSecretToken returns the decoded representation of the Webhook
// secret token held by the supplied Secret object.
func GetSecretToken(webhookSecret *corev1.Secret) ([]byte, error) {
	secretToken, exists := webhookSecret.Data[secretTokenKey]
	if !exists {
		return nil, fmt.Errorf("Key %s is missing in Secret object %s", secretTokenKey, types.NamespacedName{Namespace: webhookSecret.GetNamespace(), Name: webhookSecret.GetName()})
	}

	decodedWebhookSecret := make([]byte, base64.StdEncoding.DecodedLen(len(secretToken)))
	bytesWritten, err := base64.StdEncoding.Decode(decodedWebhookSecret, secretToken)
	if err != nil {
		return nil, fmt.Errorf("Error decoding Webhook secret from Secret %s: %w", types.NamespacedName{Namespace: webhookSecret.GetNamespace(), Name: webhookSecret.GetName()}, err)
	}

	return decodedWebhookSecret[:bytesWritten], nil
}

// OfDeployKeys constructs a Kubernetes secret object to project SSH private
// keys into Tekton tasks that need to interact with private repositories.
func OfDeployKeys(workflow *workflowsv1alpha1.Workflow, keyPairs []KeyPair) *corev1.Secret {
	deployKeys := newSecret(workflow.GetDeployKeysSecretName(), workflow)

	SetSSHPrivateKeys(deployKeys, keyPairs)

	return deployKeys
}

// SetSSHPrivateKeys assigns the supplied SSH private keys to the Secret object in question.
func SetSSHPrivateKeys(deployKeysSecret *corev1.Secret, keyPairs []KeyPair) {
	for _, keyPair := range keyPairs {
		deployKeysSecret.Data[keyPair.Repository.GetSSHPrivateKeyName()] = keyPair.PrivateKey
	}
}
