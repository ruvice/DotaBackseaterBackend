package application

import (
	"context"
	"fmt"
	"log"
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

	router.Get("/privacy", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `<!DOCTYPE html>
	<html>
	<head>
		<meta charset="UTF-8">
		<title>Privacy Policy</title>
	</head>
	<body>
		<h1>Privacy Policy</h1>
		<p><strong>Last updated:</strong> May 7, 2025</p>
		<p>Dota 2 Backseater respects your privacy. This extension transmits limited information necessary for functionality, specifically the <strong>Twitch opaque user ID</strong>, which is used to manage voting sessions within the extension.</p>
		
		<h2>Information Collected</h2>
		<ul>
			<li><strong>Opaque Twitch User ID:</strong> A non-personally identifiable identifier provided by Twitch. This is used solely to group votes and prevent abuse (e.g. multiple votes from the same user in one session).</li>
		</ul>
	
		<p>No other personal data (such as name, email, IP address) is collected. We do not use cookies, analytics, or third-party tracking tools.</p>
	
		<h2>How Information is Used</h2>
		<p>The opaque user ID is used only for vote counting and session management. It is not shared, sold, or used for profiling. All data remains internal to the service and is discarded after the voting session ends.</p>
	
		<h2>Data Retention</h2>
		<p>Vote data is temporarily stored during an active session and discarded periodically. No long-term storage or account profiling is performed.</p>
	
		<h2>Third-Party Services</h2>
		<p>This extension does not integrate with third-party services beyond Twitch. The extension runs on your device and communicates only with our backend server for voting purposes.</p>
	
		<h2>Children's Privacy</h2>
		<p>This extension does not knowingly collect data from anyone under the age of 13. The use of the extension must comply with Twitch's Terms of Service.</p>
	
		<h2>Contact</h2>
		<p>If you have questions about this privacy policy, please contact: <a href="dota2backseater@gmail.com">dota2backseater@gmail.com</a></p>
	</body>
	</html>`)
	})

	router.Route("/vote", a.loadItemVoteRoutes)
	router.Route("/vote/hero", a.loadHeroVoteRoutes)
	router.Route("/debug", a.debugRoutes)
	router.Route("/item", a.loadItemRoutes)
	router.Route("/hero", a.loadHeroRoutes)
	router.Route("/config", a.loadStreamerConfigRoutes)
	router.Route("/sse", a.voteSSERoutes)
	router.Route("/ws", a.wsRoutes)

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
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "Channel-Id", "Twitch-Id"},
		AllowCredentials: true, // Allow credentials if needed
		MaxAge:           300,  // Maximum time (in seconds) for preflight to be cached
	}
}

func (a *App) loadItemVoteRoutes(router chi.Router) {
	voteHandler := &handler.Vote{
		Redis:         a.redisRepo,
		TwitchWrapper: a.twitchWrapper,
		Broadcaster:   &handler.CombinedBroadcaster{},
	}
	router.Post("/", voteHandler.VoteItem)
	router.Get("/{channelID}", voteHandler.GetExtensionVoteStatus)
}

func (a *App) loadHeroVoteRoutes(router chi.Router) {
	voteHandler := &handler.Vote{
		Redis:         a.redisRepo,
		TwitchWrapper: a.twitchWrapper,
		Broadcaster:   &handler.CombinedBroadcaster{},
	}
	voteHandler.SessionManager = handler.NewVoteSessionManager(
		a.redisRepo.Client, // redis
		"voteHeroSession:", // key prefix
		func(channelID string) { // onExpire callback
			voteHandler.StopHeroVoteInternal(context.Background(), channelID)
		},
	)
	router.Post("/", voteHandler.VoteHero)
	router.Post("/start", voteHandler.StartHeroVote)
	router.Post("/stop", voteHandler.StopHeroVote)
	router.Get("/status", voteHandler.GetExtensionHeroVoteStatus)
}

func (a *App) loadItemRoutes(router chi.Router) {
	itemHandler := &handler.ItemHandler{
		Redis:       a.redisRepo,
		DB:          a.mongoDB,
		Broadcaster: &handler.CombinedBroadcaster{},
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

func (a *App) wsRoutes(router chi.Router) {
	log.Println("✅ WebSocket route mounted")
	wsHandler := &handler.WSHandler{}
	wsHandler.StartWSPushWorker()
	// Add your WebSocket route
	router.Get("/{channelID}", wsHandler.HandleWebSocket)
}
