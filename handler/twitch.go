package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/ruvice/dotabackseaterbackend/repository"
	"github.com/ruvice/dotabackseaterbackend/wrapper"
)

type TwitchHandler struct {
	Repo          *repository.RedisRepo
	TwitchWrapper *wrapper.TwitchWrapper
}

type StreamerConfigResponse struct {
	VoteThreshold string `json:"vote_threshold,omitempty"`
}

func (h *TwitchHandler) SendTwitchMessage(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Sending Twitch message")
	var body wrapper.TwitchMessage

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		fmt.Println("body issue", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	err := h.TwitchWrapper.SendMessage(body)
	if err != nil {
		fmt.Println("Error with send Twitch Message API: ", err)
	}
}

func (h *TwitchHandler) SendTwitchFEMessage(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Sending Twitch FE message")
	var body struct {
		Message   string `json:"message,omitempty"`
		ChannelID string `json:"channel_id,omitempty"`
		EBSToken  string `json:"ebs_token,omitempty"`
		ClientID  string `json:"client_id,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		fmt.Println("body issue", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	err := h.TwitchWrapper.SendFEMessage(body.ChannelID, body.Message, body.EBSToken, body.ClientID)
	if err != nil {
		fmt.Println("Error with send Twitch Message API: ", err)
	}
}

func (h *TwitchHandler) RefreshStreamerConfig(w http.ResponseWriter, r *http.Request) {
	channelIDParam := chi.URLParam(r, "channelID")
	fmt.Println("Fetching streamer config for:", channelIDParam)
	time.Sleep(2 * time.Second)
	voteThreshold, err := h.TwitchWrapper.GetStreamerConfig(channelIDParam)
	if err != nil {
		fmt.Println("Error retrieving configuration", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	fmt.Println("In Twitch handler:", voteThreshold)
	fmt.Println("Updating streamer config in redis")
	err = h.Repo.UpdateVoteThresholdForChannel(r.Context(), channelIDParam, voteThreshold)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := StreamerConfigResponse{
		VoteThreshold: voteThreshold,
	}

	// Write the JSON string directly to the HTTP response
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, "Unable to encode JSON", http.StatusInternalServerError)
	}
}

func (h *TwitchHandler) GetStreamerConfig(w http.ResponseWriter, r *http.Request) {
	channelIDParam := chi.URLParam(r, "channelID")
	fmt.Println("Fetching streamer config for:", channelIDParam)
	time.Sleep(2 * time.Second)

	voteThreshold, err := h.Repo.GetVoteThreshold(r.Context(), channelIDParam)
	if err != nil {
		fmt.Println("Error retrieving configuration", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := StreamerConfigResponse{
		VoteThreshold: voteThreshold,
	}

	// Write the JSON string directly to the HTTP response
	jsonEncodeErr := json.NewEncoder(w).Encode(response)
	if jsonEncodeErr != nil {
		http.Error(w, "Unable to encode JSON", http.StatusInternalServerError)
	}
}
