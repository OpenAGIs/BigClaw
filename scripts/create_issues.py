#!/usr/bin/env python3
import json
import os
import sys
import urllib.request

OWNER = "OpenAGIs"
REPO = "BigClaw"
TOKEN = os.environ.get("GITHUB_TOKEN", "")

if not TOKEN:
    raise SystemExit("GITHUB_TOKEN is required")

V1_ISSUES = [
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

V2_OPS_ISSUES = [
    ("[EPIC] BIG-EPIC-9 工程运营系统", "Epic for engineering operations system."),
    ("BIG-801 团队级执行总览 Dashboard", "P0\n\n验收:\n- 按团队查看执行中的任务、通过率、阻塞态\n- 能从总览跳转到单任务/单次运行详情"),
    ("BIG-802 队列与调度控制中心", "P0\n\n验收:\n- 展示队列深度、调度决策、执行介质分布\n- 可识别待审批、失败、重试任务"),
    ("BIG-803 Premium Orchestration Policy", "P0\n\n验收:\n- 支持 Premium 编排策略和能力边界配置\n- 策略变更有审计记录"),
    ("BIG-804 Run Detail 与执行回放页", "P0\n\n验收:\n- 可查看单次运行日志、trace、artifact、audit\n- 支持按 run_id 回放执行证据"),
    ("BIG-805 团队协作与人工接管", "P1\n\n验收:\n- 支持人工接管、转派、恢复执行\n- 交接过程保留 journal 与审批痕迹"),
    ("BIG-901 SLA 与运营看板", "P0\n\n验收:\n- 展示 SLA、吞吐、失败率、审批等待时长\n- 支持按团队/周期聚合"),
    ("BIG-902 风险评分系统", "P0\n\n验收:\n- 对任务和运行结果输出风险评分\n- 风险分可驱动审批与调度策略"),
    ("BIG-903 自动 Triage Center", "P0\n\n验收:\n- 自动聚合同类失败与阻塞原因\n- 支持建议下一步处理动作"),
    ("BIG-904 回归分析中心", "P0\n\n验收:\n- 对版本、基线、回放结果做回归比较\n- 能定位退化任务与指标"),
    ("BIG-905 工程运营周报自动生成", "P1\n\n验收:\n- 自动生成周报摘要、风险项、改进建议\n- 可复用验证报告和运营指标"),
]

PLAN_CONFIG = {
    "v1": {"issues": V1_ISSUES, "labels": ["bigclaw", "prd-v1"]},
    "v2-ops": {"issues": V2_OPS_ISSUES, "labels": ["bigclaw", "prd-v2", "ops"]},
}


def resolve_plan() -> dict[str, list]:
    selected_plan = sys.argv[1] if len(sys.argv) > 1 else os.environ.get("BIGCLAW_PLAN", "v1")
    if selected_plan not in PLAN_CONFIG:
        available = ", ".join(sorted(PLAN_CONFIG))
        raise SystemExit(f"Unknown plan '{selected_plan}'. Available plans: {available}")
    return PLAN_CONFIG[selected_plan]


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
existing_titles = {issue["title"] for issue in existing}

plan = resolve_plan()
created = []
for title, body in plan["issues"]:
    if title in existing_titles:
        continue
    payload = {"title": title, "body": body, "labels": plan["labels"]}
    created_issue = api(f"/repos/{OWNER}/{REPO}/issues", method="POST", payload=payload)
    created.append((created_issue["number"], created_issue["title"]))

print(json.dumps({"created_count": len(created), "created": created}, ensure_ascii=False, indent=2))
