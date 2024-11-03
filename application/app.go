package application

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/ruvice/dotabackseaterbackend/wrapper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type App struct {
	router         http.Handler
	rdb            *redis.Client
	config         Config
	twitchWrapper  *wrapper.TwitchWrapper
	redisAvailable bool
	mongoDB        *mongo.Client
	mongoAvailable bool
}

// Returns pointer to instance of App
func New(ctx context.Context, config Config) *App {
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(config.MongoDBConfig.URI))
	if err != nil {
		fmt.Println("Failed to establish connection with MongoDB, ", err)
	}
	app := &App{
		rdb: redis.NewClient(&redis.Options{
			Addr: config.RedisAddress,
		}),
		config:        config,
		twitchWrapper: wrapper.NewTwitchWrapper(config.TwitchConfig),
		mongoDB:       mongoClient,
	}
	app.loadRoutes()
	return app
}

func (a *App) Start(ctx context.Context) error {
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", a.config.ServerPort),
		Handler: a.router,
	}

	// MongoDB
	err := a.mongoDB.Ping(ctx, readpref.Primary())
	if err != nil {
		fmt.Println("Problem reading MongoDB, ", err)
	}

	databases, err := a.mongoDB.ListDatabaseNames(ctx, bson.M{})
	if err != nil {
		fmt.Println("Problem reading database names, ", err)
	}
	fmt.Println(databases)
	a.checkMongoDBHealth(ctx)
	a.checkRedisHealth(ctx)
	defer func() {
		if err := a.rdb.Close(); err != nil {
			fmt.Println("failed to close redis", err)
		}

		if err := a.mongoDB.Disconnect(ctx); err != nil {
			fmt.Println("failed to close connection to MongoDB", err)
		}
	}()

	fmt.Println("Starting server...")
	// Making a channel, basically a type that allows communication between goroutines
	ch := make(chan error, 1)

	// GoRoutine~
	go func() {
		err = server.ListenAndServe()
		// Error wrapping pog!
		if err != nil {
			ch <- fmt.Errorf("failed to start server:  %w", err)
		}
		close(ch)
	}()

	select {
	case err = <-ch:
		return err
	case <-ctx.Done():
		timeout, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		return server.Shutdown(timeout)
	}
}

func (a *App) checkRedisHealth(ctx context.Context) {
	err := a.rdb.Ping(ctx).Err()
	if err != nil {
		fmt.Println("failed to connect to server:  %w", err)
		a.redisAvailable = false
		fmt.Println("redisAvailability:", a.redisAvailable)
	} else {
		a.redisAvailable = true
		fmt.Println("redisAvailability:", a.redisAvailable)
	}
}

func (a *App) checkMongoDBHealth(ctx context.Context) {
	if a.mongoDB == nil {
		a.mongoAvailable = false
	} else {
		a.mongoAvailable = true
	}
}
