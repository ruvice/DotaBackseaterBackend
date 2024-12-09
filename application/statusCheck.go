package application

import (
	"context"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func (a *App) CheckMongoStatus(ctx context.Context) error {
	// MongoDB
	err := a.mongoDB.Client.Ping(ctx, readpref.Primary())
	if err != nil {
		log.Println("Problem reading MongoDB, ", err)
		return err
	}

	_, err = a.mongoDB.Client.ListDatabaseNames(ctx, bson.M{})
	if err != nil {
		log.Println("Problem reading database names, ", err)
		return err
	}
	return nil
}

func (a *App) CheckRedisStatus(ctx context.Context) error {
	err := a.redisRepo.Client.Ping(ctx).Err()
	if err != nil {
		log.Println("failed to connect to server:  %w", err)
		return err
	} else {
		a.redisAvailable = true
		log.Println("redisAvailability:", a.redisAvailable)
		return nil
	}
}
