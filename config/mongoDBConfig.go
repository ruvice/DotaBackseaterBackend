package config

import (
	"os"

	"github.com/joho/godotenv"
	"github.com/ruvice/dotabackseaterbackend/utils/dbsError"
)

type MongoDBConfig struct {
	URI string
}

func LoadMongoDBConfig() (MongoDBConfig, error) {
	err := godotenv.Load()
	if err != nil {
		return MongoDBConfig{}, dbsError.NewConfigError("LoadConfig", dbsError.ErrInvalidMongoConfig, "invalid Mongo Config", err)
	}

	MONGO_URI := os.Getenv("MONGO_URI")
	mongoDBConfig := MongoDBConfig{
		URI: MONGO_URI,
	}

	return mongoDBConfig, nil
}
