# BigClaw Issue Plan

## BigClaw PRD v2.0

### Epic
- BIG-EPIC-8: 研发自治执行平台增强
- BIG-EPIC-9: 工程运营系统
- BIG-EPIC-10: 跨部门 Agent Orchestration
- BIG-EPIC-11: 产品化 UI 与控制台
- BIG-EPIC-12: 计费、套餐与商业化控制

### Epic 10: 跨部门 Agent Orchestration (`OPE-83`)

#### Goal
- Add a cross-team flow overview page so operations can track orchestration handoffs, approvals, and blocked departments in one place.

#### Delivery shape
- Reuse `OrchestrationPlan`, premium policy decisions, and `TaskRun` execution evidence to summarize active cross-team flows.
- Provide both report and HTML page renderers so the overview can feed future console/dashboard surfaces without duplicating orchestration logic.

### Epic 9: 工程运营系统 (`OPE-65`)

#### Goal
- Build the engineering operations control plane on top of the existing BigClaw runtime, scheduler, workflow, and reporting foundation.

#### Child issues
- `OPE-69` / `BIG-801`: 团队级执行总览 Dashboard — P0, team engineering overview dashboard.
- `OPE-70` / `BIG-802`: 队列与调度控制中心 — P0, visual scheduler control center.
- `OPE-71` / `BIG-803`: Premium Orchestration Policy — P0, premium orchestration entitlement.
- `OPE-72` / `BIG-804`: Run Detail 与执行回放页 — P0, run detail replay page.
- `OPE-73` / `BIG-805`: 团队协作与人工接管 — P1, human handoff and takeover.
- `OPE-74` / `BIG-901`: SLA 与运营看板 — P0, operations and SLA dashboard.
- `OPE-75` / `BIG-902`: 风险评分系统 — P0, risk scoring for tasks and runs.
- `OPE-76` / `BIG-903`: 自动 Triage Center — P0, auto triage center.
- `OPE-77` / `BIG-904`: 回归分析中心 — P0, regression analytics center.
- `OPE-78` / `BIG-905`: 工程运营周报自动生成 — P1, weekly operations report auto generation.

#### Delivery shape
- Surface queue, run, replay, and audit data already emitted by the `Scheduler`, `WorkflowEngine`, `ObservabilityLedger`, and reporting modules.
- Add operational dashboards and control loops in slices so each child issue can reuse the same execution ledger and validation-report workflow.
- Treat entitlement, SLA, risk, triage, and regression analysis as orchestration/reporting layers over the existing execution core rather than parallel systems.

## BigClaw PRD v1.0

### Epic
- BIG-EPIC-1: 任务接入与连接器
- BIG-EPIC-2: 调度与执行控制平面
- BIG-EPIC-3: 执行层与沙箱
- BIG-EPIC-4: 工作流与验收闭环
- BIG-EPIC-5: 记忆、日志、审计与可观测
- BIG-EPIC-6: 评测、验证与效果报告
- BIG-EPIC-7: 试点落地与商业验证

### Issues
- BIG-101: 任务入口与工单接入 (P0)
- BIG-102: 统一任务模型 (P0)
- BIG-201: 持久化任务队列 (P0)
- BIG-202: Agent Scheduler (P0)
- BIG-203: Event Bus (P1)
- BIG-301: Claw Worker Runtime (P0)
- BIG-302: Sandbox 分级执行 (P0)
- BIG-303: Tool Runtime / MCP Gateway (P0)
- BIG-401: Workflow DSL (P0)
- BIG-402: Workpad / Execution Journal (P0)
- BIG-403: 验收门禁与审批流 (P0)
- BIG-501: 记忆系统 (P1)
- BIG-502: 可观测与审计 (P0)
- BIG-503: 成本与预算控制 (P1)
- BIG-601: 评测框架 (P0)
- BIG-602: 测试与效果验证报告规范 (P0)
- BIG-701: 试点客户实施 (P0)

### 本次实现范围 (v0.1 foundation)
- BIG-102: 统一任务模型基础字段与 schema 实体
- BIG-201: 持久化任务队列最小实现
- BIG-202: 按风险/工具路由的调度策略最小实现
- BIG-601: benchmark/replay/评分体系与版本对比最小实现
- BIG-602: 测试与效果报告模板最小实现
