package application

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/ruvice/dotabackseaterbackend/handler"
)

func (a *App) loadRoutes() {
	router := chi.NewRouter()
	router.Use(middleware.Logger)
	corsOptions := a.getCorsOptions()
	router.Use(cors.Handler(corsOptions))

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hi Rix!")
	})

	router.Get("/hi", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello world!")
	})

	router.Route("/vote", a.loadItemVoteRoutes)
	router.Route("/vote/hero", a.loadHeroVoteRoutes)
	router.Route("/debug", a.debugRoutes)
	router.Route("/item", a.loadItemRoutes)
	router.Route("/hero", a.loadHeroRoutes)
	router.Route("/config", a.loadStreamerConfigRoutes)
	router.Route("/sse", a.voteSSERoutes)

	a.router = router
}

func (a *App) getCorsOptions() cors.Options {
	var allowedOrigins []string
	if a.debugMode {
		allowedOrigins = []string{"https://localhost:8080", "https://" + a.config.TwitchConfig.ClientID + ".ext-twitch.tv", "http://localhost:8080"}
	} else {
		allowedOrigins = []string{"https://" + a.config.TwitchConfig.ClientID + ".ext-twitch.tv"}
	}
	return cors.Options{
		AllowedOrigins:   allowedOrigins, // Frontend origin
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "Channel-Id"},
		AllowCredentials: true, // Allow credentials if needed
		MaxAge:           300,  // Maximum time (in seconds) for preflight to be cached
	}
}

func (a *App) loadItemVoteRoutes(router chi.Router) {
	voteHandler := &handler.Vote{
		Redis:         a.redisRepo,
		TwitchWrapper: a.twitchWrapper,
	}
	router.Post("/", voteHandler.VoteItem)
	router.Get("/{channelID}", voteHandler.GetExtensionVoteStatus)
}

func (a *App) loadHeroVoteRoutes(router chi.Router) {
	voteHandler := &handler.Vote{
		Redis:         a.redisRepo,
		TwitchWrapper: a.twitchWrapper,
	}
	router.Post("/", voteHandler.VoteHero)
	router.Post("/start", voteHandler.StartHeroVote)
	// router.Post("/stop", voteHandler.StopHeroVote)
}

func (a *App) loadItemRoutes(router chi.Router) {
	itemHandler := &handler.ItemHandler{
		Redis: a.redisRepo,
		DB:    a.mongoDB,
	}
	router.Get("/", itemHandler.GetItems)
	router.Get("/refreshItems", itemHandler.RefreshItems)
}

func (a *App) loadHeroRoutes(router chi.Router) {
	heroHandler := &handler.HeroHandler{
		Redis: a.redisRepo,
		DB:    a.mongoDB,
	}
	router.Get("/", heroHandler.GetHeroes)
	router.Get("/refreshHeroes", heroHandler.RefreshHeroes)
}

func (a *App) loadStreamerConfigRoutes(router chi.Router) {
	twitchHandler := &handler.TwitchHandler{
		TwitchWrapper: a.twitchWrapper,
		Redis:         a.redisRepo,
	}
	router.Post("/{channelID}", twitchHandler.RefreshStreamerConfig)
	router.Get("/{channelID}", twitchHandler.GetStreamerConfig)
}

func (a *App) debugRoutes(router chi.Router) {
	twitchHandler := &handler.TwitchHandler{
		TwitchWrapper: a.twitchWrapper,
	}
	router.Post("/message", twitchHandler.SendTwitchMessage)
	router.Post("/messagefrontend", twitchHandler.SendTwitchFEMessage)
}

func (a *App) voteSSERoutes(router chi.Router) {
	// SSEHandler handles SSE connections
	eventHandler := &handler.EventHandler{}
	eventHandler.StartSSEPushWorker()
	router.Get("/{channelID}", eventHandler.EstablishSSEConnection)
}
