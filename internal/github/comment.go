package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/yuhuibin/oss-maintainer-kit/internal/checkrun"
)

type Comment struct {
	ID      int64  `json:"id"`
	Body    string `json:"body"`
	HTMLURL string `json:"html_url"`
}

type CheckRun struct {
	ID      int64  `json:"id"`
	HTMLURL string `json:"html_url"`
}

type ReleasePayload struct {
	TagName    string `json:"tag_name"`
	Name       string `json:"name"`
	Body       string `json:"body"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
}

type Release struct {
	ID      int64  `json:"id"`
	HTMLURL string `json:"html_url"`
	TagName string `json:"tag_name"`
}

func (c Client) CreateRelease(ctx context.Context, repo string, payload ReleasePayload) (Release, error) {
	if strings.TrimSpace(payload.TagName) == "" {
		return Release{}, fmt.Errorf("tag name is required")
	}
	path, err := repoPath(repo, "releases")
	if err != nil {
		return Release{}, err
	}
	var release Release
	err = c.requestJSON(ctx, http.MethodPost, path, nil, payload, &release)
	return release, err
}

func (c Client) CreateCheckRun(ctx context.Context, repo string, payload checkrun.Payload) (CheckRun, error) {
	if strings.TrimSpace(payload.HeadSHA) == "" {
		return CheckRun{}, fmt.Errorf("head sha is required")
	}
	path, err := repoPath(repo, "check-runs")
	if err != nil {
		return CheckRun{}, err
	}
	var run CheckRun
	err = c.requestJSON(ctx, http.MethodPost, path, nil, payload, &run)
	return run, err
}

func (c Client) UpsertIssueComment(ctx context.Context, repo string, number int, marker, body string) (Comment, error) {
	if strings.TrimSpace(marker) == "" {
		return Comment{}, fmt.Errorf("marker is required")
	}
	comments, err := c.IssueComments(ctx, repo, number)
	if err != nil {
		return Comment{}, err
	}
	for _, comment := range comments {
		if strings.Contains(comment.Body, marker) {
			return c.UpdateIssueComment(ctx, repo, comment.ID, body)
		}
	}
	return c.CreateIssueComment(ctx, repo, number, body)
}

func (c Client) IssueComments(ctx context.Context, repo string, number int) ([]Comment, error) {
	if number <= 0 {
		return nil, fmt.Errorf("issue or pull request number is required")
	}
	path, err := repoPath(repo, fmt.Sprintf("issues/%d/comments", number))
	if err != nil {
		return nil, err
	}
	var comments []Comment
	err = c.requestJSON(ctx, http.MethodGet, path, map[string]string{"per_page": "100"}, nil, &comments)
	return comments, err
}

func (c Client) CreateIssueComment(ctx context.Context, repo string, number int, body string) (Comment, error) {
	if number <= 0 {
		return Comment{}, fmt.Errorf("issue or pull request number is required")
	}
	path, err := repoPath(repo, fmt.Sprintf("issues/%d/comments", number))
	if err != nil {
		return Comment{}, err
	}
	var comment Comment
	err = c.requestJSON(ctx, http.MethodPost, path, nil, commentRequest{Body: body}, &comment)
	return comment, err
}

func (c Client) UpdateIssueComment(ctx context.Context, repo string, id int64, body string) (Comment, error) {
	if id <= 0 {
		return Comment{}, fmt.Errorf("comment id is required")
	}
	path, err := repoPath(repo, fmt.Sprintf("issues/comments/%d", id))
	if err != nil {
		return Comment{}, err
	}
	var comment Comment
	err = c.requestJSON(ctx, http.MethodPatch, path, nil, commentRequest{Body: body}, &comment)
	return comment, err
}

func (c Client) requestJSON(ctx context.Context, method, path string, query map[string]string, payload any, dst any) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("github path is required")
	}
	base := strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		base = "https://api.github.com"
	}
	endpoint, err := url.Parse(base + path)
	if err != nil {
		return err
	}
	if len(query) > 0 {
		values := endpoint.Query()
		for key, value := range query {
			values.Set(key, value)
		}
		endpoint.RawQuery = values.Encode()
	}

	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
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
		return githubError(resp.StatusCode, resp.Status, body)
	}
	if dst == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("decode github response %s %s: %w", method, path, err)
	}
	return nil
}

type commentRequest struct {
	Body string `json:"body"`
}

type errorResponse struct {
	Message string `json:"message"`
	Errors  []struct {
		Resource string `json:"resource"`
		Field    string `json:"field"`
		Code     string `json:"code"`
		Message  string `json:"message"`
	} `json:"errors"`
}

func githubError(statusCode int, status string, body []byte) error {
	detail := strings.TrimSpace(string(body))
	var parsed errorResponse
	if len(body) > 0 && json.Unmarshal(body, &parsed) == nil {
		parts := []string{}
		if parsed.Message != "" {
			parts = append(parts, parsed.Message)
		}
		for _, item := range parsed.Errors {
			fields := []string{}
			if item.Resource != "" {
				fields = append(fields, item.Resource)
			}
			if item.Field != "" {
				fields = append(fields, item.Field)
			}
			if item.Code != "" {
				fields = append(fields, item.Code)
			}
			if item.Message != "" {
				fields = append(fields, item.Message)
			}
			if len(fields) > 0 {
				parts = append(parts, strings.Join(fields, " "))
			}
		}
		if len(parts) > 0 {
			detail = strings.Join(parts, "; ")
		}
	}
	if detail == "" {
		detail = status
	}

	switch statusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("github authentication failed (%s): %s", status, detail)
	case http.StatusForbidden:
		return fmt.Errorf("github permission denied or rate limited (%s): %s", status, detail)
	case http.StatusNotFound:
		return fmt.Errorf("github resource not found or inaccessible (%s): %s", status, detail)
	case http.StatusUnprocessableEntity:
		return fmt.Errorf("github request validation failed (%s): %s", status, detail)
	default:
		return fmt.Errorf("github request failed (%s): %s", status, detail)
	}
}

func repoPath(repo, suffix string) (string, error) {
	owner, name, err := splitRepo(repo)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("/repos/%s/%s/%s", owner, name, strings.TrimLeft(suffix, "/")), nil
}

func perPage(limit int) string {
	if limit <= 0 {
		limit = 100
	}
	if limit > 100 {
		limit = 100
	}
	return strconv.Itoa(limit)
}
