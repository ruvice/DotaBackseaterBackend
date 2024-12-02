package application

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/ruvice/dotabackseaterbackend/wrapper"
)

type Config struct {
	RedisAddress  string
	ServerPort    uint64
	TwitchConfig  wrapper.TwitchConfig
	MongoDBConfig MongoDBConfig
}

func LoadConfig() Config {

	err := godotenv.Load()
	if err != nil {
		fmt.Println("Failed to get .env")
	}

	// REDIS_ADDR := os.Getenv("REDIS_ADDR")
	serverPortStr := os.Getenv("SERVER_PORT")
	if serverPortStr == "" {
		fmt.Println("SERVER_PORT is not set in the environment")
	}

	// Convert the string to uint64
	SERVER_PORT, err := strconv.ParseUint(serverPortStr, 10, 64)
	if err != nil {
		fmt.Println("Failed to parse SERVER_PORT: %v", err)
	}
	if err != nil {
		fmt.Println("Invalid PORT value: %v", err)
	}
	cfg := Config{
		RedisAddress:  "localhost:6379",
		ServerPort:    SERVER_PORT,
		TwitchConfig:  LoadTwitchConfig(),
		MongoDBConfig: LoadMongoDBConfig(),
	}

	return cfg
}
