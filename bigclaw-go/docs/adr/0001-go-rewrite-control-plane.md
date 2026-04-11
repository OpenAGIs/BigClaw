# ADR 0001: BigClaw Go 重写的控制平面边界

- 状态: Accepted (bootstrap baseline)
- 日期: 2026-03-12
- 关联事项: `BIG-GO-001`, `BIG-GO-002`, `BIG-GO-003`, `BIG-GO-004`

## 背景

BigClaw 需要从文档规划走向一个可支撑大规模并发执行的平台内核。
为了更自然地对接 `Kubernetes`、`Ray`、事件驱动状态同步和无状态 worker 扩缩容，
控制平面采用 `Go` 进行重写。

重写目标不是简单做语言替换，而是明确控制平面职责：

- 接入统一任务协议
- 维护调度状态机
- 实现队列、lease、重试、死信语义
- 将任务路由到不同 executor
- 输出可审计、可观测的运行事件

## 决策

BigClaw Go 按以下模块边界组织：

1. `domain`
   - 统一任务协议
   - run attempt
   - 状态机与事件模型
2. `queue`
   - 队列抽象
   - lease/ack/requeue/dead-letter 语义
   - 后续可接数据库或消息系统
3. `scheduler`
   - 调度循环
   - 预算、风险、配额、能力路由
4. `executor`
   - executor 抽象
   - `kubernetes` / `ray` / local adapter 的统一能力模型
5. `config`
   - 类型化配置和默认值
6. `api`
   - 健康检查、调试接口、后续控制平面 API

## 不在本阶段解决

- 生产级持久化存储实现
- 完整 Kubernetes / Ray adapter
- 多租户鉴权
- UI / Dashboard

## 迁移策略

第一阶段先冻结协议和模块边界，建立可运行骨架；
第二阶段补齐持久化、调度循环与 executor adapter；
第三阶段进行灰度迁移和 benchmark 对比。
