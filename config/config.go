package config

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/ruvice/dotabackseaterbackend/utils/configError"
	"github.com/ruvice/dotabackseaterbackend/wrapper"
)

type Config struct {
	RedisAddress  string
	ServerPort    uint64
	TwitchConfig  wrapper.TwitchConfig
	MongoDBConfig MongoDBConfig
}

func LoadConfig() (Config, error) {
	err := godotenv.Load()
	if err != nil {
		return Config{}, wrapConfigError(configError.ErrMissingEnv, "failed to get env file", err)
	}

	redisAddress, err := LoadRedisAddress()
	if err != nil {
		return Config{}, wrapConfigError(configError.ErrInvalidValue, "invalid REDIS_ADDR value", err)
	}
	serverPort, err := LoadServerPort()
	if err != nil {
		return Config{}, wrapConfigError(configError.ErrInvalidValue, "invalid SERVER_PORT value", err)
	}

	twitchConfig, err := LoadTwitchConfig()
	if err != nil {
		return Config{}, wrapConfigError(configError.ErrInvalidValue, "invalid Twitch Config value", err)
	}

	mongoDBConfig, err := LoadMongoDBConfig()
	if err != nil {
		return Config{}, wrapConfigError(configError.ErrInvalidValue, "invalid Mongo Config value", err)
	}

	cfg := Config{
		RedisAddress:  redisAddress,
		ServerPort:    serverPort,
		TwitchConfig:  twitchConfig,
		MongoDBConfig: mongoDBConfig,
	}

	return cfg, nil
}

func wrapConfigError(code configError.ConfigErrorCode, message string, err error) error {
	configErr := configError.NewConfigError("LoadConfig", code, message, err)
	log.Printf("Configuration error: %v", configErr)
	return configErr
}
