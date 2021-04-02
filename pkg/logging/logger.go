// package logging provides a factory for creating Zap loggers for components that
// don't use the injection system supplied by Knative. Nevertheless, the
// function NewLogger reads the same config-logging configmap and delegates the
// responsibility of instantiating the logger to
// knative.dev/pkg/logging/NewLogger, thus, keeping almost the same settings as
// those used by controllers and webhooks.
package logging

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"knative.dev/pkg/logging"
)

const (
	// Name of the environment variable that holds the logger configuration.
	loggerConfigKey = "LOGGER_CONFIG"

	// Name of the environment variable that holds the log level to be
	// configured.
	logLevelKey = "LOG_LEVEL"

	// Name of the environment variable that holds the pod's name to improve visibility of logger messages.
	podNameKey = "POD_NAME"
)

// NewLogger creates a new zap.SugaredLogger by reading a few settings declared
// via environment variables.
func NewLogger(component string) (*zap.SugaredLogger, error) {
	var (
		config  string
		level   string
		podName string
		exists  bool
	)

	if config, exists = os.LookupEnv(loggerConfigKey); !exists {
		return nil, fmt.Errorf("Error creating logger: environment variable %s is undefined", loggerConfigKey)
	}

	if level, exists = os.LookupEnv(logLevelKey); !exists {
		level = "info"
	}

	logger, _ := logging.NewLogger(config, level)

	if podName, exists = os.LookupEnv(podNameKey); exists {
		logger = logger.With(zap.String("pod", podName))
	}

	return logger.Named(component), nil
}
