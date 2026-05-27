/*
Copyright 2024 Swisscom (Schweiz) AG.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package config

import (
	"testing"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/stretchr/testify/assert"
)

func TestLoadConfigFromEnv(t *testing.T) {
	t.Setenv("NETBOX_HOST", "netbox_host")
	t.Setenv("AUTH_TOKEN", "auth-token")
	t.Setenv("RECONCILE_JITTER", "30m")
	t.Setenv("RECONCILE_SCHEDULE", "0 2 * * *")
	ResetForTesting()

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
