package application

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type MongoDBConfig struct {
	URI string
}

func LoadMongoDBConfig() MongoDBConfig {

	err := godotenv.Load()
	if err != nil {
		fmt.Println("Failed to get .env")
	}

	MONGO_URI := os.Getenv("MONGO_URI")
	mongoDBConfig := MongoDBConfig{
		URI: MONGO_URI,
	}

	return mongoDBConfig
}
