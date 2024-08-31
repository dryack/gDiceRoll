package config

import (
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/v2"
	"strings"
)

func Load() (*koanf.Koanf, error) {
	k := koanf.New(".")

	// Default configuration
	defaultConfig := map[string]interface{}{
		"server.address":    ":8080",
		"dragonfly.address": "dragonfly:6379",
		"postgres.host":     "postgres",
		"postgres.port":     "5432",
		"postgres.dbname":   "gdiceroll",
		"postgres.user":     "youruser",
		"postgres.password": "yourpassword",
	}
	if err := k.Load(confmap.Provider(defaultConfig, "."), nil); err != nil {
		return nil, err
	}

	// Load from environment variables
	if err := k.Load(env.Provider("GDICEROLL_", ".", func(s string) string {
		return strings.Replace(strings.ToLower(
			strings.TrimPrefix(s, "GDICEROLL_")), "_", ".", -1)
	}), nil); err != nil {
		return nil, err
	}

	return k, nil
}
