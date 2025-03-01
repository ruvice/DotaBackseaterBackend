package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/ruvice/dotabackseaterbackend/utils/dbsError"
	"github.com/ruvice/dotabackseaterbackend/wrapper"
)

var VoteHeroBody struct {
	ChannelID string `json:"channel_id,omitempty"`
	TwitchID  string `json:"twitch_id,omitempty"`
	HeroID    string `json:"hero_id,omitempty"`
}

var VoteStartBody struct {
	Duration string `json:"duration,omitempty"`
}

func (h *Vote) VoteHero(w http.ResponseWriter, r *http.Request) {
	log.Println("New vote")

	if err := json.NewDecoder(r.Body).Decode(&VoteHeroBody); err != nil {
		log.Println("body fked up")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	headerChannelID := r.Header.Get("Channel-Id")
	if headerChannelID != "" {
		VoteHeroBody.ChannelID = headerChannelID
	}

	key := "votesHero:" + VoteHeroBody.ChannelID
	h.Redis.AddVote(r.Context(), key, VoteHeroBody.HeroID)
	topVotes, err := h.Redis.GetMostVoted(r.Context(), VoteHeroBody.ChannelID, "Hero", 5)
	if err != nil {
		log.Println("Failed to retrieve most voted heroes")
	}
	jsonData, err := json.Marshal(topVotes)
	if err != nil {
		log.Println("Error encoding JSON:", err)
		return
	}

	SSEPushChannel <- SSEPushRequest{SSEMessage: SSEMessage{EventType: "votedHero", Data: string(jsonData)}, ChannelID: VoteHeroBody.ChannelID}
	w.WriteHeader(http.StatusCreated)
}

func (h *Vote) StartHeroVote(w http.ResponseWriter, r *http.Request) {
	log.Println("Starting Vote")
	channelID := r.Header.Get("Channel-Id")
	log.Println(channelID)
	if channelID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&VoteStartBody); err != nil {
		log.Println("body fked up")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	key := fmt.Sprintf("votesHero:%s", channelID)
	durationInt, err := strconv.Atoi(VoteStartBody.Duration)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println("Error converting duration to int", err)
		return
	}
	if durationInt < 10 {
		w.WriteHeader(http.StatusBadRequest)
		log.Println("Duration under 10 seconds")
		return
	}
	duration := time.Duration(durationInt) * time.Second

	// Set expiration on the key
	err = h.Redis.SetExpiry(r.Context(), key, duration)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("Failed to setup vote session with expiry")
		return
	}
	time.AfterFunc(duration, func() {
		h.stopVote(context.Background(), channelID)
	})

	fmt.Printf("Voting session started for channel %s, expires in %v seconds.\n", channelID, duration.Seconds())
	w.WriteHeader(http.StatusOK)
}

func (h *Vote) StopHeroVote(w http.ResponseWriter, r *http.Request) {
	log.Println("Stopping Vote")
	channelID := r.Header.Get("Channel-Id")
	if channelID == "" {
		VoteHeroBody.ChannelID = channelID
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	h.stopVote(r.Context(), channelID)
}

func (h *Vote) stopVote(ctx context.Context, channelID string) {
	log.Println("Stopping Vote")

	topVotes, err := h.Redis.GetMostVoted(ctx, channelID, "Hero", 5)
	if err != nil {
		log.Println("Failed to retrieve most voted heroes")
	}
	message := ""
	for id, votes := range topVotes {
		votedHero := h.Redis.GetHeroByID(ctx, id)
		message += fmt.Sprintf("%s: %d votes", votedHero.Name, votes)
	}

	var twitchMessage = wrapper.TwitchMessage{
		Message:   message,
		ChannelID: VoteHeroBody.ChannelID,
	}
	timeout := h.Redis.GetTwitchMessageAPITimeout(ctx, VoteHeroBody.ChannelID)
	if timeout < 0 {
		err := h.TwitchWrapper.SendMessage(twitchMessage)
		if vErr := new(dbsError.VoteError); errors.As(err, &vErr) {
			if vErr.Code == dbsError.CodeTwitchMessageTooManyRequests {
				log.Println("Too many requests error: %w", vErr)
				h.handleVoteMessageTooManyRequests(ctx, VoteHeroBody.ChannelID)
			}
		}
	}
	jsonData, err := json.Marshal(topVotes)
	if err != nil {
		log.Println("Error encoding JSON:", err)
		return
	}

	SSEPushChannel <- SSEPushRequest{SSEMessage: SSEMessage{EventType: "votedHero", Data: jsonData}, ChannelID: VoteHeroBody.ChannelID}
}
