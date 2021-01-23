package config

import (
	"fmt"
	"net/url"

	"github.com/ghodss/yaml"
	corev1 "k8s.io/api/core/v1"
)

const (
	// DefaultsConfigName is the name of config map for the defaults.
	DefaultsConfigName = "config-defaults"

	// fallbackImage is the image used by default when neither the task step nor the config-defaults declare one.
	fallbackImage = "gcr.io/google-containers/busybox"
)

// +k8s:deepcopy-gen=true
type Defaults struct {
	// Default image to be used in steps when the task step doesn't declare one.
	DefaultImage string

	// Default Webhook URL to be applied to workflows that do not declare a more specific one.
	Webhook string

	// Labels to be applied to all PipelineRun objects created by workflows.
	// Useful for defining common metadata observed by other controllers.
	Labels map[string]string

	// Annotations to be applied to all PipelineRun objects created by workflows.
	// Useful for defining common metadata observed by other controllers.
	Annotations map[string]string
}

// parser is a function that turns the given string into a higher object and
// sets it to the provided Defaults instance.
type parser func(defaults *Defaults, value string) error

func parseDefaultImage(defaults *Defaults, value string) error {
	defaults.DefaultImage = value
	return nil
}

func parseWebhook(defaults *Defaults, value string) error {
	_, err := url.ParseRequestURI(value)
	if err != nil {
		return fmt.Errorf("Invalid Webhook URL: %w", err)
	}
	defaults.Webhook = value

	return nil
}

func parseLabels(defaults *Defaults, value string) error {
	var labels map[string]string
	if err := yaml.Unmarshal([]byte(value), &labels); err != nil {
		return fmt.Errorf("Invalid labels: %s", err)
	}
	defaults.Labels = labels

	return nil
}

func parseAnnotations(defaults *Defaults, value string) error {
	var annotations map[string]string
	if err := yaml.Unmarshal([]byte(value), &annotations); err != nil {
		return fmt.Errorf("Invalid annotations: %s", err)
	}
	defaults.Annotations = annotations

	return nil
}

// parsers maps keys of known configs to a parser function.
var parsers = map[string]parser{
	"default-image": parseDefaultImage,
	"webhook":       parseWebhook,
	"labels":        parseLabels,
	"annotations":   parseAnnotations,
}

// NewDefaultsFromConfigMap takes a ConfigMap and returns a Defaults object.
func NewDefaultsFromConfigMap(configMap *corev1.ConfigMap) (*Defaults, error) {
	defaults := &Defaults{}

	for key, value := range configMap.Data {
		if parser, exists := parsers[key]; exists {
			if err := parser(defaults, value); err != nil {
				return nil, err
			}
		}
	}

	if defaults.DefaultImage == "" {
		// Guarantee that at least the fallback image will be set
		defaults.DefaultImage = fallbackImage
	}

	return defaults, nil
}
