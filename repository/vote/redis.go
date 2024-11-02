package vote

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/ruvice/dotabackseaterbackend/model"
)

type RedisRepo struct {
	Client *redis.Client
}

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

// Removes votes older than X minutes
func removeOldVotes(ctx context.Context, rdb *redis.Client, channelID string, minutes int) {
	minTimestamp := float64(time.Now().Add(-time.Duration(minutes) * time.Minute).Unix())
	// Remove entries older than the specified timestamp
	rdb.ZRemRangeByScore(ctx, "votes:"+channelID, "0", fmt.Sprintf("%f", minTimestamp))
	fmt.Println("Old votes removed from", channelID)
}

// Adds a vote with Twitch ID for an item in a channel using a hash
func (r *RedisRepo) AddVote(ctx context.Context, channelID string, itemID string, twitchID string) {
	// Increment vote count in a hash
	r.Client.HIncrBy(ctx, "votes:"+channelID, itemID, 1)

	// Store the Twitch ID in a hash for reference (optional)
	r.Client.HSet(ctx, "twitchIDs:"+channelID, itemID, twitchID)

	fmt.Printf("Vote added for item %d by Twitch user %s in channel %s\n", itemID, twitchID, channelID)
}

// Gets the most frequent item_id
func (r *RedisRepo) GetTopVote(ctx context.Context, channelID string) model.Item {
	// Get all votes from the hash
	votes, err := r.Client.HGetAll(ctx, "votes:"+channelID).Result()
	var votedItem model.Item
	if err != nil {
		fmt.Println("Error getting votes:", err)
		return model.Item{}
	}

	// Find the item_id with the highest vote count
	var topItemID string
	var maxVotes int64

	fmt.Println(votes)
	for itemIDStr, _ := range votes {
		count := r.Client.HIncrBy(ctx, "votes:"+channelID, itemIDStr, 0)
		if count.Val() > maxVotes {
			maxVotes = count.Val()
			topItemID = itemIDStr
		}
	}
	votedItem.ItemID = topItemID

	return votedItem
}

func (r *RedisRepo) IncrementForChannel(ctx context.Context, channelID string) (int64, error) {
	newCount, err := r.Client.Incr(ctx, channelID).Result()
	if err != nil {
		fmt.Println("Error incrementing votes for channel:", channelID, err)
		return -1, err
	}
	fmt.Println("Incremented votes for channelID: ", newCount)
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
