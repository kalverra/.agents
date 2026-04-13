package eval

import (
	"encoding/json"
	"os"
)

// HistoryEntry stores the last eval result for a case.
type HistoryEntry struct {
	AvgScore        *float64 `json:"avg_score,omitempty"`
	Tokens          *int     `json:"tokens,omitempty"`
	AvgOutputTokens *int     `json:"avg_output_tokens,omitempty"`
	TokenScore      *int     `json:"token_score,omitempty"`
	Cost            float64  `json:"cost,omitempty"`
	SubjectModel    string   `json:"subject_model,omitempty"`
	JudgeModel      string   `json:"judge_model,omitempty"`
}

// History maps case names to their last eval results.
type History map[string]HistoryEntry

// LoadHistory reads eval_history.json. Returns empty history if file doesn't exist.
func LoadHistory(path string) (History, error) {
	data, err := os.ReadFile(path) //nolint:gosec
	if err != nil {
		if os.IsNotExist(err) {
			return make(History), nil
		}
		return nil, err
	}
	h := make(History)
	if err := json.Unmarshal(data, &h); err != nil {
		return nil, err
	}
	return h, nil
}

// SaveHistory writes eval_history.json.
func SaveHistory(path string, h History) error {
	data, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600) //nolint:gosec
}

// MergeResults updates history with new eval results.
func MergeResults(h History, results []Result) {
	for _, r := range results {
		if r.Error != "" {
			continue
		}
		entry := HistoryEntry{
			AvgScore:     r.AvgScore,
			Tokens:       r.InputTokens,
			SubjectModel: r.SubjectModel,
			JudgeModel:   r.JudgeModel,
			Cost:         r.Cost,
		}
		if r.AvgOutputTokens > 0 {
			v := r.AvgOutputTokens
			entry.AvgOutputTokens = &v
		}
		entry.TokenScore = r.TokenScore
		h[r.Case] = entry
	}
}
