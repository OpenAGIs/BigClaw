#!/usr/bin/env python3
import argparse
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SRC = ROOT / 'src'
if str(SRC) not in sys.path:
    sys.path.insert(0, str(SRC))

from bigclaw.reports import generate_weekly_operations_report


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Generate a weekly operations report from an observability ledger.")
    parser.add_argument("--ledger", required=True, help="Path to observability ledger JSON")
    parser.add_argument("--out", required=True, help="Output markdown report path")
    parser.add_argument("--team", required=True, help="Team name for the report header")
    parser.add_argument("--period-start", required=True, help="Inclusive report window start in ISO-8601 format")
    parser.add_argument("--period-end", required=True, help="Inclusive report window end in ISO-8601 format")
    parser.add_argument("--generated-at", help="Optional generated-at timestamp in ISO-8601 format")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    ledger_path = Path(args.ledger)
    if not ledger_path.exists() or not ledger_path.is_file():
        raise SystemExit(f"Ledger not found: {ledger_path}")

    runs = json.loads(ledger_path.read_text())
    report = generate_weekly_operations_report(
        runs=runs,
        report_path=args.out,
        team=args.team,
        period_start=args.period_start,
        period_end=args.period_end,
        generated_at=args.generated_at,
    )

    print(
        json.dumps(
            {
                "out": str(Path(args.out)),
                "total_runs": report.total_runs,
                "successful_runs": report.successful_runs,
                "approvals_pending": report.approvals_pending,
            },
            ensure_ascii=False,
        )
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
