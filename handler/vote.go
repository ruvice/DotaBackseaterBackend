package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

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
