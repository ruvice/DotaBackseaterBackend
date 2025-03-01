package handler

import (
	"log"
	"net/http"

	"github.com/ruvice/dotabackseaterbackend/repository"
	"github.com/ruvice/dotabackseaterbackend/repository/redisRepo"
)

type HeroHandler struct {
	DB    *repository.MongoDBRepo
	Redis *redisRepo.RedisRepo
}

func (h *HeroHandler) GetHeroes(w http.ResponseWriter, r *http.Request) {
	log.Println("Get Heroes")
	heroJsonString, err := h.Redis.GetHeroMapFromCache(r.Context())
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Write the JSON string directly to the HTTP response
	w.Write([]byte(heroJsonString))
}

func (h *HeroHandler) RefreshHeroes(w http.ResponseWriter, r *http.Request) {
	log.Println("Refreshing Heroes")
	heroMap, err := h.DB.RefreshHeroes(r.Context())
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	h.Redis.CacheHeroes(r.Context(), heroMap)
	w.WriteHeader(http.StatusOK)
}
