package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yuhuibin/oss-maintainer-kit/internal/diffreview"
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

func TestPromptSanitizesInlineFindingText(t *testing.T) {
	prompt := Prompt(ReviewRequest{
		Project: "demo\n## injected",
		Diff:    "+fmt.Println(\"hello\")",
		Findings: []diffreview.Finding{{
			Severity: "critical\n- injected severity",
			File:     "main.go\n- injected file",
			Line:     3,
			Rule:     "possible-secret\n- injected rule",
			Message:  "secret\n- injected message",
		}},
	})

	for _, unwanted := range []string{
		"\n## injected",
		"\n- injected severity",
		"\n- injected file",
		"\n- injected rule",
		"\n- injected message",
	} {
		if strings.Contains(prompt, unwanted) {
			t.Fatalf("prompt contains unsanitized text %q:\n%s", unwanted, prompt)
		}
	}
	for _, want := range []string{
		"项目：demo ## injected",
		"- critical - injected severity main.go - injected file:3 possible-secret - injected rule：secret - injected message",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("missing sanitized prompt text %q:\n%s", want, prompt)
		}
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

func TestReviewFormatsProviderErrorJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"Invalid API key","type":"invalid_request_error","code":"invalid_api_key"}}`))
	}))
	defer server.Close()

	_, err := Client{BaseURL: server.URL, APIKey: "test-key", Model: "test-model"}.Review(context.Background(), ReviewRequest{
		Project: "demo",
		Diff:    "+fmt.Println(\"hello\")",
	})
	if err == nil {
		t.Fatal("expected provider error")
	}
	for _, want := range []string{"ai provider authentication failed", "Invalid API key", "invalid_request_error", "invalid_api_key"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("missing %q in error: %v", want, err)
		}
	}
	if strings.Contains(err.Error(), `{"error"`) {
		t.Fatalf("error should not expose raw JSON: %v", err)
	}
}

func TestReviewWrapsSuccessfulResponseDecodeErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html>not json</html>"))
	}))
	defer server.Close()

	_, err := Client{BaseURL: server.URL, APIKey: "test-key", Model: "test-model"}.Review(context.Background(), ReviewRequest{
		Project: "demo",
		Diff:    "+fmt.Println(\"hello\")",
	})
	if err == nil {
		t.Fatal("expected decode error")
	}
	for _, want := range []string{"decode ai provider response", "/chat/completions"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("missing %q in error: %v", want, err)
		}
	}
}
