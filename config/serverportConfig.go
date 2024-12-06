package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

func LoadServerPort() (uint64, error) {
	serverPortStr := os.Getenv("SERVER_PORT")
	if serverPortStr == "" {
		return 0, errors.New("missing SERVER_PORT environment variable")
	}

	serverPort, err := strconv.ParseUint(serverPortStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid SERVER_PORT value: %w", err)
	}

	return serverPort, nil
}
