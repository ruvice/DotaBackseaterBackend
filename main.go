package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"

	"github.com/ruvice/dotabackseaterbackend/application"
	"github.com/ruvice/dotabackseaterbackend/config"
	"github.com/ruvice/dotabackseaterbackend/utils/dbsError"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	config, err := config.LoadConfig()
	if err != nil {
		handleLoadConfigError(err)
	}

	app := application.New(ctx, config)
	defer cancel()
	err = app.Start(ctx)
	if err != nil {
		handleAppStartError(err)
	}
}

func handleLoadConfigError(err error) {
	var cfgErr *dbsError.ConfigError
	if errors.As(err, &cfgErr) {
		switch cfgErr.Code {
		case dbsError.ErrMissingEnv:
			log.Fatalf("Configuration error: %s (missing environment variable)", cfgErr.DBSError.Message)
		case dbsError.ErrInvalidValue:
			log.Fatalf("Configuration error: %s (invalid value)", cfgErr.DBSError.Message)
		case dbsError.ErrFileNotFound:
			log.Fatalf("Configuration error: %s (file not found)", cfgErr.DBSError.Message)
		case dbsError.ErrInvalidMongoConfig:
			log.Fatalf("Configuration error: %s (invalid mongo config)", cfgErr.DBSError.Message)
		case dbsError.ErrInvalidTwitchConfig:
			log.Fatalf("Configuration error: %s (invalid twitch config)", cfgErr.DBSError.Message)
		default:
			log.Fatalf("Unexpected configuration error: %v", cfgErr)
		}
	} else {
		log.Fatalf("Unexpected error: %v", err)
	}
}

func handleAppStartError(err error) {
	log.Fatalf("failed to start app: %v", err)
}
