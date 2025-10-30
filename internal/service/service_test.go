package service

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/FarisZR/knocker-cli/internal/api"
	"github.com/stretchr/testify/assert"
)

// Mocking the dependencies
type mockIPGetter struct{}

func (m *mockIPGetter) GetPublicIP(url string) (string, error) {
	return "1.2.3.4", nil
}

func TestServiceRun(t *testing.T) {
	// Mock server for the Knocker API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
		case "/knock":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(api.KnockResponse{
				WhitelistedEntry: "1.2.3.4",
				ExpiresAt:        time.Now().Unix() + 3600,
				ExpiresInSeconds: 3600,
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create a new service with mocked dependencies
	service := NewService(
		api.NewClient(server.URL, "test-key"),
		&mockIPGetter{},
		1*time.Millisecond,
		server.URL,
		3600,
		"check_interval",
		"test",
		log.New(os.Stdout, "test: ", log.LstdFlags),
	)

	// Create a quit channel for the test
	quit := make(chan struct{})

	// Run the service in a goroutine
	go service.Run(quit)

	// Allow the service to run for a short time
	time.Sleep(10 * time.Millisecond)

	// Stop the service
	service.Stop()

	// Assert that the IP was updated
	assert.Equal(t, "1.2.3.4", service.lastIP)
}

func TestServiceAdjustsCadenceFromServerTTL(t *testing.T) {
	logger := log.New(os.Stdout, "test: ", log.LstdFlags)
	service := NewService(
		nil,
		&mockIPGetter{},
		5*time.Minute,
		"",
		600,
		"ttl",
		"test",
		logger,
	)

	response := &api.KnockResponse{
		WhitelistedEntry: "1.2.3.4",
		ExpiresAt:        time.Now().Add(10 * time.Minute).Unix(),
		ExpiresInSeconds: 120,
	}

	service.handleWhitelistResponse(response, TriggerSourceSchedule)

	expected := KnockCadenceFromTTL(120)
	if service.Cadence != expected {
		t.Fatalf("expected cadence %v, got %v", expected, service.Cadence)
	}
	if service.cadenceSrc != "ttl_response" {
		t.Fatalf("expected cadence source ttl_response, got %s", service.cadenceSrc)
	}
}

func TestServiceDoesNotAdjustCadenceWhenIPCheckEnabled(t *testing.T) {
	logger := log.New(os.Stdout, "test: ", log.LstdFlags)
	service := NewService(
		nil,
		&mockIPGetter{},
		5*time.Minute,
		"https://example.com",
		600,
		"check_interval",
		"test",
		logger,
	)

	response := &api.KnockResponse{
		WhitelistedEntry: "1.2.3.4",
		ExpiresAt:        time.Now().Add(10 * time.Minute).Unix(),
		ExpiresInSeconds: 60,
	}

	service.handleWhitelistResponse(response, TriggerSourceSchedule)

	if service.Cadence != 5*time.Minute {
		t.Fatalf("expected cadence to remain unchanged, got %v", service.Cadence)
	}
	if service.cadenceSrc != "check_interval" {
		t.Fatalf("expected cadence source check_interval, got %s", service.cadenceSrc)
	}
}
