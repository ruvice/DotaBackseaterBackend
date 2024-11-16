package application

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/ruvice/dotabackseaterbackend/model"
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
	a.performInitTasks(ctx)

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

func (a *App) performInitTasks(ctx context.Context) {
	itemMap := a.getItemsFromMongo(ctx)
	a.writeItemsToCache(ctx, itemMap)
}

func (a *App) getItemsFromMongo(ctx context.Context) model.ItemMap {
	twitchExtensionDatabase := a.mongoDB.Database("itemDatabase")
	channelCollection := twitchExtensionDatabase.Collection("itemsValid")

	filter := bson.M{}
	var result bson.M
	err := channelCollection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		fmt.Println("Failed to find docucment: ", err)
		return model.ItemMap{}
	}

	fmt.Println("Found document:", result)
	itemMap := make(model.ItemMap)

	// Iterate over the bson.M map and convert keys to integers
	for key, value := range result {
		// Skip the `_id` field
		if key == "_id" {
			continue
		}
		itemID := key
		if err != nil {
			fmt.Println("Invalid item_id key:", key)
			return model.ItemMap{}
		}
		// Assert that the value is a nested object (bson.M)
		itemData, ok := value.(bson.M)
		if !ok {
			fmt.Println("Invalid value type for key:", key)
			return model.ItemMap{}
		}

		// Extract `name` and `cost` from the nested object// Extract `name`
		name, _ := itemData["name"].(string)
		itemName, _ := itemData["itemName"].(string)
		// Extract `cost`, defaulting to 0 if not present or null
		var itemCost int32
		if costValue, ok := itemData["cost"]; ok && costValue != nil {
			itemCost = costValue.(int32)
		} else {
			itemCost = 0 // Default to 0 if `cost` is absent or null
		}
		itemDetail := model.ItemDetail{
			Name:     name,
			ItemName: itemName,
			Cost:     itemCost,
		}

		itemMap[itemID] = itemDetail
	}
	return itemMap
}

func (a *App) writeItemsToCache(ctx context.Context, itemMap model.ItemMap) {
	for itemID, itemDetail := range itemMap {
		data, err := json.Marshal(itemDetail)
		if err != nil {
			fmt.Println("Failed to encode ItemDetail:", err)
		}
		// Generating unique key
		key := "itemID:" + itemID

		// Using transaction to make changes atomic
		txn := a.rdb.TxPipeline()

		res := txn.Set(ctx, key, string(data), 0)
		if err := res.Err(); err != nil {
			txn.Discard()
			fmt.Println("failed to add item: ", err)
		}

		if _, err := txn.Exec(ctx); err != nil {
			fmt.Println("failed to exec:", err)
		}
		fmt.Println("success")
	}
}
