// Copyright 2018 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package config

import (
	"github.com/spf13/viper"
)

// CredentialsCfg represents saferwall credentials.
type CredentialsCfg struct {
	URL      string `mapstructure:"url"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

// AWSS3Cfg represents AWS S3 credentials.
type AWSS3Cfg struct {
	Region    string `mapstructure:"region"`
	SecretKey string `mapstructure:"secret_key"`
	AccessKey string `mapstructure:"access_key"`
}

// MinIOCfg represents MinIO credentials.
type MinIOCfg struct {
	Endpoint  string `mapstructure:"endpoint"`
	Region    string `mapstructure:"region"`
	SecretKey string `mapstructure:"secret_key"`
	AccessKey string `mapstructure:"access_key"`
}

// LocalFsCfg represents local file system storage data.
type LocalFsCfg struct {
	RootDir string `mapstructure:"root_dir"`
}

// StorageCfg represents the object storage config.
type StorageCfg struct {
	// Deployment kind, possible values: aws, gcp, azure, local.
	DeploymentKind string     `mapstructure:"deployment_kind"`
	Bucket         string     `mapstructure:"bucket"`
	S3             AWSS3Cfg   `mapstructure:"s3"`
	MinIO          MinIOCfg   `mapstructure:"minio"`
	Local          LocalFsCfg `mapstructure:"local"`
}

// Config represents our CLI app config.
type Config struct {
	Credentials CredentialsCfg `mapstructure:"credentials"`
	Storage     StorageCfg     `mapstructure:"storage"`
}

// Load returns an application configuration which is populated
// from the given configuration file.
func Load(path, env string, c interface{}) error {

	// Adding our TOML config file.
	viper.AddConfigPath(path)

	// Load the config type depending on env variable.
	var name string
	switch env {
	case "local":
		name = "local"
	case "dev":
		name = "dev"
	case "prod":
		name = "prod"
	default:
		name = "config"
	}

	// Set the config name to choose from the config path
	// Extension not needed.
	viper.SetConfigName(name)

	// Load the configuration from disk.
	err := viper.ReadInConfig()
	if err != nil {
		return err
	}

	// Unmarshal the config into our interface.
	err = viper.Unmarshal(&c)
	if err != nil {
		return err
	}

	return err
}
