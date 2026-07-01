// Package config loads CLI configuration, merging a file, env vars, and flags.
package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Target is the deployment environment a command acts on.
type Target string

const (
	TargetLocal Target = "local"
	TargetGCP   Target = "gcp"
)

// Config is the merged CLI configuration. Flags override file/env values,
// which are applied by the root command after Load.
type Config struct {
	Target    Target `mapstructure:"target"`
	DeployDir string `mapstructure:"deploy_dir"`
	Verbose   bool   `mapstructure:"verbose"`

	GCP   GCP   `mapstructure:"gcp"`
	Admin Admin `mapstructure:"admin"`
}

// GCP holds Google Cloud provisioning settings.
type GCP struct {
	Project     string `mapstructure:"project"`
	Region      string `mapstructure:"region"`
	MQBucket    string `mapstructure:"mq_bucket"`
	CHBucket    string `mapstructure:"ch_bucket"`
	MachineType string `mapstructure:"machine_type"`
}

// Admin holds the platform super-admin credentials seeded into query.
type Admin struct {
	Email    string `mapstructure:"email"`
	Password string `mapstructure:"password"`
}

// Load reads config from the given file (or the default search paths) plus
// OPTIKK_* env vars. A missing file is not an error — defaults are returned.
func Load(file string) (Config, error) {
	v := viper.New()
	v.SetDefault("target", string(TargetLocal))
	v.SetDefault("gcp.machine_type", "e2-standard-4")
	v.SetEnvPrefix("OPTIKK")
	v.AutomaticEnv()

	if file != "" {
		v.SetConfigFile(file)
	} else {
		v.SetConfigName("optikk")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		if home, err := os.UserHomeDir(); err == nil {
			v.AddConfigPath(filepath.Join(home, ".optikk"))
		}
	}

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// A named file that fails to parse is a real error.
			if file != "" {
				return Config{}, err
			}
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
