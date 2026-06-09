package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yuhuibin/oss-maintainer-kit/internal/checkrun"
)

func TestUpsertIssueCommentUpdatesMarkedComment(t *testing.T) {
	var patchedBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer gh_test" {
			t.Fatalf("authorization = %q", got)
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/acme/demo/issues/7/comments":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[
				{"id":100,"body":"human note","html_url":"https://github.com/acme/demo/pull/7#issuecomment-100"},
				{"id":101,"body":"<!-- oss-maintainer-kit:review-diff -->\nold","html_url":"https://github.com/acme/demo/pull/7#issuecomment-101"}
			]`))
		case r.Method == http.MethodPatch && r.URL.Path == "/repos/acme/demo/issues/comments/101":
			var payload commentRequest
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			patchedBody = payload.Body
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":101,"body":"updated","html_url":"https://github.com/acme/demo/pull/7#issuecomment-101"}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	comment, err := Client{BaseURL: server.URL, Token: "gh_test"}.UpsertIssueComment(context.Background(), "acme/demo", 7, "<!-- oss-maintainer-kit:review-diff -->", "new body")
	if err != nil {
		t.Fatal(err)
	}
	if comment.ID != 101 {
		t.Fatalf("id = %d, want 101", comment.ID)
	}
	if patchedBody != "new body" {
		t.Fatalf("patched body = %q", patchedBody)
	}
}

func TestUpsertIssueCommentCreatesWhenMarkerMissing(t *testing.T) {
	var createdBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/acme/demo/issues/8/comments":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"id":200,"body":"human note"}]`))
		case r.Method == http.MethodPost && r.URL.Path == "/repos/acme/demo/issues/8/comments":
			var payload commentRequest
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			createdBody = payload.Body
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":201,"body":"created","html_url":"https://github.com/acme/demo/pull/8#issuecomment-201"}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	comment, err := Client{BaseURL: server.URL}.UpsertIssueComment(context.Background(), "acme/demo", 8, "<!-- oss-maintainer-kit:review-diff -->", "created body")
	if err != nil {
		t.Fatal(err)
	}
	if comment.ID != 201 {
		t.Fatalf("id = %d, want 201", comment.ID)
	}
	if createdBody != "created body" {
		t.Fatalf("created body = %q", createdBody)
	}
}

func TestCreateCheckRunPostsPayload(t *testing.T) {
	var requestPath string
	var payload checkrun.Payload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer gh_test" {
			t.Fatalf("authorization = %q", got)
		}
		if r.Method != http.MethodPost || r.URL.Path != "/repos/acme/demo/check-runs" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		requestPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":501,"html_url":"https://github.com/acme/demo/runs/501"}`))
	}))
	defer server.Close()

	run, err := Client{BaseURL: server.URL, Token: "gh_test"}.CreateCheckRun(context.Background(), "acme/demo", checkrun.Payload{
		Name:       "oss-maintainer-kit review-diff",
		HeadSHA:    "abc123",
		Status:     "completed",
		Conclusion: "failure",
		Output: checkrun.Output{
			Title:   "PR diff 风险检查",
			Summary: "发现 1 个风险项。",
			Annotations: []checkrun.Annotation{{
				Path:            "main.go",
				StartLine:       3,
				EndLine:         3,
				AnnotationLevel: "failure",
				Message:         "secret",
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if requestPath != "/repos/acme/demo/check-runs" {
		t.Fatalf("path = %q", requestPath)
	}
	if payload.HeadSHA != "abc123" || payload.Conclusion != "failure" || len(payload.Output.Annotations) != 1 {
		t.Fatalf("payload = %#v", payload)
	}
	if run.ID != 501 || run.HTMLURL == "" {
		t.Fatalf("run = %#v", run)
	}
}

func TestCreateReleasePostsDraftPayload(t *testing.T) {
	var payload ReleasePayload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer gh_test" {
			t.Fatalf("authorization = %q", got)
		}
		if r.Method != http.MethodPost || r.URL.Path != "/repos/acme/demo/releases" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":701,"html_url":"https://github.com/acme/demo/releases/tag/v1.0.0","tag_name":"v1.0.0"}`))
	}))
	defer server.Close()

	release, err := Client{BaseURL: server.URL, Token: "gh_test"}.CreateRelease(context.Background(), "acme/demo", ReleasePayload{
		TagName:    "v1.0.0",
		Name:       "demo v1.0.0",
		Body:       "## 功能\n\n- feat: add export (#1) by @alice\n",
		Draft:      true,
		Prerelease: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if payload.TagName != "v1.0.0" || payload.Name != "demo v1.0.0" || !payload.Draft || !payload.Prerelease {
		t.Fatalf("payload = %#v", payload)
	}
	if release.ID != 701 || release.HTMLURL == "" || release.TagName != "v1.0.0" {
		t.Fatalf("release = %#v", release)
	}
}

func TestWriteMethodsRejectInvalidRepoBeforeRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("invalid repo should fail before request: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	client := Client{BaseURL: server.URL, Token: "gh_test"}
	cases := []struct {
		name string
		run  func() error
	}{
		{
			name: "issue comments",
			run: func() error {
				_, err := client.IssueComments(context.Background(), "acme", 1)
				return err
			},
		},
		{
			name: "create issue comment",
			run: func() error {
				_, err := client.CreateIssueComment(context.Background(), "acme", 1, "body")
				return err
			},
		},
		{
			name: "upsert issue comment",
			run: func() error {
				_, err := client.UpsertIssueComment(context.Background(), "acme", 1, "<!-- marker -->", "body")
				return err
			},
		},
		{
			name: "update issue comment",
			run: func() error {
				_, err := client.UpdateIssueComment(context.Background(), "acme", 10, "body")
				return err
			},
		},
		{
			name: "check run",
			run: func() error {
				_, err := client.CreateCheckRun(context.Background(), "acme", checkrun.Payload{HeadSHA: "abc123"})
				return err
			},
		},
		{
			name: "release",
			run: func() error {
				_, err := client.CreateRelease(context.Background(), "acme", ReleasePayload{TagName: "v1.0.0"})
				return err
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run()
			if err == nil {
				t.Fatal("expected invalid repo error")
			}
			if !strings.Contains(err.Error(), "expected owner/name or repository URL") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestRequestJSONReturnsActionableGitHubErrors(t *testing.T) {
	cases := []struct {
		name       string
		status     int
		body       string
		wantParts  []string
		callClient func(Client) error
	}{
		{
			name:      "unauthorized",
			status:    http.StatusUnauthorized,
			body:      `{"message":"Bad credentials"}`,
			wantParts: []string{"github authentication failed", "Bad credentials"},
			callClient: func(client Client) error {
				_, err := client.IssueComments(context.Background(), "acme/demo", 1)
				return err
			},
		},
		{
			name:      "not found",
			status:    http.StatusNotFound,
			body:      `{"message":"Not Found"}`,
			wantParts: []string{"github resource not found or inaccessible", "Not Found"},
			callClient: func(client Client) error {
				_, err := client.IssueComments(context.Background(), "acme/demo", 1)
				return err
			},
		},
		{
			name:      "validation failed",
			status:    http.StatusUnprocessableEntity,
			body:      `{"message":"Validation Failed","errors":[{"resource":"Release","field":"tag_name","code":"already_exists"}]}`,
			wantParts: []string{"github request validation failed", "tag_name", "already_exists"},
			callClient: func(client Client) error {
				_, err := client.CreateRelease(context.Background(), "acme/demo", ReleasePayload{TagName: "v1.0.0"})
				return err
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()

			err := tc.callClient(Client{BaseURL: server.URL, Token: "gh_test"})
			if err == nil {
				t.Fatal("expected github error")
			}
			for _, want := range tc.wantParts {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("missing %q in error: %v", want, err)
				}
			}
		})
	}
}

func TestRequestJSONWrapsSuccessfulResponseDecodeErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/repos/acme/demo/issues/1/comments" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html>not json</html>"))
	}))
	defer server.Close()

	_, err := Client{BaseURL: server.URL}.IssueComments(context.Background(), "acme/demo", 1)
	if err == nil {
		t.Fatal("expected decode error")
	}
	for _, want := range []string{"decode github response", "GET", "/repos/acme/demo/issues/1/comments"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("missing %q in error: %v", want, err)
		}
	}
}
