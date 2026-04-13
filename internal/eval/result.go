package eval

// Result holds the aggregated eval results for a single test case.
type Result struct {
	Case            string      `json:"case"`
	Tags            []string    `json:"tags"`
	Description     string      `json:"description"`
	SubjectModel    string      `json:"subject_model"`
	JudgeModel      string      `json:"judge_model"`
	UserMessage     string      `json:"user_message"`
	InputTokens     *int        `json:"tokens,omitempty"`
	AvgOutputTokens int         `json:"avg_output_tokens"`
	TokenScore      *int        `json:"token_score,omitempty"`
	Cost            float64     `json:"cost,omitempty"`
	Iterations      []Iteration `json:"iterations"`
	AvgScore        *float64    `json:"avg_score,omitempty"`
	MinScore        *int        `json:"min_score,omitempty"`
	MaxScore        *int        `json:"max_score,omitempty"`
	Error           string      `json:"error,omitempty"`
}

// Iteration holds results for one iteration of a test case.
type Iteration struct {
	Num             int     `json:"iteration"`
	SubjectResponse string  `json:"subject_response"`
	InputTokens     int     `json:"input_tokens"`
	OutputTokens    int     `json:"output_tokens"`
	Cost            float64 `json:"cost,omitempty"`
	JudgeRaw        string  `json:"judge_raw,omitempty"`
	Score           *int    `json:"score,omitempty"`
}

// ScoreEmoji returns an emoji for a 1-5 score.
func ScoreEmoji(score *int) string {
	if score == nil {
		return "?"
	}
	emojis := [6]string{"", "X", "!", "~", "+", "v"}
	if *score >= 1 && *score <= 5 {
		return emojis[*score]
	}
	return "?"
}
