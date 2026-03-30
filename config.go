package main

import (
	"encoding/json"
	"os"
)

// Config holds all server configuration.
type Config struct {
	Host      string          `json:"host"`
	Port      int             `json:"port"`
	StaticDir string          `json:"static_dir"`
	LogFile   string          `json:"log_file"`
	Timeout   TimeoutConfig   `json:"timeout"`
	RateLimit RateLimitConfig `json:"rate_limit"`
	CORS      CORSConfig      `json:"cors"`
}

// TimeoutConfig holds connection timeout values in seconds.
type TimeoutConfig struct {
	Read  int `json:"read"`
	Write int `json:"write"`
	Idle  int `json:"idle"`
}

// RateLimitConfig holds rate limiter settings.
type RateLimitConfig struct {
	Enabled           bool `json:"enabled"`
	RequestsPerSecond int  `json:"requests_per_second"`
	Burst             int  `json:"burst"`
}

// CORSConfig holds CORS settings.
type CORSConfig struct {
	Enabled        bool     `json:"enabled"`
	AllowedOrigins []string `json:"allowed_origins"`
	AllowedMethods []string `json:"allowed_methods"`
	AllowedHeaders []string `json:"allowed_headers"`
}

// defaultConfig returns a Config with sensible defaults.
func defaultConfig() Config {
	return Config{
		Host:      "0.0.0.0",
		Port:      8080,
		StaticDir: "./static",
		LogFile:   "",
		Timeout: TimeoutConfig{
			Read:  10,
			Write: 10,
			Idle:  60,
		},
		RateLimit: RateLimitConfig{
			Enabled:           false,
			RequestsPerSecond: 100,
			Burst:             20,
		},
		CORS: CORSConfig{
			Enabled:        false,
			AllowedOrigins: []string{"*"},
			AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders: []string{"Content-Type", "Authorization"},
		},
	}
}

// LoadConfig reads config from a JSON file, falling back to defaults for
// any missing fields.
func LoadConfig(path string) (Config, error) {
	cfg := defaultConfig()

	if path == "" {
		return cfg, nil
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			// No config file — use defaults silently
			return cfg, nil
		}
		return cfg, err
	}
	defer f.Close()

	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}
