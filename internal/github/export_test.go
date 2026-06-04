package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
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

func TestIssuesWithOptionsPaginatesAndFilters(t *testing.T) {
	var pages []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pages = append(pages, r.URL.Query().Get("page"))
		if r.URL.Query().Get("state") != "open" {
			t.Fatalf("state = %s", r.URL.Query().Get("state"))
		}
		if r.URL.Query().Get("per_page") != "2" {
			t.Fatalf("per_page = %s", r.URL.Query().Get("per_page"))
		}
		if r.URL.Query().Get("since") != "2026-06-01T00:00:00Z" {
			t.Fatalf("since = %s", r.URL.Query().Get("since"))
		}
		w.Header().Set("Content-Type", "application/json")
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		switch page {
		case 1:
			_, _ = w.Write([]byte(`[
				{"number":1,"title":"one","state":"open","user":{"login":"alice"},"labels":[],"created_at":"2026-06-01T00:00:00Z","updated_at":"2026-06-01T00:00:00Z"},
				{"number":2,"title":"two","state":"open","user":{"login":"bob"},"labels":[],"created_at":"2026-06-01T00:00:00Z","updated_at":"2026-06-01T00:00:00Z"}
			]`))
		case 2:
			_, _ = w.Write([]byte(`[
				{"number":3,"title":"three","state":"open","user":{"login":"carol"},"labels":[],"created_at":"2026-06-01T00:00:00Z","updated_at":"2026-06-01T00:00:00Z"}
			]`))
		default:
			_, _ = w.Write([]byte(`[]`))
		}
	}))
	defer server.Close()

	since := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	issues, err := Client{BaseURL: server.URL}.IssuesWithOptions(context.Background(), "acme/demo", ExportOptions{
		Limit:   3,
		State:   "open",
		Since:   &since,
		PerPage: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 3 {
		t.Fatalf("len = %d, want 3", len(issues))
	}
	if len(pages) != 2 || pages[0] != "1" || pages[1] != "2" {
		t.Fatalf("pages = %#v", pages)
	}
}

func TestPullRequestsWithOptionsFiltersSinceClientSide(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("since") != "" {
			t.Fatalf("pulls request should not send since: %s", r.URL.RawQuery)
		}
		if r.URL.Query().Get("sort") != "updated" || r.URL.Query().Get("direction") != "desc" {
			t.Fatalf("missing updated sort query: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"number":1,"title":"new","state":"closed","user":{"login":"alice"},"labels":[],"created_at":"2026-06-01T00:00:00Z","updated_at":"2026-06-03T00:00:00Z","merged_at":"2026-06-03T00:00:00Z"},
			{"number":2,"title":"old","state":"closed","user":{"login":"bob"},"labels":[],"created_at":"2026-05-01T00:00:00Z","updated_at":"2026-05-01T00:00:00Z"}
		]`))
	}))
	defer server.Close()

	since := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	pulls, err := Client{BaseURL: server.URL}.PullRequestsWithOptions(context.Background(), "acme/demo", ExportOptions{
		Limit:   10,
		State:   "closed",
		Since:   &since,
		PerPage: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(pulls) != 1 || pulls[0].Number != 1 {
		t.Fatalf("unexpected pulls: %#v", pulls)
	}
}

func TestIssuesGraphQLPaginatesAndMapsNodes(t *testing.T) {
	var calls int
	var states []string
	var cursors []any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/graphql" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		var request graphqlRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.Variables["owner"] != "acme" || request.Variables["name"] != "demo" {
			t.Fatalf("variables = %#v", request.Variables)
		}
		rawStates, ok := request.Variables["states"].([]any)
		if !ok {
			t.Fatalf("states type = %T", request.Variables["states"])
		}
		states = append(states, rawStates[0].(string))
		cursors = append(cursors, request.Variables["after"])
		calls++
		w.Header().Set("Content-Type", "application/json")
		switch calls {
		case 1:
			_, _ = w.Write([]byte(`{
				"data":{"repository":{"issues":{
					"nodes":[{"number":1,"title":"security bug","body":"token","state":"OPEN","author":{"login":"alice"},"labels":{"nodes":[{"name":"security"}]},"createdAt":"2026-06-01T00:00:00Z","updatedAt":"2026-06-03T00:00:00Z","closedAt":null}],
					"pageInfo":{"hasNextPage":true,"endCursor":"cursor-1"}
				}}}
			}`))
		default:
			_, _ = w.Write([]byte(`{
				"data":{"repository":{"issues":{
					"nodes":[{"number":2,"title":"old bug","body":"","state":"OPEN","author":{"login":"bob"},"labels":{"nodes":[]},"createdAt":"2026-05-01T00:00:00Z","updatedAt":"2026-05-01T00:00:00Z","closedAt":null}],
					"pageInfo":{"hasNextPage":false,"endCursor":""}
				}}}
			}`))
		}
	}))
	defer server.Close()

	since := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	issues, err := Client{BaseURL: server.URL}.IssuesGraphQL(context.Background(), "acme/demo", ExportOptions{
		Limit:   2,
		State:   "open",
		Since:   &since,
		PerPage: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 1 || issues[0].Number != 1 || issues[0].State != "open" || issues[0].Labels[0] != "security" {
		t.Fatalf("unexpected issues: %#v", issues)
	}
	if calls != 2 {
		t.Fatalf("calls = %d", calls)
	}
	if states[0] != "OPEN" || cursors[0] != nil || cursors[1] != "cursor-1" {
		t.Fatalf("states=%#v cursors=%#v", states, cursors)
	}
}

func TestPullRequestsGraphQLMapsMergedState(t *testing.T) {
	var states []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request graphqlRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		rawStates, ok := request.Variables["states"].([]any)
		if !ok {
			t.Fatalf("states type = %T", request.Variables["states"])
		}
		for _, value := range rawStates {
			states = append(states, value.(string))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data":{"repository":{"pullRequests":{
				"nodes":[{"number":3,"title":"fix","body":"","state":"MERGED","author":{"login":"carol"},"labels":{"nodes":[{"name":"bug"}]},"createdAt":"2026-06-01T00:00:00Z","updatedAt":"2026-06-02T00:00:00Z","mergedAt":"2026-06-02T00:00:00Z"}],
				"pageInfo":{"hasNextPage":false,"endCursor":""}
			}}}
		}`))
	}))
	defer server.Close()

	pulls, err := Client{BaseURL: server.URL}.PullRequestsGraphQL(context.Background(), "acme/demo", ExportOptions{
		Limit: 10,
		State: "closed",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(pulls) != 1 || !pulls[0].Merged || pulls[0].Author != "carol" {
		t.Fatalf("unexpected pulls: %#v", pulls)
	}
	if len(states) != 2 || states[0] != "CLOSED" || states[1] != "MERGED" {
		t.Fatalf("states = %#v", states)
	}
}

func TestGraphQLErrorsReturnMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"errors":[{"message":"bad query"}]}`))
	}))
	defer server.Close()

	_, err := Client{BaseURL: server.URL}.IssuesGraphQL(context.Background(), "acme/demo", ExportOptions{})
	if err == nil {
		t.Fatal("expected graphql error")
	}
}
