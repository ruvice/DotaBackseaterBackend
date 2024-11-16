package handler

import (
	"fmt"
	"net/http"

	"github.com/ruvice/dotabackseaterbackend/repository"
)

type ItemHandler struct {
	DB   *repository.MongoDBRepo
	Repo *repository.RedisRepo
}

func (h *ItemHandler) GetItems(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Get Items")
	h.DB.GetItems(r.Context())
	w.WriteHeader(http.StatusOK)
}

func (h *ItemHandler) RefreshItems(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Refreshing Items")
	itemMap, err := h.DB.RefreshItems(r.Context())
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	h.Repo.CacheItems(r.Context(), itemMap)
	w.WriteHeader(http.StatusOK)
}
