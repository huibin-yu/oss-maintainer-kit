package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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
	base := strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		base = "https://api.github.com"
	}
	endpoint, err := url.Parse(fmt.Sprintf("%s/repos/%s/%s", base, repo, resource))
	if err != nil {
		return err
	}
	query := endpoint.Query()
	query.Set("state", "all")
	query.Set("per_page", strconv.Itoa(limit))
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 20 * time.Second}
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("github %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(resp.Body).Decode(dst)
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
