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
	"github.com/ruvice/dotabackseaterbackend/repository"
	"github.com/ruvice/dotabackseaterbackend/utils/voteErrors"
	"github.com/ruvice/dotabackseaterbackend/wrapper"
)

const (
	VoteThreshold = 10
)

type Vote struct {
	Redis         *repository.RedisRepo
	TwitchWrapper *wrapper.TwitchWrapper
}

var VoteBody struct {
	ChannelID string `json:"channel_id,omitempty"`
	TwitchID  string `json:"twitch_id,omitempty"`
	ItemID    string `json:"item_id,omitempty"`
}

func (h *Vote) Vote(w http.ResponseWriter, r *http.Request) {
	log.Println("New vote")

	if err := json.NewDecoder(r.Body).Decode(&VoteBody); err != nil {
		log.Println("body fked up")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ttl := h.Redis.GetVoteRelationTTL(r.Context(), VoteBody.ChannelID, VoteBody.TwitchID)
	if ttl <= 0 {
		voteError := h.Redis.AddVoteRelation(r.Context(), VoteBody.ChannelID, VoteBody.TwitchID)
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

	voteCount, incrementErr := h.Redis.IncrementForChannel(r.Context(), VoteBody.ChannelID)
	if incrementErr != nil {
		log.Println("Failed to increment count in Redis: ", incrementErr)
		// w.WriteHeader(http.StatusInternalServerError)
		// return
	}
	h.Redis.AddVote(r.Context(), VoteBody.ChannelID, VoteBody.ItemID, VoteBody.TwitchID)

	// Enough votes accumulated
	voteThreshold := h.getVoteThreshold(r.Context(), VoteBody.ChannelID)
	log.Println("voteThreshold:", voteThreshold)
	if voteCount >= voteThreshold {
		votedItem := h.handleThresholdFulfilled(r.Context(), VoteBody.ChannelID)
		message := fmt.Sprintf("Chat thinks you should buy %s!", votedItem.Name)

		var twitchMessage = wrapper.TwitchMessage{
			Message:   message,
			ChannelID: VoteBody.ChannelID,
		}
		timeout := h.Redis.GetTwitchMessageAPITimeout(r.Context(), VoteBody.ChannelID)
		if timeout < 0 {
			err := h.TwitchWrapper.SendMessage(twitchMessage)
			if vErr := new(voteErrors.VoteError); errors.As(err, &vErr) {
				if vErr.Code == voteErrors.CodeTwitchMessageTooManyRequests {
					log.Println("Too many requests error:", vErr.Message)
					h.handleVoteMessageTooManyRequests(r.Context(), VoteBody.ChannelID)
				}
			}
		}
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
		if err.Code == voteErrors.CodeMissingCacheVoteThreshold {
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
		} else {
			return VoteThreshold
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

func (h *Vote) ListV3(w http.ResponseWriter, r *http.Request) {
	channelIDParam := chi.URLParam(r, "channelID")
	res := h.Redis.GetMostVoted(r.Context(), channelIDParam)
	var response struct {
		ItemID string
	}
	response.ItemID = res

	data, err := json.Marshal(response)
	if err != nil {
		log.Println("failed to marshal:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write(data)
}

func (h *Vote) handleThresholdFulfilled(ctx context.Context, channelID string) model.Item {
	// Get the top votes, clear all votes, reset increment count
	votedItemID := h.Redis.GetMostVoted(ctx, channelID)
	votedItem := h.Redis.GetItemByID(ctx, votedItemID)
	h.Redis.ClearVotesForChannel(ctx, channelID)
	h.Redis.ClearVoteCountForChannel(ctx, channelID)
	return votedItem
}
