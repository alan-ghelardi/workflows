package config

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
)

func TestNewDefaults(t *testing.T) {
	tests := []struct {
		configMap string
		defaults  *Defaults
		valid     bool
	}{{
		configMap: "valid-config-defaults.yaml",
		defaults: &Defaults{
			DefaultImage: "ubuntu",
			Webhook:      "https://hooks.example.com",
			Labels:       map[string]string{"workflows.dev/example-label": "example"},
			Annotations:  map[string]string{"workflows.dev/example-annotation": "example"},
		},
		valid: true,
	},
		{
			configMap: "config-defaults-without-default-image.yaml",
			defaults:  &Defaults{DefaultImage: fallbackImage},
			valid:     true,
		},
		{
			configMap: "invalid-config-defaults-1.yaml",
			valid:     false,
		},
		{
			configMap: "invalid-config-defaults-2.yaml",
			valid:     false,
		},
		{
			configMap: "invalid-config-defaults-3.yaml",
			valid:     false,
		},
	}

	for _, test := range tests {
		var configMap *corev1.ConfigMap

		file, err := ioutil.ReadFile(fmt.Sprintf("testdata/%s", test.configMap))
		if err != nil {
			t.Fatalf("Error reading file %s: %v", test.configMap, err)
		}

		if err := yaml.Unmarshal(file, &configMap); err != nil {
			t.Fatalf("Error parsing config map %s: %v", test.configMap, err)
		}

		defaults, err := NewDefaultsFromConfigMap(configMap)
		if test.valid && err != nil {
			t.Fatalf("Unexpected error while parsing defaults from config map %s: %v", test.configMap, err)
		}

		if diff := cmp.Diff(test.defaults, defaults); diff != "" {
			t.Errorf("Fail while parsing config map %s\nMismatch (-want +got):\n%s", test.configMap, diff)
		}
	}
}
