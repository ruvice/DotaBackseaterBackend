package wrapper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
)

func NewTwitchWrapper(twitchConfig TwitchConfig) *TwitchWrapper {
	twitchWrapper := &TwitchWrapper{
		twitchConfig: twitchConfig,
		httpClient:   &http.Client{},
	}

	return twitchWrapper
}

// Send message to Twitch API
func (w *TwitchWrapper) SendMessage(twitchMessage TwitchMessage) error {
	log.Printf("Sending Message on Twitch:  %s\n", twitchMessage.Message)
	jwtToken, err := w.generateJWT()
	if err != nil {
		return fmt.Errorf("failed to generate JWT: %v", err)
	}
	payload := TwitchMessagePayload{
		BroadcasterID:    twitchMessage.ChannelID,
		Text:             twitchMessage.Message,
		ExtensionID:      w.twitchConfig.ClientID,
		ExtensionVersion: w.twitchConfig.ExtensionVersion,
	}
	headers := map[string]string{
		"Client-ID":     w.twitchConfig.ClientID,
		"Authorization": "Bearer " + jwtToken,
		"Content-Type":  "application/json",
	}

	resp, err := w.sendRequest("POST", "https://api.twitch.tv/helix/extensions/chat", payload, headers)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	log.Printf("Response status: %d\n", resp.StatusCode)
	log.Printf("Rate Limit Remaining: %s/%s\n", resp.Header.Get("ratelimit-remaining"), resp.Header.Get("ratelimit-limit"))

	return handleResponse(resp, err)
}

func (w *TwitchWrapper) SendFEMessage(channelID string, message string, ebs_token string, clientID string) error {
	payload := TwitchMessagePayload{
		BroadcasterID:    channelID,
		Text:             message,
		ExtensionID:      clientID,
		ExtensionVersion: w.twitchConfig.ExtensionVersion,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}

	req, err := http.NewRequest("POST", "https://api.twitch.tv/helix/extensions/chat", bytes.NewBuffer(payloadJSON))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Client-ID", w.twitchConfig.ClientID)
	req.Header.Set("Authorization", "Bearer "+ebs_token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	log.Printf("Response status: %d\n", resp.StatusCode)
	log.Printf("Rate Limit Remaining: %s/%s\n", resp.Header.Get("ratelimit-remaining"), resp.Header.Get("ratelimit-limit"))

	return handleResponse(resp, err)
}

func (w *TwitchWrapper) GetStreamerConfig(channelID string) (string, error) {
	log.Println("Getting Streamer Config")
	jwtToken, err := w.generateJWT()
	if err != nil {
		return "", fmt.Errorf("failed to generate JWT: %v", err)
	}

	u, err := w.createStreamerConfigRequestURL(channelID)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

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
	log.Printf("Response status: %d\n", resp.StatusCode)
	log.Printf("Rate Limit Remaining: %s/%s %s\n", resp.Header.Get("ratelimit-remaining"), resp.Header.Get("ratelimit-limit"), resp.Header.Get("ratelimit-reset"))

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v\n", err)
		return "", fmt.Errorf("error reading response from Twitch: %v", err)
	}

	var response TwitchStreamerConfigResponse
	err = json.Unmarshal([]byte(body), &response)
	if err != nil {
		log.Printf("Error parsing JSON: %v\n", err)
		return "", fmt.Errorf("error parsing JSON from Twitch: %v", err)
	}
	for _, content := range response.Data {
		if content.Segment == "broadcaster" {
			trimmed := strings.Trim(content.Content, `"`)
			return trimmed, nil
		}
	}
	return "", fmt.Errorf("failed to retrieve streamer vote threshold")
}

func (w *TwitchWrapper) createStreamerConfigRequestURL(channelID string) (*url.URL, error) {
	baseURL := "https://api.twitch.tv/helix/extensions/configurations"
	url, err := url.Parse(baseURL)
	if err != nil {
		log.Printf("Error parsing URL: %v\n", err)
		return nil, fmt.Errorf("error parsing URL: %v", err)
	}

	// Add query parameters
	q := url.Query()
	q.Set("broadcaster_id", channelID)
	q.Set("extension_id", w.twitchConfig.ClientID)
	q.Set("segment", "broadcaster")
	url.RawQuery = q.Encode()
	return url, nil
}
