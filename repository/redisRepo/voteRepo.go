package redisRepo

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/ruvice/dotabackseaterbackend/model"
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

// Adds a vote with Twitch ID for an item in a channel using a hash
func (r *RedisRepo) AddVote(ctx context.Context, channelID string, itemID string, twitchID string) {
	// Increment vote count in a hash
	key := "votes:" + channelID
	r.Client.HIncrBy(ctx, key, itemID, 1)
	r.Client.Expire(ctx, key, VoteTTL*time.Second)

	fmt.Printf("Vote added for item %d by Twitch user %s in channel %s\n", itemID, twitchID, channelID)
}

// Gets the most frequent item_id
func (r *RedisRepo) GetMostVoted(ctx context.Context, channelID string) string {
	// Get all votes from the hash
	votes, err := r.Client.HGetAll(ctx, "votes:"+channelID).Result()
	if err != nil {
		log.Println("Error getting votes:", err)
		return ""
	}

	// Find the item_id with the highest vote count
	var topItemID string
	var maxVotes int64

	log.Println(votes)
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
		fmt.Printf("Failed to delete vote counts for channel %s: %v\n", channelID, err)
		return
	}
}

func (r *RedisRepo) ClearVotesForChannel(ctx context.Context, channelID string) error {
	// Delete the entire hash for the given channelID
	result, err := r.Client.Del(ctx, "votes:"+channelID).Result()
	if err != nil {
		log.Println("Error clearing votes:", err)
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
