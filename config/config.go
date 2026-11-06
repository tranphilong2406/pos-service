package config

import (
	"StoreServer/utils/logger"
	"fmt"
	"sync"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

var (
	cfg  *Config
	once sync.Once
)

type Config struct {
	ServerPort      string `envconfig:"SERVER_PORT" default:"8080"`
	MONGOURL        string `envconfig:"MONGO_URL"`
	JWTSecret       string `envconfig:"JWT_SECRET"`
	JWTExpiryHours  int    `envconfig:"JWT_EXPIRY_HOURS" default:"24"`
	AnthropicAPIKey string `envconfig:"ANTHROPIC_API_KEY"`
	ClaudeModel     string `envconfig:"CLAUDE_MODEL" default:"claude-haiku-5-5"`
}

// minJWTSecretLength is the shortest HS256 secret that is not practical to brute-force offline.
const minJWTSecretLength = 32

func init() {
	GetConfig()
}

// Warnings lists settings that work but are unsafe for production.
func (c *Config) Warnings() []string {
	var warnings []string
	if len(c.JWTSecret) < minJWTSecretLength {
		warnings = append(warnings, fmt.Sprintf("JWT_SECRET is shorter than %d characters; use a long random value (e.g. openssl rand -base64 48)", minJWTSecretLength))
	}
	if c.AnthropicAPIKey == "" {
		warnings = append(warnings, "ANTHROPIC_API_KEY is not set; product image analysis is disabled")
	}
	return warnings
}

func GetConfig() *Config {
	once.Do(func() {
		cfg = &Config{}
		_ = godotenv.Load()
		err := envconfig.Process("", cfg)
		if err != nil {
			logger.GetLogger().Error(err.Error())
		}
		if cfg.JWTExpiryHours <= 0 {
			cfg.JWTExpiryHours = 24
		}
	})

	return cfg
}

