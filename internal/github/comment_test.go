package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
