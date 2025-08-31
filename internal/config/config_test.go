package config

import (
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestInitConfigWithFile(t *testing.T) {
	// Create a temporary config file
	file, err := os.CreateTemp("", "config-*.yaml")
	assert.NoError(t, err)
	defer os.Remove(file.Name())

	_, err = file.WriteString("api_key: test-key-from-file")
	assert.NoError(t, err)
	file.Close()

	CfgFile = file.Name()
	InitConfig()

	assert.Equal(t, "test-key-from-file", viper.GetString("api_key"))
}

func TestInitConfigWithEnv(t *testing.T) {
	// Set an environment variable
	os.Setenv("KNOCKER_API_KEY", "test-key-from-env")
	defer os.Unsetenv("KNOCKER_API_KEY")

	viper.SetEnvPrefix("knocker")
	viper.BindEnv("api_key")

	InitConfig()

	assert.Equal(t, "test-key-from-env", viper.GetString("api_key"))
}