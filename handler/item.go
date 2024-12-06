package handler

import (
	"log"
	"net/http"

	"github.com/ruvice/dotabackseaterbackend/repository"
)

type ItemHandler struct {
	DB    *repository.MongoDBRepo
	Redis *repository.RedisRepo
}

func (h *ItemHandler) GetItems(w http.ResponseWriter, r *http.Request) {
	log.Println("Get Items")
	itemJsonString, err := h.Redis.GetItemMapFromCache(r.Context())
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Write the JSON string directly to the HTTP response
	w.Write([]byte(itemJsonString))
}

func (h *ItemHandler) RefreshItems(w http.ResponseWriter, r *http.Request) {
	log.Println("Refreshing Items")
	itemMap, err := h.DB.RefreshItems(r.Context())
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	h.Redis.CacheItems(r.Context(), itemMap)
	w.WriteHeader(http.StatusOK)
}
