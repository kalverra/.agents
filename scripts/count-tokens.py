#!/usr/bin/env python3
"""Count tokens using tiktoken (OpenAI-style BPE).

Use the repo venv (from repo root):
  python3 -m venv scripts/.venv
  ./scripts/.venv/bin/pip install -r scripts/requirements.txt
  ./scripts/.venv/bin/python scripts/count-tokens.py ...

Usage:
  ./scripts/count-tokens.py README.md
  ./scripts/count-tokens.py < README.md
  ./scripts/count-tokens.py -s 'hello world'
  ./scripts/count-tokens.py --model gpt-4o file.txt
  ./scripts/count-tokens.py --encoding o200k_base file.txt
"""

from __future__ import annotations

import argparse
import sys
from pathlib import Path


def main() -> int:
    try:
        import tiktoken
    except ImportError:
        print(
            "Missing dependency: ./scripts/.venv/bin/pip install -r scripts/requirements.txt",
            file=sys.stderr,
        )
        return 1

    parser = argparse.ArgumentParser(description="Count LLM tokens in a file or string.")
    parser.add_argument(
        "path",
        nargs="?",
        default=None,
        metavar="FILE",
        help="File to read (if omitted and no --string, read stdin)",
    )
    parser.add_argument("-s", "--string", metavar="TEXT", help="Count tokens in this string instead of a file")
    enc_group = parser.add_mutually_exclusive_group()
    enc_group.add_argument(
        "--encoding",
        default="cl100k_base",
        metavar="NAME",
        help="tiktoken encoding name (default: cl100k_base, common for GPT-4 / 3.5-turbo)",
    )
    enc_group.add_argument(
        "--model",
        metavar="NAME",
        help="Map via tiktoken.encoding_for_model (e.g. gpt-4o, gpt-4, gpt-3.5-turbo)",
    )
    parser.add_argument(
        "-v",
        "--verbose",
        action="store_true",
        help="Print encoding, character count, and token count",
    )
    args = parser.parse_args()

    if args.string is not None:
        text = args.string
    elif args.path is not None:
        text = Path(args.path).read_text(encoding="utf-8")
    else:
        text = sys.stdin.read()

    if args.model:
        try:
            enc = tiktoken.encoding_for_model(args.model)
            enc_name = f"model:{args.model} -> {enc.name}"
        except KeyError:
            print(f"Unknown model for tiktoken: {args.model}", file=sys.stderr)
            print("Try --encoding with a known name (cl100k_base, o200k_base, …)", file=sys.stderr)
            return 1
    else:
        try:
            enc = tiktoken.get_encoding(args.encoding)
            enc_name = enc.name
        except ValueError as e:
            print(f"Unknown encoding: {args.encoding}\n{e}", file=sys.stderr)
            return 1

    n = len(enc.encode(text))

    if args.verbose:
        print(f"encoding: {enc_name}", file=sys.stderr)
        print(f"chars:  {len(text)}", file=sys.stderr)
        print(f"tokens: {n}", file=sys.stderr)
    else:
        print(n)

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
