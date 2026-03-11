# BigClaw Issue Plan

## BigClaw PRD v4.0

### Epic
- BIG-EPIC-20: v4.0 v3候选与进入条件

### Epic 20: v4.0 v3候选与进入条件 (`OPE-120`)

#### Goal
- Define the v4.0 v3 candidate backlog as an auditable planning artifact so downstream implementation starts only after the scope, validation commands, and entry conditions are explicit.

#### Candidate backlog
- Console release control center: converge console shell governance, acceptance evidence, and report output into one release-control candidate with a clear promotion contract.
- Operations command-center hardening: promote queue, triage, regression, and weekly-review surfaces as one candidate with operator-facing validation and reporting coverage.
- Agent orchestration rollout: carry entitlement-aware orchestration, handoff visibility, and commercialization evidence into a candidate that is ready for v3-level release review.

#### Candidate traceability
- Operations command-center hardening evidence should explicitly cover queue control center, approval queue handling, auto-triage, regression analysis, weekly operations reporting, dashboard builder output, saved views, and rollback simulation status so v3 planning reflects the actual operator journey.
- Link evidence to repo-native artifacts instead of prose-only references; minimum trace set for `OPE-139`:
  - command center: `src/bigclaw/operations.py`, `tests/test_control_center.py`, `tests/test_operations.py`
  - approval: `src/bigclaw/execution_contract.py`, `src/bigclaw/workflow.py`, `tests/test_workflow.py`, `tests/test_execution_flow.py`
  - saved views: `src/bigclaw/saved_views.py`, `tests/test_saved_views.py`
  - dashboard builder: `src/bigclaw/operations.py`, `tests/test_operations.py`
  - simulation / replay evidence: `src/bigclaw/evaluation.py`, `tests/test_evaluation.py`
- Keep candidate artifacts rich enough to render both named evidence requirements and clickable evidence links in the backlog report so review can happen without reconstructing supporting files manually.

#### Entry gate
- At least 3 candidates must be ready with named owners, validation commands, and no unresolved launch blockers.
- The ready set must cover `release-gate`, `ops-control`, and `commercialization` capabilities so the epic advances with both console and execution-plane scope.
- The ready set must include `acceptance-suite`, `pilot-evidence`, and `validation-report` evidence so implementation can be validated and reviewed without inventing a parallel checklist.

#### Delivery shape
- Store the candidate backlog in code so readiness can be ranked, rendered, and regression-tested like the rest of the BigClaw planning/reporting surface.
- Treat the entry gate as a deterministic evaluation over candidate capabilities, evidence, and blockers instead of a narrative-only milestone.
- Keep repo traceability in `docs/issue-plan.md` and an issue validation report so the Linear ticket, planning artifact, and validation commands stay aligned.

## BigClaw PRD v2.0

### Epic
- BIG-EPIC-8: 研发自治执行平台增强
- BIG-EPIC-9: 工程运营系统
- BIG-EPIC-10: 跨部门 Agent Orchestration
- BIG-EPIC-11: 产品化 UI 与控制台
- BIG-EPIC-12: 计费、套餐与商业化控制
- BIG-EPIC-16: v4.0 范围冻结与任务治理
- BIG-EPIC-17: v4.0 UI Design Sprint
- BIG-EPIC-18: v4.0 执行层技术契约
- BIG-EPIC-19: v4.0 设计评审与决策
- BIG-EPIC-20: v4.0 v3候选与进入条件

### Epic 16: v4.0 范围冻结与任务治理 (`OPE-116`)

#### Goal
- Freeze the v4.0 scope around the validated v2.0 baseline, govern backlog entry through explicit exceptions, and require every execution slice to capture validation evidence plus repository closeout artifacts.

#### Child issues
- `OPE-116` / `BIG-EPIC-16`: v4.0 范围冻结与任务治理 — Step1, v2.0 freeze and backlog governance.
- `OPE-117` / `BIG-EPIC-17`: v4.0 UI Design Sprint — Step2, key page design sprint.
- `OPE-118` / `BIG-EPIC-18`: v4.0 执行层技术契约 — Step3, models/APIs/permissions/metrics/audit.
- `OPE-119` / `BIG-EPIC-19`: v4.0 设计评审与决策 — Step4, cross-role review and decisions.
- `OPE-120` / `BIG-EPIC-20`: v4.0 v3候选与进入条件 — Step5, v3 candidate backlog and gate.

#### Delivery shape
- Model scope freeze as an auditable board with explicit backlog ownership, phase sequencing, and approved exceptions for any post-freeze additions.
- Require backlog items to define acceptance criteria, validation plans, and the run-closeout contract of validation evidence, `git push`, and `git log -1 --stat` capture before work can be considered governance-ready.
- Keep v4.0 planning attached to the same evidence-first workflow as execution so downstream UI, contract, and review epics inherit a shared release gate.

### Issue `OPE-137` / `BIG-4403`: 问题清单与优先级归档

#### Goal
- Archive the v4.0 review findings as one deterministic artifact so UI, IA, permission, and metric issues can be prioritized with explicit `P0`/`P1`/`P2` severity and named owners.

#### Delivery shape
- Store each finding in code with category, priority, owner, surface, impact, and evidence so the review backlog can be serialized and regression-tested.
- Audit the archive for missing owners, invalid priorities/categories, and unresolved `P0` findings so release review cannot hand-wave critical defects.
- Render a plain-text report with per-priority and per-category rollups so the issue packet is reviewable in repo-native validation output.

### Epic-Milestone Mapping
- BIG-EPIC-8: Phase 1 / M1 Foundation uplift / owner `engineering-platform`
- BIG-EPIC-9: Phase 2 / M2 Operations control plane / owner `engineering-operations`
- BIG-EPIC-10: Phase 3 / M3 Cross-team orchestration / owner `orchestration-office`
- BIG-EPIC-11: Phase 4 / M4 Productized console / owner `product-experience`
- BIG-EPIC-12: Phase 5 / M5 Billing and packaging / owner `commercialization`

### Epic 10: 跨部门 Agent Orchestration (`OPE-66`)

#### Goal
- Build the cross-department orchestration layer on top of the BigClaw scheduler, workflow engine, and observability ledger so tasks can plan handoffs, enforce collaboration policy, and render auditable coordination artifacts.

#### Child issues
- `OPE-71` / `BIG-803`: Premium Orchestration Policy — P0, entitlement-aware orchestration upgrades and approval requirements.
- `OPE-73` / `BIG-805`: 团队协作与人工接管 — P1, handoff routing and takeover queues for multi-team execution.

#### Delivery shape
- Model orchestration as a first-class planning step with explicit departments, handoffs, approvals, and collaboration modes attached to each scheduled task.
- Record orchestration policy, handoff requests, and execution traces in the observability ledger so every cross-team run can be replayed and audited.
- Render orchestration plans, per-run canvases, portfolio rollups, and overview pages from ledger-backed artifacts rather than maintaining a separate reporting store.

### Epic 11: 产品化 UI 与控制台 (`OPE-86`)

#### Goal
- Establish the reusable UI foundation for the BigClaw console so dashboard, replay, triage, and orchestration views can share one token system and governed component library.

#### Child issues
- `OPE-86` / `BIG-1103`: 设计系统与组件库 — P1, v2 design system and component library foundation.
- `OPE-93` / `BIG-1302`: 顶部全局区与命令入口 — P0, global header with search, context switching, alert entry, and command shell.
- `OPE-89` / `BIG-1106`: 信息架构与全局交互 — P1, global IA, top bar, filters, states, and actions for the console shell.
- `OPE-115` / `BIG-1701`: v3.0 UI 测试与验收体系 — P0, role-permission, data accuracy, performance, usability, and audit completeness acceptance gates.
- `OPE-112` / `BIG-1607`: Report Studio (v3) — P1, narrative report composing and export.
- `OPE-111` / `BIG-1606`: Policy/Prompt Version Center — P1, version diff/history/rollback across workflow, prompt, and policy assets.
- `OPE-128` / `BIG-4204`: UI评审包输出 — P0, compile the review objectives, wireframes, interaction notes, and open questions into one auditable review pack.

#### Delivery shape
- Treat design tokens as a platform asset rather than per-page styling so future console slices can audit consistency and release readiness.
- Model component readiness around documentation, accessibility coverage, and interactive-state completeness to keep operational UI slices shippable.
- Expose a renderer that can summarize inventory, gaps, and orphan tokens for design reviews and launch gates.
- Add governed console-chrome slices for global navigation patterns so search, time/environment switching, alerts, and command entry can be audited before they reach product surfaces.
- Model console-level navigation, top-bar actions, filter systems, and surface states as auditable assets so dashboard and control-center pages inherit one interaction contract.
- Add a UI acceptance suite that scores release readiness across role-permission coverage, data accuracy, performance budgets, usability journeys, and audit-trail completeness before v3.0 console slices ship.
- Add a report-studio layer that composes narrative sections from operational evidence and exports the same report as markdown, plain text, and HTML for downstream sharing.
- Surface version history, diff summaries, and rollback-ready targets for workflow, prompt, and policy assets as auditable control-center artifacts.
- Package UI reviews as deterministic artifacts with explicit review objectives, frame inventory, interaction coverage, and unresolved decisions so design critique can happen against one canonical packet.

### Issue `OPE-128` / `BIG-4204`: UI评审包输出

#### Goal
- Compile a review-ready packet for the `BIG-4204` console slice so reviewers can evaluate objectives, wireframes, interaction behavior, and unresolved decisions without reconstructing the scope from separate docs.

#### Review pack contract
- Capture review objectives with named personas, expected outcomes, success signals, and dependency callouts so kickoff alignment is explicit.
- Capture each wireframe surface with entry point, device context, primary layout blocks, and review notes so frame-level discussion has one manifest.
- Capture interaction flows with trigger, system response, required states, and exception paths so prototype-level feedback is not reduced to static screenshots.
- Capture open questions with owner and impact so unresolved product decisions remain visible during review instead of being hidden in comments.

#### Delivery shape
- Store the review pack in code as a serializable artifact with a deterministic completeness audit.
- Render the pack as a plain-text report so the same data can be reviewed in repo-native validation output and future exported docs.
- Keep `docs/issue-plan.md` and the issue validation report aligned with the same review-pack structure so ticket intent and shipped artifact stay traceable.

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
- `OPE-133` / `BIG-4305`: 指标口径与可复算规范 — P0, reproducible metric definitions for runs, lead time, intervention, SLA, regression, risk, and spend.

#### Delivery shape
- Surface queue, run, replay, and audit data already emitted by the `Scheduler`, `WorkflowEngine`, `ObservabilityLedger`, and reporting modules.
- Add operational dashboards and control loops in slices so each child issue can reuse the same execution ledger and validation-report workflow.
- Treat entitlement, SLA, risk, triage, and regression analysis as orchestration/reporting layers over the existing execution core rather than parallel systems.
- Keep metric definitions auditable in code with explicit formulas, source fields, and rendered evidence so operations reviews can recompute the same values without spreadsheet-only logic.

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
