package eval

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// RunSummary is a flat representation of an evaluation run for logging.
type RunSummary struct {
	Timestamp   string             `json:"timestamp"`
	Commit      string             `json:"commit"`
	Subject     string             `json:"subject"`
	Judge       string             `json:"judge"`
	AvgScore    float64            `json:"avg_score"`
	TotalCost   float64            `json:"total_cost"`
	PassedCount int                `json:"passed_count"`
	TotalCases  int                `json:"total_cases"`
	Scores      map[string]float64 `json:"scores"` // case_name -> avg_score
}

// AppendToJSONL appends a RunSummary to a .jsonl file.
func AppendToJSONL(path string, summary RunSummary) (err error) {
	data, err := json.Marshal(summary)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600) //nolint: gosec
	if err != nil {
		return err
	}
	defer func() {
		err = f.Close()
	}()

	_, err = f.Write(append(data, '\n'))
	return err
}

// CreateSummary aggregates results into a flat RunSummary.
func CreateSummary(results []Result, commit, subject, judge string) RunSummary {
	s := RunSummary{
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Commit:     commit,
		Subject:    subject,
		Judge:      judge,
		TotalCases: len(results),
		Scores:     make(map[string]float64),
	}

	totalAvg := 0.0
	scoreCount := 0
	for _, r := range results {
		if r.Error != "" {
			continue
		}
		if r.AvgScore != nil {
			s.Scores[r.Case] = *r.AvgScore
			totalAvg += *r.AvgScore
			scoreCount++
			if *r.AvgScore >= 4.0 {
				s.PassedCount++
			}
		}
		s.TotalCost += r.Cost
	}

	if scoreCount > 0 {
		s.AvgScore = totalAvg / float64(scoreCount)
	}

	return s
}

// LoadLastSummary reads the last line of a .jsonl file.
func LoadLastSummary(path string) (*RunSummary, error) {
	data, err := os.ReadFile(path) //nolint: gosec
	if err != nil {
		return nil, err
	}

	lines := splitLines(string(data))
	if len(lines) == 0 {
		return nil, fmt.Errorf("file is empty")
	}

	var last RunSummary
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &last); err != nil {
		return nil, err
	}
	return &last, nil
}

func splitLines(s string) []string {
	var lines []string
	var start int
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if line != "" {
				lines = append(lines, line)
			}
			start = i + 1
		}
	}
	return lines
}
