#!/usr/bin/env python3
import argparse
import json
import pathlib
import re
import sys


BENCHMARK_MATRIX_PATH = pathlib.Path("docs/reports/benchmark-matrix-report.json")
BENCHMARK_READINESS_PATH = pathlib.Path("docs/reports/benchmark-readiness-report.md")
BENCHMARK_REPORT_PATH = pathlib.Path("docs/reports/benchmark-report.md")
LONG_DURATION_REPORT_PATH = pathlib.Path("docs/reports/long-duration-soak-report.md")
LONG_DURATION_SOAK_PATH = pathlib.Path("docs/reports/soak-local-2000x24.json")
THOUSAND_TASK_SOAK_PATH = pathlib.Path("docs/reports/soak-local-1000x24.json")


def read_text(path: pathlib.Path) -> str:
    return path.read_text(encoding="utf-8")


def read_json(path: pathlib.Path):
    return json.loads(read_text(path))


def parse_benchmark_stdout(stdout: str) -> dict[str, dict[str, float]]:
    parsed: dict[str, dict[str, float]] = {}
    for line in stdout.splitlines():
        match = re.match(r"^(Benchmark\S+)\s+\d+\s+([0-9.]+)\s+ns/op$", line.strip())
        if match:
            parsed[match.group(1)] = {"ns_per_op": float(match.group(2))}
    return parsed


def format_ns_per_op(value: float) -> str:
    if float(value).is_integer():
        return str(int(value))
    return f"{value:.2f}".rstrip("0").rstrip(".")


def report_line_for_soak(result: dict) -> str:
    return (
        f"- `{result['count']} tasks x {result['workers']} workers`: "
        f"`{result['elapsed_seconds']:.3f}s`, "
        f"`{result['throughput_tasks_per_sec']:.3f} tasks/s`, "
        f"`{result['succeeded']} succeeded`, "
        f"`{result['failed']} failed`"
    )


def long_duration_summary_line(result: dict) -> str:
    return f"- `2000 tasks x 24 submit workers`: `{result['elapsed_seconds']:.3f}s` elapsed"


def require(condition: bool, message: str, errors: list[str]) -> None:
    if not condition:
        errors.append(message)


def validate(go_root: pathlib.Path) -> list[str]:
    errors: list[str] = []

    required_paths = [
        BENCHMARK_MATRIX_PATH,
        BENCHMARK_READINESS_PATH,
        BENCHMARK_REPORT_PATH,
        LONG_DURATION_REPORT_PATH,
        LONG_DURATION_SOAK_PATH,
        THOUSAND_TASK_SOAK_PATH,
    ]
    for relative_path in required_paths:
        require((go_root / relative_path).exists(), f"missing required artifact: {relative_path}", errors)
    if errors:
        return errors

    benchmark_matrix = read_json(go_root / BENCHMARK_MATRIX_PATH)
    long_duration_soak = read_json(go_root / LONG_DURATION_SOAK_PATH)
    thousand_task_soak = read_json(go_root / THOUSAND_TASK_SOAK_PATH)
    benchmark_readiness = read_text(go_root / BENCHMARK_READINESS_PATH)
    benchmark_report = read_text(go_root / BENCHMARK_REPORT_PATH)
    long_duration_report = read_text(go_root / LONG_DURATION_REPORT_PATH)

    parsed_baseline = parse_benchmark_stdout(benchmark_report)
    parsed_matrix = benchmark_matrix.get("benchmark", {}).get("parsed", {})
    require(parsed_baseline == parsed_matrix, "benchmark-report.md does not match benchmark-matrix-report.json parsed benchmarks", errors)

    for benchmark_name, values in parsed_matrix.items():
        expected_line = f"- `{benchmark_name}`: `{format_ns_per_op(values['ns_per_op'])} ns/op`"
        require(expected_line in benchmark_readiness, f"benchmark readiness missing line: {expected_line}", errors)

    for scenario in benchmark_matrix.get("soak_matrix", []):
        result = scenario["result"]
        require(report_line_for_soak(result) in benchmark_readiness, f"benchmark readiness missing soak matrix line for {result['count']}x{result['workers']}", errors)
        require((go_root / scenario["report_path"]).exists(), f"missing soak matrix artifact: {scenario['report_path']}", errors)

    require(report_line_for_soak(thousand_task_soak) in benchmark_readiness, "benchmark readiness missing soak matrix line for 1000x24", errors)
    require(report_line_for_soak(long_duration_soak) in benchmark_readiness, "benchmark readiness missing soak matrix line for 2000x24", errors)

    long_duration_expected_lines = [
        long_duration_summary_line(long_duration_soak),
        f"- Throughput: `{long_duration_soak['throughput_tasks_per_sec']:.3f} tasks/s`",
        f"- Terminal outcome: `{long_duration_soak['succeeded']} succeeded`, `{long_duration_soak['failed']} failed`",
        "- Sample traces preserved `trace_id`, emitted `scheduler.routed`, and reached `task.completed`",
        "- `docs/reports/soak-local-2000x24.json`",
    ]
    for line in long_duration_expected_lines:
        require(line in long_duration_report, f"long-duration soak report missing line: {line}", errors)

    artifact_mentions = {
        "docs/reports/benchmark-matrix-report.json",
        "docs/reports/soak-local-50x8.json",
        "docs/reports/soak-local-100x12.json",
        "docs/reports/soak-local-1000x24.json",
        "docs/reports/soak-local-2000x24.json",
        "docs/reports/long-duration-soak-report.md",
        "docs/reports/benchmark-report.md",
        "scripts/benchmark/run_matrix.py",
        "scripts/benchmark/validate_closeout.py",
    }
    for artifact in artifact_mentions:
        require(f"`{artifact}`" in benchmark_readiness or f"`{artifact}`" in long_duration_report, f"closeout markdown does not mention artifact: {artifact}", errors)
        require((go_root / artifact).exists(), f"mentioned artifact does not exist: {artifact}", errors)

    return errors


def main() -> int:
    parser = argparse.ArgumentParser(description="Validate benchmark and soak closeout artifacts")
    parser.add_argument("--go-root", default=str(pathlib.Path(__file__).resolve().parents[2]))
    args = parser.parse_args()

    go_root = pathlib.Path(args.go_root)
    errors = validate(go_root)
    if errors:
        for error in errors:
            print(f"ERROR: {error}", file=sys.stderr)
        return 1

    print("benchmark closeout artifacts are consistent")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
