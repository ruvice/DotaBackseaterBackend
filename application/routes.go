package application

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/ruvice/dotabackseaterbackend/handler"
	"github.com/ruvice/dotabackseaterbackend/repository/order"
	"github.com/ruvice/dotabackseaterbackend/repository/vote"
)

func (a *App) loadRoutes() {
	router := chi.NewRouter()
	router.Use(middleware.Logger)

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Doing this ensures that everything loadOrderRoutes receives will have the order prefix
	router.Route("/order", a.loadOrderRoutes)
	// Doing this ensures that everything loadVoteRoutes receives will have the vote prefix
	router.Route("/vote", a.loadVoteRoutes)

	a.router = router
}

func (a *App) loadOrderRoutes(router chi.Router) {
	orderHandler := &handler.Order{
		Repo: &order.RedisRepo{
			Client: a.rdb,
		},
	}

	router.Post("/", orderHandler.Create)
	router.Get("/", orderHandler.List)
	router.Get("/{id}", orderHandler.GetByID)
	router.Put("/{id}", orderHandler.UpdateByID)
	router.Delete("/{id}", orderHandler.DeleteByID)
}

func (a *App) loadVoteRoutes(router chi.Router) {
	voteHandler := &handler.Vote{
		Repo: &vote.RedisRepo{
			Client: a.rdb,
		},
	}

	router.Post("/", voteHandler.Vote)
}
