package wrapper

import "net/http"

type TwitchWrapper struct {
	twitchConfig TwitchConfig
	httpClient   *http.Client
}

type TwitchConfig struct {
	ExtensionSecret  string   `json:"extension_secret,omitempty"`
	ClientID         string   `json:"client_id,omitempty"`
	Owner            string   `json:"owner,omitempty"`
	ExtensionVersion string   `json:"extension_version,omitempty"`
	ClientSecret     string   `json:"client_secret,omitempty"`
	Scopes           []string `json:"scopes,omitempty"`
}

// Message Payload for sending chat message
type TwitchMessagePayload struct {
	BroadcasterID    string `json:"broadcaster_id"`
	Text             string `json:"text"`
	ExtensionID      string `json:"extension_id"`
	ExtensionVersion string `json:"extension_version"`
}

type TwitchMessage struct {
	Message   string `json:"message,omitempty"`
	ChannelID string `json:"channel_id,omitempty"`
}

type TwitchStreamerConfigPayload struct {
	BroadcasterID string `json:"broadcaster_id,omitempty"`
	ExtensionID   string `json:"extension_id,omitempty"`
	Segment       string `json:"segment,omitempty"`
}

type TwitchStreamerConfigResponse struct {
	Data []TwitchStreamerConfigContent `json:"data"`
}

type TwitchStreamerConfigContent struct {
	Segment string `json:"segment"`
	Content string `json:"content"`
	Version string `json:"version"`
}
