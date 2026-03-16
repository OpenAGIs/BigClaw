#!/usr/bin/env python3
import json
import pathlib
import sys


ROOT = pathlib.Path(__file__).resolve().parents[2]


def require(condition, message, errors):
    if not condition:
        errors.append(message)


def read_text(path):
    return path.read_text(encoding="utf-8")


def main():
    errors = []

    readiness_report = ROOT / "docs/reports/migration-readiness-report.md"
    review_notes = ROOT / "docs/reports/migration-plan-review-notes.md"
    readiness_matrix = ROOT / "docs/reports/review-readiness.md"
    live_index = ROOT / "docs/reports/live-validation-index.md"
    live_manifest = ROOT / "docs/reports/live-validation-index.json"
    shadow_compare = ROOT / "docs/reports/shadow-compare-report.json"
    shadow_matrix = ROOT / "docs/reports/shadow-matrix-report.json"

    for path in [
        readiness_report,
        review_notes,
        readiness_matrix,
        live_index,
        live_manifest,
        shadow_compare,
        shadow_matrix,
    ]:
        require(path.exists(), f"missing required artifact: {path.relative_to(ROOT)}", errors)

    if errors:
        for error in errors:
            print(error, file=sys.stderr)
        return 1

    readiness_text = read_text(readiness_report)
    review_notes_text = read_text(review_notes)
    readiness_matrix_text = read_text(readiness_matrix)
    live_index_text = read_text(live_index)
    live_manifest_payload = json.loads(read_text(live_manifest))
    shadow_compare_payload = json.loads(read_text(shadow_compare))
    shadow_matrix_payload = json.loads(read_text(shadow_matrix))

    require("docs/reports/shadow-compare-report.json" in readiness_text, "migration readiness report must reference shadow compare report", errors)
    require("docs/reports/shadow-matrix-report.json" in readiness_text, "migration readiness report must reference shadow matrix report", errors)
    require("docs/reports/live-validation-index.md" in readiness_text, "migration readiness report must reference live validation index", errors)
    require("docs/reports/migration-plan-review-notes.md" in readiness_text, "migration readiness report must reference migration review notes", errors)

    require("docs/reports/migration-readiness-report.md" in review_notes_text, "migration review notes must reference migration readiness report", errors)
    require("docs/reports/live-validation-index.md" in review_notes_text, "migration review notes must reference live validation index", errors)
    require("docs/reports/shadow-matrix-report.json" in review_notes_text, "migration review notes must reference shadow matrix report", errors)

    require("docs/reports/migration-plan-review-notes.md" in readiness_matrix_text, "review readiness matrix must reference migration plan review notes", errors)
    require("docs/reports/live-validation-index.md" in readiness_matrix_text, "review readiness matrix must reference live validation index", errors)
    require("docs/reports/migration-readiness-report.md" in readiness_matrix_text, "review readiness matrix must reference migration readiness report", errors)
    require("docs/reports/migration-readiness-report.md" in live_index_text, "live validation index must link back to migration readiness report", errors)

    require(shadow_compare_payload.get("diff", {}).get("state_equal") is True, "shadow compare report must show equal terminal state", errors)
    require(shadow_compare_payload.get("diff", {}).get("event_types_equal") is True, "shadow compare report must show equal event type sequence", errors)
    require(shadow_matrix_payload.get("matched") == shadow_matrix_payload.get("total"), "shadow matrix report must be fully matched", errors)

    latest = live_manifest_payload.get("latest", {})
    require(latest.get("status") == "succeeded", "live validation manifest latest status must be succeeded", errors)
    bundle_path = latest.get("bundle_path")
    require(bool(bundle_path), "live validation manifest must include latest bundle path", errors)
    if bundle_path:
        require((ROOT / bundle_path).exists(), f"live validation bundle path missing on disk: {bundle_path}", errors)

    for error in errors:
        print(error, file=sys.stderr)
    if errors:
        return 1

    print("migration review pack references and evidence are consistent")
    return 0


if __name__ == "__main__":
    sys.exit(main())
