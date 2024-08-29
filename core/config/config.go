package config

import (
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/v2"
)

func Load() (*koanf.Koanf, error) {
	k := koanf.New(".")

	// Default configuration
	defaultConfig := map[string]interface{}{
		"server.address":    ":8080",
		"dragonfly.address": "dragonfly:6379",
	}
	k.Load(confmap.Provider(defaultConfig, "."), nil)

	// Load from environment variables
	return k, k.Load(env.Provider("GDICEROLL_", ".", func(s string) string {
		return s
	}), nil)
}
