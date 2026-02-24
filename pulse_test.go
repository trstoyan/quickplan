package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestPulsePayloadV2(t *testing.T) {
	// 1. Setup mock server
	var receivedPayload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedPayload)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	os.Setenv("QUICKPLAN_WEB_URL", server.URL)
	defer os.Unsetenv("QUICKPLAN_WEB_URL")

	// 2. Test with prevStatus
	SendPulse("test-proj", "test-agent", "t-1", "DONE", "IN_PROGRESS")

	// Wait a bit for the goroutine
	time.Sleep(100 * time.Millisecond)

	if receivedPayload == nil {
		t.Fatal("Mock server did not receive payload")
	}

	if receivedPayload["prev_status"] != "IN_PROGRESS" {
		t.Errorf("Expected prev_status IN_PROGRESS, got %v", receivedPayload["prev_status"])
	}

	// 3. Test without prevStatus
	receivedPayload = nil
	SendPulse("test-proj", "test-agent", "t-2", "PENDING", "")

	time.Sleep(100 * time.Millisecond)

	if _, ok := receivedPayload["prev_status"]; ok {
		t.Error("prev_status should be omitted when empty")
	}
}
