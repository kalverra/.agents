package eval

import (
	"fmt"
	"regexp"
	"strconv"
)

const prometheusPrompt = `###Task Description: An instruction (might include an Input inside it), a response to evaluate, a reference answer that gets a score of 5, and a score rubric representing an evaluation criteria are given.
1. Write a detailed feedback that assesses the quality of the response strictly based on the given score rubric, not evaluating in general.
2. After writing a feedback, write a score that is an integer between 1 and 5. You should refer to the score rubric.
3. The output format should look as follows: "Feedback: (write a feedback for criteria) [RESULT] (an integer number between 1 and 5)"
4. Please do not generate any other opening, closing, and explanations.

###The instruction to evaluate:
%s

###Response to evaluate:
%s

###Reference Answer (Score 5):
%s

###Score Rubrics:
[%s]
Score 1: %s
Score 2: %s
Score 3: %s
Score 4: %s
Score 5: %s

###Feedback:`

var scoreRe = regexp.MustCompile(`\[RESULT\]\s*(\d)`)

// FormatPrometheusPrompt builds the judge prompt for a case and subject response.
func FormatPrometheusPrompt(c Case, subjectResponse string) string {
	return fmt.Sprintf(prometheusPrompt,
		c.UserMessage,
		subjectResponse,
		c.ReferenceAnswer,
		c.Criteria.Name,
		c.Criteria.Score1,
		c.Criteria.Score2,
		c.Criteria.Score3,
		c.Criteria.Score4,
		c.Criteria.Score5,
	)
}

// ParseScore extracts the integer score from the Prometheus [RESULT] N output.
func ParseScore(text string) *int {
	match := scoreRe.FindStringSubmatch(text)
	if match == nil {
		return nil
	}
	val, err := strconv.Atoi(match[1])
	if err != nil || val < 1 || val > 5 {
		return nil
	}
	return &val
}
