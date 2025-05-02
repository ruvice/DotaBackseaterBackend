package handler

import (
	"github.com/ruvice/dotabackseaterbackend/repository/redisRepo"
	"github.com/ruvice/dotabackseaterbackend/wrapper"
)

type Vote struct {
	Redis          *redisRepo.RedisRepo
	TwitchWrapper  *wrapper.TwitchWrapper
	Broadcaster    Broadcaster
	SessionManager *VoteSessionManager
}
