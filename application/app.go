package application

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/ruvice/dotabackseaterbackend/config"
	"github.com/ruvice/dotabackseaterbackend/repository"
	"github.com/ruvice/dotabackseaterbackend/repository/redisRepo"
	"github.com/ruvice/dotabackseaterbackend/wrapper"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type App struct {
	debugMode      bool
	router         http.Handler
	config         config.Config
	twitchWrapper  *wrapper.TwitchWrapper
	redisAvailable bool
	mongoDB        *repository.MongoDBRepo
	redisRepo      *redisRepo.RedisRepo
}

// Returns pointer to instance of App
func New(ctx context.Context, config config.Config) *App {
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(config.MongoDBConfig.URI))
	rdb := redis.NewClient(&redis.Options{Addr: config.RedisAddress})
	redisRepo := &redisRepo.RedisRepo{
		Client: rdb,
	}
	mongoDB := &repository.MongoDBRepo{
		Client: mongoClient,
	}
	if err != nil {
		log.Println("Failed to establish connection with MongoDB, ", err)
	}
	app := &App{
		config:        config,
		twitchWrapper: wrapper.NewTwitchWrapper(config.TwitchConfig),
		mongoDB:       mongoDB,
		debugMode:     os.Getenv("DEBUG") == "true",
		redisRepo:     redisRepo,
	}
	app.loadRoutes()
	return app
}

func (a *App) Start(ctx context.Context) error {
	// Check MongoDB and Redis statuses
	if err := a.CheckMongoStatus(ctx); err != nil {
		return fmt.Errorf("mongoDB check failed: %w", err)
	}
	if err := a.CheckRedisStatus(ctx); err != nil {
		return fmt.Errorf("redis check failed: %w", err)
	}
	defer a.Cleanup(ctx)

	server := a.getHttpServer()
	a.PerformInitTasks(ctx)

	log.Printf("Starting server on: %d", a.config.ServerPort)

	// Making a channel, basically a type that allows communication between goroutines
	ch := make(chan error, 1)

	// GoRoutine~
	go func() {
		if a.debugMode {
			ch <- server.ListenAndServe()
		} else {
			certPath := os.Getenv("SSL_CERT_PATH")
			keyPath := os.Getenv("SSL_KEY_PATH")
			ch <- server.ListenAndServeTLS(
				certPath,
				keyPath,
			)
		}
		close(ch)
	}()

	select {
	case err := <-ch:
		return fmt.Errorf("server error: %w", err)
	case <-ctx.Done():
		log.Println("Shutdown signal received")
		timeout, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		return server.Shutdown(timeout)
	}
}
func (a *App) getHttpServer() *http.Server {
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", a.config.ServerPort),
		Handler: a.router,
	}

	// Add TLS configuration for non-debug mode
	if !a.debugMode {
		server.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	return server
}
