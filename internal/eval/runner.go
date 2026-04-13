package eval

import (
	"context"
	"fmt"
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/kalverra/agents/internal/ui"
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
	AIOutput     bool
}

// Run executes the eval harness: generates subject responses, then judges them.
func Run(ctx context.Context, cfg RunConfig) ([]Result, error) {
	log.Debug().Msg("Starting eval run")
	cases, err := LoadCases(cfg.CasesDir, cfg.TagFilter)
	if err != nil {
		return nil, err
	}

	gc, err := NewGeminiClient(ctx)
	if err != nil {
		return nil, err
	}

	ui.Printf("\nPrompt Eval Harness\n")
	ui.Printf("  Subject : %s\n", cfg.SubjectModel)
	ui.Printf("  Judge   : %s\n", cfg.JudgeModel)
	ui.Printf("  Cases   : %d\n", len(cases))
	ui.Printf("  Iters   : %d\n\n", cfg.Iterations)

	type prepared struct {
		c            Case
		systemPrompt string
	}

	var active []prepared
	for _, c := range cases {
		sp, err := LoadSystemPrompt(c, cfg.RepoRoot)
		if err != nil {
			ui.WarnPrintf("Skipped %s: %v\n", c.Name, err)
			continue
		}
		active = append(active, prepared{c: c, systemPrompt: sp})
	}

	results := make([]Result, len(active))
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5) // Limit concurrency to 5 cases at a time

	ui.Println("Executing Cases...")

	for i := range active {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			a := &active[idx]
			res := Result{
				Case:         a.c.Name,
				Tags:         a.c.Tags,
				Description:  a.c.Description,
				SubjectModel: cfg.SubjectModel,
				JudgeModel:   cfg.JudgeModel,
				UserMessage:  a.c.UserMessage,
			}

			ui.Printf("[%d/%d] %s\n", idx+1, len(active), a.c.Name)

			for it := 1; it <= cfg.Iterations; it++ {
				// Phase 1: Subject
				resp, inTokens, outTokens, err := gc.CallSubject(ctx, cfg.SubjectModel, a.systemPrompt, a.c.UserMessage)
				if err != nil {
					res.Error = fmt.Sprintf("subject call failed: %v", err)
					break
				}
				cost := CalculateCost(cfg.SubjectModel, inTokens, outTokens)

				iteration := Iteration{
					Num:             it,
					SubjectResponse: resp,
					InputTokens:     inTokens,
					OutputTokens:    outTokens,
					Cost:            cost,
				}

				// Phase 2: Judge
				prompt := FormatPrometheusPrompt(a.c, resp)
				raw, judgeIn, judgeOut, err := gc.CallJudge(ctx, cfg.JudgeModel, prompt)
				if err != nil {
					res.Error = fmt.Sprintf("judge call failed: %v", err)
					res.Iterations = append(res.Iterations, iteration)
					break
				}
				score := ParseScore(raw)
				iteration.JudgeRaw = raw
				iteration.Score = score

				judgeCost := CalculateCost(cfg.JudgeModel, judgeIn, judgeOut)
				iteration.Cost += judgeCost

				res.Iterations = append(res.Iterations, iteration)

				ui.VerbosePrintf(cfg.Verbose, "  [%s] it=%d score=%v cost=$%.6f\n", a.c.Name, it, score, iteration.Cost)
			}

			results[idx] = aggregateResult(a.c, res.Iterations, cfg)

			scoreStr := "?"
			if results[idx].AvgScore != nil {
				scoreStr = fmt.Sprintf("%.1f", *results[idx].AvgScore)
			}
			ui.Printf(
				"  -> done %s score=%s cost=$%.6f\n",
				ScoreEmoji(results[idx].MaxScore),
				scoreStr,
				results[idx].Cost,
			)

		}(i)
	}

	wg.Wait()

	return results, nil
}

func aggregateResult(c Case, iterations []Iteration, cfg RunConfig) Result {
	var validScores []int
	var totalOutTokens int
	var totalInTokens int
	var totalCost float64

	for _, it := range iterations {
		if it.Score != nil {
			validScores = append(validScores, *it.Score)
		}
		totalOutTokens += it.OutputTokens
		totalInTokens += it.InputTokens
		totalCost += it.Cost
	}

	r := Result{
		Case:         c.Name,
		Tags:         c.Tags,
		Description:  c.Description,
		SubjectModel: cfg.SubjectModel,
		JudgeModel:   cfg.JudgeModel,
		UserMessage:  c.UserMessage,
		Iterations:   iterations,
		Cost:         totalCost,
	}

	if len(iterations) > 0 {
		r.AvgOutputTokens = totalOutTokens / len(iterations)
		avgIn := totalInTokens / len(iterations)
		r.InputTokens = &avgIn
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

	// TokenScore is now cost in micro-dollars for visibility in legacy fields
	ts := int(totalCost * 1_000_000)
	r.TokenScore = &ts

	return r
}
