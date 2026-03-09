package main

import (
	"net/http"
	"os"
	"testing"
)

func TestApplyWebAuthUsesPreferredPublicToken(t *testing.T) {
	t.Setenv("QUICKPLAN_API_KEY", "api-key")
	t.Setenv("QUICKPLAN_REMOTE_TOKEN", "public-token")
	t.Setenv("QUICKPLAN_WEB_TOKEN", "legacy-token")

	req, err := http.NewRequest(http.MethodGet, "http://example.test", nil)
	if err != nil {
		t.Fatalf("request creation failed: %v", err)
	}

	applyWebAuth(req)

	if got := req.Header.Get("X-API-Key"); got != "api-key" {
		t.Fatalf("unexpected API key header: %q", got)
	}
	if got := req.Header.Get("Authorization"); got != "Bearer public-token" {
		t.Fatalf("unexpected authorization header: %q", got)
	}
}

func TestApplyWebAuthFallsBackToLegacyToken(t *testing.T) {
	t.Setenv("QUICKPLAN_REMOTE_TOKEN", "")
	t.Setenv("QUICKPLAN_WEB_TOKEN", "legacy-token")

	req, err := http.NewRequest(http.MethodGet, "http://example.test", nil)
	if err != nil {
		t.Fatalf("request creation failed: %v", err)
	}

	applyWebAuth(req)

	if got := req.Header.Get("Authorization"); got != "Bearer legacy-token" {
		t.Fatalf("unexpected authorization header: %q", got)
	}
}

func TestApplyWebAuthLeavesHeadersEmptyWithoutCredentials(t *testing.T) {
	for _, key := range []string{"QUICKPLAN_API_KEY", "QUICKPLAN_REMOTE_TOKEN", "QUICKPLAN_WEB_TOKEN"} {
		t.Setenv(key, "")
	}

	req, err := http.NewRequest(http.MethodGet, "http://example.test", nil)
	if err != nil {
		t.Fatalf("request creation failed: %v", err)
	}

	applyWebAuth(req)

	if got := req.Header.Get("Authorization"); got != "" {
		t.Fatalf("expected empty authorization header, got %q", got)
	}
	if got := req.Header.Get("X-API-Key"); got != "" {
		t.Fatalf("expected empty API key header, got %q", got)
	}
}

func TestLegacyTokenEnvironmentNameStillAvailable(t *testing.T) {
	if _, ok := os.LookupEnv("QUICKPLAN_WEB_TOKEN"); ok {
		t.Fatal("test expects QUICKPLAN_WEB_TOKEN to be unset in process environment")
	}
}
