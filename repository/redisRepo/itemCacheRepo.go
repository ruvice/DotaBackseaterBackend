package redisRepo

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"

	"github.com/redis/go-redis/v9"
	"github.com/ruvice/dotabackseaterbackend/model"
	"github.com/ruvice/dotabackseaterbackend/utils/DBSError"
)

// Handling items
func (r *RedisRepo) WriteItemMapToCache(ctx context.Context, itemMap model.ItemMap) {
	var items []model.Item
	for _, item := range itemMap {
		items = append(items, item)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})

	// Step 3: Marshal the sorted items slice to JSON
	jsonData, err := json.Marshal(items)
	if err != nil {
		log.Println("Failed to marshal ItemMap: ", err)
		return
	}

	// Write the JSON string to Redis
	if err := r.Client.Set(ctx, "itemMapCache", jsonData, 0).Err(); err != nil {
		log.Println("failed to write to Redis: ", err)
		return
	}

	log.Println("ItemMap successfully saved to Redis")
	return
}

func (r *RedisRepo) GetItemMapFromCache(ctx context.Context) (string, *DBSError.VoteError) {
	jsonData, err := r.Client.Get(ctx, "itemMapCache").Result()
	if err != nil {
		log.Println("Error getting itemMapCache: ", err)
		voteError := DBSError.NewError(DBSError.CodeItemGetRedisError, "Failed to get Item Map for client from Redis")
		return "", voteError
	}

	return jsonData, nil
}

func (r *RedisRepo) CacheItems(ctx context.Context, itemMap model.ItemMap) {
	log.Println("Updating Redis Cache with items")
	err := r.clearPreviousItemCache(ctx)
	if err != nil {
		log.Println("Failed to clear previous item cache")
	}
	for itemID, itemDetail := range itemMap {
		data, err := json.Marshal(itemDetail)
		if err != nil {
			log.Println("Failed to encode ItemDetail:", err)
		}
		key := "itemID:" + itemID

		txn := r.Client.TxPipeline()
		res := txn.Set(ctx, key, string(data), 0)
		if err := res.Err(); err != nil {
			txn.Discard()
			log.Println("failed to add item: ", err)
		}

		if _, err := txn.Exec(ctx); err != nil {
			log.Println("failed to exec:", err)
		}
	}
}

func (r *RedisRepo) clearPreviousItemCache(ctx context.Context) error {
	var cursor uint64
	var keysToDelete []string
	prefix := "itemID:"

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
		fmt.Printf("Deleted %d keys with prefix '%s'\n", len(keysToDelete), prefix)
	} else {
		log.Println("No keys found with the specified prefix")
	}

	return nil
}

func (r *RedisRepo) GetItemByID(ctx context.Context, itemID string) model.Item {
	// Retrieve the value for the given key

	data, err := r.Client.Get(ctx, "itemID:"+itemID).Result()
	if err == redis.Nil {
		log.Println("Error retrieving itemID from redis: ", err)
		return model.Item{}
	} else if err != nil {
		log.Println("Error retrieving itemID from redis: ", err)
		return model.Item{}
	}

	var item model.Item
	// Deserialize the JSON string back to the struct
	if err := json.Unmarshal([]byte(data), &item); err != nil {
		log.Println("Failed to unmarshal itemDetail JSON: %w", err)
	}

	return item
}
