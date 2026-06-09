package sarif

import "github.com/yuhuibin/oss-maintainer-kit/internal/diffreview"

type Log struct {
	Version string `json:"version"`
	Schema  string `json:"$schema"`
	Runs    []Run  `json:"runs"`
}

type Run struct {
	Tool    Tool     `json:"tool"`
	Results []Result `json:"results"`
}

type Tool struct {
	Driver Driver `json:"driver"`
}

type Driver struct {
	Name           string          `json:"name"`
	InformationURI string          `json:"informationUri"`
	Rules          []ReportingRule `json:"rules"`
}

type ReportingRule struct {
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	ShortDescription Message        `json:"shortDescription"`
	Properties       RuleProperties `json:"properties"`
}

type RuleProperties struct {
	Tags []string `json:"tags"`
}

type Result struct {
	RuleID    string     `json:"ruleId"`
	Level     string     `json:"level"`
	Message   Message    `json:"message"`
	Locations []Location `json:"locations"`
}

type Message struct {
	Text string `json:"text"`
}

type Location struct {
	PhysicalLocation PhysicalLocation `json:"physicalLocation"`
}

type PhysicalLocation struct {
	ArtifactLocation ArtifactLocation `json:"artifactLocation"`
	Region           Region           `json:"region"`
}

type ArtifactLocation struct {
	URI string `json:"uri"`
}

type Region struct {
	StartLine int `json:"startLine"`
}

func FromFindings(findings []diffreview.Finding) Log {
	ruleMap := map[string]ReportingRule{}
	results := make([]Result, 0, len(findings))
	for _, finding := range findings {
		ruleMap[finding.Rule] = ReportingRule{
			ID:               finding.Rule,
			Name:             finding.Rule,
			ShortDescription: Message{Text: finding.Message},
			Properties:       RuleProperties{Tags: []string{"security", "maintainability"}},
		}
		results = append(results, Result{
			RuleID:  finding.Rule,
			Level:   level(finding.Severity),
			Message: Message{Text: finding.Message},
			Locations: []Location{{
				PhysicalLocation: PhysicalLocation{
					ArtifactLocation: ArtifactLocation{URI: finding.File},
					Region:           Region{StartLine: positiveLine(finding.Line)},
				},
			}},
		})
	}

	rules := make([]ReportingRule, 0, len(ruleMap))
	for _, rule := range ruleMap {
		rules = append(rules, rule)
	}

	return Log{
		Version: "2.1.0",
		Schema:  "https://json.schemastore.org/sarif-2.1.0.json",
		Runs: []Run{{
			Tool: Tool{Driver: Driver{
				Name:           "oss-maintainer-kit",
				InformationURI: "https://github.com/yuhuibin/oss-maintainer-kit",
				Rules:          rules,
			}},
			Results: results,
		}},
	}
}

func positiveLine(line int) int {
	if line <= 0 {
		return 1
	}
	return line
}

func level(severity string) string {
	switch severity {
	case "critical", "high":
		return "error"
	case "medium":
		return "warning"
	default:
		return "note"
	}
}
