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

	key := "votedHero:" + VoteHeroBody.ChannelID
	sessionKey := "voteHeroSession:" + VoteHeroBody.ChannelID
	// Check if key exists - if it does then there's an ongoing vote session
	hasActiveVoteSession := h.hasActiveVoteSession(r.Context(), sessionKey)
	if !hasActiveVoteSession {
		log.Println("Received vote with no active vote session")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// Handling vote relation
	// voteRelationKey := "voteRelation:" + VoteHeroBody.ChannelID + ":" + VoteHeroBody.TwitchID
	// hasVoteRelation := h.Redis.GetHeroVoteRelation(r.Context(), voteRelationKey)
	// if hasVoteRelation {
	// 	log.Println("User already voted")
	// 	w.WriteHeader(http.StatusBadRequest)
	// 	return
	// }
	// voteError := h.Redis.AddHeroVoteRelation(r.Context(), voteRelationKey)
	// if voteError != nil {
	// 	log.Println(voteError)
	// 	w.WriteHeader(http.StatusInternalServerError)
	// 	return
	// }
	// Adding actual vote
	h.Redis.AddVote(r.Context(), key, VoteHeroBody.HeroID)
	topVotes, err := h.Redis.GetMostVoted(r.Context(), key, 10)
	if err != nil {
		log.Println("Failed to retrieve most voted heroes", err)
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
	if channelID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&VoteStartBody); err != nil {
		log.Println("body fked up")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

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
	sessionKey := "voteHeroSession:" + channelID
	log.Println("Adding sessionkey", sessionKey)
	err = h.Redis.Client.SetEx(r.Context(), sessionKey, "started", duration).Err()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("Failed to setup vote session with expiry")
		return
	}
	hasActiveVoteSession, err := h.Redis.KeyExists(r.Context(), sessionKey)
	if err != nil {
		log.Println("Eror getting active vote session", err)
		return
	}
	log.Println(hasActiveVoteSession)
	time.AfterFunc(duration, func() {
		h.stopVote(context.Background(), channelID)
	})

	fmt.Printf("Voting session started for channel %s, expires in %v seconds.\n", channelID, duration.Seconds())
	w.WriteHeader(http.StatusOK)
	SSEPushChannel <- SSEPushRequest{SSEMessage: SSEMessage{EventType: "voteSession", Data: "started"}, ChannelID: channelID}
}

func (h *Vote) StopHeroVote(w http.ResponseWriter, r *http.Request) {
	log.Println("Stopping Vote")
	channelID := r.Header.Get("Channel-Id")
	if channelID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	h.stopVote(r.Context(), channelID)
}

func (h *Vote) stopVote(ctx context.Context, channelID string) {
	log.Println("Stopping Vote")
	sessionKey := "voteHeroSession:" + channelID
	err := h.Redis.Client.Del(ctx, sessionKey).Err()
	if err != nil {
		log.Printf("Failed to delete session key %s: %v\n", sessionKey, err)
	}

	key := "votedHero:" + channelID
	topVotes, err := h.Redis.GetMostVoted(ctx, key, 10)
	if err != nil {
		log.Println("Failed to retrieve most voted heroes")
	}
	message := ""
	for id, votes := range topVotes {
		votedHero := h.Redis.GetHeroByID(ctx, id)
		message += fmt.Sprintf("%s: %d votes\n", votedHero.Name, votes)
	}

	var twitchMessage = wrapper.TwitchMessage{
		Message:   message,
		ChannelID: channelID,
	}
	timeout := h.Redis.GetTwitchMessageAPITimeout(ctx, channelID)
	if timeout < 0 {
		err := h.TwitchWrapper.SendMessage(twitchMessage)
		if vErr := new(dbsError.VoteError); errors.As(err, &vErr) {
			if vErr.Code == dbsError.CodeTwitchMessageTooManyRequests {
				log.Println("Too many requests error: %w", vErr)
				h.handleVoteMessageTooManyRequests(ctx, channelID)
			}
		}
	}
	err = h.Redis.Client.Del(ctx, key).Err()
	if err != nil {
		log.Printf("Failed to delete key %s: %v\n", key, err)
	}

	voteRelationKey := "voteRelation:" + VoteHeroBody.ChannelID + ":"
	err = h.Redis.ClearHeroVoteRelation(context.Background(), voteRelationKey)
	if err != nil {
		log.Println("Failed to clear hero vote relation", err)
	}

	jsonData, err := json.Marshal(topVotes)
	if err != nil {
		log.Println("Error encoding JSON:", err)
		return
	}

	h.Redis.UpdateLastVotedID(ctx, "lastVotedHeroMap:"+channelID, string(jsonData), 120*time.Second)

	SSEPushChannel <- SSEPushRequest{SSEMessage: SSEMessage{EventType: "votedHero", Data: string(jsonData)}, ChannelID: channelID}
	SSEPushChannel <- SSEPushRequest{SSEMessage: SSEMessage{EventType: "voteSession", Data: "stopped"}, ChannelID: channelID}
}

func (h *Vote) GetExtensionHeroVoteStatus(w http.ResponseWriter, r *http.Request) {
	channelID := r.Header.Get("Channel-Id")
	twitchID := r.Header.Get("Twitch-Id")
	if channelID == "" {
		log.Println("Missing channelID in URL")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if twitchID == "" {
		log.Println("Missing twitchID in URL")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	key := "votedHero:" + channelID
	sessionKey := "voteHeroSession:" + channelID
	// Check if key exists - if it does then there's an ongoing vote sessio
	hasActiveVoteSession := h.hasActiveVoteSession(r.Context(), sessionKey)
	topVotes, err := h.Redis.GetMostVoted(r.Context(), key, 10)
	if err != nil {
		log.Println("Failed to retrieve most voted heroes")
	}

	if len(topVotes) == 0 {
		// Use last HeroVoteMap instead (if any)
		key := "lastVotedHeroMap:" + channelID
		res, err := h.Redis.GetLastVotedID(r.Context(), key)
		if err != nil {
			log.Println("Failed to retrieve last HeroVoteMap")
		} else {
			var lastHeroVoteMap map[string]int
			err = json.Unmarshal([]byte(res), &lastHeroVoteMap)
			if err != nil {
				log.Println("Failed to parse last HeroVoteMap:", err)
			} else if !hasActiveVoteSession {
				topVotes = lastHeroVoteMap
			}
		}
	}

	var response struct {
		HeroVoteMap          map[string]int `json:"hero_vote_map"`
		HasActiveVoteSession bool           `json:"has_active_vote_session"`
		HasVoted             bool           `json:"has_voted"`
	}

	voteRelationKey := "voteRelation:" + channelID + ":" + twitchID
	hasVoted := h.Redis.GetHeroVoteRelation(r.Context(), voteRelationKey)

	response.HeroVoteMap = topVotes
	response.HasVoted = hasVoted
	response.HasActiveVoteSession = hasActiveVoteSession
	data, err := json.Marshal(response)
	log.Println("ANDREW WTF", response, data)
	if err != nil {
		log.Println("failed to marshal:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func (h *Vote) hasActiveVoteSession(ctx context.Context, key string) bool {
	hasActiveVoteSession, err := h.Redis.KeyExists(ctx, key)
	if err != nil {
		return false
	}
	return hasActiveVoteSession
}
