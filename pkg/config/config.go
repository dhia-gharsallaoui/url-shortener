package config

import (
	"log/slog"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Port         string        // Port where the server will run
	Domain       string        // Domain used for generating short URLs
	Expiry       time.Duration // Duration for which a URL should remain active
	DATABASE_URL string
}

func LoadConfig(logger *slog.Logger) (*Config, error) {
	viper.SetDefault("PORT", "8080")
	viper.SetDefault("DOMAIN", "http://tiny.io")
	viper.SetDefault("EXPIRY", "168h")
	viper.SetDefault("DATABASE_URL", "postgres://username:password@localhost:5432/database_name")

	viper.SetEnvPrefix("URLSHORTENER")
	viper.AutomaticEnv()

	// we can also use config file

	var config Config
	err := viper.Unmarshal(&config)
	if err != nil {
		logger.Error("Unable to decode into struct, %v", err)
		return nil, err
	}

	expiryDuration, err := time.ParseDuration(viper.GetString("EXPIRY"))
	if err != nil {
		logger.Error("Invalid format for EXPIRY, use valid time units, %v", err)
		return nil, err
	}
	config.Expiry = expiryDuration

	return &config, nil
}
