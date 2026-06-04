package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunGitHubCommentCreatesPRComment(t *testing.T) {
	var createdBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/acme/demo/issues/9/comments":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[]`))
		case r.Method == http.MethodPost && r.URL.Path == "/repos/acme/demo/issues/9/comments":
			data, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}
			createdBody = string(data)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":301,"html_url":"https://github.com/acme/demo/pull/9#issuecomment-301"}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	diffPath := filepath.Join(t.TempDir(), "pr.diff")
	if err := os.WriteFile(diffPath, []byte(`diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,2 +1,3 @@
 package main
+const token = "sk_live_123456"
`), 0644); err != nil {
		t.Fatal(err)
	}

	err := run([]string{
		"github-comment",
		"--repo", "acme/demo",
		"--pr", "9",
		"--diff", diffPath,
		"--base-url", server.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(createdBody, "oss-maintainer-kit:review-diff") {
		t.Fatalf("missing marker in created body: %q", createdBody)
	}
}
