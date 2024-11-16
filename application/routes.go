package application

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/ruvice/dotabackseaterbackend/handler"
	"github.com/ruvice/dotabackseaterbackend/repository"
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

	router.Get("/hi", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello world!")
	})

	// Doing this ensures that everything loadVoteRoutes receives will have the vote prefix
	router.Route("/vote", a.loadVoteRoutes)
	// Doing this ensures that everything debugRoutes receives will have the debug prefix
	router.Route("/debug", a.debugRoutes)
	router.Route("/item", a.loadItemRoutes)

	a.router = router
}

func (a *App) loadVoteRoutes(router chi.Router) {
	voteHandler := &handler.Vote{
		Repo: &repository.RedisRepo{
			Client: a.rdb,
		},
		TwitchWrapper: a.twitchWrapper,
		DB: &repository.MongoDBRepo{
			Client: a.mongoDB,
		},
	}
	if a.mongoDB == nil {
		fmt.Println("Lost reference")
	}
	fmt.Println("redisAvailability: ", a.redisAvailable)

	router.Get("/", voteHandler.InsertMongo)
	router.Post("/", voteHandler.Vote)
	router.Get("/{channelID}", voteHandler.ListV3)
}

func (a *App) loadItemRoutes(router chi.Router) {
	itemHandler := &handler.ItemHandler{
		Repo: &repository.RedisRepo{
			Client: a.rdb,
		},
		DB: &repository.MongoDBRepo{
			Client: a.mongoDB,
		},
	}
	router.Get("/", itemHandler.GetItems)
	router.Get("/refreshItems", itemHandler.RefreshItems)
}

func (a *App) debugRoutes(router chi.Router) {
	twitchHandler := &handler.TwitchHandler{
		TwitchWrapper: a.twitchWrapper,
	}
	router.Post("/message", twitchHandler.SendTwitchMessage)
	router.Post("/messagev2", twitchHandler.SendTwitchFEMessage)
}
