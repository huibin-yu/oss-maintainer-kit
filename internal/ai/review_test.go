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
