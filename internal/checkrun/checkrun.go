package checkrun

import (
	"fmt"

	"github.com/yuhuibin/oss-maintainer-kit/internal/diffreview"
)

const maxAnnotations = 50

type Payload struct {
	Name       string `json:"name"`
	HeadSHA    string `json:"head_sha,omitempty"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	Output     Output `json:"output"`
}

type Output struct {
	Title       string       `json:"title"`
	Summary     string       `json:"summary"`
	Annotations []Annotation `json:"annotations"`
}

type Annotation struct {
	Path            string `json:"path"`
	StartLine       int    `json:"start_line"`
	EndLine         int    `json:"end_line"`
	AnnotationLevel string `json:"annotation_level"`
	Message         string `json:"message"`
	RawDetails      string `json:"raw_details,omitempty"`
}

func FromFindings(findings []diffreview.Finding) Payload {
	annotations := make([]Annotation, 0, min(len(findings), maxAnnotations))
	for _, finding := range findings {
		if len(annotations) >= maxAnnotations {
			break
		}
		annotations = append(annotations, Annotation{
			Path:            finding.File,
			StartLine:       positiveLine(finding.Line),
			EndLine:         positiveLine(finding.Line),
			AnnotationLevel: annotationLevel(finding.Severity),
			Message:         fmt.Sprintf("[%s] %s", finding.Rule, finding.Message),
			RawDetails:      finding.Snippet,
		})
	}

	return Payload{
		Name:       "oss-maintainer-kit review-diff",
		Status:     "completed",
		Conclusion: conclusion(findings),
		Output: Output{
			Title:       "PR diff 风险检查",
			Summary:     summary(findings),
			Annotations: annotations,
		},
	}
}

func conclusion(findings []diffreview.Finding) string {
	if len(findings) == 0 {
		return "success"
	}
	for _, finding := range findings {
		if finding.Severity == "critical" || finding.Severity == "high" {
			return "failure"
		}
	}
	return "neutral"
}

func summary(findings []diffreview.Finding) string {
	if len(findings) == 0 {
		return "未发现需要阻塞合并的新增代码风险。"
	}
	return fmt.Sprintf(
		"发现 %d 个风险项：critical=%d，high=%d，medium=%d，low=%d。",
		len(findings),
		count(findings, "critical"),
		count(findings, "high"),
		count(findings, "medium"),
		count(findings, "low"),
	)
}

func count(findings []diffreview.Finding, severity string) int {
	total := 0
	for _, finding := range findings {
		if finding.Severity == severity {
			total++
		}
	}
	return total
}

func annotationLevel(severity string) string {
	switch severity {
	case "critical", "high":
		return "failure"
	case "medium":
		return "warning"
	default:
		return "notice"
	}
}

func positiveLine(line int) int {
	if line <= 0 {
		return 1
	}
	return line
}
