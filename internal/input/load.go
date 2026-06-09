package input

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/yuhuibin/oss-maintainer-kit/internal/model"
)

func Issues(path string) ([]model.Issue, error) {
	var issues []model.Issue
	if err := load(path, &issues); err != nil {
		return nil, err
	}
	return issues, nil
}

func PullRequests(path string) ([]model.PullRequest, error) {
	var pulls []model.PullRequest
	if err := load(path, &pulls); err != nil {
		return nil, err
	}
	return pulls, nil
}

func load(path string, dst any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("parse %s: unexpected trailing JSON value", path)
		}
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}
