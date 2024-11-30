package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ruvice/dotabackseaterbackend/model"
	"github.com/ruvice/dotabackseaterbackend/repository"
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
	if voteCount > VoteThreshold {
		votedItem := h.handleThresholdFulfilled(r.Context(), VoteBody.ChannelID)
		message := fmt.Sprintf("Chat thinks you should buy %s!", votedItem.Name)

		var twitchMessage = wrapper.TwitchMessage{
			Message:   message,
			ChannelID: VoteBody.ChannelID,
		}
		h.TwitchWrapper.SendMessage(twitchMessage)
	}

	w.WriteHeader(http.StatusCreated)
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
