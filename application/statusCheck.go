package application

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func (a *App) CheckMongoStatus(ctx context.Context) error {
	// MongoDB
	err := a.mongoDB.Ping(ctx, readpref.Primary())
	if err != nil {
		fmt.Println("Problem reading MongoDB, ", err)
		return err
	}

	_, err = a.mongoDB.ListDatabaseNames(ctx, bson.M{})
	if err != nil {
		fmt.Println("Problem reading database names, ", err)
		return err
	}
	return nil
}

func (a *App) CheckRedisStatus(ctx context.Context) error {
	err := a.rdb.Ping(ctx).Err()
	if err != nil {
		fmt.Println("failed to connect to server:  %w", err)
		return err
	} else {
		a.redisAvailable = true
		fmt.Println("redisAvailability:", a.redisAvailable)
		return nil
	}
}
