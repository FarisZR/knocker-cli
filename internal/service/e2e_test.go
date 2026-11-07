package service

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/FarisZR/knocker-cli/internal/api"
)

// Run against a real Knocker server configured to trust loopback as a proxy.
// The transport changes only the forwarded source address; all API responses,
// authentication, CIDR matching, and whitelist mutations come from Knocker.
func TestServerEndToEnd(t *testing.T) {
	url, key := os.Getenv("KNOCKER_E2E_URL"), os.Getenv("KNOCKER_E2E_KEY")
	if url == "" || key == "" {
		t.Skip("set KNOCKER_E2E_URL and KNOCKER_E2E_KEY for a local test server")
	}
	client := api.NewClient(url, key)
	transport := &sourceTransport{ip: "198.51.100.10", events: make(chan string, 100)}
	client.HTTPClient.Transport = transport
	if _, err := client.Knock("198.51.100.0/24", 60); err != nil {
		t.Fatal(err)
	}
	<-transport.events
	svc := NewService(client, time.Hour, true, time.Hour, 4, "ttl", "e2e", log.New(io.Discard, "", 0))
	changes := make(chan struct{}, 10)
	svc.networkChanges = changes
	svc.networkDebounce = 10 * time.Millisecond
	svc.retryDelay = 20 * time.Millisecond
	quit := make(chan struct{})
	done := make(chan struct{})
	go func() { defer close(done); svc.Run(quit) }()
	defer func() { close(quit); <-done }()
	next := func(want string) {
		t.Helper()
		select {
		case got := <-transport.events:
			if got != want {
				t.Fatalf("got %s, want %s", got, want)
			}
		case <-time.After(6 * time.Second):
			t.Fatalf("timed out waiting for %s", want)
		}
	}
	next("GET /status 200") // existing CIDR coverage: no startup knock
	transport.set("198.51.100.20", false)
	changes <- struct{}{}
	next("GET /status 200") // different IP, same CIDR: no knock
	transport.set("203.0.113.30", false)
	changes <- struct{}{}
	next("GET /status 200")
	next("POST /knock 200")
	// After switching to an uncovered IP, renewal follows the returned 4s TTL,
	// even without any further network event.
	next("POST /knock 200")
	transport.set("203.0.113.40", true)
	changes <- struct{}{}
	next("offline") // disconnect must not immediately attempt a knock
	transport.set("203.0.113.40", false)
	next("GET /status 200")
	next("POST /knock 200")
	// Query the actual server independently to confirm persisted coverage.
	req, _ := http.NewRequest(http.MethodGet, url+"/status", nil)
	req.Header.Set("X-Api-Key", key)
	req.Header.Set("X-Forwarded-For", "203.0.113.40")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var status api.StatusResponse
	if err := json.NewDecoder(res.Body).Decode(&status); err != nil {
		t.Fatal(err)
	}
	if !status.Whitelisted || status.IPAddress != "203.0.113.40" || status.ExpiresAt == nil {
		t.Fatalf("unexpected status: %+v", status)
	}
}

type sourceTransport struct {
	mu      sync.Mutex
	ip      string
	offline bool
	events  chan string
}

func (t *sourceTransport) set(ip string, offline bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.ip = ip
	t.offline = offline
}
func (t *sourceTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.mu.Lock()
	ip, offline := t.ip, t.offline
	t.mu.Unlock()
	if offline {
		t.events <- "offline"
		return nil, fmt.Errorf("simulated disconnect")
	}
	req = req.Clone(req.Context())
	req.Header.Set("X-Forwarded-For", ip)
	res, err := http.DefaultTransport.RoundTrip(req)
	if err == nil {
		t.events <- fmt.Sprintf("%s %s %d", req.Method, req.URL.Path, res.StatusCode)
	}
	return res, err
}
