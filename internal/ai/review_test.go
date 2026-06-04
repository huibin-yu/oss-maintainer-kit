package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPromptIncludesDiffAndRules(t *testing.T) {
	prompt := Prompt(ReviewRequest{
		Project: "demo",
		Diff:    "+const token = \"abc\"",
	})
	if !strings.Contains(prompt, "demo") || !strings.Contains(prompt, "+const token") {
		t.Fatalf("prompt missing content:\n%s", prompt)
	}
}

func TestReviewCallsOpenAICompatibleEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("authorization = %q", got)
		}
		var req chatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req.Model != "test-model" {
			t.Fatalf("model = %s", req.Model)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"review ok"}}]}`))
	}))
	defer server.Close()

	result, err := Client{BaseURL: server.URL, APIKey: "test-key", Model: "test-model"}.Review(context.Background(), ReviewRequest{
		Project: "demo",
		Diff:    "+fmt.Println(\"hello\")",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Content != "review ok" {
		t.Fatalf("content = %q", result.Content)
	}
}

func TestReviewRetriesTransientStatus(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			http.Error(w, "rate limited", http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"retry ok"}}]}`))
	}))
	defer server.Close()

	result, err := Client{BaseURL: server.URL, APIKey: "test-key", Model: "test-model", Retries: 1}.Review(context.Background(), ReviewRequest{
		Project: "demo",
		Diff:    "+fmt.Println(\"hello\")",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Content != "retry ok" || calls != 2 {
		t.Fatalf("content=%q calls=%d", result.Content, calls)
	}
}

func TestReviewDoesNotRetryClientError(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer server.Close()

	_, err := Client{BaseURL: server.URL, APIKey: "test-key", Model: "test-model", Retries: 2}.Review(context.Background(), ReviewRequest{
		Project: "demo",
		Diff:    "+fmt.Println(\"hello\")",
	})
	if err == nil {
		t.Fatal("expected client error")
	}
	if calls != 1 {
		t.Fatalf("calls=%d, want 1", calls)
	}
}
