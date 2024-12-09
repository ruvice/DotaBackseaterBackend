package wrapper

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ruvice/dotabackseaterbackend/utils/dbsError"
)

func (w *TwitchWrapper) generateJWT() (string, error) {
	decodedSecret, err := base64.StdEncoding.DecodeString(w.twitchConfig.ExtensionSecret)
	if err != nil {
		return "", err
	}

	claims := jwt.MapClaims{
		"exp":     time.Now().Add(4 * time.Second).Unix(),
		"user_id": w.twitchConfig.Owner,
		"role":    "external",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	jwtToken, err := token.SignedString(decodedSecret)
	if err != nil {
		return "", err
	}

	return jwtToken, nil
}

// Create and send HTTP requests
func (w *TwitchWrapper) sendRequest(method, url string, payload interface{}, headers map[string]string) (*http.Response, error) {
	var body io.Reader
	if payload != nil {
		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %w", err)
		}
		body = bytes.NewBuffer(payloadJSON)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return w.httpClient.Do(req)
}

func handleResponse(resp *http.Response, err error) error {
	if resp.StatusCode == http.StatusTooManyRequests {
		log.Println("Too many requests: ", resp)
		return dbsError.NewVoteError("SendMessage", dbsError.CodeTwitchMessageTooManyRequests, "Too many requests", err)
	}

	if resp.StatusCode != 204 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error response from Twitch: %s", string(body))
	}
	return nil
}
