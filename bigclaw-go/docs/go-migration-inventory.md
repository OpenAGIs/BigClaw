# BIG-GO-901 Go迁移盘点与目标架构

## 范围与口径

- 盘点对象: 所有仍会影响 Go 主线收敛的非 Go 可执行资产。
- 纳入口径: `src/bigclaw/*.py` 中的 Python 模块，`bigclaw-go/scripts/**/*.py` 中的 Python 脚本，以及 `bigclaw-go/scripts/**/*.sh` 中的 shell 包装器。
- 排除项: `src/bigclaw/__init__.py` 这类包标记文件、Markdown/JSON 报告、示例数据和纯证据产物；这些不是迁移阻塞代码面。
- 全量账本: 见 `bigclaw-go/docs/reports/go-migration-ledger.json`，共 76 个资产，覆盖率目标为 100%。
- 机器可读计划摘要: 见 `bigclaw-go/docs/reports/go-migration-plan-summary.json`，用于消费 Wave、lane、验证命令和风险。

## 当前资产总览

| 资产类型 | 数量 | 说明 |
| --- | ---: | --- |
| Python 模块 | 49 | 主要位于 `src/bigclaw`，包含遗留 runtime、契约、产品面和报告逻辑 |
| Python 脚本 | 23 | 主要位于 `bigclaw-go/scripts`，承担迁移 shadow、验证 bundle、benchmark 编排 |
| Shell 包装器 | 4 | `run_all.sh`、`kubernetes_smoke.sh`、`ray_smoke.sh`、`run_suite.sh` |
| P0 | 22 | 直接阻塞 Go 成为唯一主线的入口、契约、调度/runtime、迁移脚本 |
| P1 | 44 | 与现有 Go 包一一对应的控制面、repo、治理、验证资产 |
| P2 | 10 | 产品信息架构、设计系统、评审视图、规划类辅助逻辑 |

优先级按以下原则排序:

- `P0`: 会阻塞 Go 成为唯一 operator path，或直接影响 shadow、bundle、run-all、核心契约与运行时。
- `P1`: 已经存在明确 Go 包落点，但还需要补齐行为和回归面。
- `P2`: 面向产品呈现和规划报告，依赖控制面稳定后再迁移更稳妥。

## 目标架构

最终状态不是 “Python 改写成 Go 文件名对位”，而是收敛到 Go 控制平面单主线:

| 架构层 | Go 落点 | 最终职责 |
| --- | --- | --- |
| Operator 入口 | `cmd/bigclawd`, `cmd/bigclawctl` | 唯一服务启动、诊断、迁移、验证、benchmark 入口 |
| 核心域模型 | `internal/domain`, `internal/contract`, `internal/flow` | 统一任务模型、执行契约、DSL 与状态机 |
| 控制面内核 | `internal/queue`, `internal/scheduler`, `internal/worker`, `internal/executor`, `internal/events` | 队列、调度、运行时、执行器、事件日志 |
| 控制与策略 | `internal/workflow`, `internal/orchestrator`, `internal/policy`, `internal/governance`, `internal/risk`, `internal/billing` | 工作流编排、策略裁决、冻结/治理、风险与预算 |
| 接入与仓库面 | `internal/intake`, `internal/githubsync`, `internal/repo`, `internal/refill`, `internal/bootstrap` | issue intake、GitHub 同步、repo 视图、refill、本地 bootstrap |
| 产品与报告面 | `internal/product`, `internal/reporting`, `internal/api`, `internal/regression`, `internal/prd` | dashboard/readiness surface、报告生成、HTTP surface、回归门禁 |
| 兼容层 | `internal/legacyshim` | 仅保留编译时或 CLI 兼容提示，不再承载业务逻辑 |

硬边界:

- `src/bigclaw` 不再新增业务逻辑，只允许短期兼容提示，最终整体移除。
- `bigclaw-go/scripts` 下的 Python 和 shell 编排器全部下沉到 `cmd/bigclawctl` 子命令。
- 报告与 bundle 产物的生成逻辑统一由 Go 命令驱动，`internal/regression` 负责 schema 与文档回归。

## 可执行迁移方案

### Wave 0: 账本与门禁

- 固化本 issue 产出的全量资产账本和目标架构。
- 对账本跑完整性校验，后续 PR 若新增 Python 或 shell 资产必须同步更新账本。
- 将 “Go-only mainline” 作为迁移准则，禁止新增新的 Python control-plane 逻辑。

### Wave 1: 先切断 operator 入口和迁移必经路径

这批资产优先级最高，建议先做:

| 主题 | 资产 | Go 落点 | 完成定义 |
| --- | --- | --- | --- |
| 唯一入口 | `src/bigclaw/__main__.py`, `src/bigclaw/service.py`, `src/bigclaw/deprecation.py`, `src/bigclaw/legacy_shim.py` | `cmd/bigclawd`, `cmd/bigclawctl`, `internal/legacyshim` | operator 不再需要 Python 命令；兼容提示由 Go 输出 |
| 核心契约 | `src/bigclaw/models.py`, `src/bigclaw/execution_contract.py`, `src/bigclaw/observability.py` | `internal/domain`, `internal/contract`, `internal/observability`, `internal/api` | Go 成为唯一 schema 与 run telemetry authority |
| 核心控制面 | `src/bigclaw/queue.py`, `src/bigclaw/scheduler.py`, `src/bigclaw/runtime.py`, `src/bigclaw/workflow.py`, `src/bigclaw/orchestration.py` | `internal/queue`, `internal/scheduler`, `internal/worker`, `internal/workflow`, `internal/orchestrator` | Python 冻结面可删除，只留 Go 测试和回归 |
| 迁移 shadow | `bigclaw-go/scripts/migration/*.py` | `cmd/bigclawctl migration ...` | `shadow-compare`/`shadow-matrix`/`live-shadow-scorecard`/`export-live-shadow` 全部走 Go |
| 统一验证入口 | `bigclaw-go/scripts/e2e/run_all.sh`, `export_validation_bundle.py`, `multi_node_shared_queue.py` 及其 Python tests | `cmd/bigclawctl validation ...`, `internal/reporting`, `internal/regression` | review bundle、run-all、shared-queue 证据链全部由 Go 命令生成 |

### Wave 2: 收敛控制面剩余 Python 模块

- `connectors.py`, `mapping.py`, `workspace_bootstrap*.py` -> `internal/intake`, `internal/bootstrap`, `cmd/bigclawctl`
- `governance.py`, `cost_control.py`, `risk.py`, `validation_policy.py` -> `internal/governance`, `internal/policy`, `internal/risk`, `internal/billing`
- `repo_*.py`, `github_sync.py`, `parallel_refill.py`, `operations.py`, `repo_triage.py` -> `internal/repo`, `internal/githubsync`, `internal/refill`, `internal/triage`
- 完成定义: `src/bigclaw` 仅剩文档性或待删兼容壳，不再被主流程 import。

### Wave 3: 收敛产品与报告面

- `reports.py`, `run_detail.py`, `saved_views.py`, `dashboard_run_contract.py` -> `internal/reporting`, `internal/product`, `internal/api`
- `console_ia.py`, `ui_review.py`, `design_system.py`, `planning.py`, `evaluation.py`, `pilot.py`, `roadmap.py`, `memory.py`, `issue_archive.py` -> `internal/product`, `internal/reporting`, `internal/prd`
- 完成定义: 所有 reviewer/operator 可见 JSON、Markdown、HTTP surface 都由 Go 生成。

## 首批实现/改造清单

建议按 5 个并行 PR lane 落地，避免互相阻塞:

1. `lane/core-cutover`
   - 处理 `src/bigclaw/__main__.py`
   - 处理 `src/bigclaw/service.py`
   - 处理 `src/bigclaw/deprecation.py`
   - 处理 `src/bigclaw/legacy_shim.py`

2. `lane/contracts-runtime`
   - 处理 `src/bigclaw/models.py`
   - 处理 `src/bigclaw/execution_contract.py`
   - 处理 `src/bigclaw/observability.py`
   - 处理 `src/bigclaw/queue.py`
   - 处理 `src/bigclaw/scheduler.py`
   - 处理 `src/bigclaw/runtime.py`
   - 处理 `src/bigclaw/workflow.py`
   - 处理 `src/bigclaw/orchestration.py`

3. `lane/migration-cli`
   - 处理 `bigclaw-go/scripts/migration/shadow_compare.py`
   - 处理 `bigclaw-go/scripts/migration/shadow_matrix.py`
   - 处理 `bigclaw-go/scripts/migration/live_shadow_scorecard.py`
   - 处理 `bigclaw-go/scripts/migration/export_live_shadow_bundle.py`

4. `lane/validation-cli`
   - 处理 `bigclaw-go/scripts/e2e/run_all.sh`
   - 处理 `bigclaw-go/scripts/e2e/export_validation_bundle.py`
   - 处理 `bigclaw-go/scripts/e2e/multi_node_shared_queue.py`
   - 同步替换对应 Python 测试为 `internal/regression` Go 测试

5. `lane/control-surface`
   - 处理 `src/bigclaw/connectors.py`
   - 处理 `src/bigclaw/workspace_bootstrap*.py`
   - 处理 `src/bigclaw/governance.py`, `cost_control.py`, `risk.py`, `validation_policy.py`
   - 处理 `src/bigclaw/repo_*.py`, `github_sync.py`, `parallel_refill.py`, `operations.py`, `repo_triage.py`

建议顺序:

1. 先合并 `lane/core-cutover`
2. 再并行 `lane/contracts-runtime` 与 `lane/migration-cli`
3. 紧接 `lane/validation-cli`
4. 最后处理 `lane/control-surface` 与产品/报告面 `P2`

## 验证命令与回归面

迁移过程中每一类资产都要带着回归面移动，不接受“先迁移再补测试”:

| 迁移面 | 命令 |
| --- | --- |
| 入口与 HTTP surface | `cd bigclaw-go && go test ./cmd/bigclawd ./cmd/bigclawctl ./internal/api` |
| 核心域/契约 | `cd bigclaw-go && go test ./internal/domain ./internal/contract ./internal/flow` |
| 控制面内核 | `cd bigclaw-go && go test ./internal/queue ./internal/scheduler ./internal/worker ./internal/executor ./internal/events` |
| 工作流/策略/治理 | `cd bigclaw-go && go test ./internal/workflow ./internal/orchestrator ./internal/policy ./internal/governance ./internal/risk ./internal/billing` |
| intake/repo/refill/bootstrap | `cd bigclaw-go && go test ./internal/intake ./internal/repo ./internal/refill ./internal/bootstrap ./internal/githubsync ./internal/triage` |
| 报告/产品/回归门禁 | `cd bigclaw-go && go test ./internal/reporting ./internal/product ./internal/regression ./internal/prd` |

重点回归面:

- CLI flag 与 operator 入口兼容
- task schema、execution schema、dashboard schema
- queue lease/ack/requeue/dead-letter 语义
- scheduler fairness、quota、policy store
- worker sandbox/tool policy 与 executor adapter
- event replay、delivery、durability、checkpoint
- shadow compare/matrix 与 live shadow bundle 的产物布局
- run-all/export bundle/shared-queue 的报告、索引、证据包格式
- repo/governance/risk/policy 输出的 JSON surface

## 分支与 PR 建议

- 本 issue 建议分支: `codex/big-go-901-migration-inventory`
- 后续实现分支建议按 lane 独立:
  - `codex/big-go-901-core-cutover`
  - `codex/big-go-901-contracts-runtime`
  - `codex/big-go-901-migration-cli`
  - `codex/big-go-901-validation-cli`
  - `codex/big-go-901-control-surface`
- PR 策略:
  - 每个 PR 只跨一个 lane，避免同时改 `cmd`、`internal`、`scripts` 的多个不相关面
  - 每个 PR 要删除对应 Python/shell 入口，而不是只增加 Go 副本
  - 每个 PR 必须附带被替换资产的回归命令结果和产物路径

## 主要风险

| 风险 | 影响 | 缓解方式 |
| --- | --- | --- |
| Python 脚本持续生成 canonical 报告，Go 只能旁路输出 | Go 迟迟不能成为唯一主线 | 优先迁移 `migration/*.py`、`export_validation_bundle.py`、`run_all.sh` |
| 冻结态 Python 模块仍被 operator 或文档引用 | 切换后出现隐性入口断裂 | 先统一 CLI 和 legacy shim，再删 Python 入口 |
| schema 在 Python 与 Go 双写期间漂移 | shadow/report bundle 无法比较 | 契约类资产纳入 `P0`，以 Go schema 为唯一权威 |
| P2 产品面模块过大 | 一次性迁移成本高、PR 过胖 | 先保 API payload parity，再决定是否完整搬运内部实现 |
| shell wrapper 残留 | 本地跑法不一致，review 证据不可重复 | 把所有 wrapper 收敛成 `bigclawctl` 子命令 |

## 全量账本说明

- `bigclaw-go/docs/reports/go-migration-ledger.json` 是 100% 迁移总表。
- `bigclaw-go/docs/reports/go-migration-plan-summary.json` 是面向自动化和 reviewer 的计划摘要，收敛了目标架构、Wave、首批 lane、验证命令与主要风险。
- 每个条目都包含 `asset`、`kind`、`priority`、`target`、`action`、`validation`、`regression`。
- 后续如新增或删除非 Go 可执行资产，必须先更新该账本，再合并实现 PR。
