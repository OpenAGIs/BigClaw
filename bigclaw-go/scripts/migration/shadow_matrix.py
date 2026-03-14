#!/usr/bin/env python3
import argparse
import json
import pathlib
import sys

import shadow_compare


def load_tasks(task_files):
    tasks = []
    for task_file in task_files:
        with open(task_file, 'r', encoding='utf-8') as fh:
            task = json.load(fh)
        task['_source_file'] = task_file
        tasks.append(task)
    return tasks


def main():
    parser = argparse.ArgumentParser(description='Run a shadow-comparison matrix across multiple task files')
    parser.add_argument('--primary', required=True)
    parser.add_argument('--shadow', required=True)
    parser.add_argument('--task-file', action='append', required=True)
    parser.add_argument('--timeout-seconds', type=int, default=180)
    parser.add_argument('--health-timeout-seconds', type=int, default=60)
    parser.add_argument('--report-path')
    args = parser.parse_args()

    tasks = load_tasks(args.task_file)
    results = []
    for index, task in enumerate(tasks, start=1):
        base_id = task.get('id', f'matrix-task-{index}')
        task['id'] = f'{base_id}-m{index}'
        result = shadow_compare.compare_task(args.primary, args.shadow, task, args.timeout_seconds, args.health_timeout_seconds)
        result['source_file'] = task['_source_file']
        results.append(result)

    matched = sum(1 for item in results if item['diff']['state_equal'] and item['diff']['event_types_equal'])
    report = {
        'total': len(results),
        'matched': matched,
        'mismatched': len(results) - matched,
        'results': results,
    }
    if args.report_path:
        path = pathlib.Path(args.report_path)
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(json.dumps(report, ensure_ascii=False, indent=2), encoding='utf-8')
    print(json.dumps(report, ensure_ascii=False, indent=2))
    return 0 if matched == len(results) else 1


if __name__ == '__main__':
    sys.exit(main())
