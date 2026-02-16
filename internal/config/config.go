package config

import (
	"errors"
	"os"
	"strings"
)

// Config contains runtime configuration required by the service.
type Config struct {
	DBURL   string
	APIKeys map[string]string // apiKey -> tenantID
}

// Load reads required values from environment variables.
// API_KEYS format: "tenant1:key1,tenant2:key2"
func Load() (Config, error) {
	dbURL := strings.TrimSpace(os.Getenv("DB_URL"))
	if dbURL == "" {
		return Config{}, errors.New("DB_URL required")
	}

	apiKeysRaw := strings.TrimSpace(os.Getenv("API_KEYS"))
	apiKeys := map[string]string{}

	if apiKeysRaw != "" {
		pairs := strings.Split(apiKeysRaw, ",")
		for _, p := range pairs {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			parts := strings.SplitN(p, ":", 2)
			if len(parts) != 2 {
				return Config{}, errors.New(`API_KEYS must be "tenant:key,tenant:key"`)
			}
			tenant := strings.TrimSpace(parts[0])
			key := strings.TrimSpace(parts[1])
			if tenant == "" || key == "" {
				return Config{}, errors.New(`API_KEYS must be "tenant:key,tenant:key"`)
			}
			apiKeys[key] = tenant
		}
	}

	// Local dev fallback so the service runs out-of-the-box.
	if len(apiKeys) == 0 {
		apiKeys["tenant-key-123"] = "tenant1"
	}

	return Config{
		DBURL:   dbURL,
		APIKeys: apiKeys,
	}, nil
}
