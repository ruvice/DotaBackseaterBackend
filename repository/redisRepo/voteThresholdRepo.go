package redisRepo

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/ruvice/dotabackseaterbackend/utils/voteErrors"
)

func (r *RedisRepo) UpdateVoteThresholdForChannel(ctx context.Context, channelID string, newThreshold string) error {
	key := "voteThreshold:" + channelID
	err := r.Client.Set(ctx, key, newThreshold, VoteThresholdTTL*time.Second).Err()
	if err != nil {
		log.Println("failed to write to Redis with expiry: %w", err)
		voteError := voteErrors.NewError(voteErrors.CodeVoteRelationCreationError, "Unable to add Vote Relation")
		return voteError
	}

	fmt.Printf("Successfully set key '%s' with value '%s' and 1 week expiry\n", key, newThreshold)
	return nil
}

func (r *RedisRepo) GetVoteThreshold(ctx context.Context, channelID string) (string, *voteErrors.VoteError) {
	key := "voteThreshold:" + channelID
	log.Println("In Redis GetVoteThreshold", key)
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
