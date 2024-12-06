package redisRepo

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ruvice/dotabackseaterbackend/utils/DBSError"
)

func (r *RedisRepo) AddVoteRelation(ctx context.Context, channelID string, twitchID string) *DBSError.VoteError {
	// Set the key with a 30-second expiration
	key := channelID + ":" + twitchID
	value := time.Now()
	err := r.Client.Set(ctx, key, value, VoteRelationTTL*time.Second).Err()
	if err != nil {
		log.Println("failed to write to Redis with expiry: %w", err)
		voteError := DBSError.NewError(DBSError.CodeVoteRelationCreationError, "Unable to add Vote Relation")
		return voteError
	}

	fmt.Printf("Successfully set key '%s' with value '%s' and 30-second expiry\n", key, value)
	return nil
}

func (r *RedisRepo) GetVoteRelationTTL(ctx context.Context, channelID string, twitchID string) int64 {
	// Retrieve the value for the given key
	key := channelID + ":" + twitchID
	ttl, err := r.Client.TTL(ctx, key).Result()
	if err != nil {
		log.Println("Unable to get TTL for vote relation: ", err)
		return 0
	}
	// Check the TTL value
	if ttl == -1 {
		fmt.Printf("Key '%s' does not have an expiry set\n", key)
		return -1
	} else if ttl == -2 {
		fmt.Printf("Key '%s' does not exist\n", key)
		return -2
	}

	// Return the TTL in seconds
	return int64(ttl.Seconds())
}

// Handling too many requests
func (r *RedisRepo) SetTwitchMessageAPITimeout(ctx context.Context, channelID string) *DBSError.VoteError {
	// Set the key with a 60-second expiration
	key := "timeout:" + channelID
	value := time.Now()
	err := r.Client.Set(ctx, key, value, APIBackoffTTL*time.Second).Err()
	if err != nil {
		log.Println("failed to write to Redis with expiry: %w", err)
		voteError := DBSError.NewError(DBSError.CodeVoteRelationCreationError, "Unable to add Vote Relation")
		return voteError
	}

	fmt.Printf("Successfully set key '%s' with value '%s' and 30-second expiry\n", key, value)
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
		fmt.Printf("Key '%s' does not have an expiry set\n", key)
		return -1
	} else if ttl == -2 {
		fmt.Printf("Key '%s' does not exist\n", key)
		return -2
	}

	// Return the TTL in seconds
	return int64(ttl.Seconds())
}
