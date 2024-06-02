package config

import (
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

var mockLogger = slog.New(slog.NewJSONHandler(io.Discard, nil))

func TestLoadConfigDefaults(t *testing.T) {
	viper.Reset()
	os.Clearenv()

	config, err := LoadConfig(mockLogger)
	assert.Nil(t, err)
	assert.Equal(t, "8080", config.Port)
	assert.Equal(t, "http://tiny.io", config.Domain)
	assert.Equal(t, 168*time.Hour, config.Expiry)
	assert.Equal(t, "postgres://username:password@localhost:5432/database_name", config.DATABASE_URL)
}

func TestLoadConfigEnvVars(t *testing.T) {
	os.Setenv("URLSHORTENER_PORT", "9090")
	os.Setenv("URLSHORTENER_DOMAIN", "http://shorty.io")
	os.Setenv("URLSHORTENER_EXPIRY", "72h")
	os.Setenv("URLSHORTENER_DATABASE_URL", "postgres://user:pass@remotehost:5432/db")

	config, err := LoadConfig(mockLogger)
	assert.Nil(t, err)
	assert.Equal(t, "9090", config.Port)
	assert.Equal(t, "http://shorty.io", config.Domain)
	assert.Equal(t, 72*time.Hour, config.Expiry)
	assert.Equal(t, "postgres://user:pass@remotehost:5432/db", config.DATABASE_URL)
}

func TestLoadConfigInvalidExpiry(t *testing.T) {
	os.Setenv("URLSHORTENER_EXPIRY", "invalid_duration")
	_, err := LoadConfig(mockLogger)
	assert.NotNil(t, err)
}
