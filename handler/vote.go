package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/ruvice/dotabackseaterbackend/model"
	"github.com/ruvice/dotabackseaterbackend/repository/vote"
)

type Vote struct {
	Repo *vote.RedisRepo
}

func (h *Vote) Vote(w http.ResponseWriter, r *http.Request) {
	fmt.Println("New vote")

	var body struct {
		ChannelID string `json:"channel_id,omitempty"`
		TwitchID  string `json:"twitch_id,omitempty"`
		ItemID    uint64 `json:"item_id,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		fmt.Println("body fked up")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	now := time.Now().UTC()

	vote := model.Vote{
		ChannelID: body.ChannelID,
		TwitchID:  body.TwitchID,
		ItemID:    body.ItemID,
		VotedAt:   &now,
	}
	err := h.Repo.Insert(r.Context(), vote)
	if err != nil {
		fmt.Println("failed to insert:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	res, err := json.Marshal(vote)
	if err != nil {
		fmt.Println("failed to marshall:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write(res)
}

func (h *Vote) List(w http.ResponseWriter, r *http.Request) {
	channelIDParam := chi.URLParam(r, "channelID")
	cursorStr := r.URL.Query().Get("cursor")
	if cursorStr == "" {
		cursorStr = "0"
	}

	const decimal = 10
	const bitSize = 64
	cursor, err := strconv.ParseUint(cursorStr, decimal, bitSize)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	const size = 50
	res, err := h.Repo.FindAll(r.Context(), vote.FindAllPage{
		Offset:    cursor,
		Size:      size,
		ChannelID: channelIDParam,
	})
	if err != nil {
		fmt.Println("failed to find all:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var response struct {
		Votes []model.Vote `json:"votes,omitempty"`
		Next  uint64       `json:"next,omitempty"`
	}
	response.Votes = res.Votes
	response.Next = res.Cursor

	data, err := json.Marshal(response)
	if err != nil {
		fmt.Println("failed to marshal:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write(data)
}

func (h *Vote) VoteV2(w http.ResponseWriter, r *http.Request) {
	fmt.Println("New vote")

	var body struct {
		ChannelID string `json:"channel_id,omitempty"`
		TwitchID  string `json:"twitch_id,omitempty"`
		ItemID    string `json:"item_id,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		fmt.Println("body fked up")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	h.Repo.AddVote(r.Context(), body.ChannelID, body.ItemID)

	w.WriteHeader(http.StatusCreated)
}

func (h *Vote) ListV2(w http.ResponseWriter, r *http.Request) {
	channelIDParam := chi.URLParam(r, "channelID")
	res := h.Repo.GetTopVote(r.Context(), channelIDParam, 10)
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

func (h *Vote) VoteV3(w http.ResponseWriter, r *http.Request) {
	fmt.Println("New vote")

	var body struct {
		ChannelID string `json:"channel_id,omitempty"`
		TwitchID  string `json:"twitch_id,omitempty"`
		ItemID    string `json:"item_id,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		fmt.Println("body fked up")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	h.Repo.AddVoteV2(r.Context(), body.ChannelID, body.ItemID, body.TwitchID)

	w.WriteHeader(http.StatusCreated)
}

func (h *Vote) ListV3(w http.ResponseWriter, r *http.Request) {
	channelIDParam := chi.URLParam(r, "channelID")
	res := h.Repo.GetTopVoteV2(r.Context(), channelIDParam)
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
