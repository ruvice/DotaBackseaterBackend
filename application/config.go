package application

import "github.com/ruvice/dotabackseaterbackend/wrapper"

type Config struct {
	RedisAddress  string
	ServerPort    uint64
	TwitchConfig  wrapper.TwitchConfig
	MongoDBConfig MongoDBConfig
}

func LoadConfig() Config {
	cfg := Config{
		// RedisAddress: "localhost:6379",
		RedisAddress:  "redis:6379",
		ServerPort:    3000,
		TwitchConfig:  LoadTwitchConfig(),
		MongoDBConfig: LoadMongoDBConfig(),
	}

	return cfg
}
