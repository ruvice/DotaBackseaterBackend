package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/ruvice/dotabackseaterbackend/model"
	"github.com/ruvice/dotabackseaterbackend/repository/vote"
	"github.com/ruvice/dotabackseaterbackend/wrapper"
)

const (
	VoteThreshold = 10
)

type Vote struct {
	Repo          *vote.RedisRepo
	TwitchWrapper *wrapper.TwitchWrapper
}

func (h *Vote) VoteV3(w http.ResponseWriter, r *http.Request) {
	fmt.Println("New vote")

	var body struct {
		ChannelID string `json:"channel_id,omitempty"`
		TwitchID  string `json:"twitch_id,omitempty"`
		ItemID    int64  `json:"item_id,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		fmt.Println("body fked up")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	itemIDString := strconv.FormatInt(body.ItemID, 10) // Base 10 for decimal
	voteCount, err := h.Repo.IncrementForChannel(r.Context(), body.ChannelID)
	if err != nil {
		fmt.Println("Failed to increment count")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	h.Repo.AddVote(r.Context(), body.ChannelID, itemIDString, body.TwitchID)

	// Enough votes accumulated
	if voteCount > VoteThreshold {
		votedItem := h.handleThresholdFulfilled(r.Context(), body.ChannelID)
		message := fmt.Sprintf("Chat thinks you  should buy %s!", votedItem.ItemName())

		var twitchMessage = wrapper.TwitchMessage{
			Message:   message,
			ChannelID: body.ChannelID,
		}
		h.TwitchWrapper.SendMessage(twitchMessage)
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *Vote) ListV3(w http.ResponseWriter, r *http.Request) {
	channelIDParam := chi.URLParam(r, "channelID")
	res := h.Repo.GetTopVote(r.Context(), channelIDParam)
	var response struct {
		ItemID string
	}
	response.ItemID = res.ItemID

	data, err := json.Marshal(response)
	if err != nil {
		fmt.Println("failed to marshal:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write(data)
}

func (h *Vote) handleThresholdFulfilled(ctx context.Context, channelID string) model.Item {
	// Get the top votes
	// Clear all votes
	// Reset increment count
	votedItem := h.getVotesForChannelId(ctx, channelID)
	h.Repo.ClearVotesForChannel(ctx, channelID)
	h.Repo.ClearVoteCountForChannel(ctx, channelID)
	return votedItem
}

func (h *Vote) getVotesForChannelId(ctx context.Context, channelID string) model.Item {
	res := h.Repo.GetTopVote(ctx, channelID)
	return res
}
