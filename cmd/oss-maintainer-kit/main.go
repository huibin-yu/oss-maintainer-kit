package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/yuhuibin/oss-maintainer-kit/internal/ai"
	"github.com/yuhuibin/oss-maintainer-kit/internal/applicationpack"
	"github.com/yuhuibin/oss-maintainer-kit/internal/codexplan"
	"github.com/yuhuibin/oss-maintainer-kit/internal/diffreview"
	"github.com/yuhuibin/oss-maintainer-kit/internal/github"
	"github.com/yuhuibin/oss-maintainer-kit/internal/health"
	"github.com/yuhuibin/oss-maintainer-kit/internal/healthtrend"
	"github.com/yuhuibin/oss-maintainer-kit/internal/input"
	"github.com/yuhuibin/oss-maintainer-kit/internal/prcomment"
	"github.com/yuhuibin/oss-maintainer-kit/internal/releasecheck"
	"github.com/yuhuibin/oss-maintainer-kit/internal/report"
	"github.com/yuhuibin/oss-maintainer-kit/internal/reviewconfig"
	"github.com/yuhuibin/oss-maintainer-kit/internal/sarif"
	"github.com/yuhuibin/oss-maintainer-kit/internal/sbom"
	"github.com/yuhuibin/oss-maintainer-kit/internal/triage"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		usage()
		return nil
	}

	switch args[0] {
	case "triage":
		return runTriage(args[1:])
	case "release-notes":
		return runReleaseNotes(args[1:])
	case "release-check":
		return runReleaseCheck(args[1:])
	case "report":
		return runReport(args[1:])
	case "health":
		return runHealth(args[1:])
	case "health-snapshot":
		return runHealthSnapshot(args[1:])
	case "health-trend":
		return runHealthTrend(args[1:])
	case "sbom":
		return runSBOM(args[1:])
	case "codex-plan":
		return runCodexPlan(args[1:])
	case "application-pack":
		return runApplicationPack(args[1:])
	case "github-export":
		return runGitHubExport(args[1:])
	case "review-diff":
		return runReviewDiff(args[1:])
	case "ai-review":
		return runAIReview(args[1:])
	case "github-comment":
		return runGitHubComment(args[1:])
	case "help", "-h", "--help":
		usage()
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runTriage(args []string) error {
	fs := flag.NewFlagSet("triage", flag.ContinueOnError)
	inputPath := fs.String("input", "examples/issues.json", "issues JSON file")
	format := fs.String("format", "table", "output format: table or json")
	if err := fs.Parse(args); err != nil {
		return err
	}

	issues, err := input.Issues(*inputPath)
	if err != nil {
		return err
	}
	results := triage.RuleSet{}.Issues(issues)
	if *format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ISSUE\tPRIORITY\tSTALE_DAYS\tLABELS\tTITLE")
	for _, result := range results {
		fmt.Fprintf(w, "#%d\t%s\t%d\t%s\t%s\n", result.Number, result.Priority, result.StaleDays, strings.Join(result.Suggested, ","), result.Title)
	}
	return w.Flush()
}

func runReleaseNotes(args []string) error {
	fs := flag.NewFlagSet("release-notes", flag.ContinueOnError)
	inputPath := fs.String("input", "examples/pulls.json", "pull requests JSON file")
	version := fs.String("version", "v0.1.0", "release version")
	if err := fs.Parse(args); err != nil {
		return err
	}

	pulls, err := input.PullRequests(*inputPath)
	if err != nil {
		return err
	}
	fmt.Print(report.ReleaseNotes(*version, pulls))
	return nil
}

func runReleaseCheck(args []string) error {
	fs := flag.NewFlagSet("release-check", flag.ContinueOnError)
	issuesPath := fs.String("issues", "examples/issues.json", "issues JSON file")
	pullsPath := fs.String("pulls", "examples/pulls.json", "pull requests JSON file")
	root := fs.String("root", ".", "repository root for health checks")
	project := fs.String("project", "oss-maintainer-kit", "project name")
	version := fs.String("version", "v0.1.0", "release version")
	policyPath := fs.String("policy", "", "optional release policy JSON file")
	format := fs.String("format", "markdown", "output format: markdown or json")
	output := fs.String("output", "", "write release check report to file")
	failOnBlocked := fs.Bool("fail-on-blocked", false, "exit with non-zero status when release readiness is blocked")
	if err := fs.Parse(args); err != nil {
		return err
	}

	issues, err := input.Issues(*issuesPath)
	if err != nil {
		return err
	}
	pulls, err := input.PullRequests(*pullsPath)
	if err != nil {
		return err
	}
	policy, err := releasecheck.LoadPolicy(*policyPath)
	if err != nil {
		return err
	}
	result := releasecheck.BuildWithPolicy(releasecheck.Input{
		Project: *project,
		Version: *version,
		Issues:  issues,
		Pulls:   pulls,
		Health:  health.Repository(*root),
	}, policy)
	var content []byte
	if *format == "json" {
		content, err = json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		content = append(content, '\n')
	} else {
		content = []byte(releasecheck.Markdown(result))
	}
	if *output == "" {
		fmt.Print(string(content))
	} else if err := os.WriteFile(*output, content, 0644); err != nil {
		return err
	}
	if *failOnBlocked && !result.Ready {
		return fmt.Errorf("release blocked by policy: %d blocker(s)", len(result.Blockers))
	}
	return nil
}

func runReport(args []string) error {
	fs := flag.NewFlagSet("report", flag.ContinueOnError)
	issuesPath := fs.String("issues", "examples/issues.json", "issues JSON file")
	pullsPath := fs.String("pulls", "examples/pulls.json", "pull requests JSON file")
	project := fs.String("project", "oss-maintainer-kit", "project name")
	output := fs.String("output", "", "write markdown report to file")
	if err := fs.Parse(args); err != nil {
		return err
	}

	issues, err := input.Issues(*issuesPath)
	if err != nil {
		return err
	}
	pulls, err := input.PullRequests(*pullsPath)
	if err != nil {
		return err
	}
	doc := report.Markdown(*project, report.Maintainer(issues, pulls, triage.RuleSet{}))
	if *output == "" {
		fmt.Print(doc)
		return nil
	}
	return os.WriteFile(*output, []byte(doc), 0644)
}

func runReviewDiff(args []string) error {
	fs := flag.NewFlagSet("review-diff", flag.ContinueOnError)
	diffPath := fs.String("diff", "", "unified diff file, reads stdin when empty")
	configPath := fs.String("config", "", "optional JSON review rules config")
	format := fs.String("format", "markdown", "output format: markdown, json, sarif, or comment")
	if err := fs.Parse(args); err != nil {
		return err
	}

	diff, err := readAll(*diffPath)
	if err != nil {
		return err
	}
	cfg, err := loadReviewConfig(*configPath)
	if err != nil {
		return err
	}
	findings, err := diffreview.ReviewWithConfig(strings.NewReader(diff), cfg)
	if err != nil {
		return err
	}
	if *format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(findings)
	}
	if *format == "sarif" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(sarif.FromFindings(findings))
	}
	if *format == "comment" {
		fmt.Print(prcomment.Markdown(findings))
		return nil
	}
	fmt.Print(diffreview.Markdown(findings))
	return nil
}

func runAIReview(args []string) error {
	fs := flag.NewFlagSet("ai-review", flag.ContinueOnError)
	diffPath := fs.String("diff", "", "unified diff file, reads stdin when empty")
	configPath := fs.String("config", "", "optional JSON review rules config")
	project := fs.String("project", "oss-maintainer-kit", "project name")
	baseURL := fs.String("base-url", "https://api.openai.com/v1", "OpenAI-compatible API base URL")
	model := fs.String("model", "gpt-4.1-mini", "review model")
	apiKeyEnv := fs.String("api-key-env", "OPENAI_API_KEY", "environment variable that stores API key")
	promptOnly := fs.Bool("prompt-only", true, "print review prompt instead of calling API")
	output := fs.String("output", "", "write review output to file")
	if err := fs.Parse(args); err != nil {
		return err
	}

	diff, err := readAll(*diffPath)
	if err != nil {
		return err
	}
	cfg, err := loadReviewConfig(*configPath)
	if err != nil {
		return err
	}
	findings, err := diffreview.ReviewWithConfig(strings.NewReader(diff), cfg)
	if err != nil {
		return err
	}
	req := ai.ReviewRequest{Project: *project, Diff: diff, Findings: findings}
	var content string
	if *promptOnly {
		content = ai.Prompt(req)
	} else {
		result, err := ai.Client{
			BaseURL: *baseURL,
			APIKey:  os.Getenv(*apiKeyEnv),
			Model:   *model,
		}.Review(context.Background(), req)
		if err != nil {
			return err
		}
		content = result.Content
	}
	if *output == "" {
		fmt.Print(content)
		if !strings.HasSuffix(content, "\n") {
			fmt.Println()
		}
		return nil
	}
	return os.WriteFile(*output, []byte(content), 0644)
}

func runGitHubComment(args []string) error {
	fs := flag.NewFlagSet("github-comment", flag.ContinueOnError)
	repo := fs.String("repo", "", "GitHub repository in owner/name format")
	number := fs.Int("pr", 0, "pull request number")
	diffPath := fs.String("diff", "", "unified diff file, reads stdin when empty")
	configPath := fs.String("config", "", "optional JSON review rules config")
	tokenEnv := fs.String("token-env", "GITHUB_TOKEN", "environment variable that stores GitHub token")
	baseURL := fs.String("base-url", "https://api.github.com", "GitHub API base URL")
	if err := fs.Parse(args); err != nil {
		return err
	}

	diff, err := readAll(*diffPath)
	if err != nil {
		return err
	}
	cfg, err := loadReviewConfig(*configPath)
	if err != nil {
		return err
	}
	findings, err := diffreview.ReviewWithConfig(strings.NewReader(diff), cfg)
	if err != nil {
		return err
	}
	body := prcomment.Markdown(findings)
	comment, err := github.Client{
		BaseURL: *baseURL,
		Token:   os.Getenv(*tokenEnv),
	}.UpsertIssueComment(context.Background(), *repo, *number, "<!-- oss-maintainer-kit:review-diff -->", body)
	if err != nil {
		return err
	}
	if comment.HTMLURL != "" {
		fmt.Println(comment.HTMLURL)
		return nil
	}
	fmt.Printf("updated comment %d\n", comment.ID)
	return nil
}

func loadReviewConfig(path string) (reviewconfig.Config, error) {
	if path == "" {
		return reviewconfig.Config{}, nil
	}
	return reviewconfig.Load(path)
}

func readAll(path string) (string, error) {
	var data []byte
	var err error
	if path == "" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func runHealth(args []string) error {
	fs := flag.NewFlagSet("health", flag.ContinueOnError)
	root := fs.String("root", ".", "repository root")
	format := fs.String("format", "markdown", "output format: markdown or json")
	if err := fs.Parse(args); err != nil {
		return err
	}

	summary := health.Repository(*root)
	if *format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(summary)
	}
	fmt.Print(health.Markdown(summary))
	return nil
}

func runHealthSnapshot(args []string) error {
	fs := flag.NewFlagSet("health-snapshot", flag.ContinueOnError)
	root := fs.String("root", ".", "repository root")
	project := fs.String("project", "oss-maintainer-kit", "project name")
	history := fs.String("history", "health-history.jsonl", "append health snapshot to JSONL file")
	ref := fs.String("ref", "", "git ref for this snapshot")
	if err := fs.Parse(args); err != nil {
		return err
	}

	gitRef := *ref
	if gitRef == "" {
		gitRef = currentGitRef(*root)
	}
	snapshot := healthtrend.NewSnapshot(*project, gitRef, health.Repository(*root), time.Now().UTC())
	if err := healthtrend.Append(*history, snapshot); err != nil {
		return err
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func runHealthTrend(args []string) error {
	fs := flag.NewFlagSet("health-trend", flag.ContinueOnError)
	history := fs.String("history", "health-history.jsonl", "read health snapshots from JSONL file")
	format := fs.String("format", "markdown", "output format: markdown or json")
	if err := fs.Parse(args); err != nil {
		return err
	}

	snapshots, err := healthtrend.Load(*history)
	if err != nil {
		return err
	}
	trend := healthtrend.Analyze(snapshots)
	if *format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(trend)
	}
	fmt.Print(healthtrend.Markdown(trend))
	return nil
}

func currentGitRef(root string) string {
	cmd := exec.Command("git", "-C", root, "rev-parse", "--short", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func runSBOM(args []string) error {
	fs := flag.NewFlagSet("sbom", flag.ContinueOnError)
	root := fs.String("root", ".", "repository root")
	project := fs.String("project", "oss-maintainer-kit", "project name")
	namespace := fs.String("namespace", "", "SPDX document namespace")
	output := fs.String("output", "", "write SPDX JSON SBOM to file")
	if err := fs.Parse(args); err != nil {
		return err
	}

	doc, err := sbom.Build(sbom.Options{
		Root:      *root,
		Name:      *project,
		Namespace: *namespace,
	})
	if err != nil {
		return err
	}
	data, err := sbom.JSON(doc)
	if err != nil {
		return err
	}
	if *output == "" {
		fmt.Print(string(data))
		return nil
	}
	return os.WriteFile(*output, data, 0644)
}

func runCodexPlan(args []string) error {
	fs := flag.NewFlagSet("codex-plan", flag.ContinueOnError)
	issuesPath := fs.String("issues", "examples/issues.json", "issues JSON file")
	pullsPath := fs.String("pulls", "examples/pulls.json", "pull requests JSON file")
	project := fs.String("project", "oss-maintainer-kit", "project name")
	repository := fs.String("repo-url", "", "public repository URL")
	output := fs.String("output", "", "write markdown plan to file")
	if err := fs.Parse(args); err != nil {
		return err
	}

	issues, err := input.Issues(*issuesPath)
	if err != nil {
		return err
	}
	pulls, err := input.PullRequests(*pullsPath)
	if err != nil {
		return err
	}
	summary := report.Maintainer(issues, pulls, triage.RuleSet{})
	doc := codexplan.Markdown(codexplan.Build(*project, *repository, summary))
	if *output == "" {
		fmt.Print(doc)
		return nil
	}
	return os.WriteFile(*output, []byte(doc), 0644)
}

func runApplicationPack(args []string) error {
	fs := flag.NewFlagSet("application-pack", flag.ContinueOnError)
	issuesPath := fs.String("issues", "examples/issues.json", "issues JSON file")
	pullsPath := fs.String("pulls", "examples/pulls.json", "pull requests JSON file")
	root := fs.String("root", ".", "repository root for health checks")
	project := fs.String("project", "oss-maintainer-kit", "project name")
	repository := fs.String("repo-url", "", "public repository URL")
	output := fs.String("output", "", "write markdown application pack to file")
	if err := fs.Parse(args); err != nil {
		return err
	}

	issues, err := input.Issues(*issuesPath)
	if err != nil {
		return err
	}
	pulls, err := input.PullRequests(*pullsPath)
	if err != nil {
		return err
	}
	doc := applicationpack.Markdown(applicationpack.Build(applicationpack.Input{
		Project:    *project,
		Repository: *repository,
		Issues:     issues,
		Pulls:      pulls,
		Health:     health.Repository(*root),
	}))
	if *output == "" {
		fmt.Print(doc)
		return nil
	}
	return os.WriteFile(*output, []byte(doc), 0644)
}

func runGitHubExport(args []string) error {
	fs := flag.NewFlagSet("github-export", flag.ContinueOnError)
	repo := fs.String("repo", "", "GitHub repository in owner/name format")
	kind := fs.String("kind", "issues", "export kind: issues or pulls")
	limit := fs.Int("limit", 100, "max items to export, capped at 100")
	tokenEnv := fs.String("token-env", "GITHUB_TOKEN", "environment variable that stores GitHub token")
	baseURL := fs.String("base-url", "https://api.github.com", "GitHub API base URL")
	output := fs.String("output", "", "write JSON output to file")
	if err := fs.Parse(args); err != nil {
		return err
	}

	client := github.Client{
		BaseURL: *baseURL,
		Token:   os.Getenv(*tokenEnv),
	}
	var data any
	var err error
	switch *kind {
	case "issues":
		data, err = client.Issues(context.Background(), *repo, *limit)
	case "pulls":
		data, err = client.PullRequests(context.Background(), *repo, *limit)
	default:
		return fmt.Errorf("unknown kind %q", *kind)
	}
	if err != nil {
		return err
	}

	encoded, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	if *output == "" {
		fmt.Print(string(encoded))
		return nil
	}
	return os.WriteFile(*output, encoded, 0644)
}

func usage() {
	fmt.Println(`oss-maintainer-kit - open source maintainer automation CLI

Usage:
  oss-maintainer-kit triage --input examples/issues.json [--format table|json]
  oss-maintainer-kit release-notes --input examples/pulls.json --version v0.1.0
  oss-maintainer-kit release-check --issues examples/issues.json --pulls examples/pulls.json --root . --version v0.1.0 [--policy examples/release-policy.json] [--fail-on-blocked]
  oss-maintainer-kit report --issues examples/issues.json --pulls examples/pulls.json --output report.md
  oss-maintainer-kit health --root .
  oss-maintainer-kit health-snapshot --root . --history health-history.jsonl
  oss-maintainer-kit health-trend --history health-history.jsonl
  oss-maintainer-kit sbom --root . --project oss-maintainer-kit --output sbom.spdx.json
  oss-maintainer-kit codex-plan --issues examples/issues.json --pulls examples/pulls.json --output codex-plan.md
  oss-maintainer-kit application-pack --issues examples/issues.json --pulls examples/pulls.json --root . --output codex-oss-application.md
  oss-maintainer-kit github-export --repo owner/name --kind issues --output examples/issues.json
  oss-maintainer-kit review-diff --diff examples/pr.diff [--config examples/review-rules.json] [--format markdown|json|sarif|comment]
  oss-maintainer-kit ai-review --diff examples/pr.diff --prompt-only
  oss-maintainer-kit github-comment --repo owner/name --pr 123 --diff examples/pr.diff --config examples/review-rules.json`)
}
