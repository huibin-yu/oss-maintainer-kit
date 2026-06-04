package github

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/yuhuibin/oss-maintainer-kit/internal/model"
)

type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

func (c Client) Issues(ctx context.Context, repo string, limit int) ([]model.Issue, error) {
	var raw []issueResponse
	if err := c.fetch(ctx, repo, "issues", limit, &raw); err != nil {
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
	var raw []pullResponse
	if err := c.fetch(ctx, repo, "pulls", limit, &raw); err != nil {
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

func (c Client) fetch(ctx context.Context, repo, resource string, limit int, dst any) error {
	if repo == "" {
		return fmt.Errorf("repo is required, expected owner/name")
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 100 {
		limit = 100
	}
	return c.requestJSON(ctx, "GET", fmt.Sprintf("/repos/%s/%s", repo, resource), map[string]string{
		"state":    "all",
		"per_page": perPage(limit),
	}, nil, dst)
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
