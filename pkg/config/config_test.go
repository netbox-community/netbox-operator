//go:build unit

package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	filePath   = "."
	fileName   = "test"
	fileFormat = "env"
)

func TestLoadConfigFromEnv(t *testing.T) {
	os.Setenv("NETBOX_HOST", "netbox_host")
	os.Setenv("AUTH_TOKEN", "auth-token")

	configuration := GetOperatorConfig()

	assert.Equal(t, "netbox_host", configuration.NetboxHost)
	assert.Equal(t, "auth-token", configuration.AuthToken)
}
