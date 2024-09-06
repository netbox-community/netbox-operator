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
	"log"
	"os"
	"sync"

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
}

func (c *OperatorConfig) setDefaults() {
	c.viper.SetDefault("CA_CERT", "")
	c.viper.SetDefault("NETBOX_HOST", "")
	c.viper.SetDefault("AUTH_TOKEN", "")
	c.viper.SetDefault("HTTPS_ENABLE", true)
	c.viper.SetDefault("DEBUG_ENABLE", false)
	c.viper.SetDefault("NETBOX_RESTORATION_HASH_FIELD_NAME", "netboxOperatorRestorationHash")
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

		configuration = c
	})

	return configuration
}

func GetProtocol() string {
	if GetOperatorConfig().HttpsEnable {
		return "https"
	} else {
		return "http"
	}
}

func GetBaseUrl() string {
	return GetProtocol() + "://" + GetOperatorConfig().NetboxHost
}
