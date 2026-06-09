package healthtrend

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yuhuibin/oss-maintainer-kit/internal/health"
)

type Snapshot struct {
	Timestamp   string `json:"timestamp"`
	Project     string `json:"project"`
	Ref         string `json:"ref,omitempty"`
	Score       int    `json:"score"`
	TotalChecks int    `json:"total_checks"`
	Passed      int    `json:"passed"`
	Failed      int    `json:"failed"`
}

type Trend struct {
	Project      string
	Snapshots    []Snapshot
	FirstScore   int
	LatestScore  int
	ScoreDelta   int
	LatestFailed int
}

func NewSnapshot(project, ref string, summary health.Summary, now time.Time) Snapshot {
	passed := 0
	for _, check := range summary.Checks {
		if check.Passed {
			passed++
		}
	}
	return Snapshot{
		Timestamp:   now.UTC().Format(time.RFC3339),
		Project:     project,
		Ref:         ref,
		Score:       summary.Score,
		TotalChecks: len(summary.Checks),
		Passed:      passed,
		Failed:      len(summary.Checks) - passed,
	}
}

func Append(path string, snapshot Snapshot) error {
	if path == "" {
		return fmt.Errorf("history path is required")
	}
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	data, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		return err
	}
	return nil
}

func Load(path string) ([]Snapshot, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var snapshots []Snapshot
	scanner := bufio.NewScanner(file)
	line := 0
	for scanner.Scan() {
		line++
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}
		var snapshot Snapshot
		if err := json.Unmarshal([]byte(text), &snapshot); err != nil {
			return nil, fmt.Errorf("parse %s line %d: %w", path, line, err)
		}
		snapshots = append(snapshots, snapshot)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return snapshots, nil
}

func Analyze(snapshots []Snapshot) Trend {
	trend := Trend{Snapshots: snapshots}
	if len(snapshots) == 0 {
		return trend
	}
	first := snapshots[0]
	latest := snapshots[len(snapshots)-1]
	trend.Project = latest.Project
	trend.FirstScore = first.Score
	trend.LatestScore = latest.Score
	trend.ScoreDelta = latest.Score - first.Score
	trend.LatestFailed = latest.Failed
	return trend
}

func Markdown(trend Trend) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# 仓库健康度趋势报告\n\n")
	if len(trend.Snapshots) == 0 {
		fmt.Fprintf(&b, "暂无健康度快照。\n")
		return b.String()
	}
	fmt.Fprintf(&b, "项目：%s\n\n", trend.Project)
	fmt.Fprintf(&b, "快照数量：%d\n\n", len(trend.Snapshots))
	fmt.Fprintf(&b, "首个评分：%d/100\n\n", trend.FirstScore)
	fmt.Fprintf(&b, "最新评分：%d/100\n\n", trend.LatestScore)
	fmt.Fprintf(&b, "评分变化：%+d\n\n", trend.ScoreDelta)
	fmt.Fprintf(&b, "最新失败项：%d\n\n", trend.LatestFailed)
	fmt.Fprintf(&b, "| 时间 | Git Ref | 评分 | 通过 | 失败 |\n")
	fmt.Fprintf(&b, "| --- | --- | ---: | ---: | ---: |\n")
	for _, snapshot := range trend.Snapshots {
		ref := snapshot.Ref
		if ref == "" {
			ref = "unknown"
		}
		fmt.Fprintf(&b, "| %s | `%s` | %d | %d | %d |\n",
			markdownCell(snapshot.Timestamp),
			markdownCell(ref),
			snapshot.Score,
			snapshot.Passed,
			snapshot.Failed,
		)
	}
	return b.String()
}

func markdownCell(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	value = strings.ReplaceAll(value, "\n", "<br>")
	value = strings.ReplaceAll(value, "|", "\\|")
	return value
}
