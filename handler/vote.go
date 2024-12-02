package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	Repo          *repository.RedisRepo
	TwitchWrapper *wrapper.TwitchWrapper
}

var VoteBody struct {
	ChannelID string `json:"channel_id,omitempty"`
	TwitchID  string `json:"twitch_id,omitempty"`
	ItemID    string `json:"item_id,omitempty"`
}

func (h *Vote) Vote(w http.ResponseWriter, r *http.Request) {
	fmt.Println("New vote")

	if err := json.NewDecoder(r.Body).Decode(&VoteBody); err != nil {
		fmt.Println("body fked up")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ttl := h.Repo.GetVoteRelationTTL(r.Context(), VoteBody.ChannelID, VoteBody.TwitchID)
	if ttl <= 0 {
		voteError := h.Repo.AddVoteRelation(r.Context(), VoteBody.ChannelID, VoteBody.TwitchID)
		if voteError != nil {
			fmt.Println(voteError)
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

	voteCount, incrementErr := h.Repo.IncrementForChannel(r.Context(), VoteBody.ChannelID)
	if incrementErr != nil {
		fmt.Println("Failed to increment count in Redis: ", incrementErr)
		// w.WriteHeader(http.StatusInternalServerError)
		// return
	}
	h.Repo.AddVote(r.Context(), VoteBody.ChannelID, VoteBody.ItemID, VoteBody.TwitchID)

	// Enough votes accumulated
	voteThreshold := h.getVoteThreshold(r.Context(), VoteBody.ChannelID)
	fmt.Println("voteThreshold:", voteThreshold)
	if voteCount >= voteThreshold {
		votedItem := h.handleThresholdFulfilled(r.Context(), VoteBody.ChannelID)
		message := fmt.Sprintf("Chat thinks you should buy %s!", votedItem.Name)

		var twitchMessage = wrapper.TwitchMessage{
			Message:   message,
			ChannelID: VoteBody.ChannelID,
		}
		timeout := h.Repo.GetTwitchMessageAPITimeout(r.Context(), VoteBody.ChannelID)
		if timeout < 0 {
			err := h.TwitchWrapper.SendMessage(twitchMessage)
			if vErr := new(voteErrors.VoteError); errors.As(err, &vErr) {
				if vErr.Code == voteErrors.CodeTwitchMessageTooManyRequests {
					fmt.Println("Too many requests error:", vErr.Message)
					h.handleVoteMessageTooManyRequests(r.Context(), VoteBody.ChannelID)
				}
			}
		}
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *Vote) handleVoteMessageTooManyRequests(ctx context.Context, channelID string) {
	fmt.Println("Handling backoff")
	h.Repo.SetTwitchMessageAPITimeout(ctx, channelID)
}

func (h *Vote) getVoteThreshold(ctx context.Context, channelID string) int64 {
	fmt.Println("Getting vote threshold for:", channelID)
	voteThresholdString, err := h.Repo.GetVoteThreshold(ctx, channelID)
	if err != nil {
		if err.Code == voteErrors.CodeMissingCacheVoteThreshold {
			fmt.Println("missing vote threshold cache:", err)
			voteThresholdString, twitchGetConfigErr := h.TwitchWrapper.GetStreamerConfig(channelID)
			if twitchGetConfigErr != nil {
				h.Repo.UpdateVoteThresholdForChannel(ctx, channelID, strconv.Itoa(VoteThreshold))
				return VoteThreshold
			} else {
				h.Repo.UpdateVoteThresholdForChannel(ctx, channelID, voteThresholdString)
				fmt.Println("retrieved vote threshold:", voteThresholdString)
				voteThreshold, stringConvErr := strconv.ParseInt(voteThresholdString, 10, 64)
				if stringConvErr != nil {
					fmt.Println("Failed to convert vote threshold to int64")
					return VoteThreshold
				}
				return voteThreshold
			}
		} else {
			return VoteThreshold
		}
	}
	fmt.Println("retrieved vote threshold:", voteThresholdString)
	voteThreshold, stringConvErr := strconv.ParseInt(voteThresholdString, 10, 64)
	if stringConvErr != nil {
		fmt.Println("Failed to convert vote threshold to int64")
		return VoteThreshold
	}
	return voteThreshold
}

func (h *Vote) ListV3(w http.ResponseWriter, r *http.Request) {
	channelIDParam := chi.URLParam(r, "channelID")
	res := h.Repo.GetMostVoted(r.Context(), channelIDParam)
	var response struct {
		ItemID string
	}
	response.ItemID = res

	data, err := json.Marshal(response)
	if err != nil {
		fmt.Println("failed to marshal:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write(data)
}

func (h *Vote) handleThresholdFulfilled(ctx context.Context, channelID string) model.Item {
	// Get the top votes, clear all votes, reset increment count
	votedItemID := h.Repo.GetMostVoted(ctx, channelID)
	votedItem := h.Repo.GetItemByID(ctx, votedItemID)
	h.Repo.ClearVotesForChannel(ctx, channelID)
	h.Repo.ClearVoteCountForChannel(ctx, channelID)
	return votedItem
}
