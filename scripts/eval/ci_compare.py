#!/usr/bin/env python3
import json
import sys
from pathlib import Path


def format_diff(current, previous, invert_color=False):
    if current is None or previous is None:
        return ""
    diff = current - previous
    if abs(diff) < 0.01:
        return ""

    sign = "+" if diff > 0 else ""
    val_str = f"{sign}{diff:.2f}" if isinstance(diff, float) else f"{sign}{int(diff)}"

    # We use simple github markdown, no rich formatting needed, no BBCode
    return f"({val_str})"


def score_emoji(score):
    if score is None:
        return "❓"
    return ["", "🔴", "🟠", "🟡", "🟢", "✅"][round(score)]


def main():
    if len(sys.argv) != 4:
        print("Usage: ci_compare.py <main_json> <pr_json> <output_md>")
        sys.exit(1)

    main_path = Path(sys.argv[1])
    pr_path = Path(sys.argv[2])
    out_path = Path(sys.argv[3])

    if not main_path.exists():
        print(f"Main json not found at {main_path}, proceeding blindly.")
        main_history = {}
    else:
        main_history = json.loads(main_path.read_text())

    if not pr_path.exists():
        print(f"PR json not found at {pr_path}. Fail.")
        sys.exit(1)

    pr_history = json.loads(pr_path.read_text())

    pr_scores = [
        v.get("avg_score")
        for v in pr_history.values()
        if v.get("avg_score") is not None
    ]
    pr_total = sum(pr_scores) / len(pr_scores) if pr_scores else 0

    main_scores = [
        v.get("avg_score")
        for v in main_history.values()
        if v.get("avg_score") is not None
    ]
    main_total = sum(main_scores) / len(main_scores) if main_scores else 0
    total_diff = format_diff(pr_total, main_total)

    lines = [
        "## Agent Prompt Evaluation",
        f"**Overall Score:** {score_emoji(pr_total)} {pr_total:.2f}/5 {total_diff}",
        "",
    ]

    if pr_total < 4.0:
        lines.append("> 🔴 **FAILED:** Overall score is below the 4.0 threshold.")
        lines.append("")
    elif pr_total < main_total:
        lines.append(
            f"> 🟠 **WARNING:** Quality regression detected! Score dropped from {main_total:.2f} to {pr_total:.2f}."
        )
        lines.append("")

    lines.append("| Case | Score | Token Score |")
    lines.append("|------|-------|-------------|")

    for case_id, pr_data in pr_history.items():
        main_data = main_history.get(case_id, {})

        pr_score = pr_data.get("avg_score")
        main_score = main_data.get("avg_score")
        score_df = format_diff(pr_score, main_score)

        pr_token_score = pr_data.get("token_score")
        main_token_score = main_data.get("token_score")
        token_score_df = format_diff(
            pr_token_score, main_token_score, invert_color=True
        )

        sc_str = (
            f"{score_emoji(pr_score)} {pr_score:.2f} {score_df}".strip()
            if pr_score is not None
            else "ERR"
        )
        ts_str = f"{pr_token_score or '?'} {token_score_df}".strip()

        lines.append(f"| {case_id} | {sc_str} | {ts_str} |")

    out_path.write_text("\n".join(lines))

    # Determine exit status
    if pr_total < 4.0:
        print("Score below 4.0 threshold.")
        sys.exit(1)

    print("Evaluation checks passed.")
    sys.exit(0)


if __name__ == "__main__":
    main()
