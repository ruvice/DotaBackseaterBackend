package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"

	"github.com/ruvice/dotabackseaterbackend/application"
	"github.com/ruvice/dotabackseaterbackend/config"
	"github.com/ruvice/dotabackseaterbackend/utils/DBSError"
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
	var cfgErr *DBSError.ConfigError
	if errors.As(err, &cfgErr) {
		switch cfgErr.Code {
		case DBSError.ErrMissingEnv:
			log.Fatalf("Configuration error: %s (missing environment variable)", cfgErr.Message)
		case DBSError.ErrInvalidValue:
			log.Fatalf("Configuration error: %s (invalid value)", cfgErr.Message)
		case DBSError.ErrFileNotFound:
			log.Fatalf("Configuration error: %s (file not found)", cfgErr.Message)
		case DBSError.ErrInvalidMongoConfig:
			log.Fatalf("Configuration error: %s (invalid mongo config)", cfgErr.Message)
		case DBSError.ErrInvalidTwitchConfig:
			log.Fatalf("Configuration error: %s (invalid twitch config)", cfgErr.Message)
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
