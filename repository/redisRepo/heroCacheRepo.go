package redisRepo

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"

	"github.com/redis/go-redis/v9"
	"github.com/ruvice/dotabackseaterbackend/model"
	"github.com/ruvice/dotabackseaterbackend/utils/dbsError"
)

// Handling heroes
func (r *RedisRepo) WriteHeroMapToCache(ctx context.Context, heroMap model.HeroMap) {
	var heroes []model.Hero
	for _, hero := range heroMap {
		heroes = append(heroes, hero)
	}

	sort.Slice(heroes, func(i, j int) bool {
		return heroes[i].Name < heroes[j].Name
	})

	jsonData, err := json.Marshal(heroes)
	if err != nil {
		log.Println("Failed to marshal HeroMap: ", err)
		return
	}

	// Write the JSON string to Redis
	if err := r.Client.Set(ctx, "heroMapCache", jsonData, 0).Err(); err != nil {
		log.Println("failed to write to Redis: ", err)
		return
	}

	log.Println("HeroMap successfully saved to Redis")
	return
}

func (r *RedisRepo) GetHeroMapFromCache(ctx context.Context) (string, error) {
	jsonData, err := r.Client.Get(ctx, "heroMapCache").Result()
	if err != nil {
		log.Println("Error getting heroMapCache: ", err)
		voteError := dbsError.NewVoteError("GetHeroMapFromCache", dbsError.CodeHeroGetRedisError, "Failed to get Hero Map for client from Redis", err)
		return "", voteError
	}

	return jsonData, nil
}

func (r *RedisRepo) CacheHeroes(ctx context.Context, heroMap model.HeroMap) {
	log.Println("Updating Redis Cache with heroes")
	err := r.clearPreviousHeroCache(ctx)
	if err != nil {
		log.Println("Failed to clear previous hero cache")
	}
	for heroID, heroDetail := range heroMap {
		data, err := json.Marshal(heroDetail)
		if err != nil {
			log.Println("Failed to encode HeroDetail:", err)
		}
		key := "heroID:" + heroID

		txn := r.Client.TxPipeline()
		res := txn.Set(ctx, key, string(data), 0)
		if err := res.Err(); err != nil {
			txn.Discard()
			log.Println("failed to add hero: ", err)
		}

		if _, err := txn.Exec(ctx); err != nil {
			log.Println("failed to exec:", err)
		}
	}
}

func (r *RedisRepo) clearPreviousHeroCache(ctx context.Context) error {
	var cursor uint64
	var keysToDelete []string
	prefix := "heroID:"

	for {
		keys, newCursor, err := r.Client.Scan(ctx, cursor, prefix+"*", 100).Result()
		if err != nil {
			return fmt.Errorf("error scanning keys: %w", err)
		}

		keysToDelete = append(keysToDelete, keys...)
		cursor = newCursor

		if cursor == 0 {
			break
		}
	}

	if len(keysToDelete) > 0 {
		if err := r.Client.Del(ctx, keysToDelete...).Err(); err != nil {
			return fmt.Errorf("error deleting keys: %w", err)
		}
		log.Printf("Deleted %d keys with prefix '%s'\n", len(keysToDelete), prefix)
	} else {
		log.Println("No keys found with the specified prefix")
	}

	return nil
}

func (r *RedisRepo) GetHeroByID(ctx context.Context, heroID string) model.Hero {
	// Retrieve the value for the given key

	data, err := r.Client.Get(ctx, "heroID:"+heroID).Result()
	if err == redis.Nil {
		log.Println("Error retrieving heroID from redis: ", err)
		return model.Hero{}
	} else if err != nil {
		log.Println("Error retrieving heroID from redis: ", err)
		return model.Hero{}
	}

	var hero model.Hero
	// Deserialize the JSON string back to the struct
	if err := json.Unmarshal([]byte(data), &hero); err != nil {
		log.Println("Failed to unmarshal heroDetail JSON: %w", err)
	}

	return hero
}
