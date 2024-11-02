package wrapper

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"golang.org/x/oauth2/twitch"
)

type TwitchWrapper struct {
	token        *oauth2.Token
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
	fmt.Println("Getting access token")

	token, err := oauth2Config.Token(context.Background())
	if err != nil {
		fmt.Println("Access token: ", err)
		log.Fatal(err)
	}

	fmt.Printf("Access token: %s\n", token.AccessToken)

	// Trying to send a message
	expirationTime := time.Now().Add(5 * time.Second).Unix()
	claims := jwt.MapClaims{
		"exp":          expirationTime,
		"user_id":      "40825038",
		"role":         "external",
		"channel_id":   "40825038",
		"pubsub_perms": map[string][]string{"send": {"broadcast"}},
	}

	// Create the JWT with the claims and sign it
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token with the secret key
	decodedSecret, err := base64.StdEncoding.DecodeString(twitchConfig.ExtensionSecret)
	if err != nil {
		log.Fatalf("Error decoding base64 secret: %v", err)
	}
	jwtTokenString, err := jwtToken.SignedString([]byte(decodedSecret))
	if err != nil {
		log.Fatalf("Error signing token: %v", err)
	}

	// Print the signed token
	fmt.Println("Signed JWT:", jwtTokenString)

	twitchWrapper := &TwitchWrapper{
		token:        token,
		twitchConfig: twitchConfig,
	}

	// Sending a test message
	testMessage := TwitchMessage{
		Message:   "THIS IS A TEST MESSAGE, WRAPPER INIT SENT SWIMMINGLY",
		ChannelID: "40825038",
	}
	twitchWrapper.SendMessage(testMessage)
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
func (w *TwitchWrapper) sendMessage(twitchMessage TwitchMessage) error {
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
