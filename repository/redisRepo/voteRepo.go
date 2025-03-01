package redisRepo

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/ruvice/dotabackseaterbackend/model"
	"github.com/ruvice/dotabackseaterbackend/utils/dbsError"
)

func (r *RedisRepo) Insert(ctx context.Context, vote model.Vote) error {
	data, err := json.Marshal(vote)
	if err != nil {
		return fmt.Errorf("failed to encode vote: %w", err)
	}

	// Generating unique key
	key := vote.TwitchID
	channelID := vote.ChannelID

	// Using transaction to make changes atomic
	txn := r.Client.TxPipeline()

	res := txn.SetNX(ctx, key, string(data), 0)
	if err := res.Err(); err != nil {
		txn.Discard()
		return fmt.Errorf("failed to add vote: %w", err)
	}

	if err := txn.SAdd(ctx, channelID, key).Err(); err != nil {
		txn.Discard()
		return fmt.Errorf("failed to add vote set: %w", err)
	}

	if _, err := txn.Exec(ctx); err != nil {
		return fmt.Errorf("failed to exec: %w", err)
	}

	return nil
}

// Adds a vote with Twitch ID for a key in a channel using a hash
func (r *RedisRepo) AddVote(ctx context.Context, key string, id string) {
	// Increment vote count in a hash
	log.Println("Adding vote for", key)
	err := r.Client.ZIncrBy(ctx, key, 1, id).Err()
	if err != nil {
		log.Println("Failed to add vote", err)
	}
}

func (r *RedisRepo) SetExpiry(ctx context.Context, key string, duration time.Duration) error {
	if duration == 0 {
		duration = VoteTTL * time.Second
	}

	err := r.Client.Expire(ctx, key, duration).Err()
	if err != nil {
		return err
	}
	return nil
}

// Gets the most frequent item_id
func (r *RedisRepo) GetMostVoted(ctx context.Context, key string, topN int64) (map[string]int, error) {
	results, err := r.Client.ZRevRangeWithScores(ctx, key, 0, int64(topN-1)).Result()
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		fmt.Printf("No votes recorded for channel %s.\n", key)
		return nil, err
	}

	// Convert results to a structured format
	tally := make(map[string]int)
	for _, result := range results {
		itemID, ok := result.Member.(string)
		if !ok {
			fmt.Println("Error converting Member to string")
			continue
		}
		tally[itemID] = int(result.Score) // Convert float64 to int
	}

	return tally, nil
}

func (r *RedisRepo) IncrementForChannel(ctx context.Context, channelID string) (int64, error) {
	newCount, err := r.Client.Incr(ctx, channelID).Result()
	if err != nil {
		log.Println("Error incrementing votes for channel:", channelID, err)
		return -1, err
	}
	log.Println("Incremented votes for channelID: ", newCount)

	r.Client.Expire(ctx, channelID, VoteTTL*time.Second)
	return newCount, nil
}

func (r *RedisRepo) ClearVoteCountForChannel(ctx context.Context, channelID string) {
	_, err := r.Client.Del(ctx, channelID).Result()
	if err != nil {
		log.Printf("Failed to delete vote counts for channel %s: %v\n", channelID, err)
		return
	}
}

func (r *RedisRepo) ClearVotesForChannel(ctx context.Context, key string) error {
	// Delete the entire hash for the given channelID
	result, err := r.Client.Del(ctx, key).Result()
	if err != nil {
		log.Println("Error clearing votes:", err)
		return err
	}

	// Check if any keys were actually deleted
	if result == 0 {
		log.Printf("No votes found for channel %s\n", key)
	} else {
		log.Printf("Votes cleared for channel %s\n", key)
	}

	return nil
}

func (r *RedisRepo) UpdateLastVotedID(ctx context.Context, key string, id string) error {
	value := id
	err := r.Client.Set(ctx, key, value, LastVotedItemTTL*time.Second).Err()
	if err != nil {
		log.Println("failed to write to Redis with expiry: %w", err)
		voteError := dbsError.NewVoteError("AddVoteRelation", dbsError.CodeUpdateLastVotedError, "Unable to add Last Voted Item", err)
		return voteError
	}

	log.Printf("Successfully set key '%s' with value '%s' and 1 hour expiry\n", key, value)
	return nil
}

func (r *RedisRepo) GetLastVotedItem(ctx context.Context, channelID string) (string, error) {
	key := "lastVotedItem:" + channelID
	value, err := r.Client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			// Key does not exist
			log.Printf("Key '%s' does not exist in Redis\n", key)
			return "", nil
		}

		// Other Redis errors
		log.Println("failed to read from Redis: %w", err)
		getError := dbsError.NewVoteError("GetLastVotedItem", dbsError.CodeRetrieveLastVotedError, "Unable to get Last Voted Item", err)
		return "", getError
	}

	log.Printf("Successfully retrieved key '%s' with value '%s'\n", key, value)
	return value, nil
}

func (r *RedisRepo) GetCurrentCountForChannel(ctx context.Context, channelID string) (int64, error) {
	// Retrieve the value from Redis
	value, err := r.Client.Get(ctx, channelID).Result()
	if err != nil {
		if err == redis.Nil {
			// Key does not exist, treat as zero count
			log.Printf("Key '%s' does not exist in Redis, returning count as 0\n", channelID)
			getError := dbsError.NewVoteError("GetCurrentCountForChannel", dbsError.CodeRetrieveVoteCountNoKey, "Key does not exist", err)
			return 0, getError
		}

		// Other Redis errors
		log.Printf("Error retrieving current count for channel '%s': %v\n", channelID, err)

		getError := dbsError.NewVoteError("GetCurrentCountForChannel", dbsError.CodeRetrieveVoteCountError, "Error retrieving current vote count", err)
		return -1, getError
	}

	// Convert the value to int64
	count, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		log.Printf("Error parsing current count for channel '%s': %v\n", channelID, err)
		parseError := dbsError.NewVoteError("GetCurrentCountForChannel", dbsError.CodeRetrieveVoteCountError, "Error retrieving current vote count", err)
		return -1, parseError
	}

	log.Printf("Current count for channel '%s': %d\n", channelID, count)
	return count, nil
}

func (r *RedisRepo) KeyExists(ctx context.Context, key string) (bool, error) {
	// Check if the key exists in Redis
	exists, err := r.Client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}

	// Exists() returns an integer, 1 means the key exists
	return exists > 0, nil
}
