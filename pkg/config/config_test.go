//go:build unit

package config

import (
	"os"
	"testing"
	"time"

	"github.com/robfig/cron/v3"
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

	os.Setenv("RECONCILE_JITTER", "30m")
	os.Setenv("RECONCILE_SCHEDULE", "0 2 * * *")

	configuration := GetOperatorConfig()

	expectedSchedule, _ := cron.ParseStandard("0 2 * * *")

	assert.Equal(t, "netbox_host", configuration.NetboxHost)
	assert.Equal(t, "auth-token", configuration.AuthToken)

	assert.Equal(t, 30*time.Minute, configuration.ReconcileJitterDuration)
	assert.Equal(t, configuration.ReconcileSchedule, expectedSchedule)

}

func TestParseScheduleAndJitter_Defaults(t *testing.T) {
	c := &OperatorConfig{
		ReconcileJitterRaw:   "",
		ReconcileScheduleRaw: "",
	}
	err := c.parseScheduleAndJitter()
	assert.NoError(t, err)
	assert.Equal(t, time.Hour, c.ReconcileJitterDuration)
	assert.Equal(t, nil, c.ReconcileSchedule)
}

func TestParseScheduleAndJitter_CustomValues(t *testing.T) {
	c := &OperatorConfig{
		ReconcileJitterRaw:   "30m",
		ReconcileScheduleRaw: "0 2 * * *",
	}
	expectedSchedule, _ := cron.ParseStandard("0 2 * * *")
	err := c.parseScheduleAndJitter()
	assert.NoError(t, err)
	assert.Equal(t, 30*time.Minute, c.ReconcileJitterDuration)
	assert.Equal(t, expectedSchedule, c.ReconcileSchedule)
}

func TestParseScheduleAndJitter_InvalidJitter(t *testing.T) {
	c := &OperatorConfig{
		ReconcileJitterRaw: "notaduration",
	}
	err := c.parseScheduleAndJitter()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid reconcile jitter")
}

func TestParseScheduleAndJitter_InvalidSchedule(t *testing.T) {
	c := &OperatorConfig{
		ReconcileScheduleRaw: "bad cron",
	}
	err := c.parseScheduleAndJitter()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid cron schedule")
}

func TestParseCronSchedule_InvalidHour(t *testing.T) {
	_, err := parseCronSchedule("0 xx * * *")
	assert.Error(t, err)
}
