package github

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/yuhuibin/oss-maintainer-kit/internal/model"
)

type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

type ExportOptions struct {
	Limit   int
	State   string
	Since   *time.Time
	PerPage int
}

func (c Client) Issues(ctx context.Context, repo string, limit int) ([]model.Issue, error) {
	return c.IssuesWithOptions(ctx, repo, ExportOptions{Limit: limit})
}

func (c Client) IssuesWithOptions(ctx context.Context, repo string, options ExportOptions) ([]model.Issue, error) {
	var raw []issueResponse
	if err := c.fetchAll(ctx, repo, "issues", options, &raw); err != nil {
		return nil, err
	}
	issues := make([]model.Issue, 0, len(raw))
	for _, item := range raw {
		if item.PullRequest.URL != "" {
			continue
		}
		issues = append(issues, model.Issue{
			Number:    item.Number,
			Title:     item.Title,
			Body:      item.Body,
			State:     item.State,
			Author:    item.User.Login,
			Labels:    labels(item.Labels),
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
			ClosedAt:  item.ClosedAt,
		})
	}
	return issues, nil
}

func (c Client) PullRequests(ctx context.Context, repo string, limit int) ([]model.PullRequest, error) {
	return c.PullRequestsWithOptions(ctx, repo, ExportOptions{Limit: limit})
}

func (c Client) PullRequestsWithOptions(ctx context.Context, repo string, options ExportOptions) ([]model.PullRequest, error) {
	var raw []pullResponse
	if err := c.fetchAll(ctx, repo, "pulls", options, &raw); err != nil {
		return nil, err
	}
	pulls := make([]model.PullRequest, 0, len(raw))
	for _, item := range raw {
		pulls = append(pulls, model.PullRequest{
			Number:    item.Number,
			Title:     item.Title,
			Body:      item.Body,
			State:     item.State,
			Author:    item.User.Login,
			Labels:    labels(item.Labels),
			Merged:    item.MergedAt != nil,
			MergedAt:  item.MergedAt,
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
		})
	}
	return pulls, nil
}

func (c Client) fetchAll(ctx context.Context, repo, resource string, options ExportOptions, dst any) error {
	if repo == "" {
		return fmt.Errorf("repo is required, expected owner/name")
	}
	options = options.withDefaults()
	switch out := dst.(type) {
	case *[]issueResponse:
		var all []issueResponse
		for page := 1; len(all) < options.Limit; page++ {
			var batch []issueResponse
			if err := c.requestJSON(ctx, "GET", fmt.Sprintf("/repos/%s/%s", repo, resource), exportQuery(resource, options, page), nil, &batch); err != nil {
				return err
			}
			if len(batch) == 0 {
				break
			}
			for _, item := range batch {
				if matchesSince(item.UpdatedAt, options.Since) {
					all = append(all, item)
				}
				if len(all) >= options.Limit {
					break
				}
			}
			if len(batch) < options.PerPage {
				break
			}
		}
		if len(all) > options.Limit {
			all = all[:options.Limit]
		}
		*out = all
	case *[]pullResponse:
		var all []pullResponse
		for page := 1; len(all) < options.Limit; page++ {
			var batch []pullResponse
			if err := c.requestJSON(ctx, "GET", fmt.Sprintf("/repos/%s/%s", repo, resource), exportQuery(resource, options, page), nil, &batch); err != nil {
				return err
			}
			if len(batch) == 0 {
				break
			}
			olderThanSince := 0
			for _, item := range batch {
				if matchesSince(item.UpdatedAt, options.Since) {
					all = append(all, item)
				} else {
					olderThanSince++
				}
				if len(all) >= options.Limit {
					break
				}
			}
			if len(batch) < options.PerPage {
				break
			}
			if options.Since != nil && olderThanSince == len(batch) {
				break
			}
		}
		if len(all) > options.Limit {
			all = all[:options.Limit]
		}
		*out = all
	default:
		return fmt.Errorf("unsupported export destination %T", dst)
	}
	return nil
}

func (o ExportOptions) withDefaults() ExportOptions {
	if o.Limit <= 0 {
		o.Limit = 100
	}
	if o.State == "" {
		o.State = "all"
	}
	if o.PerPage <= 0 || o.PerPage > 100 {
		o.PerPage = 100
	}
	return o
}

func exportQuery(resource string, options ExportOptions, page int) map[string]string {
	query := map[string]string{
		"state":    options.State,
		"per_page": strconv.Itoa(options.PerPage),
		"page":     strconv.Itoa(page),
	}
	if options.Since != nil && resource == "issues" {
		query["since"] = options.Since.UTC().Format(time.RFC3339)
	}
	if options.Since != nil && resource == "pulls" {
		query["sort"] = "updated"
		query["direction"] = "desc"
	}
	return query
}

func matchesSince(value time.Time, since *time.Time) bool {
	if since == nil {
		return true
	}
	return !value.Before(*since)
}

func labels(values []labelResponse) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value.Name)
	}
	return out
}

type issueResponse struct {
	Number      int             `json:"number"`
	Title       string          `json:"title"`
	Body        string          `json:"body"`
	State       string          `json:"state"`
	User        user            `json:"user"`
	Labels      []labelResponse `json:"labels"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	ClosedAt    *time.Time      `json:"closed_at"`
	PullRequest struct {
		URL string `json:"url"`
	} `json:"pull_request"`
}

type pullResponse struct {
	Number    int             `json:"number"`
	Title     string          `json:"title"`
	Body      string          `json:"body"`
	State     string          `json:"state"`
	User      user            `json:"user"`
	Labels    []labelResponse `json:"labels"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
	MergedAt  *time.Time      `json:"merged_at"`
}

type user struct {
	Login string `json:"login"`
}

type labelResponse struct {
	Name string `json:"name"`
}
