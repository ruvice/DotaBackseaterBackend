package wrapper

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2/clientcredentials"
	"golang.org/x/oauth2/twitch"
)

type TwitchWrapper struct {
	twitchConfig TwitchConfig
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

var (
	// Consider storing the secret in an environment variable or a dedicated storage system.
	oauth2Config *clientcredentials.Config
)

func NewTwitchWrapper(twitchConfig TwitchConfig) *TwitchWrapper {
	oauth2Config = &clientcredentials.Config{
		ClientID:     twitchConfig.ClientID,
		ClientSecret: twitchConfig.ClientSecret,
		TokenURL:     twitch.Endpoint.TokenURL,
		Scopes:       twitchConfig.Scopes,
	}

	twitchWrapper := &TwitchWrapper{
		twitchConfig: twitchConfig,
	}

	return twitchWrapper
}

func (w *TwitchWrapper) generateJWT() (string, error) {
	// Decode the base64 encoded secret
	decodedSecret, err := base64.StdEncoding.DecodeString(w.twitchConfig.ExtensionSecret)
	if err != nil {
		return "", err
	}

	// Create JWT claims
	claims := jwt.MapClaims{
		"exp":     time.Now().Add(4 * time.Second).Unix(),
		"user_id": w.twitchConfig.Owner,
		"role":    "external",
	}

	// Create JWT token and sign it with the decoded secret
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	jwtToken, err := token.SignedString(decodedSecret)
	if err != nil {
		return "", err
	}

	return jwtToken, nil
}

// Send message to Twitch API
func (w *TwitchWrapper) SendMessage(twitchMessage TwitchMessage) error {
	// Load configuration
	// Generate JWT for authentication
	jwtToken, err := w.generateJWT()
	if err != nil {
		return fmt.Errorf("failed to generate JWT: %v", err)
	}

	// Create message payload
	payload := TwitchMessagePayload{
		BroadcasterID:    twitchMessage.ChannelID,
		Text:             twitchMessage.Message,
		ExtensionID:      w.twitchConfig.ClientID,
		ExtensionVersion: w.twitchConfig.ExtensionVersion,
	}

	// Convert payload to JSON
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}

	// Make the HTTP request to the Twitch API
	req, err := http.NewRequest("POST", "https://api.twitch.tv/helix/extensions/chat", bytes.NewBuffer(payloadJSON))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	req.Header.Set("Client-ID", w.twitchConfig.ClientID)
	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Log the response, including rate limits
	fmt.Printf("Response status: %d\n", resp.StatusCode)
	fmt.Printf("Rate Limit Remaining: %s/%s\n", resp.Header.Get("ratelimit-remaining"), resp.Header.Get("ratelimit-limit"))

	// Check if the request failed
	if resp.StatusCode != 204 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error response from Twitch: %s", string(body))
	}

	return nil
}

// Send message to Twitch API
func (w *TwitchWrapper) SendFEMessage(channelID string, message string, ebs_token string, clientID string) error {
	// Create message payload
	payload := TwitchMessagePayload{
		BroadcasterID:    channelID,
		Text:             message,
		ExtensionID:      clientID,
		ExtensionVersion: w.twitchConfig.ExtensionVersion,
	}

	// Convert payload to JSON
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}

	// Make the HTTP request to the Twitch API
	req, err := http.NewRequest("POST", "https://api.twitch.tv/helix/extensions/chat", bytes.NewBuffer(payloadJSON))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	req.Header.Set("Client-ID", clientID)
	req.Header.Set("Authorization", "Bearer "+ebs_token)
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Log the response, including rate limits
	fmt.Printf("Response status: %d\n", resp.StatusCode)
	fmt.Printf("Rate Limit Remaining: %s/%s\n", resp.Header.Get("ratelimit-remaining"), resp.Header.Get("ratelimit-limit"))

	// Check if the request failed
	if resp.StatusCode != 204 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error response from Twitch: %s", string(body))
	}

	return nil
}

// Send message to Twitch API
func (w *TwitchWrapper) GetStreamerConfig(channelID string) (string, error) {
	// Generate JWT for authentication
	fmt.Println("In Getting Streamer Config")
	jwtToken, err := w.generateJWT()
	if err != nil {
		return "", fmt.Errorf("failed to generate JWT: %v", err)
	}

	// Make the HTTP request to the Twitch API
	baseURL := "https://api.twitch.tv/helix/extensions/configurations"

	// Create a URL object
	u, err := url.Parse(baseURL)
	if err != nil {
		fmt.Printf("Error parsing URL: %v\n", err)
		return "", fmt.Errorf("error parsing URL: %v", err)
	}

	// Add query parameters
	q := u.Query()
	q.Set("broadcaster_id", channelID)
	q.Set("extension_id", w.twitchConfig.ClientID)
	q.Set("segment", "broadcaster")
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	fmt.Println("Twitch Config URI", u.String())
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	req.Header.Set("Client-ID", w.twitchConfig.ClientID)
	req.Header.Set("Authorization", "Bearer "+jwtToken)

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Log the response, including rate limits
	fmt.Printf("Response status: %d\n", resp.StatusCode)
	fmt.Printf("Rate Limit Remaining: %s/%s\n", resp.Header.Get("ratelimit-remaining"), resp.Header.Get("ratelimit-limit"))

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response body: %v\n", err)
		return "", fmt.Errorf("error reading response from Twitch: %v", err)
	}

	var response TwitchStreamerConfigResponse
	err = json.Unmarshal([]byte(body), &response)
	if err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		return "", fmt.Errorf("error parsing JSON from Twitch: %v", err)
	}
	for _, content := range response.Data {
		if content.Segment == "broadcaster" {
			fmt.Printf("Segment: %s, Content: %s, Version: %s\n", content.Segment, content.Content, content.Version)

			// Remove quotes from the string
			trimmed := strings.Trim(content.Content, `"`)
			return trimmed, nil
		}
	}
	return "", fmt.Errorf("failed to retrieve streamer vote threshold")
}
