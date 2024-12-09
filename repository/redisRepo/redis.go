package redisRepo

import (
	"github.com/redis/go-redis/v9"
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
