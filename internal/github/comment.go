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
)

type Comment struct {
	ID      int64  `json:"id"`
	Body    string `json:"body"`
	HTMLURL string `json:"html_url"`
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
	var comments []Comment
	err := c.requestJSON(ctx, http.MethodGet, fmt.Sprintf("/repos/%s/issues/%d/comments", repo, number), map[string]string{"per_page": "100"}, nil, &comments)
	return comments, err
}

func (c Client) CreateIssueComment(ctx context.Context, repo string, number int, body string) (Comment, error) {
	if number <= 0 {
		return Comment{}, fmt.Errorf("issue or pull request number is required")
	}
	var comment Comment
	err := c.requestJSON(ctx, http.MethodPost, fmt.Sprintf("/repos/%s/issues/%d/comments", repo, number), nil, commentRequest{Body: body}, &comment)
	return comment, err
}

func (c Client) UpdateIssueComment(ctx context.Context, repo string, id int64, body string) (Comment, error) {
	if id <= 0 {
		return Comment{}, fmt.Errorf("comment id is required")
	}
	var comment Comment
	err := c.requestJSON(ctx, http.MethodPatch, fmt.Sprintf("/repos/%s/issues/comments/%d", repo, id), nil, commentRequest{Body: body}, &comment)
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
		return fmt.Errorf("github %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	if dst == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(dst)
}

type commentRequest struct {
	Body string `json:"body"`
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
