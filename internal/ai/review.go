package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/yuhuibin/oss-maintainer-kit/internal/diffreview"
)

type Client struct {
	BaseURL string
	APIKey  string
	Model   string
	HTTP    *http.Client
}

type ReviewRequest struct {
	Project  string               `json:"project"`
	Diff     string               `json:"diff"`
	Findings []diffreview.Finding `json:"findings"`
}

type ReviewResult struct {
	Content string `json:"content"`
}

func Prompt(req ReviewRequest) string {
	var b strings.Builder
	fmt.Fprintf(&b, "你是开源项目维护者的 PR review 助手。请用中文审查以下 diff。\n\n")
	fmt.Fprintf(&b, "项目：%s\n\n", req.Project)
	fmt.Fprintf(&b, "要求：\n")
	fmt.Fprintf(&b, "- 优先指出安全、稳定性、边界条件和测试缺口。\n")
	fmt.Fprintf(&b, "- 每条建议必须可执行。\n")
	fmt.Fprintf(&b, "- 不要输出无依据的泛泛建议。\n")
	fmt.Fprintf(&b, "- 如果本地规则发现风险，请先复核这些风险。\n\n")
	if len(req.Findings) > 0 {
		fmt.Fprintf(&b, "本地规则发现：\n")
		for _, finding := range req.Findings {
			fmt.Fprintf(&b, "- %s %s:%d %s：%s\n", finding.Severity, finding.File, finding.Line, finding.Rule, finding.Message)
		}
		fmt.Fprintln(&b)
	}
	fmt.Fprintf(&b, "Diff：\n```diff\n%s\n```\n", req.Diff)
	return b.String()
}

func (c Client) Review(ctx context.Context, req ReviewRequest) (ReviewResult, error) {
	if c.APIKey == "" {
		return ReviewResult{}, fmt.Errorf("api key is required")
	}
	model := c.Model
	if model == "" {
		model = "gpt-4.1-mini"
	}
	base := strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		base = "https://api.openai.com/v1"
	}

	payload := chatRequest{
		Model: model,
		Messages: []message{
			{Role: "system", Content: "你是严谨的开源项目维护者，负责 PR review。"},
			{Role: "user", Content: Prompt(req)},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return ReviewResult{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return ReviewResult{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return ReviewResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return ReviewResult{}, fmt.Errorf("ai review %s: %s", resp.Status, strings.TrimSpace(string(data)))
	}

	var out chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return ReviewResult{}, err
	}
	if len(out.Choices) == 0 {
		return ReviewResult{}, fmt.Errorf("ai review returned no choices")
	}
	return ReviewResult{Content: out.Choices[0].Message.Content}, nil
}

type chatRequest struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message message `json:"message"`
	} `json:"choices"`
}
