// Package config provides configuration for the application.
package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Config is the configuration for the application.
type Config struct {
	LogLevel string `mapstructure:"log_level"`
	AIOutput bool   `mapstructure:"ai_output"`
}

const (
	// DefaultLogLevel is the default log level.
	DefaultLogLevel = "info"
)

// LoadOption is a function that can be used to load configuration.
type LoadOption func(*viper.Viper) error

// WithConfigFile sets a specific config file to load.
func WithConfigFile(path string) LoadOption {
	return func(v *viper.Viper) error {
		v.SetConfigFile(path)
		return nil
	}
}

// WithFlags binds flags to the viper instance.
func WithFlags(flags *pflag.FlagSet) LoadOption {
	return func(v *viper.Viper) error {
		normalizeFunc := flags.GetNormalizeFunc()
		flags.SetNormalizeFunc(func(fs *pflag.FlagSet, name string) pflag.NormalizedName {
			result := normalizeFunc(fs, name)
			name = strings.ReplaceAll(string(result), "-", "_") // Replace hyphens with underscores
			return pflag.NormalizedName(name)
		})
		return v.BindPFlags(flags)
	}
}

// Load loads configuration from file, env vars, and optionally flags.
func Load(opts ...LoadOption) (*Config, error) {
	v := viper.New()

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	v.AddConfigPath(filepath.Join(home, ".config", "mentat"))

	v.SetDefault("log_level", DefaultLogLevel)

	// Bind all configuration fields to environment variables
	typ := reflect.TypeFor[Config]()
	for field := range typ.Fields() {
		tag := field.Tag.Get("mapstructure")
		if tag != "" {
			if err := v.BindEnv(tag); err != nil {
				return nil, err
			}
		}
	}

	for _, opt := range opts {
		if err := opt(v); err != nil {
			return nil, err
		}
	}

	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
