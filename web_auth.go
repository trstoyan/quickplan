package main

import (
	"net/http"
	"os"
	"strings"
	"time"
)

func applyWebAuth(req *http.Request) {
	if req == nil {
		return
	}

	if key := strings.TrimSpace(os.Getenv("QUICKPLAN_API_KEY")); key != "" {
		req.Header.Set("X-API-Key", key)
	}

	if token := strings.TrimSpace(os.Getenv("QUICKPLAN_WEB_TOKEN")); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
}

func newWebClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &http.Client{Timeout: timeout}
}
