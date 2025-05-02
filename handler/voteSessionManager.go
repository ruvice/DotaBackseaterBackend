package handler

import (
	"context"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// VoteSessionManager manages vote session timers and cleanup logic
type VoteSessionManager struct {
	lock       sync.Mutex
	timers     map[string]*time.Timer
	onExpire   func(channelID string)
	redis      *redis.Client
	keyPrefix  string
	expiryTime time.Duration
}

// NewVoteSessionManager creates a new instance
func NewVoteSessionManager(
	redis *redis.Client,
	keyPrefix string,
	onExpire func(channelID string),
) *VoteSessionManager {
	return &VoteSessionManager{
		timers:    make(map[string]*time.Timer),
		onExpire:  onExpire,
		redis:     redis,
		keyPrefix: keyPrefix,
	}
}

// Start starts or resets a vote session timer for a channel
func (v *VoteSessionManager) Start(channelID string, duration time.Duration) {
	v.lock.Lock()
	defer v.lock.Unlock()

	// Set Redis key
	key := v.keyPrefix + channelID
	v.redis.Set(context.Background(), key, "started", duration)

	// Stop existing timer if present
	if timer, exists := v.timers[channelID]; exists {
		timer.Stop()
	}

	// Create new timer
	v.timers[channelID] = time.AfterFunc(duration, func() {
		v.lock.Lock()
		delete(v.timers, channelID)
		v.lock.Unlock()

		// Cleanup Redis
		v.redis.Del(context.Background(), key)

		// Trigger expiration logic
		v.onExpire(channelID)
	})
}

// Stop immediately stops and removes the vote session timer
func (v *VoteSessionManager) Stop(channelID string) {
	v.lock.Lock()
	defer v.lock.Unlock()

	key := v.keyPrefix + channelID
	v.redis.Del(context.Background(), key)

	if timer, exists := v.timers[channelID]; exists {
		timer.Stop()
		delete(v.timers, channelID)
	}
}

func (v *VoteSessionManager) HasActive(ctx context.Context, channelID string) bool {
	key := v.keyPrefix + channelID
	exists, err := v.redis.Exists(ctx, key).Result()
	return err == nil && exists == 1
}
