package config

import (
	"errors"
	"os"
	"strings"
)

// Config holds runtime configuration values.
// In production this could include:
// - API keys
// - logging settings
// - ports
// - feature flags
type Config struct {
	DBURL string
}

// Load reads environment variables and validates them.
func Load() (Config, error) {

	// Database connection string.
	db := strings.TrimSpace(os.Getenv("DB_URL"))

	if db == "" {
		return Config{}, errors.New("DB_URL required")
	}

	return Config{
		DBURL: db,
	}, nil
}
