package redisRepo

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/ruvice/dotabackseaterbackend/utils/dbsError"
)

func (r *RedisRepo) UpdateVoteThresholdForChannel(ctx context.Context, channelID string, newThreshold string) error {
	key := "voteThreshold:" + channelID
	err := r.Client.Set(ctx, key, newThreshold, VoteThresholdTTL*time.Second).Err()
	if err != nil {
		log.Println("failed to write to Redis with expiry: %w", err)
		voteError := dbsError.NewVoteError("UpdateVoteThresholdForChannel", dbsError.CodeVoteRelationCreationError, "unable to add vote relation", err)
		return voteError
	}

	log.Printf("Successfully set key '%s' with value '%s' and 1 week expiry\n", key, newThreshold)
	return nil
}

func (r *RedisRepo) GetVoteThreshold(ctx context.Context, channelID string) (string, error) {
	key := "voteThreshold:" + channelID
	log.Println("In Redis GetVoteThreshold", key)
	voteThreshold, err := r.Client.Get(ctx, key).Result()
	if err == redis.Nil {
		log.Printf("Could not find key: %v\n", err)
		return "", dbsError.NewVoteError("GetVoteThreshold", dbsError.CodeMissingCacheVoteThreshold, "could not find vote threshold cache", err)
	} else if err != nil {
		log.Printf("Error getting key: %v\n", err)
		return "", dbsError.NewVoteError("GetVoteThreshold", dbsError.CodeMissingCacheVoteThreshold, "error getting key", err)
	}
	return voteThreshold, nil
}
