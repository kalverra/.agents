#!/usr/bin/env python3
"""Prompt eval harness for .agents repo.

Tests whether a subject model follows agent instructions by having a judge
model (Prometheus) score responses 1–5 against a rubric.

Usage:
    python scripts/eval/eval.py [OPTIONS]

Options:
    --subject MODEL    ollama model to test (default: llama3.1:8b)
    --judge MODEL      ollama judge model (default: prometheus model)
    --cases DIR        path to test cases dir (default: scripts/eval/cases/)
    --filter TAG       only run cases with this tag
    --iterations INT   number of times to run each case (default: 1)
    --output FILE      write JSON results to file (default: stdout summary)
    --report FILE      write full markdown report to file (e.g. eval_results.md)
    --verbose          show full subject response and judge feedback
    --repo ROOT        repo root for resolving system_prompt_file paths
                       (default: parent of this script's directory)
"""

import argparse
import json
import re
import subprocess
import sys
import time
from datetime import datetime, timezone
from pathlib import Path
from typing import Optional

try:
    import ollama
    HAS_OLLAMA = True
except Exception:
    HAS_OLLAMA = False

try:
    import yaml
except ImportError:
    print("ERROR: pyyaml not installed. Run: pip install pyyaml", file=sys.stderr)
    sys.exit(1)

try:
    from rich.console import Console
    from rich.table import Table
    from rich.panel import Panel
    from rich import print as rprint
    HAS_RICH = True
except ImportError:
    HAS_RICH = False

try:
    import tiktoken
    HAS_TIKTOKEN = True
except ImportError:
    HAS_TIKTOKEN = False

try:
    from google import genai
    from google.genai import types
    HAS_GENAI = True
except ImportError:
    HAS_GENAI = False

PROMETHEUS_MODEL = "hf.co/divish/M-Prometheus-7B-Q4_K_M-GGUF:latest"
DEFAULT_SUBJECT = "llama3.1:8b"

PROMETHEUS_PROMPT = """\
###Task Description: An instruction (might include an Input inside it), a response to evaluate, a reference answer that gets a score of 5, and a score rubric representing an evaluation criteria are given.
1. Write a detailed feedback that assesses the quality of the response strictly based on the given score rubric, not evaluating in general.
2. After writing a feedback, write a score that is an integer between 1 and 5. You should refer to the score rubric.
3. The output format should look as follows: "Feedback: (write a feedback for criteria) [RESULT] (an integer number between 1 and 5)"
4. Please do not generate any other opening, closing, and explanations.

###The instruction to evaluate:
{instruction}

###Response to evaluate:
{response}

###Reference Answer (Score 5):
{reference_answer}

###Score Rubrics:
[{criteria_name}]
Score 1: {score_1}
Score 2: {score_2}
Score 3: {score_3}
Score 4: {score_4}
Score 5: {score_5}

###Feedback:\
"""


def resolve_placeholders(content: str, repo_root: Path) -> str:
    """Substitute USER_AGENTS.md into template text."""
    user_src = repo_root / "USER_AGENTS.md"
    if not user_src.is_file():
        return content

    user_content = user_src.read_text(encoding="utf-8").strip()
    if not user_content:
        return content

    USER_AGENTS_PLACEHOLDER = "<!-- Instructions from USER_AGENTS.md are appended here during install -->"
    if USER_AGENTS_PLACEHOLDER in content:
        return content.replace(USER_AGENTS_PLACEHOLDER, user_content)

    m = re.search(r"(<user>).*?(</user>)", content, re.DOTALL)
    if m:
        return re.sub(
            r"(<user>).*?(</user>)",
            lambda match: match.group(1) + "\n" + user_content + "\n" + match.group(2),
            content,
            flags=re.DOTALL
        )

    return content.rstrip() + "\n\n" + user_content + "\n"


def load_file(path: Path, repo_root: Path) -> str:
    """Read a file and resolve any install-time placeholders."""
    return resolve_placeholders(path.read_text(encoding="utf-8"), repo_root)


def load_system_prompt(case: dict, repo_root: Path) -> str:
    """Resolve system prompt from inline text or one/many file references."""
    if "system_prompt" in case:
        return case["system_prompt"].strip()
    if "system_prompt_files" in case:
        parts = []
        for rel in case["system_prompt_files"]:
            p = repo_root / rel
            if not p.exists():
                raise FileNotFoundError(f"system_prompt_file not found: {p}")
            parts.append(load_file(p, repo_root).strip())
        return "\n\n---\n\n".join(parts)
    if "system_prompt_file" in case:
        p = repo_root / case["system_prompt_file"]
        if not p.exists():
            raise FileNotFoundError(f"system_prompt_file not found: {p}")
        return load_file(p, repo_root).strip()
    return ""


def call_gemini(model: str, system_prompt: str, user_message: str, max_retries: int = 3) -> tuple[str, int]:
    """Send system + user message to Gemini model, return (response_text, output_tokens)."""
    if not HAS_GENAI:
        raise RuntimeError("google-genai package is required. Run: pip install google-genai")

    client = genai.Client()

    config = types.GenerateContentConfig(
        system_instruction=system_prompt if system_prompt else None,
        temperature=0.0,
        safety_settings=[
            types.SafetySetting(
                category=cat,
                threshold='BLOCK_NONE'
            ) for cat in [
                'HARM_CATEGORY_HATE_SPEECH',
                'HARM_CATEGORY_HARASSMENT',
                'HARM_CATEGORY_SEXUALLY_EXPLICIT',
                'HARM_CATEGORY_DANGEROUS_CONTENT',
                'HARM_CATEGORY_CIVIC_INTEGRITY'
            ]
        ]
    )

    for attempt in range(max_retries):
        try:
            response = client.models.generate_content(
                model=model,
                contents=user_message,
                config=config
            )

            output = ""
            if response.candidates:
                cand = response.candidates[0]
                if cand.content and cand.content.parts:
                    for part in cand.content.parts:
                        if part.text:
                            output += part.text
                        elif part.call:
                            # Convert tool call to a string representation
                            args = ", ".join(f"{k}={v}" for k, v in part.call.args.items())
                            output += f"\n[Tool Call: {part.call.name}({args})]"

            output_tokens = 0
            if response.usage_metadata:
                output_tokens = response.usage_metadata.candidates_token_count or 0

            return output or "", output_tokens
        except Exception as e:
            if attempt < max_retries - 1:
                wait = min(2 ** attempt, 30)
                print(f"\n  [warn] Gemini error: {e}, retrying in {wait}s...", file=sys.stderr)
                time.sleep(wait)
            else:
                raise

def call_subject(model: str, model_type: str, system_prompt: str, user_message: str, max_retries: int = 3) -> tuple[str, int]:
    """Send system + user message to subject model, return (response_text, output_tokens)."""
    if model_type == "gemini":
        return call_gemini(model, system_prompt, user_message, max_retries)

    if not HAS_OLLAMA:
        raise RuntimeError("ollama package is required for local models. Run: pip install ollama")

    messages = []
    if system_prompt:
        messages.append({"role": "system", "content": system_prompt})
    messages.append({"role": "user", "content": user_message})

    for attempt in range(max_retries):
        try:
            resp = ollama.chat(model=model, messages=messages)
            return resp.message.content or "", resp.eval_count or 0
        except ollama.ResponseError as e:
            if attempt < max_retries - 1:
                wait = min(2 ** attempt, 30)
                print(f"\n  [warn] Ollama error ({e.status_code}), retrying in {wait}s...", file=sys.stderr)
                time.sleep(wait)
            else:
                raise


def call_judge(judge_model: str, judge_type: str, case: dict, subject_response: str, max_retries: int = 5) -> tuple[str, Optional[int]]:
    """Call judge, return (feedback_text, score)."""
    criteria = case["criteria"]
    prompt = PROMETHEUS_PROMPT.format(
        instruction=case["user_message"],
        response=subject_response,
        reference_answer=case["reference_answer"],
        criteria_name=criteria["name"],
        score_1=criteria["score_1"],
        score_2=criteria["score_2"],
        score_3=criteria["score_3"],
        score_4=criteria["score_4"],
        score_5=criteria["score_5"],
    )

    if judge_type == "gemini":
        for attempt in range(max_retries):
            try:
                raw, _ = call_gemini(judge_model, "", prompt, max_retries=1)
                score = parse_score(raw)
                return raw, score
            except Exception as e:
                if attempt < max_retries - 1:
                    wait = min(2 ** attempt, 30)
                    print(f"\n  [warn] Gemini error: {e}, retrying in {wait}s...", file=sys.stderr)
                    time.sleep(wait)
                else:
                    raise

    if not HAS_OLLAMA:
        raise RuntimeError("ollama package is required for local judge models. Run: pip install ollama")

    for attempt in range(max_retries):
        try:
            resp = ollama.chat(
                model=judge_model,
                messages=[{"role": "user", "content": prompt}],
            )
            raw = resp.message.content or ""
            score = parse_score(raw)
            return raw, score
        except ollama.ResponseError as e:
            if attempt < max_retries - 1:
                wait = min(2 ** attempt, 30)
                print(f"\n  [warn] Ollama error ({e.status_code}), retrying in {wait}s...", file=sys.stderr)
                time.sleep(wait)
            else:
                raise


def parse_score(text: str) -> Optional[int]:
    """Extract integer score from Prometheus [RESULT] N output."""
    match = re.search(r"\[RESULT\]\s*(\d)", text)
    if match:
        val = int(match.group(1))
        if 1 <= val <= 5:
            return val
    return None


def score_emoji(score: Optional[int]) -> str:
    if score is None:
        return "❓"
    return ["", "🔴", "🟠", "🟡", "🟢", "✅"][score]


def load_cases(cases_dir: Path, tag_filter: Optional[str]) -> list[dict]:
    cases = []
    for f in sorted(cases_dir.glob("*.yaml")):
        c = yaml.safe_load(f.read_text())
        c["_file"] = f.name
        if tag_filter and tag_filter not in c.get("tags", []):
            continue
        cases.append(c)
    if not cases:
        raise ValueError(f"No cases found in {cases_dir}" + (f" with tag '{tag_filter}'" if tag_filter else ""))
    return cases


def get_git_info(repo_root: Path) -> dict:
    """Get current git commit and dirty status."""
    try:
        commit = subprocess.check_output(
            ["git", "rev-parse", "--short", "HEAD"],
            cwd=repo_root,
            stderr=subprocess.DEVNULL,
            text=True
        ).strip()

        # Check for uncommitted changes in agent-related files
        dirty = subprocess.call(
            ["git", "diff", "--quiet", "--", "GLOBAL_AGENTS.md", "USER_AGENTS.md", "scripts/eval/"],
            cwd=repo_root,
            stderr=subprocess.DEVNULL
        ) != 0

        return {"commit": commit, "dirty": dirty}
    except Exception:
        return {"commit": "unknown", "dirty": False}


def clear_vram(console=None):
    """Unloads all models currently loaded in Ollama VRAM."""
    if not HAS_OLLAMA:
        return
    try:
        active = ollama.ps()
        # Handle ProcessResponse object (0.6.x) or dict (older)
        models_list = getattr(active, "models", []) if not isinstance(active, dict) else active.get("models", [])

        if not models_list:
            return

        for model in models_list:
            # Safely extract model name from ProcessModel object, dict, or tuple
            model_name = getattr(model, "model", getattr(model, "name", None))
            if isinstance(model, dict) and not model_name:
                model_name = model.get("name") or model.get("model")
            if isinstance(model, tuple) and not model_name:
                model_name = model[0]

            if not model_name:
                continue

            try:
                # keep_alive=0 unloads immediately
                ollama.generate(model=model_name, keep_alive=0)
                if console:
                    console.print(f"  [dim]Unloaded: {model_name}[/dim]")
            except Exception:
                pass

        # Essential pause for GPU driver/Ollama to reclaim memory
        time.sleep(2)
    except Exception as e:
        if console:
            console.print(f"  [dim](warn) Failed to clear VRAM: {e}[/dim]")


def run_eval(args: argparse.Namespace) -> list[dict]:
    repo_root = Path(args.repo)
    cases_dir = Path(args.cases)
    console = Console() if HAS_RICH else None

    cases = load_cases(cases_dir, args.filter)

    use_local = args.subject_type == "local" or args.judge_type == "local"

    if console:
        console.print(f"\n[bold cyan]🧪 Prompt Eval Harness[/bold cyan]")
        console.print(f"  Subject : [yellow]{args.subject}[/yellow]")
        console.print(f"  Judge   : [yellow]{args.judge}[/yellow]")
        console.print(f"  Cases   : [yellow]{len(cases)}[/yellow]")
        console.print(f"  Iters   : [yellow]{args.iterations}[/yellow]\n")

        if use_local:
            console.print("[bold cyan]▶ Pre-run: Cleaning VRAM[/bold cyan]")
            clear_vram(console)
    else:
        print(f"\nPrompt Eval Harness")
        print(f"  Subject : {args.subject}")
        print(f"  Judge   : {args.judge}")
        print(f"  Cases   : {len(cases)}")
        print(f"  Iters   : {args.iterations}\n")

        if use_local:
            print("Cleaning VRAM...")
            clear_vram()

    results = []
    active_cases = []

    # PREPARE: Load prompts and handle file errors
    for case in cases:
        name = case.get("name", case["_file"])
        desc = case.get("description", "")
        try:
            system_prompt = load_system_prompt(case, repo_root)
            tokens = None
            if HAS_TIKTOKEN:
                try:
                    tokens = len(tiktoken.get_encoding("cl100k_base").encode(system_prompt))
                except Exception:
                    pass
            active_cases.append({
                "case": case,
                "name": name,
                "desc": desc,
                "system_prompt": system_prompt,
                "tokens": tokens,
                "iterations": []
            })
        except FileNotFoundError as e:
            print(f"  ⚠ Skipped {name}: {e}", file=sys.stderr)
            results.append({"case": name, "error": str(e), "tags": case.get("tags", [])})

    # PHASE 1: Generate subject responses
    if console and active_cases:
        console.print("\n[bold cyan]▶ Phase 1: Generating Responses (Subject)[/bold cyan]")

    for i, ac in enumerate(active_cases, 1):
        name = ac["name"]
        if console:
            console.print(f"[{i}/{len(active_cases)}] [bold]{name}[/bold]")
        else:
            print(f"\n[{i}/{len(active_cases)}] {name} (Generating)")

        for it in range(1, args.iterations + 1):
            iter_prefix = f" [{it}/{args.iterations}]" if args.iterations > 1 else ""
            if console:
                console.print(f"  → calling subject{iter_prefix}...", end="")

            subject_response, output_tokens = call_subject(args.subject, args.subject_type, ac["system_prompt"], ac["case"]["user_message"])

            if console:
                console.print(" done")
                if args.verbose:
                    console.print(Panel(subject_response, title=f"Subject Response{iter_prefix}", border_style="dim"))
            else:
                if args.verbose:
                    print(f"  SUBJECT{iter_prefix}:\n{subject_response}\n")

            ac["iterations"].append({
                "iteration": it,
                "subject_response": subject_response,
                "output_tokens": output_tokens,
            })

    # PHASE 2: Evaluate responses
    if active_cases:
        if console:
            console.print(f"\n[bold cyan]▶ Phase 2: Evaluating Responses ({args.judge})[/bold cyan]")
        else:
            print(f"\nPhase 2: Evaluating Responses ({args.judge})")

        # Unload subject model to clear VRAM for judge
        if use_local:
            clear_vram(console)

    for i, ac in enumerate(active_cases, 1):
        name = ac["name"]
        if console:
            console.print(f"[{i}/{len(active_cases)}] [bold]{name}[/bold]")
        else:
            print(f"\n[{i}/{len(active_cases)}] {name} (Evaluating)")

        for it_data in ac["iterations"]:
            it = it_data["iteration"]
            iter_prefix = f" [{it}/{args.iterations}]" if args.iterations > 1 else ""

            if console:
                console.print(f"  → calling judge{iter_prefix}...", end="")

            judge_raw, score = call_judge(args.judge, args.judge_type, ac["case"], it_data["subject_response"])

            if console:
                console.print(f" done  {score_emoji(score)} score={score}")
                if args.verbose:
                    console.print(Panel(judge_raw, title=f"Judge Feedback{iter_prefix}", border_style="dim"))
            else:
                print(f"  score: {score}")
                if args.verbose:
                    print(f"  JUDGE{iter_prefix}:\n{judge_raw}\n")

            it_data["judge_raw"] = judge_raw
            it_data["score"] = score


        # Aggregate results for this case
        valid_scores = [it["score"] for it in ac["iterations"] if it["score"] is not None]
        avg_score = sum(valid_scores) / len(valid_scores) if valid_scores else None
        min_score = min(valid_scores) if valid_scores else None
        max_score = max(valid_scores) if valid_scores else None

        output_tokens_list = [it.get("output_tokens", 0) for it in ac["iterations"]]
        avg_output_tokens = sum(output_tokens_list) / len(output_tokens_list) if output_tokens_list else 0
        input_tokens = ac["tokens"]
        token_score = round(input_tokens + avg_output_tokens * 5) if input_tokens is not None else None

        results.append({
            "case": ac["name"],
            "tags": ac["case"].get("tags", []),
            "description": ac["desc"],
            "subject_model": args.subject,
            "subject_type": args.subject_type,
            "judge_model": args.judge,
            "judge_type": args.judge_type,
            "user_message": ac["case"]["user_message"],
            "tokens": input_tokens,
            "avg_output_tokens": round(avg_output_tokens),
            "token_score": token_score,
            "iterations": ac["iterations"],
            "avg_score": avg_score,
            "min_score": min_score,
            "max_score": max_score,
        })

    if use_local:
        clear_vram(console)

    return results


def format_diff(current, previous, is_fmt: bool = True, invert_color: bool = False, label: str = "") -> str:
    if current is None or previous is None:
        return ""
    diff = current - previous
    if abs(diff) < 0.01:
        return ""

    sign = "+" if diff > 0 else ""
    val_str = f"{sign}{diff:.2f}" if isinstance(diff, float) else f"{sign}{int(diff)}"

    if not is_fmt:
        res = f"({val_str})"
        return f"{res} {label}".strip() if label else res

    color = "green" if diff > 0 else "red"
    if invert_color:
        color = "red" if diff > 0 else "green"

    res = f"[[{color}]{val_str}[/{color}]]"
    return f"{res} [dim]{label}[/dim]" if label else res


def print_summary(results: list[dict], history: dict, console, iterations: int):
    avg_scores = [r["avg_score"] for r in results if r.get("avg_score") is not None]
    total_avg = sum(avg_scores) / len(avg_scores) if avg_scores else 0
    passed = sum(1 for r in results if r.get("avg_score") is not None and r["avg_score"] >= 4.0)
    total_input_tokens = sum(r.get("tokens") or 0 for r in results if "error" not in r)
    total_output_tokens = sum(r.get("avg_output_tokens") or 0 for r in results if "error" not in r)
    total_token_score = sum(r.get("token_score") or 0 for r in results if "error" not in r)

    if HAS_RICH and console:
        table = Table(title="\nResults Summary", show_lines=True)
        table.add_column("Case", style="cyan", no_wrap=True)
        table.add_column("Tags", style="dim")
        table.add_column("In Tokens", justify="right")
        table.add_column("Out Tokens", justify="right")
        table.add_column("Token Score", justify="right")
        if iterations > 1:
            table.add_column("Avg", justify="center")
            table.add_column("Min", justify="center")
            table.add_column("Max", justify="center")
        else:
            table.add_column("Score", justify="center")
        table.add_column("Grade", justify="center")

        for r in results:
            if "error" in r:
                cols = [r["case"], "", "ERR", "ERR", "ERR"]
                if iterations > 1: cols += ["ERR", "ERR", "ERR"]
                else: cols += ["ERR"]
                cols += ["⚠"]
                table.add_row(*cols)
                continue

            score = r.get("avg_score")
            grade = score_emoji(round(score) if score is not None else None)
            color = "red" if (score or 0) <= 2 else "yellow" if (score or 0) < 4.0 else "green"

            prev = history.get(r["case"], {})
            score_diff = format_diff(score, prev.get("avg_score"))
            token_score_diff = format_diff(r.get("token_score"), prev.get("token_score"), invert_color=True)

            in_tok = str(r.get("tokens", "?"))
            out_tok = str(r.get("avg_output_tokens", "?"))
            ts = r.get("token_score", "?")
            token_score_str = f"{ts} {token_score_diff}".strip()
            row = [r["case"], ", ".join(r.get("tags", [])), in_tok, out_tok, token_score_str]

            score_val = f"{score:.2f}" if iterations > 1 and score is not None else str(int(score)) if score is not None else "ERR"
            score_str = f"[{color}]{score_val}[/{color}] {score_diff}".strip()

            if iterations > 1:
                row += [score_str, str(r['min_score']), str(r['max_score'])]
            else:
                row += [score_str]

            row += [grade]
            table.add_row(*row)

        console.print(table)
        console.print(f"\n[bold]Overall Average:[/bold] {total_avg:.2f}/5  |  Passed (Avg >= 4.0): {passed}/{len(avg_scores)}")
        console.print(f"[bold]Total Token Score:[/bold] {total_token_score}  [dim](in={total_input_tokens} out={total_output_tokens})[/dim]\n")
    else:
        print("\n--- Results ---")
        for r in results:
            if "error" in r:
                print(f"  {r['case']}: ERR")
            elif iterations > 1:
                avg = f"{r['avg_score']:.2f}" if r.get("avg_score") is not None else "ERR"
                print(f"  {r['case']}: Avg={avg} Min={r['min_score']} Max={r['max_score']} In={r.get('tokens')} Out={r.get('avg_output_tokens')} TokenScore={r.get('token_score')}")
            else:
                avg = f"{r['avg_score']:.0f}/5" if r.get("avg_score") is not None else "ERR"
                print(f"  {r['case']}: {avg} In={r.get('tokens')} Out={r.get('avg_output_tokens')} TokenScore={r.get('token_score')}")
        print(f"\nOverall Average: {total_avg:.2f}/5  |  Passed (Avg >= 4.0): {passed}/{len(avg_scores)}")
        print(f"Total Token Score: {total_token_score} (in={total_input_tokens} out={total_output_tokens})")


SCORE_BADGE = {
    1: "🔴 1/5",
    2: "🟠 2/5",
    3: "🟡 3/5",
    4: "🟢 4/5",
    5: "✅ 5/5",
}


def write_markdown_report(results: list[dict], history: dict, git_info: dict, path: Path, args: argparse.Namespace) -> None:
    from datetime import datetime, timezone

    avg_scores = [r["avg_score"] for r in results if r.get("avg_score") is not None]
    total_avg = sum(avg_scores) / len(avg_scores) if avg_scores else 0
    passed = sum(1 for r in results if r.get("avg_score") is not None and r["avg_score"] >= 4.0)
    total_input_tokens = sum(r.get("tokens") or 0 for r in results if "error" not in r)
    total_output_tokens = sum(r.get("avg_output_tokens") or 0 for r in results if "error" not in r)
    total_token_score = sum(r.get("token_score") or 0 for r in results if "error" not in r)
    now = datetime.now(timezone.utc).strftime("%Y-%m-%d %H:%M UTC")

    git_status = f"`{git_info['commit']}`"
    if git_info["dirty"]:
        git_status += " (with local changes)"

    lines = [
        "# Eval Results\n",
        f"**Run:** {now}  ",
        f"**Commit:** {git_status}  ",
        f"**Subject:** `{args.subject}`  ",
        f"**Judge:** `{args.judge}`  ",
        f"**Cases:** {len(results)}  ",
        f"**Iterations:** {args.iterations}  ",
        f"**Average score:** {total_avg:.2f}/5  |  **Passed (Avg >= 4.0):** {passed}/{len(avg_scores)}  ",
        f"**Total Token Score:** {total_token_score} (in={total_input_tokens} out={total_output_tokens})\n",
        "---\n",
        "## Summary\n",
    ]

    if args.iterations > 1:
        lines.append("| # | Case | Tags | In | Out | Token Score | Avg | Min | Max |")
        lines.append("|---|------|------|----|-----|-------------|-----|-----|-----|")
    else:
        lines.append("| # | Case | Tags | In | Out | Token Score | Score |")
        lines.append("|---|------|------|----|-----|-------------|-------|")

    for i, r in enumerate(results, 1):
        tags = ", ".join(f"`{t}`" for t in r.get("tags", []))
        if "error" in r:
            if args.iterations > 1:
                lines.append(f"| {i} | [{r['case']}](#{r['case']}) | {tags} | ERR | ERR | ERR | ERR | ERR | ERR |")
            else:
                lines.append(f"| {i} | [{r['case']}](#{r['case']}) | {tags} | ERR | ERR | ERR | ⚠ error |")
            continue

        prev = history.get(r["case"], {})
        score = r.get("avg_score")
        score_diff = format_diff(score, prev.get("avg_score"), is_fmt=False)
        token_score_diff = format_diff(r.get("token_score"), prev.get("token_score"), is_fmt=False)

        in_tok = r.get("tokens", "?")
        out_tok = r.get("avg_output_tokens", "?")
        token_score_str = f"{r.get('token_score', '?')} {token_score_diff}".strip()

        if args.iterations > 1:
            badge = SCORE_BADGE.get(round(score), "❓")
            score_str = f"{badge} {score:.2f} {score_diff}".strip()
            lines.append(f"| {i} | [{r['case']}](#{r['case']}) | {tags} | {in_tok} | {out_tok} | {token_score_str} | {score_str} | {r['min_score']} | {r['max_score']} |")
        else:
            badge = SCORE_BADGE.get(int(score or 0), "❓") if score is not None else "❓"
            score_str = f"{badge} {score_diff}".strip()
            lines.append(f"| {i} | [{r['case']}](#{r['case']}) | {tags} | {in_tok} | {out_tok} | {token_score_str} | {score_str} |")

    lines.append("\n---\n")
    lines.append("## Case Details\n")

    for r in results:
        tags = " ".join(f"`{t}`" for t in r.get("tags", []))
        prev = history.get(r.get("case", ""), {})
        token_score_diff = format_diff(r.get("token_score"), prev.get("token_score"), is_fmt=False)
        token_score_str = f"{r.get('token_score', '?')} {token_score_diff}".strip()

        lines += [
            f"### {r['case']}",
            f"",
            f"{tags}  ",
            f"**Token Score:** {token_score_str} (in={r.get('tokens', '?')} out={r.get('avg_output_tokens', '?')}×5)  ",
            f"**Description:** {r.get('description', '')}\n",
        ]

        score = r.get("avg_score")
        score_diff = format_diff(score, prev.get("avg_score"), is_fmt=False)

        if args.iterations > 1:
            badge = SCORE_BADGE.get(round(score or 0), "❓")
            score_str_md = f"{score:.2f}" if score is not None else "ERR"
            lines.append(f"**Average Score:** {badge} {score_str_md} {score_diff} (Min: {r.get('min_score')}, Max: {r.get('max_score')})  ")
        else:
            badge = SCORE_BADGE.get(int(score or 0), "❓") if score is not None else "❓"
            lines.append(f"**Score:** {badge} {score_diff}  ")

        lines += [
            f"#### User Message\n",
            f"```",
            r.get("user_message", "").strip(),
            f"```\n",
        ]

        if "error" in r:
            lines.append(f"> ⚠ **Error:** {r['error']}\n")
            lines.append("---\n")
            continue

        for it in r.get("iterations", []):
            iter_header = f"Iteration {it['iteration']}" if args.iterations > 1 else "Response"
            lines += [
                f"#### {iter_header} (`{r.get('subject_model', '')}`)",
                f"**Score:** {SCORE_BADGE.get(it['score'], '❓')}",
                f"",
                (it.get("subject_response") or "").strip(),
                f"\n**Judge Feedback**",
                f"",
                (it.get("judge_raw") or "").strip(),
                f"\n",
            ]
        lines.append("---\n")

    path.write_text("\n".join(lines))


def main():
    parser = argparse.ArgumentParser(
        description="Eval harness: test whether models follow .agents instructions"
    )
    parser.add_argument("--subject-type", choices=["local", "gemini"], default="local", help="Type of subject model")
    parser.add_argument("--judge-type", choices=["local", "gemini"], default="local", help="Type of judge model")
    parser.add_argument("--subject", help="Subject model (default varies by type)")
    parser.add_argument("--judge", help="Judge model (default varies by type)")
    parser.add_argument("--cases", default=str(Path(__file__).parent / "cases"), help="Test cases directory")
    parser.add_argument("--filter", help="Only run cases with this tag")
    parser.add_argument("--iterations", type=int, default=1, help="Number of times to run each case")
    parser.add_argument("--output", help="Write JSON results to this file")
    parser.add_argument("--report", help="Write full markdown report to this file (e.g. eval_results.md)")
    parser.add_argument("--verbose", action="store_true", help="Show full responses")
    parser.add_argument("--check-dirty", action="store_true", help="Instantly verify history validity (used in pre-commit hooks).")
    parser.add_argument(
        "--repo",
        default=str(Path(__file__).parent.parent.parent),
        help="Repo root (for system_prompt_file resolution)",
    )
    args = parser.parse_args()

    if args.subject is None:
        args.subject = DEFAULT_SUBJECT if args.subject_type == "local" else "gemini-2.5-flash" if args.subject_type == "gemini" else None
    if args.judge is None:
        args.judge = PROMETHEUS_MODEL if args.judge_type == "local" else "gemini-2.5-pro" if args.judge_type == "gemini" else None

    repo_root = Path(args.repo)
    history_file = Path(__file__).parent / "eval_history.json"

    if args.check_dirty:
        console = Console() if HAS_RICH else None
        def pr_err(msg):
            if console: console.print(f"[bold red]❌ {msg}[/bold red]")
            else: print(f"ERROR: {msg}")

        if not history_file.exists():
            pr_err("No eval_history.json found. You must run `just eval` before committing.")
            sys.exit(1)

        try:
            history = json.loads(history_file.read_text())
        except Exception as e:
            pr_err(f"Corrupt eval_history.json: {e}")
            sys.exit(1)

        global_agents_path = repo_root / "GLOBAL_AGENTS.md"
        if global_agents_path.exists():
            global_mtime = global_agents_path.stat().st_mtime
            hist_mtime = history_file.stat().st_mtime
            if global_mtime > hist_mtime:
                pr_err("GLOBAL_AGENTS.md has been modified since the last eval run. Please run `just eval-multi` to update scores.")
                sys.exit(1)

        scores = [v.get("avg_score") for v in history.values() if v.get("avg_score") is not None]
        total = sum(scores) / len(scores) if scores else 0
        if total < 4.0:
            pr_err(f"Overall average score is {total:.2f}. Must be >= 4.0 to commit.")
            sys.exit(1)

        sys.exit(0)

    git_info = get_git_info(repo_root)

    history = {}
    if history_file.exists():
        try:
            history = json.loads(history_file.read_text())
        except Exception:
            pass

    console = Console() if HAS_RICH else None
    results = run_eval(args)
    print_summary(results, history, console, args.iterations)

    # Merge new results into existing history
    for r in results:
        if "error" not in r:
            history[r["case"]] = {
                "avg_score": r.get("avg_score"),
                "tokens": r.get("tokens"),
                "avg_output_tokens": r.get("avg_output_tokens"),
                "token_score": r.get("token_score"),
                "subject_model": r.get("subject_model"),
                "subject_type": r.get("subject_type"),
                "judge_model": r.get("judge_model"),
                "judge_type": r.get("judge_type")
            }

    history_file.write_text(json.dumps(history, indent=2))

    if args.output:
        out = Path(args.output)
        out.write_text(json.dumps(results, indent=2))
        msg = f"JSON written to {out}"
        if console:
            console.print(f"\n[dim]{msg}[/dim]")
        else:
            print(f"\n{msg}")

    if args.report:
        rep = Path(args.report)
        write_markdown_report(results, history, git_info, rep, args)
        msg = f"Report written to {rep}"
        if console:
            console.print(f"[dim]{msg}[/dim]")
        else:
            print(msg)


if __name__ == "__main__":
    main()
