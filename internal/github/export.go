package github

import (
	"context"
	"fmt"
	"net/http"
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
			Author:    authorLogin(item.User.Login),
			Labels:    labels(item.Labels),
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
			ClosedAt:  item.ClosedAt,
		})
	}
	return issues, nil
}

func (c Client) IssuesGraphQL(ctx context.Context, repo string, options ExportOptions) ([]model.Issue, error) {
	var raw []graphqlIssueNode
	if err := c.fetchGraphQL(ctx, repo, "issues", options, &raw); err != nil {
		return nil, err
	}
	issues := make([]model.Issue, 0, len(raw))
	for _, item := range raw {
		issues = append(issues, model.Issue{
			Number:    item.Number,
			Title:     item.Title,
			Body:      item.Body,
			State:     strings.ToLower(item.State),
			Author:    authorLogin(item.Author.Login),
			Labels:    graphqlLabels(item.Labels.Nodes),
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
			Author:    authorLogin(item.User.Login),
			Labels:    labels(item.Labels),
			Merged:    item.MergedAt != nil,
			MergedAt:  item.MergedAt,
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
		})
	}
	return pulls, nil
}

func (c Client) PullRequestsGraphQL(ctx context.Context, repo string, options ExportOptions) ([]model.PullRequest, error) {
	var raw []graphqlPullNode
	if err := c.fetchGraphQL(ctx, repo, "pulls", options, &raw); err != nil {
		return nil, err
	}
	pulls := make([]model.PullRequest, 0, len(raw))
	for _, item := range raw {
		pulls = append(pulls, model.PullRequest{
			Number:    item.Number,
			Title:     item.Title,
			Body:      item.Body,
			State:     strings.ToLower(item.State),
			Author:    authorLogin(item.Author.Login),
			Labels:    graphqlLabels(item.Labels.Nodes),
			Merged:    item.MergedAt != nil,
			MergedAt:  item.MergedAt,
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
		})
	}
	return pulls, nil
}

func (c Client) fetchAll(ctx context.Context, repo, resource string, options ExportOptions, dst any) error {
	owner, name, err := splitRepo(repo)
	if err != nil {
		return err
	}
	options = options.withDefaults()
	path := fmt.Sprintf("/repos/%s/%s/%s", owner, name, resource)
	switch out := dst.(type) {
	case *[]issueResponse:
		var all []issueResponse
		for page := 1; len(all) < options.Limit; page++ {
			var batch []issueResponse
			if err := c.requestJSON(ctx, "GET", path, exportQuery(resource, options, page), nil, &batch); err != nil {
				return err
			}
			if len(batch) == 0 {
				break
			}
			for _, item := range batch {
				if item.PullRequest.URL != "" {
					continue
				}
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
			if err := c.requestJSON(ctx, "GET", path, exportQuery(resource, options, page), nil, &batch); err != nil {
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

func (c Client) fetchGraphQL(ctx context.Context, repo, resource string, options ExportOptions, dst any) error {
	owner, name, err := splitRepo(repo)
	if err != nil {
		return err
	}
	options = options.withDefaults()
	switch out := dst.(type) {
	case *[]graphqlIssueNode:
		var all []graphqlIssueNode
		cursor := ""
		for len(all) < options.Limit {
			var response graphqlIssuesResponse
			if err := c.requestJSON(ctx, http.MethodPost, "/graphql", nil, graphqlRequest{
				Query: graphqlIssuesQuery,
				Variables: map[string]any{
					"owner":  owner,
					"name":   name,
					"first":  min(options.PerPage, options.Limit-len(all)),
					"after":  nullableCursor(cursor),
					"states": graphqlIssueStates(options.State),
				},
			}, &response); err != nil {
				return err
			}
			if err := response.error(); err != nil {
				return err
			}
			olderThanSince := 0
			for _, item := range response.Data.Repository.Issues.Nodes {
				if matchesSince(item.UpdatedAt, options.Since) {
					all = append(all, item)
				} else {
					olderThanSince++
				}
				if len(all) >= options.Limit {
					break
				}
			}
			pageInfo := response.Data.Repository.Issues.PageInfo
			if !pageInfo.HasNextPage || pageInfo.EndCursor == "" {
				break
			}
			if options.Since != nil && olderThanSince == len(response.Data.Repository.Issues.Nodes) {
				break
			}
			cursor = pageInfo.EndCursor
		}
		*out = all
	case *[]graphqlPullNode:
		var all []graphqlPullNode
		cursor := ""
		for len(all) < options.Limit {
			var response graphqlPullsResponse
			if err := c.requestJSON(ctx, http.MethodPost, "/graphql", nil, graphqlRequest{
				Query: graphqlPullsQuery,
				Variables: map[string]any{
					"owner":  owner,
					"name":   name,
					"first":  min(options.PerPage, options.Limit-len(all)),
					"after":  nullableCursor(cursor),
					"states": graphqlPullStates(options.State),
				},
			}, &response); err != nil {
				return err
			}
			if err := response.error(); err != nil {
				return err
			}
			olderThanSince := 0
			for _, item := range response.Data.Repository.PullRequests.Nodes {
				if matchesSince(item.UpdatedAt, options.Since) {
					all = append(all, item)
				} else {
					olderThanSince++
				}
				if len(all) >= options.Limit {
					break
				}
			}
			pageInfo := response.Data.Repository.PullRequests.PageInfo
			if !pageInfo.HasNextPage || pageInfo.EndCursor == "" {
				break
			}
			if options.Since != nil && olderThanSince == len(response.Data.Repository.PullRequests.Nodes) {
				break
			}
			cursor = pageInfo.EndCursor
		}
		*out = all
	default:
		return fmt.Errorf("unsupported graphql export destination %T", dst)
	}
	return nil
}

func splitRepo(repo string) (string, string, error) {
	parts := strings.Split(repo, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("repo is required, expected owner/name")
	}
	owner := strings.TrimSpace(parts[0])
	name := strings.TrimSpace(parts[1])
	if owner == "" || name == "" {
		return "", "", fmt.Errorf("repo is required, expected owner/name")
	}
	return owner, name, nil
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

func authorLogin(login string) string {
	if strings.TrimSpace(login) == "" {
		return "unknown"
	}
	return login
}

func labels(values []labelResponse) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value.Name)
	}
	return out
}

func graphqlLabels(values []graphqlLabelNode) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value.Name)
	}
	return out
}

func graphqlIssueStates(state string) []string {
	switch strings.ToLower(state) {
	case "open":
		return []string{"OPEN"}
	case "closed":
		return []string{"CLOSED"}
	default:
		return []string{"OPEN", "CLOSED"}
	}
}

func graphqlPullStates(state string) []string {
	switch strings.ToLower(state) {
	case "open":
		return []string{"OPEN"}
	case "closed":
		return []string{"CLOSED", "MERGED"}
	default:
		return []string{"OPEN", "CLOSED", "MERGED"}
	}
}

func nullableCursor(cursor string) any {
	if cursor == "" {
		return nil
	}
	return cursor
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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

type graphqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

type graphqlError struct {
	Message    string `json:"message"`
	Path       []any  `json:"path"`
	Extensions struct {
		Code string `json:"code"`
	} `json:"extensions"`
}

type graphqlPageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

type graphqlAuthor struct {
	Login string `json:"login"`
}

type graphqlLabelNode struct {
	Name string `json:"name"`
}

type graphqlLabelsConnection struct {
	Nodes []graphqlLabelNode `json:"nodes"`
}

type graphqlIssueNode struct {
	Number    int                     `json:"number"`
	Title     string                  `json:"title"`
	Body      string                  `json:"body"`
	State     string                  `json:"state"`
	Author    graphqlAuthor           `json:"author"`
	Labels    graphqlLabelsConnection `json:"labels"`
	CreatedAt time.Time               `json:"createdAt"`
	UpdatedAt time.Time               `json:"updatedAt"`
	ClosedAt  *time.Time              `json:"closedAt"`
}

type graphqlPullNode struct {
	Number    int                     `json:"number"`
	Title     string                  `json:"title"`
	Body      string                  `json:"body"`
	State     string                  `json:"state"`
	Author    graphqlAuthor           `json:"author"`
	Labels    graphqlLabelsConnection `json:"labels"`
	CreatedAt time.Time               `json:"createdAt"`
	UpdatedAt time.Time               `json:"updatedAt"`
	MergedAt  *time.Time              `json:"mergedAt"`
}

type graphqlIssuesResponse struct {
	Data struct {
		Repository struct {
			Issues struct {
				Nodes    []graphqlIssueNode `json:"nodes"`
				PageInfo graphqlPageInfo    `json:"pageInfo"`
			} `json:"issues"`
		} `json:"repository"`
	} `json:"data"`
	Errors []graphqlError `json:"errors"`
}

func (r graphqlIssuesResponse) error() error {
	return graphqlErrors(r.Errors)
}

type graphqlPullsResponse struct {
	Data struct {
		Repository struct {
			PullRequests struct {
				Nodes    []graphqlPullNode `json:"nodes"`
				PageInfo graphqlPageInfo   `json:"pageInfo"`
			} `json:"pullRequests"`
		} `json:"repository"`
	} `json:"data"`
	Errors []graphqlError `json:"errors"`
}

func (r graphqlPullsResponse) error() error {
	return graphqlErrors(r.Errors)
}

func graphqlErrors(errors []graphqlError) error {
	if len(errors) == 0 {
		return nil
	}
	messages := make([]string, 0, len(errors))
	for _, item := range errors {
		parts := []string{}
		if item.Message != "" {
			parts = append(parts, item.Message)
		}
		if path := graphqlErrorPath(item.Path); path != "" {
			parts = append(parts, path)
		}
		if item.Extensions.Code != "" {
			parts = append(parts, item.Extensions.Code)
		}
		if len(parts) > 0 {
			messages = append(messages, strings.Join(parts, " "))
		}
	}
	if len(messages) == 0 {
		return fmt.Errorf("github graphql: unknown error")
	}
	return fmt.Errorf("github graphql: %s", strings.Join(messages, "; "))
}

func graphqlErrorPath(values []any) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		text := fmt.Sprint(value)
		if text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, ".")
}

const graphqlIssuesQuery = `
query ExportIssues($owner: String!, $name: String!, $first: Int!, $after: String, $states: [IssueState!]) {
  repository(owner: $owner, name: $name) {
    issues(first: $first, after: $after, states: $states, orderBy: {field: UPDATED_AT, direction: DESC}) {
      nodes {
        number
        title
        body
        state
        author { login }
        labels(first: 20) { nodes { name } }
        createdAt
        updatedAt
        closedAt
      }
      pageInfo { hasNextPage endCursor }
    }
  }
}`

const graphqlPullsQuery = `
query ExportPullRequests($owner: String!, $name: String!, $first: Int!, $after: String, $states: [PullRequestState!]) {
  repository(owner: $owner, name: $name) {
    pullRequests(first: $first, after: $after, states: $states, orderBy: {field: UPDATED_AT, direction: DESC}) {
      nodes {
        number
        title
        body
        state
        author { login }
        labels(first: 20) { nodes { name } }
        createdAt
        updatedAt
        mergedAt
      }
      pageInfo { hasNextPage endCursor }
    }
  }
}`
