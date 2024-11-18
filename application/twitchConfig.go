package application

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/ruvice/dotabackseaterbackend/wrapper"
)

func LoadTwitchConfig() wrapper.TwitchConfig {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Failed to get .env")
	}

	// Access environment variables
	TWITCH_EXTENSION_SECRET := os.Getenv("TWITCH_EXTENSION_SECRET")
	TWITCH_OWNER := os.Getenv("TWITCH_OWNER")
	TWITCH_CLIENT_ID := os.Getenv("TWITCH_CLIENT_ID")
	TWITCH_CLIENT_SECRET := os.Getenv("TWITCH_CLIENT_SECRET")
	twitchConfig := wrapper.TwitchConfig{
		ExtensionSecret:  TWITCH_EXTENSION_SECRET,
		ExtensionVersion: "0.0.1",
		Owner:            TWITCH_OWNER,
		ClientID:         TWITCH_CLIENT_ID,
		ClientSecret:     TWITCH_CLIENT_SECRET,
		Scopes:           []string{"user:write:chat", "user:bot", "channel:bot"},
	}

	return twitchConfig
}
