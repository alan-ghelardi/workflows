package github

import (
	"fmt"
	"os"
	"time"

	"github.com/bradleyfalzon/ghinstallation"

	"net/http"

	"strconv"

	"github.com/google/go-github/v33/github"
	"github.com/gregjones/httpcache"
)

const (

	// Name of the environment variable that contains the Github App ID.
	githubAppID = "GITHUB_APP_ID"

	// Name of the environment variable that contains the Github installation ID.
	githubInstallationID = "GITHUB_INSTALLATION_ID"

	// Path where the Github App private key is mounted.
	githubPrivateKeyPath = "/var/run/secrets/github/private-key"

	// Default timeout for calling Github services.
	timeout = time.Second * 15
)

// NotFoundError is returned when certain Github resources aren't found.
type NotFoundError struct {
	msg string
}

// Error satisfies the error interface.
func (n *NotFoundError) Error() string {
	return n.msg
}

// IsNotFound returns true if the supplied error is of the type NotFoundError
// otherwise it returns false.
func IsNotFound(e error) bool {
	switch e.(type) {
	case *NotFoundError:
		return true
	}
	return false
}

// NewClientOrDie returns a client to talk to Github APIs.
// It panics if the client cannot be created.
func NewClientOrDie() *github.Client {
	const errorMessage = "Error initializing Github REST client: %w"

	appID, err := parseID(githubAppID)
	if err != nil {
		panic(fmt.Errorf(errorMessage, err))
	}

	installationID, err := parseID(githubInstallationID)
	if err != nil {
		panic(fmt.Errorf(errorMessage, err))
	}

	transport := httpcache.NewTransport(httpcache.NewMemoryCache())
	installationTransport, err := ghinstallation.NewKeyFromFile(transport, appID, installationID, githubPrivateKeyPath)
	if err != nil {
		panic(fmt.Errorf(errorMessage, err))
	}

	return github.NewClient(&http.Client{
		Transport: installationTransport,
		Timeout:   timeout,
	})
}

// parseID attempts to read the environment variable whose name was given,
// returning it as an int64. It returns an error if the variable isn't set or if its value
// cannot be converted to an int64.
func parseID(varName string) (int64, error) {
	if value, exists := os.LookupEnv(varName); !exists {
		return 0, fmt.Errorf("Missing environment variable %s", varName)
	} else if id, err := strconv.ParseInt(value, 10, 64); err != nil {
		return 0, fmt.Errorf("Invalid value %s for environment variable %s: %w", value, varName, err)
	} else {
		return id, nil
	}
}
