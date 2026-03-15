#!/usr/bin/env python3
import argparse
import json
import os
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path
from typing import Dict, List, Set

REPO_ROOT = Path(__file__).resolve().parents[2]
SRC_ROOT = REPO_ROOT / "src"
if str(SRC_ROOT) not in sys.path:
    sys.path.insert(0, str(SRC_ROOT))

from bigclaw.parallel_refill import ParallelIssueQueue, issue_state_map

POLL_QUERY = """
query RefillIssues($projectSlug: String!, $stateNames: [String!]!, $first: Int!, $after: String) {
  issues(filter: {project: {slugId: {eq: $projectSlug}}, state: {name: {in: $stateNames}}}, first: $first, after: $after) {
    nodes {
      id
      identifier
      title
      state {
        name
      }
    }
    pageInfo {
      hasNextPage
      endCursor
    }
  }
}
"""

PROMOTE_MUTATION = """
mutation PromoteIssue($id: String!, $input: IssueUpdateInput!) {
  issueUpdate(id: $id, input: $input) {
    success
    issue {
      identifier
      state {
        name
      }
    }
  }
}
"""


class LinearClient:
    def __init__(self, api_key: str):
        self.api_key = api_key

    def graphql(self, query: str, variables: dict) -> dict:
        payload = json.dumps({"query": query, "variables": variables}).encode("utf-8")
        request = urllib.request.Request(
            "https://api.linear.app/graphql",
            data=payload,
            method="POST",
            headers={
                "Content-Type": "application/json",
                "Authorization": self.api_key,
            },
        )
        try:
            with urllib.request.urlopen(request, timeout=30) as response:
                body = response.read().decode("utf-8")
        except urllib.error.HTTPError as exc:
            body = exc.read().decode("utf-8", errors="replace")
            raise RuntimeError(f"Linear HTTP {exc.code}: {body}") from exc
        data = json.loads(body)
        if data.get("errors"):
            raise RuntimeError(json.dumps(data["errors"], ensure_ascii=False))
        return data["data"]

    def fetch_issue_states(self, project_slug: str, state_names: List[str]) -> List[dict]:
        issues: List[dict] = []
        cursor = None
        while True:
            data = self.graphql(
                POLL_QUERY,
                {
                    "projectSlug": project_slug,
                    "stateNames": state_names,
                    "first": 50,
                    "after": cursor,
                },
            )
            page = data["issues"]
            issues.extend(page["nodes"])
            page_info = page["pageInfo"]
            if not page_info.get("hasNextPage"):
                return issues
            cursor = page_info.get("endCursor")

    def promote_issue(self, identifier: str, state_id: str) -> dict:
        data = self.graphql(
            PROMOTE_MUTATION,
            {"id": identifier, "input": {"stateId": state_id}},
        )
        return data["issueUpdate"]


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Refill the BigClaw parallel Linear queue.")
    parser.add_argument(
        "--queue",
        default="docs/parallel-refill-queue.json",
        help="Path to the queue definition JSON.",
    )
    parser.add_argument(
        "--target-in-progress",
        type=int,
        default=None,
        help="Override the target number of In Progress issues.",
    )
    parser.add_argument(
        "--watch",
        action="store_true",
        help="Keep watching and refill when the active issue count drops.",
    )
    parser.add_argument(
        "--interval",
        type=int,
        default=20,
        help="Polling interval in seconds when --watch is enabled.",
    )
    parser.add_argument(
        "--apply",
        action="store_true",
        help="Actually promote issues; without this flag the command is dry-run.",
    )
    parser.add_argument(
        "--refresh-url",
        default="",
        help="Optional dashboard refresh endpoint to POST after a promotion.",
    )
    return parser.parse_args()


def load_client() -> LinearClient:
    api_key = os.environ.get("LINEAR_API_KEY", "").strip()
    if not api_key:
        raise SystemExit("LINEAR_API_KEY is required")
    return LinearClient(api_key)


def trigger_refresh(refresh_url: str) -> None:
    if not refresh_url:
        return
    request = urllib.request.Request(refresh_url, data=b"{}", method="POST")
    request.add_header("Content-Type", "application/json")
    with urllib.request.urlopen(request, timeout=10):
        return


def run_once(queue: ParallelIssueQueue, client: LinearClient, apply: bool, refresh_url: str, target_override: int | None) -> int:
    refill_state_names = sorted(queue.refill_states())
    issue_states = client.fetch_issue_states(queue.project_slug(), ["In Progress", *refill_state_names])
    state_map = issue_state_map(issue_states)
    active_identifiers: Set[str] = {identifier for identifier, state in state_map.items() if state == "In Progress"}
    candidates = queue.select_candidates(active_identifiers, state_map, target_override)

    print(
        json.dumps(
            {
                "active_in_progress": sorted(active_identifiers),
                "target_in_progress": queue.target_in_progress() if target_override is None else int(target_override),
                "candidates": candidates,
                "mode": "apply" if apply else "dry-run",
            },
            ensure_ascii=False,
            indent=2,
        )
    )

    if not apply:
        return 0

    promoted = 0
    for identifier in candidates:
        result = client.promote_issue(identifier, queue.activate_state_id())
        if result.get("success"):
            promoted += 1
            print(f"promoted {identifier} -> {result['issue']['state']['name']}")
            trigger_refresh(refresh_url)
    return promoted


def main() -> int:
    args = parse_args()
    queue = ParallelIssueQueue(args.queue)
    client = load_client()

    if not args.watch:
        run_once(queue, client, args.apply, args.refresh_url, args.target_in_progress)
        return 0

    while True:
        try:
            run_once(queue, client, args.apply, args.refresh_url, args.target_in_progress)
        except Exception as exc:  # pragma: no cover - operator path
            print(f"watcher error: {exc}", file=sys.stderr)
        time.sleep(args.interval)


if __name__ == "__main__":
    raise SystemExit(main())
