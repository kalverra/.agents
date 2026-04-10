package eval

import (
	"context"
	"fmt"
	"os"

	tiktoken "github.com/pkoukk/tiktoken-go"
)

// RunConfig holds all the parameters for an eval run.
type RunConfig struct {
	SubjectModel string
	JudgeModel   string
	CasesDir     string
	TagFilter    string
	Iterations   int
	RepoRoot     string
	Verbose      bool
}

// Run executes the eval harness: generates subject responses, then judges them.
func Run(ctx context.Context, cfg RunConfig) ([]Result, error) {
	cases, err := LoadCases(cfg.CasesDir, cfg.TagFilter)
	if err != nil {
		return nil, err
	}

	gc, err := NewGeminiClient(ctx)
	if err != nil {
		return nil, err
	}

	fmt.Printf("\nPrompt Eval Harness\n")
	fmt.Printf("  Subject : %s\n", cfg.SubjectModel)
	fmt.Printf("  Judge   : %s\n", cfg.JudgeModel)
	fmt.Printf("  Cases   : %d\n", len(cases))
	fmt.Printf("  Iters   : %d\n\n", cfg.Iterations)

	type prepared struct {
		c            Case
		systemPrompt string
		inputTokens  *int
		iterations   []Iteration
	}

	var active []prepared
	var results []Result

	// Load system prompts
	for _, c := range cases {
		sp, err := LoadSystemPrompt(c, cfg.RepoRoot)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: Skipped %s: %v\n", c.Name, err)
			results = append(results, Result{Case: c.Name, Error: err.Error(), Tags: c.Tags})
			continue
		}
		var tokens *int
		if enc, err := tiktoken.GetEncoding("cl100k_base"); err == nil {
			n := len(enc.Encode(sp, nil, nil))
			tokens = &n
		}
		active = append(active, prepared{c: c, systemPrompt: sp, inputTokens: tokens})
	}

	// Phase 1: Generate subject responses
	fmt.Println("Phase 1: Generating Responses")
	for i := range active {
		a := &active[i]
		fmt.Printf("[%d/%d] %s\n", i+1, len(active), a.c.Name)

		for it := 1; it <= cfg.Iterations; it++ {
			fmt.Printf("  -> calling subject...")
			resp, outTokens, err := gc.CallSubject(ctx, cfg.SubjectModel, a.systemPrompt, a.c.UserMessage)
			if err != nil {
				return nil, fmt.Errorf("subject call for %s: %w", a.c.Name, err)
			}
			fmt.Println(" done")

			if cfg.Verbose {
				fmt.Printf("  --- Subject Response ---\n%s\n  ---\n", resp)
			}

			a.iterations = append(a.iterations, Iteration{
				Num:             it,
				SubjectResponse: resp,
				OutputTokens:    outTokens,
			})
		}
	}

	// Phase 2: Judge responses
	fmt.Printf("\nPhase 2: Evaluating Responses (%s)\n", cfg.JudgeModel)
	for i := range active {
		a := &active[i]
		fmt.Printf("[%d/%d] %s\n", i+1, len(active), a.c.Name)

		for j := range a.iterations {
			it := &a.iterations[j]
			fmt.Printf("  -> calling judge...")
			prompt := FormatPrometheusPrompt(a.c, it.SubjectResponse)
			raw, err := gc.CallJudge(ctx, cfg.JudgeModel, prompt)
			if err != nil {
				return nil, fmt.Errorf("judge call for %s: %w", a.c.Name, err)
			}
			score := ParseScore(raw)
			it.JudgeRaw = raw
			it.Score = score
			scoreStr := "?"
			if score != nil {
				scoreStr = fmt.Sprintf("%d", *score)
			}
			fmt.Printf(" done  %s score=%s\n", ScoreEmoji(score), scoreStr)

			if cfg.Verbose {
				fmt.Printf("  --- Judge Feedback ---\n%s\n  ---\n", raw)
			}
		}

		// Aggregate
		r := aggregateResult(a.c, a.iterations, a.inputTokens, cfg)
		results = append(results, r)
	}

	return results, nil
}

func aggregateResult(c Case, iterations []Iteration, inputTokens *int, cfg RunConfig) Result {
	var validScores []int
	var totalOutTokens int

	for _, it := range iterations {
		if it.Score != nil {
			validScores = append(validScores, *it.Score)
		}
		totalOutTokens += it.OutputTokens
	}

	r := Result{
		Case:         c.Name,
		Tags:         c.Tags,
		Description:  c.Description,
		SubjectModel: cfg.SubjectModel,
		JudgeModel:   cfg.JudgeModel,
		UserMessage:  c.UserMessage,
		InputTokens:  inputTokens,
		Iterations:   iterations,
	}

	if len(iterations) > 0 {
		avg := totalOutTokens / len(iterations)
		r.AvgOutputTokens = avg
	}

	if len(validScores) > 0 {
		sum := 0
		minS, maxS := validScores[0], validScores[0]
		for _, s := range validScores {
			sum += s
			if s < minS {
				minS = s
			}
			if s > maxS {
				maxS = s
			}
		}
		avg := float64(sum) / float64(len(validScores))
		r.AvgScore = &avg
		r.MinScore = &minS
		r.MaxScore = &maxS
	}

	if inputTokens != nil {
		ts := *inputTokens + r.AvgOutputTokens*5
		r.TokenScore = &ts
	}

	return r
}
