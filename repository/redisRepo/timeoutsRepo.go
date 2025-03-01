package redisRepo

import (
	"context"
	"log"
	"time"

	"github.com/ruvice/dotabackseaterbackend/utils/dbsError"
)

func (r *RedisRepo) AddHeroVoteRelation(ctx context.Context, voteRelationKey string) error {
	// Set the key with a 30-second expiration
	value := time.Now()
	err := r.Client.Set(ctx, voteRelationKey, value, 0).Err()
	if err != nil {
		log.Println("Failed to set hero vote relation: %w", err)
		voteError := dbsError.NewVoteError("AddHeroVoteRelation", dbsError.CodeVoteRelationCreationError, "Unable to add Hero Vote Relation", err)
		return voteError
	}

	log.Printf("Successfully set key '%s' with value '%s' and 30-second expiry\n", voteRelationKey, value)
	return nil
}

func (r *RedisRepo) GetHeroVoteRelation(ctx context.Context, voteRelationKey string) bool {
	// Retrieve the value for the given key
	_, err := r.Client.Get(ctx, voteRelationKey).Result()
	if err != nil {
		log.Println("Error getting vote relation: ", err)
		return false
	}
	// Return the TTL in seconds
	return true
}

func (r *RedisRepo) ClearHeroVoteRelation(ctx context.Context, prefix string) error {
	var cursor uint64 = 0
	var keys []string
	var err error
	for {
		var result []string
		result, cursor, err = r.Client.Scan(ctx, cursor, prefix+"*", 100).Result()
		if err != nil {
			return err
		}

		keys = append(keys, result...)

		if cursor == 0 { // No more keys to scan
			break
		}
	}

	if len(keys) == 0 {
		log.Println("No matching keys found.")
		return nil
	}

	// Delete all matching keys
	err = r.Client.Del(ctx, keys...).Err()
	if err != nil {
		return err
	}

	log.Printf("Deleted %d keys with prefix '%s'\n", len(keys), prefix)
	return nil
}

func (r *RedisRepo) AddItemVoteRelation(ctx context.Context, voteRelationKey string) error {
	// Set the key with a 30-second expiration
	value := time.Now()
	err := r.Client.Set(ctx, voteRelationKey, value, VoteRelationTTL*time.Second).Err()
	if err != nil {
		log.Println("failed to write to Redis with expiry: %w", err)
		voteError := dbsError.NewVoteError("AddItemVoteRelation", dbsError.CodeVoteRelationCreationError, "Unable to add Item Vote Relation", err)
		return voteError
	}

	log.Printf("Successfully set key '%s' with value '%s' and 30-second expiry\n", voteRelationKey, value)
	return nil
}

func (r *RedisRepo) GetItemVoteRelationTTL(ctx context.Context, voteRelationKey string) int64 {
	// Retrieve the value for the given key
	ttl, err := r.Client.TTL(ctx, voteRelationKey).Result()
	if err != nil {
		log.Println("Unable to get TTL for vote relation: ", err)
		return 0
	}
	// Check the TTL value
	if ttl == -1 {
		log.Printf("Key '%s' does not have an expiry set\n", voteRelationKey)
		return -1
	} else if ttl == -2 {
		log.Printf("Key '%s' does not exist\n", voteRelationKey)
		return -2
	}

	// Return the TTL in seconds
	return int64(ttl.Seconds())
}

// Handling too many requests
func (r *RedisRepo) SetTwitchMessageAPITimeout(ctx context.Context, channelID string) error {
	// Set the key with a 60-second expiration
	key := "timeout:" + channelID
	value := time.Now()
	err := r.Client.Set(ctx, key, value, APIBackoffTTL*time.Second).Err()
	if err != nil {
		log.Println("failed to write to Redis with expiry: %w", err)
		voteError := dbsError.NewVoteError("SetTwitchMessageAPITimeout", dbsError.CodeVoteRelationCreationError, "Unable to add Vote Relation", err)
		return voteError
	}

	log.Printf("Successfully set key '%s' with value '%s' and 30-second expiry\n", key, value)
	return nil
}

func (r *RedisRepo) GetTwitchMessageAPITimeout(ctx context.Context, channelID string) int64 {
	// Set the key with a 60-second expirations
	key := "timeout:" + channelID
	ttl, err := r.Client.TTL(ctx, key).Result()
	if err != nil {
		log.Println("Unable to get TTL for vote relation: ", err)
		return 0
	}
	// Check the TTL value
	if ttl == -1 {
		log.Printf("Key '%s' does not have an expiry set\n", key)
		return -1
	} else if ttl == -2 {
		log.Printf("Key '%s' does not exist\n", key)
		return -2
	}

	// Return the TTL in seconds
	return int64(ttl.Seconds())
}
