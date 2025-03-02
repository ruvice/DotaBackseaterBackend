package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/ruvice/dotabackseaterbackend/model"
	"github.com/ruvice/dotabackseaterbackend/utils/dbsError"
	"github.com/ruvice/dotabackseaterbackend/wrapper"
)

const (
	VoteThreshold = 10
)

var VoteItemBody struct {
	ChannelID string `json:"channel_id,omitempty"`
	TwitchID  string `json:"twitch_id,omitempty"`
	ItemID    string `json:"item_id,omitempty"`
}

func (h *Vote) VoteItem(w http.ResponseWriter, r *http.Request) {
	log.Println("New vote")

	if err := json.NewDecoder(r.Body).Decode(&VoteItemBody); err != nil {
		log.Println("body fked up")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// Preferentially use channel_id from header for newer versions
	// TODO: Deprecate old channel_id in request body
	headerChannelID := r.Header.Get("Channel-Id")
	if headerChannelID != "" {
		VoteItemBody.ChannelID = headerChannelID
	}

	ttl := h.Redis.GetVoteRelationTTL(r.Context(), VoteItemBody.ChannelID, VoteItemBody.TwitchID)
	if ttl <= 0 {
		voteError := h.Redis.AddVoteRelation(r.Context(), VoteItemBody.ChannelID, VoteItemBody.TwitchID)
		if voteError != nil {
			log.Println(voteError)
		}
	} else {
		w.Header().Set("Access-Control-Expose-Headers", "Retry-After") // Expose Retry-After header
		// Set the `Retry-After` header to indicate the backoff period in seconds
		w.Header().Set("Retry-After", fmt.Sprintf("%d", int(ttl)))

		w.WriteHeader(http.StatusTooManyRequests)
		// Optionally, include a message in the response body
		response := fmt.Sprintf("Please retry after %d seconds", int(ttl))
		w.Write([]byte(response))
		return
	}

	voteCount, incrementErr := h.Redis.IncrementForChannel(r.Context(), VoteItemBody.ChannelID)
	if incrementErr != nil {
		log.Println("Failed to increment count in Redis: ", incrementErr)
		// w.WriteHeader(http.StatusInternalServerError)
		// return
	}

	key := "votesItem:" + VoteItemBody.ChannelID
	h.Redis.AddVote(r.Context(), key, VoteItemBody.ItemID)
	h.Redis.SetExpiry(r.Context(), key, 0)

	// Enough votes accumulated
	voteThreshold := h.getVoteThreshold(r.Context(), VoteItemBody.ChannelID)
	log.Println("voteThreshold:", voteThreshold)
	if voteCount >= voteThreshold {
		votedItem := h.handleThresholdFulfilled(r.Context(), VoteItemBody.ChannelID)
		message := fmt.Sprintf("Chat thinks you should buy %s!", votedItem.Name)

		var twitchMessage = wrapper.TwitchMessage{
			Message:   message,
			ChannelID: VoteItemBody.ChannelID,
		}
		timeout := h.Redis.GetTwitchMessageAPITimeout(r.Context(), VoteItemBody.ChannelID)
		if timeout < 0 {
			err := h.TwitchWrapper.SendMessage(twitchMessage)
			if vErr := new(dbsError.VoteError); errors.As(err, &vErr) {
				if vErr.Code == dbsError.CodeTwitchMessageTooManyRequests {
					log.Println("Too many requests error: %w", vErr)
					h.handleVoteMessageTooManyRequests(r.Context(), VoteItemBody.ChannelID)
				}
			}
		}
		log.Println("Sending voteUpdate: ", voteThreshold)
		log.Println("Sending votedItem: ", votedItem.ID)

		SSEPushChannel <- SSEPushRequest{SSEMessage: SSEMessage{EventType: "votedItem", Data: votedItem.ID}, ChannelID: VoteItemBody.ChannelID}
		SSEPushChannel <- SSEPushRequest{SSEMessage: SSEMessage{EventType: "voteUpdate", Data: 0}, ChannelID: VoteItemBody.ChannelID}
	} else {
		SSEPushChannel <- SSEPushRequest{SSEMessage: SSEMessage{EventType: "voteUpdate", Data: voteCount}, ChannelID: VoteItemBody.ChannelID}
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *Vote) handleVoteMessageTooManyRequests(ctx context.Context, channelID string) {
	log.Println("Handling backoff")
	h.Redis.SetTwitchMessageAPITimeout(ctx, channelID)
}

func (h *Vote) getVoteThreshold(ctx context.Context, channelID string) int64 {
	log.Println("Getting vote threshold for:", channelID)
	voteThresholdString, err := h.Redis.GetVoteThreshold(ctx, channelID)
	if err != nil {
		var voteErr *dbsError.VoteError
		if errors.As(err, &voteErr) {
			switch voteErr.Code {
			case dbsError.CodeMissingCacheVoteThreshold:
				log.Println("missing vote threshold cache:", err)
				voteThresholdString, twitchGetConfigErr := h.TwitchWrapper.GetStreamerConfig(channelID)
				if twitchGetConfigErr != nil {
					h.Redis.UpdateVoteThresholdForChannel(ctx, channelID, strconv.Itoa(VoteThreshold))
					return VoteThreshold
				} else {
					h.Redis.UpdateVoteThresholdForChannel(ctx, channelID, voteThresholdString)
					log.Println("retrieved vote threshold:", voteThresholdString)
					voteThreshold, stringConvErr := strconv.ParseInt(voteThresholdString, 10, 64)
					if stringConvErr != nil {
						log.Println("Failed to convert vote threshold to int64")
						return VoteThreshold
					}
					return voteThreshold
				}
			default:
				return VoteThreshold
			}
		}
	}
	log.Println("retrieved vote threshold:", voteThresholdString)
	voteThreshold, stringConvErr := strconv.ParseInt(voteThresholdString, 10, 64)
	if stringConvErr != nil {
		log.Println("Failed to convert vote threshold to int64")
		return VoteThreshold
	}
	return voteThreshold
}

func (h *Vote) GetExtensionVoteStatus(w http.ResponseWriter, r *http.Request) {
	channelID := chi.URLParam(r, "channelID")
	if channelID == "" {
		log.Println("Missing channelID in URL")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// Preferentially use channel_id from header for newer versions
	// TODO: Deprecate old channel_id in request body
	headerChannelID := r.Header.Get("Channel-Id")
	if headerChannelID != "" {
		channelID = headerChannelID
	}
	var response struct {
		ItemID       string `json:"item_id,omitempty"`
		CurrentCount int64  `json:"current_count"`
	}
	lastVotedID, err := h.Redis.GetLastVotedItem(r.Context(), channelID)
	if err != nil {
		var voteErr *dbsError.VoteError
		if errors.As(err, &voteErr) {
			switch voteErr.Code {
			case dbsError.CodeRetrieveLastVotedError:
				log.Println("Failed to find last voted item from cache cache:", err)
			default:
				log.Println("Unknown error occurred:", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
	}

	currentCount, err := h.Redis.GetCurrentCountForChannel(r.Context(), channelID)
	if err != nil {
		var voteErr *dbsError.VoteError
		if errors.As(err, &voteErr) {
			switch voteErr.Code {
			case dbsError.CodeRetrieveVoteCountNoKey:
				log.Println("Key not in cache:", err)
			default:
				log.Println("Unknown error occurred:", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
	}
	log.Printf("CurrentCount: %d, LastVotedID: %s\n", currentCount, lastVotedID)
	response.CurrentCount = currentCount
	response.ItemID = lastVotedID

	data, err := json.Marshal(response)
	if err != nil {
		log.Println("failed to marshal:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Println("DEBUG:", data)
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func (h *Vote) handleThresholdFulfilled(ctx context.Context, channelID string) model.Item {
	// Get the top votes, clear all votes, reset increment count
	topVotes, err := h.Redis.GetMostVoted(ctx, channelID, "Item", 1)
	if err != nil {
		log.Println(err)
		return model.Item{}
	}
	mostVotedID := h.GetTopVotedId(topVotes)
	votedItem := h.Redis.GetItemByID(ctx, mostVotedID)
	h.Redis.UpdateLastVotedItem(ctx, channelID, mostVotedID)
	h.Redis.ClearVotesForChannel(ctx, channelID)
	h.Redis.ClearVoteCountForChannel(ctx, channelID)
	return votedItem
}

func (h *Vote) GetTopVotedId(voteResults map[string]int) string {
	if len(voteResults) == 0 {
		return "" // Return empty if no votes
	}
	var topItem string
	var maxVotes int
	for item, votes := range voteResults {
		if votes > maxVotes {
			topItem = item
			maxVotes = votes
		}
	}
	return topItem
}
