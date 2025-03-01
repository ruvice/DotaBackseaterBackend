package application

import (
	"context"
	"log"
)

func (a *App) PerformInitTasks(ctx context.Context) error {
	itemMap, err := a.mongoDB.RefreshItems(ctx)
	if err != nil {
		return err
	}
	a.redisRepo.WriteItemMapToCache(ctx, itemMap)
	a.redisRepo.CacheItems(ctx, itemMap)

	heroMap, err := a.mongoDB.RefreshHeroes(ctx)
	if err != nil {
		return err
	}
	a.redisRepo.WriteHeroMapToCache(ctx, heroMap)
	a.redisRepo.CacheHeroes(ctx, heroMap)
	return nil
}

func (a *App) Cleanup(ctx context.Context) {
	log.Println("Cleaning up resources")
	if err := a.redisRepo.Client.Close(); err != nil {
		log.Printf("Failed to close Redis: %v", err)
	}

	if err := a.mongoDB.Client.Disconnect(ctx); err != nil {
		log.Printf("Failed to disconnect MongoDB: %v", err)
	}
}
