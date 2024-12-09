package config

import (
	"errors"
	"os"
)

func LoadRedisAddress() (string, error) {
	debugMode := os.Getenv("DEBUG") == "true"
	var redisAddress string
	if debugMode {
		redisAddress = os.Getenv("DEBUG_REDIS_ADDR")
	} else {
		redisAddress = os.Getenv("REDIS_ADDR")
	}
	if redisAddress == "" {
		return "", errors.New("missing REDIS_ADDR environment variable")
	}
	return redisAddress, nil
}
