package model

import (
	"time"
)

type Vote struct {
	ChannelID string     `json:"channel_id,omitempty"`
	TwitchID  string     `json:"twitch_id,omitempty"`
	ItemID    uint64     `json:"item_id,omitempty"`
	VotedAt   *time.Time `json:"voted_at,omitempty"`
}
