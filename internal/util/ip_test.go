package util

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetPublicIP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "8.8.8.8")
	}))
	defer server.Close()

	ipGetter := NewIPGetter()
	ip, err := ipGetter.GetPublicIP(server.URL)
	assert.NoError(t, err)
	assert.Equal(t, "8.8.8.8", ip)
}