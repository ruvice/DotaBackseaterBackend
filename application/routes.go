package application

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/ruvice/dotabackseaterbackend/handler"
	"github.com/ruvice/dotabackseaterbackend/repository/order"
	"github.com/ruvice/dotabackseaterbackend/repository/vote"
)

func (a *App) loadRoutes() {
	router := chi.NewRouter()
	router.Use(middleware.Logger)
	// CORS middleware configuration
	corsOptions := cors.Options{
		AllowedOrigins:   []string{"https://localhost:8080"}, // Frontend origin
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true, // Allow credentials if needed
		MaxAge:           300,  // Maximum time (in seconds) for preflight to be cached
	}

	// Apply the CORS middleware
	router.Use(cors.Handler(corsOptions))

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Doing this ensures that everything loadOrderRoutes receives will have the order prefix
	router.Route("/order", a.loadOrderRoutes)
	// Doing this ensures that everything loadVoteRoutes receives will have the vote prefix
	router.Route("/vote", a.loadVoteRoutes)
	// Doing this ensures that everything debugRoutes receives will have the debug prefix
	router.Route("/debug", a.debugRoutes)

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
	router.Get("/{channelID}", voteHandler.List)
	router.Post("/v2", voteHandler.VoteV2)
	router.Get("/v2/{channelID}", voteHandler.ListV2)

	router.Post("/v3", voteHandler.VoteV3)
	router.Get("/v3/{channelID}", voteHandler.ListV3)
}

func (a *App) debugRoutes(router chi.Router) {
	twitchHandler := &handler.TwitchHandler{
		TwitchWrapper: a.twitchWrapper,
	}
	router.Post("/message", twitchHandler.SendTwitchMessage)
	router.Post("/messagev2", twitchHandler.SendTwitchFEMessage)
}
