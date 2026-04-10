package config

import (
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadFile(t *testing.T) {
	t.Parallel()

	cfg, err := Load(WithConfigFile("./testdata/config.valid.yaml"))
	require.NoError(t, err)
	assert.Equal(t, "debug", cfg.LogLevel)
}

func TestLoadEnv(t *testing.T) {
	t.Setenv("LOG_LEVEL", "debug")
	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "debug", cfg.LogLevel)
}

func TestLoadFlags(t *testing.T) {
	t.Parallel()

	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.String("log-level", "", "Log level")
	err := flags.Set("log-level", "debug")
	require.NoError(t, err)
	cfg, err := Load(WithFlags(flags))
	require.NoError(t, err)
	assert.Equal(t, "debug", cfg.LogLevel)
}

func TestFlagsOverrideEnv(t *testing.T) {
	t.Setenv("LOG_LEVEL", "debug")

	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.String("log-level", "", "Log level")
	err := flags.Set("log-level", "trace")
	require.NoError(t, err)
	cfg, err := Load(WithFlags(flags))
	require.NoError(t, err)
	assert.Equal(t, "trace", cfg.LogLevel)
}
