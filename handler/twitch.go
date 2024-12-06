package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/ruvice/dotabackseaterbackend/repository"
	"github.com/ruvice/dotabackseaterbackend/wrapper"
)

type TwitchHandler struct {
	Redis         *repository.RedisRepo
	TwitchWrapper *wrapper.TwitchWrapper
}

type StreamerConfigResponse struct {
	VoteThreshold string `json:"vote_threshold,omitempty"`
}

func (h *TwitchHandler) SendTwitchMessage(w http.ResponseWriter, r *http.Request) {
	log.Println("Sending Twitch message")
	var body wrapper.TwitchMessage

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		log.Println("body issue", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	err := h.TwitchWrapper.SendMessage(body)
	if err != nil {
		log.Println("Error with send Twitch Message API: ", err)
	}
}

func (h *TwitchHandler) SendTwitchFEMessage(w http.ResponseWriter, r *http.Request) {
	log.Println("Sending Twitch FE message")
	var body struct {
		Message   string `json:"message,omitempty"`
		ChannelID string `json:"channel_id,omitempty"`
		EBSToken  string `json:"ebs_token,omitempty"`
		ClientID  string `json:"client_id,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		log.Println("body issue", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	err := h.TwitchWrapper.SendFEMessage(body.ChannelID, body.Message, body.EBSToken, body.ClientID)
	if err != nil {
		log.Println("Error with send Twitch Message API: ", err)
	}
}

func (h *TwitchHandler) RefreshStreamerConfig(w http.ResponseWriter, r *http.Request) {
	channelIDParam := chi.URLParam(r, "channelID")
	log.Println("Fetching streamer config for:", channelIDParam)
	time.Sleep(2 * time.Second)
	voteThreshold, err := h.TwitchWrapper.GetStreamerConfig(channelIDParam)
	if err != nil {
		log.Println("Error retrieving configuration", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Println("In Twitch handler:", voteThreshold)
	log.Println("Updating streamer config in redis")
	err = h.Redis.UpdateVoteThresholdForChannel(r.Context(), channelIDParam, voteThreshold)
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
	log.Println("Fetching streamer config for:", channelIDParam)
	time.Sleep(2 * time.Second)

	voteThreshold, err := h.Redis.GetVoteThreshold(r.Context(), channelIDParam)
	if err != nil {
		voteThreshold, err := h.TwitchWrapper.GetStreamerConfig(channelIDParam)
		if err != nil {
			log.Println("Error retrieving configuration", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		err = h.Redis.UpdateVoteThresholdForChannel(r.Context(), channelIDParam, voteThreshold)
		if err != nil {
			log.Println("Couldn't write to redis cache", err)
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
