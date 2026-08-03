// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package config

import (
	"github.com/spf13/viper"
)

// CredentialsCfg represents saferwall credentials.
type CredentialsCfg struct {
	URL    string `mapstructure:"url"`
	APIKey string `mapstructure:"api_key"`
}

// Config represents our CLI app config.
type Config struct {
	Credentials CredentialsCfg `mapstructure:"credentials"`
}

// Load returns an application configuration which is populated
// from the config file in the given directory.
func Load(path string, c any) error {

	// Adding our TOML config file.
	viper.AddConfigPath(path)

	// Set the config name to choose from the config path.
	// Extension not needed.
	viper.SetConfigName("config")

	// Load the configuration from disk.
	if err := viper.ReadInConfig(); err != nil {
		return err
	}

	// Unmarshal the config into our interface.
	return viper.Unmarshal(&c)
}
