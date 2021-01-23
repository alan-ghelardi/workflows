package config

import (
	"context"

	"knative.dev/pkg/configmap"
)

// configKey identifies a Config object within a context.Context.
type configKey struct{}

// Config holds the collection of configurations that we attach to contexts.
type Config struct {
	Defaults *Defaults
}

// Get extracts a Config from the provided context.
func Get(ctx context.Context) *Config {
	config, exists := ctx.Value(configKey{}).(*Config)
	if exists {
		return config
	}
	return nil
}

// withConfig attaches the provided Config to the provided context, returning the
// new context with the Config attached.
func withConfig(ctx context.Context, c *Config) context.Context {
	return context.WithValue(ctx, configKey{}, c)
}

// Store is a typed wrapper around configmap.Untyped store to handle our configmaps.
// +k8s:deepcopy-gen=false
type Store struct {
	*configmap.UntypedStore
}

// NewStore creates a new store of Configs and optionally calls functions when ConfigMaps are updated.
func NewStore(logger configmap.Logger, onAfterStore ...func(name string, value interface{})) *Store {
	store := &Store{
		UntypedStore: configmap.NewUntypedStore(
			"apis",
			logger,
			configmap.Constructors{
				DefaultsConfigName: NewDefaultsFromConfigMap,
			},
			onAfterStore...,
		),
	}

	return store
}

// WithConfig attaches the current Config state to the provided context.
func (s *Store) WithConfig(ctx context.Context) context.Context {
	return withConfig(ctx, s.Load())
}

// Load creates a Config from the current config state of the Store.
func (s *Store) Load() *Config {
	config := &Config{}
	if def, ok := s.UntypedLoad(DefaultsConfigName).(*Defaults); ok {
		config.Defaults = def.DeepCopy()
	}
	return config
}
