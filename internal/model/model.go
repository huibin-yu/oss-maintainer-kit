package model

import "time"

type Issue struct {
	Number    int        `json:"number"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	State     string     `json:"state"`
	Author    string     `json:"author"`
	Labels    []string   `json:"labels"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	ClosedAt  *time.Time `json:"closed_at,omitempty"`
}

type PullRequest struct {
	Number    int        `json:"number"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	State     string     `json:"state"`
	Author    string     `json:"author"`
	Labels    []string   `json:"labels"`
	Merged    bool       `json:"merged"`
	MergedAt  *time.Time `json:"merged_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type TriageResult struct {
	Number        int      `json:"number"`
	Title         string   `json:"title"`
	Priority      string   `json:"priority"`
	Suggested     []string `json:"suggested_labels"`
	Reasons       []string `json:"reasons"`
	StaleDays     int      `json:"stale_days"`
	NeedsReview   bool     `json:"needs_review"`
	NeedsSecurity bool     `json:"needs_security_review"`
}

type MaintainerReport struct {
	OpenIssues       int
	StaleIssues      int
	SecurityIssues   int
	MergedPulls      int
	NeedsReview      int
	TopSuggestedWork []TriageResult
}
