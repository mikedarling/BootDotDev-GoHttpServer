package auth

import (
	"errors"
	"net/http"
	"strings"
)

func GetAPIKey(headers http.Header) (string, error) {
	apiKey := headers.Get("Authorization")
	if apiKey == "" {
		return "", errors.New("no authorization header available")
	}

	apiKey = strings.TrimSpace(strings.ReplaceAll(apiKey, "ApiKey", ""))
	if apiKey == "" {
		return "", errors.New("api key token must not be empty")
	}

	return apiKey, nil
}
