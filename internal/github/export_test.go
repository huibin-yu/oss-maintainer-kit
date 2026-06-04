package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIssuesSkipsPullRequests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/acme/demo/issues" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"number":1,"title":"bug","body":"panic","state":"open","user":{"login":"alice"},"labels":[{"name":"bug"}],"created_at":"2026-06-01T00:00:00Z","updated_at":"2026-06-01T00:00:00Z"},
			{"number":2,"title":"pr","state":"open","user":{"login":"bob"},"labels":[],"created_at":"2026-06-01T00:00:00Z","updated_at":"2026-06-01T00:00:00Z","pull_request":{"url":"https://api.github.com/repos/acme/demo/pulls/2"}}
		]`))
	}))
	defer server.Close()

	issues, err := Client{BaseURL: server.URL}.Issues(context.Background(), "acme/demo", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 1 {
		t.Fatalf("len = %d, want 1", len(issues))
	}
	if issues[0].Author != "alice" || issues[0].Labels[0] != "bug" {
		t.Fatalf("unexpected issue: %#v", issues[0])
	}
}

func TestPullRequestsMapsMergedState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"number":3,"title":"fix","body":"","state":"closed","user":{"login":"carol"},"labels":[{"name":"bug"}],"created_at":"2026-06-01T00:00:00Z","updated_at":"2026-06-01T00:00:00Z","merged_at":"2026-06-02T00:00:00Z"}
		]`))
	}))
	defer server.Close()

	pulls, err := Client{BaseURL: server.URL}.PullRequests(context.Background(), "acme/demo", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(pulls) != 1 || !pulls[0].Merged {
		t.Fatalf("unexpected pulls: %#v", pulls)
	}
}
