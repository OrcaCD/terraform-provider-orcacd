package api

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBaseURLAndAuthorization(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/hub/api/v1/agents/test-id" {
			t.Errorf("unexpected request path %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer fixture-token" {
			t.Error("expected bearer authentication")
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"test-id","name":"fixture"}`)
	}))
	t.Cleanup(server.Close)

	client, err := New(server.URL+"/hub/", "fixture-token")
	if err != nil {
		t.Fatal(err)
	}
	agent, err := Decode[AgentResponse](client.GetAgent(t.Context(), "test-id"))
	if err != nil {
		t.Fatal(err)
	}
	if agent.Id != "test-id" || agent.Name != "fixture" {
		t.Errorf("unexpected agent %#v", agent)
	}
}

func TestResponseErrors(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		mediaType string
		missing   bool
	}{
		{"missing on hub", http.StatusNotFound, "application/problem+json", true},
		{"missing on proxy", http.StatusNotFound, "text/html", false},
		{"forbidden", http.StatusForbidden, "application/json", false},
		{"invalid secret", http.StatusBadRequest, "application/json", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body := `{"detail":"bad credential fixture-secret"}`
			response := &http.Response{
				StatusCode: test.status,
				Header:     http.Header{"Content-Type": []string{test.mediaType}},
				Body:       io.NopCloser(strings.NewReader(body)),
			}
			_, err := Decode[AgentResponse](response, nil)
			if err == nil {
				t.Fatal("expected API error")
			}
			if errors.Is(err, ErrNotFound) != test.missing {
				t.Errorf("unexpected missing resource status: %v", err)
			}
			if strings.Contains(err.Error(), "fixture-secret") {
				t.Error("API error leaked response contents")
			}
		})
	}
}
