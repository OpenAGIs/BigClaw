#!/usr/bin/env python3
import json
import os
import urllib.request

OWNER = "OpenAGIs"
REPO = "BigClaw"
TOKEN = os.environ.get("GITHUB_TOKEN", "")

if not TOKEN:
    raise SystemExit("GITHUB_TOKEN is required")

issues = [
    ("[EPIC] BIG-EPIC-1 任务接入与连接器", "PRD Epic: 任务接入与连接器"),
    ("[EPIC] BIG-EPIC-2 调度与执行控制平面", "PRD Epic: 调度与执行控制平面"),
    ("[EPIC] BIG-EPIC-3 执行层与沙箱", "PRD Epic: 执行层与沙箱"),
    ("[EPIC] BIG-EPIC-4 工作流与验收闭环", "PRD Epic: 工作流与验收闭环"),
    ("[EPIC] BIG-EPIC-5 记忆、日志、审计与可观测", "PRD Epic: 记忆、日志、审计与可观测"),
    ("[EPIC] BIG-EPIC-6 评测、验证与效果报告", "PRD Epic: 评测、验证与效果报告"),
    ("[EPIC] BIG-EPIC-7 试点落地与商业验证", "PRD Epic: 试点落地与商业验证"),
    ("BIG-101 任务入口与工单接入", "P0\n\n验收:\n- 可从真实项目拉取 issue\n- 可回写 issue/PR\n- 失败回写有错误报告"),
    ("BIG-102 统一任务模型", "P0\n\n验收:\n- GitHub/Linear/Jira 数据可映射统一模型\n- 模型可驱动调度执行回写"),
    ("BIG-201 持久化任务队列", "P0\n\n验收:\n- 重启恢复\n- 重试和死信正确"),
    ("BIG-202 Agent Scheduler", "P0\n\n验收:\n- 低风险->Docker\n- 高风险->VM/审批\n- 超预算暂停/降级"),
    ("BIG-203 Event Bus", "P1\n\n验收:\n- PR评论/CI完成/任务失败触发状态流转"),
    ("BIG-301 Claw Worker Runtime", "P0\n\n验收:\n- worker 生命周期稳定\n- 任务结果可完整回传"),
    ("BIG-302 Sandbox 分级执行", "P0\n\n验收:\n- 风险等级到执行介质路由正确"),
    ("BIG-303 Tool Runtime / MCP Gateway", "P0\n\n验收:\n- 工具权限可控\n- 调用可审计"),
    ("BIG-401 Workflow DSL", "P0\n\n验收:\n- 可表达研发闭环与运维runbook"),
    ("BIG-402 Workpad / Execution Journal", "P0\n\n验收:\n- 每任务完整journal,可回放"),
    ("BIG-403 验收门禁与审批流", "P0\n\n验收:\n- 未满足gate不得完成"),
    ("BIG-501 记忆系统", "P1\n\n验收:\n- 可复用历史方案,自动注入项目规则"),
    ("BIG-502 可观测与审计", "P0\n\n验收:\n- 任一失败任务可追踪根因\n- 任一写操作有审计"),
    ("BIG-503 成本与预算控制", "P1\n\n验收:\n- 超预算自动降级/暂停"),
    ("BIG-601 评测框架", "P0\n\n验收:\n- 核心能力有基准评测,版本可对比"),
    ("BIG-602 测试与效果验证报告规范", "P0\n\n验收:\n- 无报告不得关闭issue"),
    ("BIG-701 试点客户实施", "P0\n\n验收:\n- 形成真实生产数据与试点复盘"),
]

def api(path, method="GET", payload=None):
    data = None
    if payload is not None:
        data = json.dumps(payload).encode("utf-8")
    req = urllib.request.Request(
        f"https://api.github.com{path}",
        data=data,
        method=method,
        headers={
            "Accept": "application/vnd.github+json",
            "Authorization": f"Bearer {TOKEN}",
            "User-Agent": "bigclaw-bootstrap",
        },
    )
    with urllib.request.urlopen(req) as resp:
        return json.loads(resp.read().decode("utf-8"))

existing = api(f"/repos/{OWNER}/{REPO}/issues?state=all&per_page=100")
existing_titles = {i["title"] for i in existing}

created = []
for title, body in issues:
    if title in existing_titles:
        continue
    payload = {"title": title, "body": body, "labels": ["bigclaw", "prd-v1"]}
    created_issue = api(f"/repos/{OWNER}/{REPO}/issues", method="POST", payload=payload)
    created.append((created_issue["number"], created_issue["title"]))

print(json.dumps({"created_count": len(created), "created": created}, ensure_ascii=False, indent=2))
