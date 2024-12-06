package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/ruvice/dotabackseaterbackend/model"
	"github.com/ruvice/dotabackseaterbackend/utils/voteErrors"
)

type RedisRepo struct {
	Client *redis.Client
}

const (
	VoteTTL          = 3600
	VoteRelationTTL  = 10
	VoteThresholdTTL = 604800 // 1 week
	APIBackoffTTL    = 60
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

type FindResult struct {
	Votes  []model.Vote
	Cursor uint64
}

// Adds a vote with Twitch ID for an item in a channel using a hash
func (r *RedisRepo) AddVote(ctx context.Context, channelID string, itemID string, twitchID string) {
	// Increment vote count in a hash
	key := "votes:" + channelID
	r.Client.HIncrBy(ctx, key, itemID, 1)
	r.Client.Expire(ctx, key, VoteTTL*time.Second)

	fmt.Printf("Vote added for item %d by Twitch user %s in channel %s\n", itemID, twitchID, channelID)
}

func (r *RedisRepo) AddVoteRelation(ctx context.Context, channelID string, twitchID string) *voteErrors.VoteError {
	// Set the key with a 30-second expiration
	key := channelID + ":" + twitchID
	value := time.Now()
	err := r.Client.Set(ctx, key, value, VoteRelationTTL*time.Second).Err()
	if err != nil {
		fmt.Println("failed to write to Redis with expiry: %w", err)
		voteError := voteErrors.NewError(voteErrors.CodeVoteRelationCreationError, "Unable to add Vote Relation")
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
		fmt.Println("Unable to get TTL for vote relation: ", err)
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

// Gets the most frequent item_id
func (r *RedisRepo) GetMostVoted(ctx context.Context, channelID string) string {
	// Get all votes from the hash
	votes, err := r.Client.HGetAll(ctx, "votes:"+channelID).Result()
	if err != nil {
		fmt.Println("Error getting votes:", err)
		return ""
	}

	// Find the item_id with the highest vote count
	var topItemID string
	var maxVotes int64

	fmt.Println(votes)
	for item, countStr := range votes {
		count, err := strconv.ParseInt(countStr, 10, 64)
		if err != nil {
			fmt.Printf("Skipping invalid vote count for item %s: %v\n", item, err)
			continue
		}
		if count > maxVotes {
			maxVotes = count
			topItemID = item
		}
	}
	return topItemID
}

func (r *RedisRepo) IncrementForChannel(ctx context.Context, channelID string) (int64, error) {
	newCount, err := r.Client.Incr(ctx, channelID).Result()
	if err != nil {
		fmt.Println("Error incrementing votes for channel:", channelID, err)
		return -1, err
	}
	fmt.Println("Incremented votes for channelID: ", newCount)

	r.Client.Expire(ctx, channelID, VoteTTL*time.Second)
	return newCount, nil
}

func (r *RedisRepo) ClearVoteCountForChannel(ctx context.Context, channelID string) {
	_, err := r.Client.Del(ctx, channelID).Result()
	if err != nil {
		fmt.Printf("Failed to delete vote counts for channel %s: %v\n", channelID, err)
		return
	}
}

func (r *RedisRepo) ClearVotesForChannel(ctx context.Context, channelID string) error {
	// Delete the entire hash for the given channelID
	result, err := r.Client.Del(ctx, "votes:"+channelID).Result()
	if err != nil {
		fmt.Println("Error clearing votes:", err)
		return err
	}

	// Check if any keys were actually deleted
	if result == 0 {
		fmt.Printf("No votes found for channel %s\n", channelID)
	} else {
		fmt.Printf("Votes cleared for channel %s\n", channelID)
	}

	return nil
}

func (r *RedisRepo) UpdateVoteThresholdForChannel(ctx context.Context, channelID string, newThreshold string) error {
	key := "voteThreshold:" + channelID
	err := r.Client.Set(ctx, key, newThreshold, VoteThresholdTTL*time.Second).Err()
	if err != nil {
		fmt.Println("failed to write to Redis with expiry: %w", err)
		voteError := voteErrors.NewError(voteErrors.CodeVoteRelationCreationError, "Unable to add Vote Relation")
		return voteError
	}

	fmt.Printf("Successfully set key '%s' with value '%s' and 1 week expiry\n", key, newThreshold)
	return nil
}

func (r *RedisRepo) GetVoteThreshold(ctx context.Context, channelID string) (string, *voteErrors.VoteError) {
	key := "voteThreshold:" + channelID
	fmt.Println("In Redis GetVoteThreshold", key)
	voteThreshold, err := r.Client.Get(ctx, key).Result()
	if err == redis.Nil {
		fmt.Printf("Could not find key: %v\n", err)
		return "", voteErrors.NewError(voteErrors.CodeMissingCacheVoteThreshold, "Could not find vote threshold cache")
	} else if err != nil {
		fmt.Printf("Error getting key: %v\n", err)
		return "", voteErrors.NewError(voteErrors.CodeUnknown, "Error getting key")
	}
	return voteThreshold, nil
}

// Handling items
func (r *RedisRepo) WriteItemMapToCache(ctx context.Context, itemMap model.ItemMap) {
	// Step 1: Convert itemMap to a slice of Items
	// Sort the 'Items' array by the 'Name' field
	var items []model.Item
	for _, item := range itemMap {
		items = append(items, item)
	}

	// Step 2: Sort the items slice by the Name field in ItemDetail
	sort.Slice(items, func(i, j int) bool {
		// Handle empty or null Name values by treating them as empty strings
		return items[i].Name < items[j].Name
	})

	// Step 3: Marshal the sorted items slice to JSON
	jsonData, err := json.Marshal(items)
	if err != nil {
		fmt.Println("Failed to marshal ItemMap: ", err)
		return
	}

	// Write the JSON string to Redis
	if err := r.Client.Set(ctx, "itemMapCache", jsonData, 0).Err(); err != nil {
		fmt.Println("failed to write to Redis: ", err)
		return
	}

	fmt.Println("ItemMap successfully saved to Redis")
	return
}

func (r *RedisRepo) GetItemMapFromCache(ctx context.Context) (string, *voteErrors.VoteError) {
	// Get the JSON string from Redis
	jsonData, err := r.Client.Get(ctx, "itemMapCache").Result()
	if err != nil {
		fmt.Println("Error getting itemMapCache: ", err)
		voteError := voteErrors.NewError(voteErrors.CodeItemGetRedisError, "Failed to get Item Map for client from Redis")
		return "", voteError
	}

	return jsonData, nil
}

func (r *RedisRepo) CacheItems(ctx context.Context, itemMap model.ItemMap) {
	fmt.Println("Updating Redis Cache with items")
	err := r.clearPreviousItemCache(ctx)
	if err != nil {
		fmt.Println("Failed to clear previous item cache")
	}
	for itemID, itemDetail := range itemMap {
		data, err := json.Marshal(itemDetail)
		if err != nil {
			fmt.Println("Failed to encode ItemDetail:", err)
		}
		// Generating unique key
		key := "itemID:" + itemID

		// Using transaction to make changes atomic
		txn := r.Client.TxPipeline()

		res := txn.Set(ctx, key, string(data), 0)
		if err := res.Err(); err != nil {
			txn.Discard()
			fmt.Println("failed to add item: ", err)
		}

		if _, err := txn.Exec(ctx); err != nil {
			fmt.Println("failed to exec:", err)
		}
	}
}

func (r *RedisRepo) clearPreviousItemCache(ctx context.Context) error {
	var cursor uint64
	var keysToDelete []string
	prefix := "itemID:"

	// Use SCAN to find keys with the specified prefix
	for {
		keys, newCursor, err := r.Client.Scan(ctx, cursor, prefix+"*", 100).Result()
		if err != nil {
			return fmt.Errorf("error scanning keys: %w", err)
		}

		// Collect the keys to delete
		keysToDelete = append(keysToDelete, keys...)
		cursor = newCursor

		// If cursor is 0, the scan is complete
		if cursor == 0 {
			break
		}
	}

	// If there are keys to delete, use DEL command
	if len(keysToDelete) > 0 {
		if err := r.Client.Del(ctx, keysToDelete...).Err(); err != nil {
			return fmt.Errorf("error deleting keys: %w", err)
		}
		fmt.Printf("Deleted %d keys with prefix '%s'\n", len(keysToDelete), prefix)
	} else {
		fmt.Println("No keys found with the specified prefix")
	}

	return nil
}

func (r *RedisRepo) GetItemByID(ctx context.Context, itemID string) model.Item {
	// Retrieve the value for the given key

	data, err := r.Client.Get(ctx, "itemID:"+itemID).Result()
	if err == redis.Nil {
		fmt.Println("Error retrieving itemID from redis: ", err)
		return model.Item{}
	} else if err != nil {
		fmt.Println("Error retrieving itemID from redis: ", err)
		return model.Item{}
	}

	var item model.Item
	// Deserialize the JSON string back to the struct
	if err := json.Unmarshal([]byte(data), &item); err != nil {
		fmt.Println("Failed to unmarshal itemDetail JSON: %w", err)
	}

	return item
}

// Handling too many requests
func (r *RedisRepo) SetTwitchMessageAPITimeout(ctx context.Context, channelID string) *voteErrors.VoteError {
	// Set the key with a 60-second expiration
	key := "timeout:" + channelID
	value := time.Now()
	err := r.Client.Set(ctx, key, value, APIBackoffTTL*time.Second).Err()
	if err != nil {
		fmt.Println("failed to write to Redis with expiry: %w", err)
		voteError := voteErrors.NewError(voteErrors.CodeVoteRelationCreationError, "Unable to add Vote Relation")
		return voteError
	}

	fmt.Printf("Successfully set key '%s' with value '%s' and 30-second expiry\n", key, value)
	return nil
}

func (r *RedisRepo) GetTwitchMessageAPITimeout(ctx context.Context, channelID string) int64 {
	// Set the key with a 60-second expiration
	key := "timeout:" + channelID
	ttl, err := r.Client.TTL(ctx, key).Result()
	if err != nil {
		fmt.Println("Unable to get TTL for vote relation: ", err)
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
