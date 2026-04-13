---
name: iterate-improve
description: Eval-edit loop to raise prompt scores and reduce token cost.
---

<goal>
Phase A: Raise all eval case avg_score >= 4.
Phase B: Minimize token_score for all cases while keeping avg_score >= 4.
</goal>

<constraints>
Editable files: GLOBAL_AGENTS.md
Do not edit: eval cases, Go source, justfile, workflows.
Do not add eval cases.
Create a git branch before first edit.
Revert any change that drops a case below avg_score 4.
You have a spend-cap of $1.00. Do not exceed this cap.
If the user specifies a set amount of cycles, stick that amount.
</constraints>

<phase name="A" trigger="any case avg_score < 4">
1. Sort cases by score ascending.
2. Read lowest-scoring case YAML and its system_prompt_file.
3. Edit prompt file to address failure.
4. Re-eval. Commit if target case improved and no case dropped below 4. Else revert.
</phase>

<phase name="B" trigger="all cases avg_score >= 4">
1. Find case with highest token_score.
2. Read its system_prompt_file. Identify verbose or redundant sections.
3. Tighten phrasing to reduce token count.
4. Re-eval. Accept only if all scores >= 4 and token_score decreased. Else revert.
</phase>

<workflow>
1. Budget check: `go run . eval --ai-output --spend-check --spend-cap 1.00`. Stop if over budget.
2. Run eval: `go run . eval --iterations 3 --ai-output`. Record avg_score and token_score per case.
3. If any avg_score < 4: execute Phase A. Else: execute Phase B. If all scores >= 4 and no token_score reduction possible: stop.
4. Loop to step 1. Stop after no improvement path, spend limit reached, or user specified cycle count.
5. Report: list changes, before/after scores and token_scores, total spend.
6. If no improvements, revert all changes and exit.
7. If improvements have been made, commit changes and make a PR to the repo.
</workflow>
