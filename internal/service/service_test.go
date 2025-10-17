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
	service := &Service{
		APIClient:  api.NewClient(server.URL, "test-key"),
		IPGetter:   &mockIPGetter{},
		Interval:   1 * time.Millisecond, // Run once and exit
		Logger:     log.New(os.Stdout, "test: ", log.LstdFlags),
		stop:       make(chan struct{}),
		lastIP:     "",
		ipCheckURL: server.URL,
		ttl:        3600,
	}

	// Create a quit channel for the test
	quit := make(chan struct{})

	// Run the service in a goroutine
	go service.Run(quit)

	// Allow the service to run for a short time
	time.Sleep(10 * time.Millisecond)

	// Stop the service
	close(service.stop)

	// Assert that the IP was updated
	assert.Equal(t, "1.2.3.4", service.lastIP)
}