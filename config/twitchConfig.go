package config

import (
	"os"

	"github.com/joho/godotenv"
	"github.com/ruvice/dotabackseaterbackend/utils/dbsError"
	"github.com/ruvice/dotabackseaterbackend/wrapper"
)

func LoadTwitchConfig() (wrapper.TwitchConfig, error) {
	err := godotenv.Load()
	if err != nil {
		return wrapper.TwitchConfig{}, dbsError.NewConfigError("LoadConfig", dbsError.ErrInvalidTwitchConfig, "invalid Twitch Config", err)
	}

	// Access environment variables
	TWITCH_EXTENSION_SECRET := os.Getenv("TWITCH_EXTENSION_SECRET")
	TWITCH_OWNER := os.Getenv("TWITCH_OWNER")
	TWITCH_CLIENT_ID := os.Getenv("TWITCH_CLIENT_ID")
	TWITCH_CLIENT_SECRET := os.Getenv("TWITCH_CLIENT_SECRET")
	TWITCH_EXTENSION_VERSION := os.Getenv("TWITCH_EXTENSION_VERSION")
	twitchConfig := wrapper.TwitchConfig{
		ExtensionSecret:  TWITCH_EXTENSION_SECRET,
		ExtensionVersion: TWITCH_EXTENSION_VERSION,
		Owner:            TWITCH_OWNER,
		ClientID:         TWITCH_CLIENT_ID,
		ClientSecret:     TWITCH_CLIENT_SECRET,
		Scopes:           []string{"user:write:chat", "user:bot", "channel:bot"},
	}

	return twitchConfig, nil
}
