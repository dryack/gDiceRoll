package config

import (
	"fmt"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/v2"
	"log"
	"strings"
)

func Load() (*koanf.Koanf, error) {
	k := koanf.New(".")

	// Default configuration
	defaultConfig := map[string]interface{}{
		"server.address":     ":8080",
		"dragonfly.address":  "dragonfly:6379",
		"postgres.host":      "postgres",
		"postgres.port":      "5432",
		"postgres.dbname":    "gdiceroll",
		"postgres.user":      "youruser",
		"postgres.password":  "yourpassword",
		"jwt.accesss.secret": "",
		"jwt.refresh.secret": "",
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

	// Debug logging
	log.Printf("Loaded configuration:")
	log.Println(k.All())
	log.Printf("jwt.access.secret: %s", maskSecret(k.String("jwt.access.secret")))
	log.Printf("jwt.refresh.secret: %s", maskSecret(k.String("jwt.refresh.secret")))

	// JWT secrets must be set
	if k.String("jwt.access.secret") == "" {
		return nil, fmt.Errorf("JWT access secret is not set")
	}
	if k.String("jwt.refresh.secret") == "" {
		return nil, fmt.Errorf("JWT refresh secret is not set")
	}

	return k, nil
}

func maskSecret(s string) string {
	if len(s) > 4 {
		return s[:2] + "..." + s[len(s)-2:]
	}
	return "..."
}
