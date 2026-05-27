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

/*
Package config contains all global configuration parameters.
*/
package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/spf13/viper"
)

var configuration *OperatorConfig

type OperatorConfig struct {
	viper                          *viper.Viper
	CaCert                         string `mapstructure:"CA_CERT"`
	NetboxHost                     string `mapstructure:"NETBOX_HOST"`
	AuthToken                      string `mapstructure:"AUTH_TOKEN"`
	HttpsEnable                    bool   `mapstructure:"HTTPS_ENABLE"`
	DebugEnable                    bool   `mapstructure:"DEBUG_ENABLE"`
	NetboxRestorationHashFieldName string `mapstructure:"NETBOX_RESTORATION_HASH_FIELD_NAME"`

	// cron schedule for scheduled reconciliation of all custom resources
	// if set, all custom resources will be reconciled at the defined schedule, in addition to the regular event-based reconciliation
	// if empty, scheduled reconciliation is disabled
	// format: cron, see https://pkg.go.dev/github.com/robfig/cron/v3 for examples
	// defaults to empty (disabled)
	ReconcileScheduleRaw string `mapstructure:"RECONCILE_SCHEDULE"`
	// jitter which is added to the defined reconcile schedule, only used if RECONCILE_SCHEDULE is defined
	// it adds a random amount of time within a given window to the next scheduled reconcile and can help reducing load on backend systems
	// all reconciles are randomly distributed across the window [scheduled trigger time, scheduled trigger time + RECONCILE_JITTER]
	// format: duration, needs to be parseable by time.ParseDuration, e.g. "30s", "30m"
	// defaults to 1 hour
	ReconcileJitterRaw string `mapstructure:"RECONCILE_JITTER"`

	// Parsed fields (not from config file/env)
	ReconcileSchedule       cron.Schedule
	ReconcileJitterDuration time.Duration
}

func (c *OperatorConfig) setDefaults() {
	c.viper.SetDefault("CA_CERT", "")
	c.viper.SetDefault("NETBOX_HOST", "")
	c.viper.SetDefault("AUTH_TOKEN", "")
	c.viper.SetDefault("HTTPS_ENABLE", true)
	c.viper.SetDefault("DEBUG_ENABLE", false)
	c.viper.SetDefault("NETBOX_RESTORATION_HASH_FIELD_NAME", "netboxOperatorRestorationHash")

	c.viper.SetDefault("RECONCILE_JITTER", "")
	c.viper.SetDefault("RECONCILE_SCHEDULE", "")

}

func (c *OperatorConfig) LoadCaCert() (cert []byte, err error) {
	caCert, err := os.ReadFile(c.CaCert)
	if err != nil {
		return nil, err
	}
	return caCert, nil
}

var once sync.Once

func GetOperatorConfig() *OperatorConfig {
	once.Do(func() {
		c := &OperatorConfig{}
		c.viper = viper.New()
		c.setDefaults()

		c.viper.SetConfigName("config")
		c.viper.SetConfigType("env")
		c.viper.AddConfigPath("/")
		c.viper.AddConfigPath(".")
		if err := c.viper.ReadInConfig(); err != nil {
			var cfnferr viper.ConfigFileNotFoundError
			if !errors.As(err, &cfnferr) {
				// Config file was found but another error was produced
				log.Fatalf("error reading config file: %s", err)
				return
			}
		} else {
			log.Printf("No config file found: %s - continuing...", c.viper.ConfigFileUsed())
		}
		c.viper.AutomaticEnv()

		err := c.viper.Unmarshal(c)
		if err != nil {
			log.Fatalf("error unmarshalling config: %s", err)
			return
		}

		err = c.parseScheduleAndJitter()
		if err != nil {
			log.Fatalf("error parsing schedule and jitter: %s", err)
			return
		}

		configuration = c
	})

	return configuration
}

func GetProtocol() string {
	if GetOperatorConfig().HttpsEnable {
		return "https"
	}
	return "http"
}

func GetBaseUrl() string {
	return GetProtocol() + "://" + GetOperatorConfig().NetboxHost
}

func (c *OperatorConfig) parseScheduleAndJitter() (err error) {
	// Parse jitter duration
	c.ReconcileJitterDuration = time.Hour // default
	if c.ReconcileJitterRaw != "" {
		c.ReconcileJitterDuration, err = time.ParseDuration(c.ReconcileJitterRaw)
		if err != nil {
			return fmt.Errorf("invalid reconcile jitter %q: %w", c.ReconcileJitterRaw, err)
		}
	}

	// Parse cron schedule
	c.ReconcileSchedule = nil // default
	if c.ReconcileScheduleRaw != "" {
		c.ReconcileSchedule, err = parseCronSchedule(c.ReconcileScheduleRaw)
		if err != nil {
			return fmt.Errorf("invalid cron schedule %q: %w", c.ReconcileScheduleRaw, err)
		}
	}

	return nil
}

func parseCronSchedule(cronExpr string) (cron.Schedule, error) {
	schedule, err := cron.ParseStandard(cronExpr)
	if err != nil {
		return nil, err
	}

	return schedule, nil
}

func ResetForTesting() {
	once = sync.Once{}
	configuration = nil
}
