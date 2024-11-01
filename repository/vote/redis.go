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

// Adding pagination~
type FindAllPage struct {
	Size      uint64
	Offset    uint64
	ChannelID string
}

type FindResult struct {
	Votes  []model.Vote
	Cursor uint64
}

func (r *RedisRepo) FindAll(ctx context.Context, page FindAllPage) (FindResult, error) {
	res := r.Client.SScan(ctx, page.ChannelID, page.Offset, "*", int64(page.Size))

	keys, cursor, err := res.Result()
	if err != nil {
		return FindResult{}, fmt.Errorf("failed to get vote ids: %w", err)
	}
	fmt.Println("keys", keys)

	if len(keys) == 0 {
		return FindResult{
			Votes: []model.Vote{},
		}, nil
	}

	xs, err := r.Client.MGet(ctx, keys...).Result()
	if err != nil {
		return FindResult{}, fmt.Errorf("failed to get votes: %w", err)
	}

	votes := make([]model.Vote, len(xs))

	for i, x := range xs {
		x := x.(string)
		var vote model.Vote

		err := json.Unmarshal([]byte(x), &vote)
		if err != nil {
			return FindResult{}, fmt.Errorf("failed to decode order json: %w", err)
		}

		votes[i] = vote
	}

	return FindResult{
		Votes:  votes,
		Cursor: cursor,
	}, nil
}

// Adds a vote for a specific item in a channel
func (r *RedisRepo) AddVote(ctx context.Context, channelID string, itemID string) {
	timestamp := float64(time.Now().Unix()) // Use Unix timestamp as score
	// ZADD key (channel) with score (timestamp) and value (itemID)
	r.Client.ZAdd(ctx, "votes:"+channelID, redis.Z{
		Score:  timestamp,
		Member: itemID,
	})
	fmt.Println("Vote added for", itemID, "in", channelID)
}

// Gets the most frequent item_id in the last X minutes
func (r *RedisRepo) GetTopVote(ctx context.Context, channelID string, minutes int) string {
	now := time.Now()
	minTimestamp := float64(now.Add(-time.Duration(minutes) * time.Minute).Unix())
	maxTimestamp := float64(now.Unix())

	// Get all votes within the time range
	items, err := r.Client.ZRangeByScoreWithScores(ctx, "votes:"+channelID, &redis.ZRangeBy{
		Min: fmt.Sprintf("%f", minTimestamp),
		Max: fmt.Sprintf("%f", maxTimestamp),
	}).Result()

	if err != nil {
		fmt.Println("Error getting votes:", err)
		return ""
	}

	// Count occurrences of each item_id
	voteCounts := make(map[string]int)
	for _, item := range items {
		fmt.Println("item", item.Member)
		voteCounts[item.Member.(string)]++
	}

	// Find the item_id with the highest count
	var topItemID string
	var maxVotes int
	for itemID, count := range voteCounts {
		if count > maxVotes {
			maxVotes = count
			topItemID = itemID
		}
	}

	return topItemID
}

// Removes votes older than X minutes
func removeOldVotes(ctx context.Context, rdb *redis.Client, channelID string, minutes int) {
	minTimestamp := float64(time.Now().Add(-time.Duration(minutes) * time.Minute).Unix())
	// Remove entries older than the specified timestamp
	rdb.ZRemRangeByScore(ctx, "votes:"+channelID, "0", fmt.Sprintf("%f", minTimestamp))
	fmt.Println("Old votes removed from", channelID)
}

// Adds a vote with Twitch ID for an item in a channel using a hash
func (r *RedisRepo) AddVoteV2(ctx context.Context, channelID string, itemID string, twitchID string) {
	// Increment vote count in a hash
	r.Client.HIncrBy(ctx, "votes:"+channelID, itemID, 1)

	// Store the Twitch ID in a hash for reference (optional)
	r.Client.HSet(ctx, "twitchIDs:"+channelID, itemID, twitchID)

	fmt.Printf("Vote added for item %d by Twitch user %s in channel %s\n", itemID, twitchID, channelID)
}

// Gets the most frequent item_id
func (r *RedisRepo) GetTopVoteV2(ctx context.Context, channelID string) string {
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
	for itemIDStr, _ := range votes {
		count := r.Client.HIncrBy(ctx, "votes:"+channelID, itemIDStr, 0)
		if count.Val() > maxVotes {
			maxVotes = count.Val()
			topItemID = itemIDStr
		}
	}
	return topItemID
}
