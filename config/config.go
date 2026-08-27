package config

import (
	"StoreServer/utils/logger"
	"sync"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

var (
	cfg  *Config
	once sync.Once
)

type Config struct {
	ServerPort     string `envconfig:"SERVER_PORT" default:"8080"`
	MONGOURL       string `envconfig:"MONGO_URL"`
	JWTSecret      string `envconfig:"JWT_SECRET"`
	JWTExpiryHours int    `envconfig:"JWT_EXPIRY_HOURS" default:"24"`
}

func init() {
	GetConfig()
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

