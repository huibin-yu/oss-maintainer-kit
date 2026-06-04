package healthtrend

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yuhuibin/oss-maintainer-kit/internal/health"
)

func TestNewSnapshotCountsPassedAndFailed(t *testing.T) {
	snapshot := NewSnapshot("demo", "abc123", health.Summary{
		Score: 50,
		Checks: []health.Check{
			{Name: "README", Passed: true},
			{Name: "CI", Passed: false},
		},
	}, time.Date(2026, 6, 4, 1, 2, 3, 0, time.UTC))

	if snapshot.Timestamp != "2026-06-04T01:02:03Z" {
		t.Fatalf("timestamp = %s", snapshot.Timestamp)
	}
	if snapshot.Passed != 1 || snapshot.Failed != 1 || snapshot.TotalChecks != 2 {
		t.Fatalf("unexpected counts: %#v", snapshot)
	}
}

func TestAppendLoadAndMarkdownTrend(t *testing.T) {
	path := filepath.Join(t.TempDir(), "health-history.jsonl")
	first := Snapshot{Timestamp: "2026-06-01T00:00:00Z", Project: "demo", Ref: "a1", Score: 80, TotalChecks: 10, Passed: 8, Failed: 2}
	second := Snapshot{Timestamp: "2026-06-02T00:00:00Z", Project: "demo", Ref: "b2", Score: 100, TotalChecks: 10, Passed: 10, Failed: 0}
	if err := Append(path, first); err != nil {
		t.Fatal(err)
	}
	if err := Append(path, second); err != nil {
		t.Fatal(err)
	}

	snapshots, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	trend := Analyze(snapshots)
	if trend.ScoreDelta != 20 || trend.LatestFailed != 0 {
		t.Fatalf("unexpected trend: %#v", trend)
	}
	doc := Markdown(trend)
	for _, want := range []string{"仓库健康度趋势报告", "评分变化：+20", "`b2`"} {
		if !strings.Contains(doc, want) {
			t.Fatalf("missing %q:\n%s", want, doc)
		}
	}
}

func TestLoadReportsInvalidJSONLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.jsonl")
	if err := os.WriteFile(path, []byte("{bad}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected parse error")
	}
	if !strings.Contains(err.Error(), "line 1") {
		t.Fatalf("unexpected error: %v", err)
	}
}
