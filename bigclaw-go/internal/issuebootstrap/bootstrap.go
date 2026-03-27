package issuebootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type IssueSpec struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type Plan struct {
	Issues []IssueSpec `json:"issues"`
	Labels []string    `json:"labels"`
}

type CreatedIssue struct {
	Number int    `json:"number,omitempty"`
	Title  string `json:"title"`
}

type SyncOptions struct {
	Owner      string
	Repo       string
	PlanName   string
	APIBaseURL string
	Token      string
	DryRun     bool
	HTTPClient *http.Client
}

type SyncReport struct {
	Plan         string         `json:"plan"`
	Owner        string         `json:"owner"`
	Repo         string         `json:"repo"`
	DryRun       bool           `json:"dry_run"`
	Labels       []string       `json:"labels"`
	Existing     int            `json:"existing_count"`
	Skipped      int            `json:"skipped_count"`
	CreatedCount int            `json:"created_count"`
	Created      []CreatedIssue `json:"created"`
}

func DefaultPlans() map[string]Plan {
	return map[string]Plan{
		"v1": {
			Labels: []string{"bigclaw", "prd-v1"},
			Issues: []IssueSpec{
				{Title: "[EPIC] BIG-EPIC-1 任务接入与连接器", Body: "PRD Epic: 任务接入与连接器"},
				{Title: "[EPIC] BIG-EPIC-2 调度与执行控制平面", Body: "PRD Epic: 调度与执行控制平面"},
				{Title: "[EPIC] BIG-EPIC-3 执行层与沙箱", Body: "PRD Epic: 执行层与沙箱"},
				{Title: "[EPIC] BIG-EPIC-4 工作流与验收闭环", Body: "PRD Epic: 工作流与验收闭环"},
				{Title: "[EPIC] BIG-EPIC-5 记忆、日志、审计与可观测", Body: "PRD Epic: 记忆、日志、审计与可观测"},
				{Title: "[EPIC] BIG-EPIC-6 评测、验证与效果报告", Body: "PRD Epic: 评测、验证与效果报告"},
				{Title: "[EPIC] BIG-EPIC-7 试点落地与商业验证", Body: "PRD Epic: 试点落地与商业验证"},
				{Title: "BIG-101 任务入口与工单接入", Body: "P0\n\n验收:\n- 可从真实项目拉取 issue\n- 可回写 issue/PR\n- 失败回写有错误报告"},
				{Title: "BIG-102 统一任务模型", Body: "P0\n\n验收:\n- GitHub/Linear/Jira 数据可映射统一模型\n- 模型可驱动调度执行回写"},
				{Title: "BIG-201 持久化任务队列", Body: "P0\n\n验收:\n- 重启恢复\n- 重试和死信正确"},
				{Title: "BIG-202 Agent Scheduler", Body: "P0\n\n验收:\n- 低风险->Docker\n- 高风险->VM/审批\n- 超预算暂停/降级"},
				{Title: "BIG-203 Event Bus", Body: "P1\n\n验收:\n- PR评论/CI完成/任务失败触发状态流转"},
				{Title: "BIG-301 Claw Worker Runtime", Body: "P0\n\n验收:\n- worker 生命周期稳定\n- 任务结果可完整回传"},
				{Title: "BIG-302 Sandbox 分级执行", Body: "P0\n\n验收:\n- 风险等级到执行介质路由正确"},
				{Title: "BIG-303 Tool Runtime / MCP Gateway", Body: "P0\n\n验收:\n- 工具权限可控\n- 调用可审计"},
				{Title: "BIG-401 Workflow DSL", Body: "P0\n\n验收:\n- 可表达研发闭环与运维runbook"},
				{Title: "BIG-402 Workpad / Execution Journal", Body: "P0\n\n验收:\n- 每任务完整journal,可回放"},
				{Title: "BIG-403 验收门禁与审批流", Body: "P0\n\n验收:\n- 未满足gate不得完成"},
				{Title: "BIG-501 记忆系统", Body: "P1\n\n验收:\n- 可复用历史方案,自动注入项目规则"},
				{Title: "BIG-502 可观测与审计", Body: "P0\n\n验收:\n- 任一失败任务可追踪根因\n- 任一写操作有审计"},
				{Title: "BIG-503 成本与预算控制", Body: "P1\n\n验收:\n- 超预算自动降级/暂停"},
				{Title: "BIG-601 评测框架", Body: "P0\n\n验收:\n- 核心能力有基准评测,版本可对比"},
				{Title: "BIG-602 测试与效果验证报告规范", Body: "P0\n\n验收:\n- 无报告不得关闭issue"},
				{Title: "BIG-701 试点客户实施", Body: "P0\n\n验收:\n- 形成真实生产数据与试点复盘"},
			},
		},
		"v2-ops": {
			Labels: []string{"bigclaw", "prd-v2", "ops"},
			Issues: []IssueSpec{
				{Title: "[EPIC] BIG-EPIC-9 工程运营系统", Body: "Epic for engineering operations system."},
				{Title: "BIG-801 团队级执行总览 Dashboard", Body: "P0\n\n验收:\n- 按团队查看执行中的任务、通过率、阻塞态\n- 能从总览跳转到单任务/单次运行详情"},
				{Title: "BIG-802 队列与调度控制中心", Body: "P0\n\n验收:\n- 展示队列深度、调度决策、执行介质分布\n- 可识别待审批、失败、重试任务"},
				{Title: "BIG-803 Premium Orchestration Policy", Body: "P0\n\n验收:\n- 支持 Premium 编排策略和能力边界配置\n- 策略变更有审计记录"},
				{Title: "BIG-804 Run Detail 与执行回放页", Body: "P0\n\n验收:\n- 可查看单次运行日志、trace、artifact、audit\n- 支持按 run_id 回放执行证据"},
				{Title: "BIG-805 团队协作与人工接管", Body: "P1\n\n验收:\n- 支持人工接管、转派、恢复执行\n- 交接过程保留 journal 与审批痕迹"},
				{Title: "BIG-901 SLA 与运营看板", Body: "P0\n\n验收:\n- 展示 SLA、吞吐、失败率、审批等待时长\n- 支持按团队/周期聚合"},
				{Title: "BIG-902 风险评分系统", Body: "P0\n\n验收:\n- 对任务和运行结果输出风险评分\n- 风险分可驱动审批与调度策略"},
				{Title: "BIG-903 自动 Triage Center", Body: "P0\n\n验收:\n- 自动聚合同类失败与阻塞原因\n- 支持建议下一步处理动作"},
				{Title: "BIG-904 回归分析中心", Body: "P0\n\n验收:\n- 对版本、基线、回放结果做回归比较\n- 能定位退化任务与指标"},
				{Title: "BIG-905 工程运营周报自动生成", Body: "P1\n\n验收:\n- 自动生成周报摘要、风险项、改进建议\n- 可复用验证报告和运营指标"},
			},
		},
	}
}

func Sync(ctx context.Context, options SyncOptions) (SyncReport, error) {
	plans := DefaultPlans()
	plan, ok := plans[strings.TrimSpace(options.PlanName)]
	if !ok {
		return SyncReport{}, fmt.Errorf("unknown plan %q", options.PlanName)
	}
	if strings.TrimSpace(options.Owner) == "" || strings.TrimSpace(options.Repo) == "" {
		return SyncReport{}, errors.New("owner and repo are required")
	}
	baseURL := strings.TrimRight(strings.TrimSpace(options.APIBaseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.github.com"
	}
	client := options.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	report := SyncReport{
		Plan:    options.PlanName,
		Owner:   options.Owner,
		Repo:    options.Repo,
		DryRun:  options.DryRun,
		Labels:  append([]string{}, plan.Labels...),
		Created: []CreatedIssue{},
	}
	if options.DryRun {
		for _, issue := range plan.Issues {
			report.Created = append(report.Created, CreatedIssue{Title: issue.Title})
			report.CreatedCount++
		}
		return report, nil
	}
	existingIssues, err := fetchIssues(ctx, client, baseURL, options.Owner, options.Repo, options.Token)
	if err != nil {
		return SyncReport{}, err
	}
	existingTitles := map[string]struct{}{}
	for _, issue := range existingIssues {
		existingTitles[issue.Title] = struct{}{}
	}
	report.Existing = len(existingIssues)
	for _, issue := range plan.Issues {
		if _, found := existingTitles[issue.Title]; found {
			report.Skipped++
			continue
		}
		created, err := createIssue(ctx, client, baseURL, options.Owner, options.Repo, options.Token, issue, plan.Labels)
		if err != nil {
			return SyncReport{}, err
		}
		report.Created = append(report.Created, created)
		report.CreatedCount++
	}
	return report, nil
}

func fetchIssues(ctx context.Context, client *http.Client, baseURL string, owner string, repo string, token string) ([]CreatedIssue, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/repos/%s/%s/issues?state=all&per_page=100", baseURL, owner, repo), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "bigclaw-bootstrap")
	if strings.TrimSpace(token) != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		body, _ := io.ReadAll(response.Body)
		return nil, fmt.Errorf("github issues list failed: %s", strings.TrimSpace(string(body)))
	}
	var issues []struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
	}
	if err := json.NewDecoder(response.Body).Decode(&issues); err != nil {
		return nil, err
	}
	out := make([]CreatedIssue, 0, len(issues))
	for _, issue := range issues {
		out = append(out, CreatedIssue{Number: issue.Number, Title: issue.Title})
	}
	return out, nil
}

func createIssue(ctx context.Context, client *http.Client, baseURL string, owner string, repo string, token string, issue IssueSpec, labels []string) (CreatedIssue, error) {
	payload := map[string]any{
		"title":  issue.Title,
		"body":   issue.Body,
		"labels": labels,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return CreatedIssue{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/repos/%s/%s/issues", baseURL, owner, repo), bytes.NewReader(body))
	if err != nil {
		return CreatedIssue{}, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "bigclaw-bootstrap")
	if strings.TrimSpace(token) != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := client.Do(request)
	if err != nil {
		return CreatedIssue{}, err
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		raw, _ := io.ReadAll(response.Body)
		return CreatedIssue{}, fmt.Errorf("github issue create failed: %s", strings.TrimSpace(string(raw)))
	}
	var created struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
	}
	if err := json.NewDecoder(response.Body).Decode(&created); err != nil {
		return CreatedIssue{}, err
	}
	return CreatedIssue{Number: created.Number, Title: created.Title}, nil
}
