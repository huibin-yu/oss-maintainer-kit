package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/yuhuibin/oss-maintainer-kit/internal/ai"
	"github.com/yuhuibin/oss-maintainer-kit/internal/applicationpack"
	"github.com/yuhuibin/oss-maintainer-kit/internal/checkrun"
	"github.com/yuhuibin/oss-maintainer-kit/internal/codexplan"
	"github.com/yuhuibin/oss-maintainer-kit/internal/diffreview"
	"github.com/yuhuibin/oss-maintainer-kit/internal/github"
	"github.com/yuhuibin/oss-maintainer-kit/internal/health"
	"github.com/yuhuibin/oss-maintainer-kit/internal/healthtrend"
	"github.com/yuhuibin/oss-maintainer-kit/internal/input"
	"github.com/yuhuibin/oss-maintainer-kit/internal/prcomment"
	"github.com/yuhuibin/oss-maintainer-kit/internal/releasecheck"
	"github.com/yuhuibin/oss-maintainer-kit/internal/releasedraft"
	"github.com/yuhuibin/oss-maintainer-kit/internal/releasesummary"
	"github.com/yuhuibin/oss-maintainer-kit/internal/report"
	"github.com/yuhuibin/oss-maintainer-kit/internal/reviewconfig"
	"github.com/yuhuibin/oss-maintainer-kit/internal/sarif"
	"github.com/yuhuibin/oss-maintainer-kit/internal/sbom"
	"github.com/yuhuibin/oss-maintainer-kit/internal/securityreport"
	"github.com/yuhuibin/oss-maintainer-kit/internal/testplan"
	"github.com/yuhuibin/oss-maintainer-kit/internal/triage"
	"github.com/yuhuibin/oss-maintainer-kit/internal/triagecomment"
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
	case "triage-comment":
		return runTriageComment(args[1:])
	case "release-notes":
		return runReleaseNotes(args[1:])
	case "release-summary":
		return runReleaseSummary(args[1:])
	case "release-draft":
		return runReleaseDraft(args[1:])
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
	case "security-report":
		return runSecurityReport(args[1:])
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
	case "test-plan":
		return runTestPlan(args[1:])
	case "github-check-run":
		return runGitHubCheckRun(args[1:])
	case "github-release":
		return runGitHubRelease(args[1:])
	case "github-comment":
		return runGitHubComment(args[1:])
	case "github-triage-comment":
		return runGitHubTriageComment(args[1:])
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
	if err := validateFormat(*format, "table", "json"); err != nil {
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
	fmt.Fprintln(w, "ISSUE\tPRIORITY\tSTALE_DAYS\tLABELS\tACTION\tTITLE")
	for _, result := range results {
		fmt.Fprintf(w, "#%d\t%s\t%d\t%s\t%s\t%s\n", result.Number, result.Priority, result.StaleDays, strings.Join(result.Suggested, ","), result.Action, result.Title)
	}
	return w.Flush()
}

func runTriageComment(args []string) error {
	fs := flag.NewFlagSet("triage-comment", flag.ContinueOnError)
	inputPath := fs.String("input", "examples/issues.json", "issues JSON file")
	output := fs.String("output", "", "write Markdown comment to file")
	if err := fs.Parse(args); err != nil {
		return err
	}

	issues, err := input.Issues(*inputPath)
	if err != nil {
		return err
	}
	content := triagecomment.Markdown(triage.RuleSet{}.Issues(issues))
	if *output == "" {
		fmt.Print(content)
		return nil
	}
	return writeFile(*output, []byte(content), 0644)
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

func runReleaseSummary(args []string) error {
	fs := flag.NewFlagSet("release-summary", flag.ContinueOnError)
	inputPath := fs.String("input", "examples/pulls.json", "pull requests JSON file")
	project := fs.String("project", "oss-maintainer-kit", "project name")
	version := fs.String("version", "v0.1.0", "release version")
	providerPath := fs.String("provider-config", "", "optional OpenAI-compatible provider JSON config")
	baseURL := fs.String("base-url", "https://api.openai.com/v1", "OpenAI-compatible API base URL")
	model := fs.String("model", "gpt-4.1-mini", "summary model")
	apiKeyEnv := fs.String("api-key-env", "OPENAI_API_KEY", "environment variable that stores API key")
	retries := fs.Int("retries", 0, "retry count for transient AI provider failures")
	promptOnly := fs.Bool("prompt-only", true, "print summary prompt instead of calling API")
	output := fs.String("output", "", "write release summary to file")
	if err := fs.Parse(args); err != nil {
		return err
	}

	pulls, err := input.PullRequests(*inputPath)
	if err != nil {
		return err
	}
	summary := releasesummary.Build(releasesummary.Input{Project: *project, Version: *version, Pulls: pulls})
	var content string
	if *promptOnly {
		content = releasesummary.Markdown(summary)
	} else {
		provider, err := providerFromFlags(fs, *providerPath, *baseURL, *model, *apiKeyEnv, *retries)
		if err != nil {
			return err
		}
		apiKey, err := aiKeyFromEnv(provider.APIKeyEnv)
		if err != nil {
			return err
		}
		result, err := ai.Client{
			BaseURL: provider.BaseURL,
			APIKey:  apiKey,
			Model:   provider.Model,
			Retries: provider.Retries,
		}.Complete(context.Background(), "你是严谨的开源项目维护者，负责生成发布摘要。", summary.Prompt)
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
	return writeFile(*output, []byte(content), 0644)
}

func runReleaseDraft(args []string) error {
	fs := flag.NewFlagSet("release-draft", flag.ContinueOnError)
	inputPath := fs.String("input", "examples/pulls.json", "pull requests JSON file")
	project := fs.String("project", "oss-maintainer-kit", "project name")
	version := fs.String("version", "v0.1.0", "release version")
	previousTag := fs.String("previous-tag", "", "previous release tag for compare range")
	format := fs.String("format", "markdown", "output format: markdown or json")
	output := fs.String("output", "", "write release draft to file")
	prerelease := fs.Bool("prerelease", false, "mark the GitHub release payload as prerelease")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := validateFormat(*format, "markdown", "json"); err != nil {
		return err
	}

	pulls, err := input.PullRequests(*inputPath)
	if err != nil {
		return err
	}
	draft := releasedraft.Build(releasedraft.Input{
		Project:     *project,
		Version:     *version,
		PreviousTag: *previousTag,
		Pulls:       pulls,
		Prerelease:  *prerelease,
	})
	var content []byte
	if *format == "json" {
		content, err = json.MarshalIndent(draft, "", "  ")
		if err != nil {
			return err
		}
		content = append(content, '\n')
	} else {
		content = []byte(releasedraft.Markdown(draft))
	}
	if *output == "" {
		fmt.Print(string(content))
		return nil
	}
	return writeFile(*output, content, 0644)
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
	if err := validateFormat(*format, "markdown", "json"); err != nil {
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
	} else if err := writeFile(*output, content, 0644); err != nil {
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
	return writeFile(*output, []byte(doc), 0644)
}

func runReviewDiff(args []string) error {
	fs := flag.NewFlagSet("review-diff", flag.ContinueOnError)
	diffPath := fs.String("diff", "", "unified diff file, reads stdin when empty")
	configPath := fs.String("config", "", "optional JSON review rules config")
	format := fs.String("format", "markdown", "output format: markdown, json, sarif, comment, or check-run")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := validateFormat(*format, "markdown", "json", "sarif", "comment", "check-run"); err != nil {
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
	if *format == "check-run" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(checkrun.FromFindings(findings))
	}
	fmt.Print(diffreview.Markdown(findings))
	return nil
}

func runAIReview(args []string) error {
	fs := flag.NewFlagSet("ai-review", flag.ContinueOnError)
	diffPath := fs.String("diff", "", "unified diff file, reads stdin when empty")
	configPath := fs.String("config", "", "optional JSON review rules config")
	project := fs.String("project", "oss-maintainer-kit", "project name")
	providerPath := fs.String("provider-config", "", "optional OpenAI-compatible provider JSON config")
	baseURL := fs.String("base-url", "https://api.openai.com/v1", "OpenAI-compatible API base URL")
	model := fs.String("model", "gpt-4.1-mini", "review model")
	apiKeyEnv := fs.String("api-key-env", "OPENAI_API_KEY", "environment variable that stores API key")
	retries := fs.Int("retries", 0, "retry count for transient AI provider failures")
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
		provider, err := providerFromFlags(fs, *providerPath, *baseURL, *model, *apiKeyEnv, *retries)
		if err != nil {
			return err
		}
		apiKey, err := aiKeyFromEnv(provider.APIKeyEnv)
		if err != nil {
			return err
		}
		result, err := ai.Client{
			BaseURL: provider.BaseURL,
			APIKey:  apiKey,
			Model:   provider.Model,
			Retries: provider.Retries,
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
	return writeFile(*output, []byte(content), 0644)
}

func providerFromFlags(fs *flag.FlagSet, path, baseURL, model, apiKeyEnv string, retries int) (ai.ProviderConfig, error) {
	setFlags := map[string]bool{}
	fs.Visit(func(flag *flag.Flag) {
		setFlags[flag.Name] = true
	})
	provider, err := ai.LoadProviderConfig(path)
	if err != nil {
		return ai.ProviderConfig{}, err
	}
	if setFlags["base-url"] {
		provider.BaseURL = baseURL
	}
	if setFlags["model"] {
		provider.Model = model
	}
	if setFlags["api-key-env"] {
		provider.APIKeyEnv = apiKeyEnv
	}
	if setFlags["retries"] {
		provider.Retries = retries
	}
	if err := provider.Validate(); err != nil {
		return ai.ProviderConfig{}, err
	}
	return provider, nil
}

func runTestPlan(args []string) error {
	fs := flag.NewFlagSet("test-plan", flag.ContinueOnError)
	diffPath := fs.String("diff", "", "unified diff file, reads stdin when empty")
	configPath := fs.String("config", "", "optional JSON review rules config")
	project := fs.String("project", "oss-maintainer-kit", "project name")
	format := fs.String("format", "markdown", "output format: markdown or json")
	output := fs.String("output", "", "write test plan to file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := validateFormat(*format, "markdown", "json"); err != nil {
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
	plan := testplan.Build(testplan.Input{Project: *project, Diff: diff, Findings: findings})
	var content []byte
	if *format == "json" {
		content, err = json.MarshalIndent(plan, "", "  ")
		if err != nil {
			return err
		}
		content = append(content, '\n')
	} else {
		content = []byte(testplan.Markdown(plan))
	}
	if *output == "" {
		fmt.Print(string(content))
		return nil
	}
	return writeFile(*output, content, 0644)
}

func runGitHubComment(args []string) error {
	fs := flag.NewFlagSet("github-comment", flag.ContinueOnError)
	repo := fs.String("repo", "", "GitHub repository as owner/name or URL")
	number := fs.Int("pr", 0, "pull request number")
	diffPath := fs.String("diff", "", "unified diff file, reads stdin when empty")
	configPath := fs.String("config", "", "optional JSON review rules config")
	tokenEnv := fs.String("token-env", "GITHUB_TOKEN", "environment variable that stores GitHub token")
	baseURL := fs.String("base-url", "https://api.github.com", "GitHub API base URL")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := validateRepo(*repo); err != nil {
		return err
	}
	if *number <= 0 {
		return fmt.Errorf("pull request number is required")
	}
	if err := validateBaseURL(*baseURL); err != nil {
		return err
	}
	token, err := githubTokenFromEnv(*tokenEnv)
	if err != nil {
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
		Token:   token,
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

func runGitHubCheckRun(args []string) error {
	fs := flag.NewFlagSet("github-check-run", flag.ContinueOnError)
	repo := fs.String("repo", "", "GitHub repository as owner/name or URL")
	sha := fs.String("sha", "", "GitHub commit SHA for the check run head")
	diffPath := fs.String("diff", "", "unified diff file, reads stdin when empty")
	configPath := fs.String("config", "", "optional JSON review rules config")
	tokenEnv := fs.String("token-env", "GITHUB_TOKEN", "environment variable that stores GitHub token")
	baseURL := fs.String("base-url", "https://api.github.com", "GitHub API base URL")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := validateRepo(*repo); err != nil {
		return err
	}
	if strings.TrimSpace(*sha) == "" {
		return fmt.Errorf("head sha is required")
	}
	if err := validateBaseURL(*baseURL); err != nil {
		return err
	}
	token, err := githubTokenFromEnv(*tokenEnv)
	if err != nil {
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
	payload := checkrun.FromFindings(findings)
	payload.HeadSHA = *sha
	run, err := github.Client{
		BaseURL: *baseURL,
		Token:   token,
	}.CreateCheckRun(context.Background(), *repo, payload)
	if err != nil {
		return err
	}
	if run.HTMLURL != "" {
		fmt.Println(run.HTMLURL)
		return nil
	}
	fmt.Printf("created check run %d\n", run.ID)
	return nil
}

func runGitHubRelease(args []string) error {
	fs := flag.NewFlagSet("github-release", flag.ContinueOnError)
	repo := fs.String("repo", "", "GitHub repository as owner/name or URL")
	inputPath := fs.String("input", "examples/pulls.json", "pull requests JSON file")
	project := fs.String("project", "oss-maintainer-kit", "project name")
	version := fs.String("version", "v0.1.0", "release version")
	previousTag := fs.String("previous-tag", "", "previous release tag for compare range")
	prerelease := fs.Bool("prerelease", false, "mark the GitHub release as prerelease")
	dryRun := fs.Bool("dry-run", false, "print GitHub release payload without creating the release")
	tokenEnv := fs.String("token-env", "GITHUB_TOKEN", "environment variable that stores GitHub token")
	baseURL := fs.String("base-url", "https://api.github.com", "GitHub API base URL")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := validateRepo(*repo); err != nil {
		return err
	}
	if err := validateBaseURL(*baseURL); err != nil {
		return err
	}
	var token string
	if !*dryRun {
		var err error
		token, err = githubTokenFromEnv(*tokenEnv)
		if err != nil {
			return fmt.Errorf("%w; use --dry-run to preview the payload without a token", err)
		}
	}

	pulls, err := input.PullRequests(*inputPath)
	if err != nil {
		return err
	}
	draft := releasedraft.Build(releasedraft.Input{
		Project:     *project,
		Version:     *version,
		PreviousTag: *previousTag,
		Pulls:       pulls,
		Prerelease:  *prerelease,
	})
	payload := github.ReleasePayload{
		TagName:    draft.TagName,
		Name:       draft.Name,
		Body:       draft.Body,
		Draft:      true,
		Prerelease: draft.Prerelease,
	}
	if *dryRun {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(payload)
	}
	release, err := github.Client{
		BaseURL: *baseURL,
		Token:   token,
	}.CreateRelease(context.Background(), *repo, payload)
	if err != nil {
		return err
	}
	if release.HTMLURL != "" {
		fmt.Println(release.HTMLURL)
		return nil
	}
	fmt.Printf("created release %d\n", release.ID)
	return nil
}

func runGitHubTriageComment(args []string) error {
	fs := flag.NewFlagSet("github-triage-comment", flag.ContinueOnError)
	repo := fs.String("repo", "", "GitHub repository as owner/name or URL")
	number := fs.Int("issue", 0, "issue or pull request number")
	inputPath := fs.String("input", "examples/issues.json", "issues JSON file")
	tokenEnv := fs.String("token-env", "GITHUB_TOKEN", "environment variable that stores GitHub token")
	baseURL := fs.String("base-url", "https://api.github.com", "GitHub API base URL")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := validateRepo(*repo); err != nil {
		return err
	}
	if *number <= 0 {
		return fmt.Errorf("issue or pull request number is required")
	}
	if err := validateBaseURL(*baseURL); err != nil {
		return err
	}
	token, err := githubTokenFromEnv(*tokenEnv)
	if err != nil {
		return err
	}

	issues, err := input.Issues(*inputPath)
	if err != nil {
		return err
	}
	body := triagecomment.Markdown(triage.RuleSet{}.Issues(issues))
	comment, err := github.Client{
		BaseURL: *baseURL,
		Token:   token,
	}.UpsertIssueComment(context.Background(), *repo, *number, triagecomment.Marker, body)
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

func githubTokenFromEnv(name string) (string, error) {
	token := os.Getenv(name)
	if strings.TrimSpace(token) == "" {
		return "", fmt.Errorf("%s is required for GitHub write operations", name)
	}
	return token, nil
}

func githubAPITokenFromEnv(name string) (string, error) {
	token := os.Getenv(name)
	if strings.TrimSpace(token) == "" {
		return "", fmt.Errorf("%s is required for GitHub API calls", name)
	}
	return token, nil
}

func validateRepo(repo string) error {
	repo = github.NormalizeRepo(repo)
	parts := strings.Split(repo, "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return fmt.Errorf("repo is required, expected owner/name or repository URL")
	}
	return nil
}

func validateBaseURL(baseURL string) error {
	if strings.TrimSpace(baseURL) == "" {
		return fmt.Errorf("base-url must not be empty")
	}
	return nil
}

func aiKeyFromEnv(name string) (string, error) {
	token := os.Getenv(name)
	if strings.TrimSpace(token) == "" {
		return "", fmt.Errorf("%s is required for AI provider calls; use --prompt-only to preview the prompt without a token", name)
	}
	return token, nil
}

func validateFormat(format string, allowed ...string) error {
	for _, value := range allowed {
		if format == value {
			return nil
		}
	}
	return fmt.Errorf("unknown format %q, allowed: %s", format, strings.Join(allowed, ", "))
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

func writeFile(path string, data []byte, perm os.FileMode) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("output path is required")
	}
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, data, perm)
}

func runHealth(args []string) error {
	fs := flag.NewFlagSet("health", flag.ContinueOnError)
	root := fs.String("root", ".", "repository root")
	format := fs.String("format", "markdown", "output format: markdown or json")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := validateFormat(*format, "markdown", "json"); err != nil {
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
	if strings.TrimSpace(*project) == "" {
		return fmt.Errorf("project is required")
	}
	if strings.TrimSpace(*history) == "" {
		return fmt.Errorf("history path is required")
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
	if err := validateFormat(*format, "markdown", "json"); err != nil {
		return err
	}
	if strings.TrimSpace(*history) == "" {
		return fmt.Errorf("history path is required")
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

func runSecurityReport(args []string) error {
	fs := flag.NewFlagSet("security-report", flag.ContinueOnError)
	issuesPath := fs.String("issues", "examples/issues.json", "issues JSON file")
	diffPath := fs.String("diff", "", "optional unified diff file")
	configPath := fs.String("config", "", "optional JSON review rules config for diff scanning")
	root := fs.String("root", ".", "repository root for governance checks")
	project := fs.String("project", "oss-maintainer-kit", "project name")
	format := fs.String("format", "markdown", "output format: markdown or json")
	output := fs.String("output", "", "write security report to file")
	failOnRisk := fs.Bool("fail-on-risk", false, "exit with non-zero status when security report has blockers")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := validateFormat(*format, "markdown", "json"); err != nil {
		return err
	}

	issues, err := input.Issues(*issuesPath)
	if err != nil {
		return err
	}
	var findings []diffreview.Finding
	if *diffPath != "" {
		diff, err := readAll(*diffPath)
		if err != nil {
			return err
		}
		cfg, err := loadReviewConfig(*configPath)
		if err != nil {
			return err
		}
		findings, err = diffreview.ReviewWithConfig(strings.NewReader(diff), cfg)
		if err != nil {
			return err
		}
	}
	report := securityreport.Build(securityreport.Input{
		Project:  *project,
		Issues:   issues,
		Findings: findings,
		Health:   health.Repository(*root),
	})
	var content []byte
	if *format == "json" {
		content, err = json.MarshalIndent(report, "", "  ")
		if err != nil {
			return err
		}
		content = append(content, '\n')
	} else {
		content = []byte(securityreport.Markdown(report))
	}
	if *output == "" {
		fmt.Print(string(content))
	} else if err := writeFile(*output, content, 0644); err != nil {
		return err
	}
	if *failOnRisk && report.Blocked {
		return fmt.Errorf("security report blocked by risk: %d blocker(s)", len(report.Blockers))
	}
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
	return writeFile(*output, data, 0644)
}

func runCodexPlan(args []string) error {
	fs := flag.NewFlagSet("codex-plan", flag.ContinueOnError)
	issuesPath := fs.String("issues", "examples/issues.json", "issues JSON file")
	pullsPath := fs.String("pulls", "examples/pulls.json", "pull requests JSON file")
	project := fs.String("project", "oss-maintainer-kit", "project name")
	var repository string
	fs.StringVar(&repository, "repo-url", "", "public repository URL")
	fs.StringVar(&repository, "repository", "", "public repository URL (alias for --repo-url)")
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
	doc := codexplan.Markdown(codexplan.Build(*project, repository, summary))
	if *output == "" {
		fmt.Print(doc)
		return nil
	}
	return writeFile(*output, []byte(doc), 0644)
}

func runApplicationPack(args []string) error {
	fs := flag.NewFlagSet("application-pack", flag.ContinueOnError)
	issuesPath := fs.String("issues", "examples/issues.json", "issues JSON file")
	pullsPath := fs.String("pulls", "examples/pulls.json", "pull requests JSON file")
	root := fs.String("root", ".", "repository root for health checks")
	project := fs.String("project", "oss-maintainer-kit", "project name")
	var repository string
	fs.StringVar(&repository, "repo-url", "", "public repository URL")
	fs.StringVar(&repository, "repository", "", "public repository URL (alias for --repo-url)")
	version := fs.String("version", "v0.1.0", "release version for readiness evidence")
	policyPath := fs.String("policy", "examples/release-policy.json", "release policy JSON file for readiness evidence")
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
	repositoryHealth := health.Repository(*root)
	policy, err := releasecheck.LoadPolicy(*policyPath)
	if err != nil {
		return err
	}
	release := releasecheck.BuildWithPolicy(releasecheck.Input{
		Project: *project,
		Version: *version,
		Issues:  issues,
		Pulls:   pulls,
		Health:  repositoryHealth,
	}, policy)
	security := securityreport.Build(securityreport.Input{
		Project: *project,
		Issues:  issues,
		Health:  repositoryHealth,
	})
	doc := applicationpack.Markdown(applicationpack.Build(applicationpack.Input{
		Project:    *project,
		Repository: repository,
		Issues:     issues,
		Pulls:      pulls,
		Health:     repositoryHealth,
		Release:    release,
		Security:   security,
	}))
	if *output == "" {
		fmt.Print(doc)
		return nil
	}
	return writeFile(*output, []byte(doc), 0644)
}

func runGitHubExport(args []string) error {
	fs := flag.NewFlagSet("github-export", flag.ContinueOnError)
	repo := fs.String("repo", "", "GitHub repository as owner/name or URL")
	kind := fs.String("kind", "issues", "export kind: issues or pulls")
	api := fs.String("api", "rest", "GitHub API type: rest or graphql")
	limit := fs.Int("limit", 100, "max items to export")
	perPage := fs.Int("per-page", 100, "GitHub API page size, capped at 100")
	state := fs.String("state", "all", "GitHub state filter: open, closed, or all")
	since := fs.String("since", "", "only export items updated at or after RFC3339 time")
	tokenEnv := fs.String("token-env", "GITHUB_TOKEN", "environment variable that stores GitHub token")
	baseURL := fs.String("base-url", "https://api.github.com", "GitHub API base URL")
	graphqlURL := fs.String("graphql-url", "", "GitHub GraphQL API URL, defaults to base-url + /graphql")
	output := fs.String("output", "", "write JSON output to file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := validateGitHubExportOptions(*kind, *api, *state, *limit, *perPage); err != nil {
		return err
	}
	if err := validateBaseURL(*baseURL); err != nil {
		return err
	}

	client := github.Client{
		BaseURL: *baseURL,
		Token:   os.Getenv(*tokenEnv),
	}
	options := github.ExportOptions{
		Limit:   *limit,
		State:   *state,
		PerPage: *perPage,
	}
	if *since != "" {
		value, err := time.Parse(time.RFC3339, *since)
		if err != nil {
			return fmt.Errorf("invalid --since, expected RFC3339: %w", err)
		}
		options.Since = &value
	}
	var data any
	var err error
	switch *api {
	case "rest":
		switch *kind {
		case "issues":
			data, err = client.IssuesWithOptions(context.Background(), *repo, options)
		case "pulls":
			data, err = client.PullRequestsWithOptions(context.Background(), *repo, options)
		default:
			return fmt.Errorf("unknown kind %q", *kind)
		}
	case "graphql":
		token, err := githubAPITokenFromEnv(*tokenEnv)
		if err != nil {
			return err
		}
		client.Token = token
		if *graphqlURL != "" {
			client.BaseURL = githubGraphQLBaseURL(*graphqlURL)
		}
		switch *kind {
		case "issues":
			data, err = client.IssuesGraphQL(context.Background(), *repo, options)
		case "pulls":
			data, err = client.PullRequestsGraphQL(context.Background(), *repo, options)
		default:
			return fmt.Errorf("unknown kind %q", *kind)
		}
	default:
		return fmt.Errorf("unknown api %q", *api)
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
	return writeFile(*output, encoded, 0644)
}

func githubGraphQLBaseURL(raw string) string {
	return strings.TrimSuffix(strings.TrimSuffix(strings.TrimRight(raw, "/"), "/graphql"), "/")
}

func validateGitHubExportOptions(kind, api, state string, limit, perPage int) error {
	if err := validateChoice("kind", kind, "issues", "pulls"); err != nil {
		return err
	}
	if err := validateChoice("api", api, "rest", "graphql"); err != nil {
		return err
	}
	if err := validateChoice("state", state, "open", "closed", "all"); err != nil {
		return err
	}
	if limit <= 0 {
		return fmt.Errorf("limit must be greater than 0")
	}
	if perPage < 1 || perPage > 100 {
		return fmt.Errorf("per-page must be between 1 and 100")
	}
	return nil
}

func validateChoice(name, value string, allowed ...string) error {
	for _, item := range allowed {
		if value == item {
			return nil
		}
	}
	return fmt.Errorf("unknown %s %q, allowed: %s", name, value, strings.Join(allowed, ", "))
}

func usage() {
	fmt.Println(`oss-maintainer-kit - open source maintainer automation CLI

Usage:
  oss-maintainer-kit triage --input examples/issues.json [--format table|json]
  oss-maintainer-kit triage-comment --input examples/issues.json [--output triage-comment.md]
  oss-maintainer-kit release-notes --input examples/pulls.json --version v0.1.0
  oss-maintainer-kit release-summary --input examples/pulls.json --version v0.1.0 [--provider-config examples/ai-provider.json] --prompt-only
  oss-maintainer-kit release-draft --input examples/pulls.json --version v0.1.0 [--previous-tag v0.0.9] [--format markdown|json]
  oss-maintainer-kit release-check --issues examples/issues.json --pulls examples/pulls.json --root . --version v0.1.0 [--policy examples/release-policy.json] [--fail-on-blocked]
  oss-maintainer-kit report --issues examples/issues.json --pulls examples/pulls.json --output report.md
  oss-maintainer-kit health --root .
  oss-maintainer-kit health-snapshot --root . --history health-history.jsonl
  oss-maintainer-kit health-trend --history health-history.jsonl
  oss-maintainer-kit security-report --issues examples/issues.json [--diff examples/pr.diff] --root . [--fail-on-risk]
  oss-maintainer-kit sbom --root . --project oss-maintainer-kit --output sbom.spdx.json
  oss-maintainer-kit codex-plan --issues examples/issues.json --pulls examples/pulls.json --output codex-plan.md
  oss-maintainer-kit application-pack --issues examples/issues.json --pulls examples/pulls.json --root . [--version v0.1.0] [--policy examples/release-policy.json] --output codex-oss-application.md
  oss-maintainer-kit github-export --repo owner/name|URL --kind issues [--api rest|graphql] [--state open|closed|all] [--since RFC3339] [--limit 200] --output examples/issues.json
  oss-maintainer-kit review-diff --diff examples/pr.diff [--config examples/review-rules.json] [--format markdown|json|sarif|comment|check-run]
  oss-maintainer-kit ai-review --diff examples/pr.diff [--provider-config examples/ai-provider.json] --prompt-only
  oss-maintainer-kit test-plan --diff examples/pr.diff [--config examples/review-rules.json] [--format markdown|json]
  oss-maintainer-kit github-check-run --repo owner/name|URL --sha HEAD_SHA --diff examples/pr.diff [--config examples/review-rules.json]
  oss-maintainer-kit github-release --repo owner/name|URL --input examples/pulls.json --version v0.1.0 [--previous-tag v0.0.9] [--dry-run]
  oss-maintainer-kit github-comment --repo owner/name|URL --pr 123 --diff examples/pr.diff --config examples/review-rules.json
  oss-maintainer-kit github-triage-comment --repo owner/name|URL --issue 123 --input examples/issues.json`)
}
