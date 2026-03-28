package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/scheduler"
	"bigclaw-go/internal/testharness"
)

type plannedIssue struct {
	Title string
	Body  string
}

var createIssuePlans = map[string][]plannedIssue{
	"v1": {
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
	"v2-ops": {
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
}

var createIssueLabels = map[string][]string{
	"v1":     {"bigclaw", "prd-v1"},
	"v2-ops": {"bigclaw", "prd-v2", "ops"},
}

func runDevSmoke(args []string) error {
	flags := flag.NewFlagSet("dev-smoke", flag.ContinueOnError)
	_ = flags.String("repo", "..", "repo root")
	asJSON := flags.Bool("json", false, "json")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl dev-smoke [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	task := domain.Task{
		ID:          "SMOKE-1",
		Source:      "local",
		Title:       "smoke",
		Description: "smoke test",
	}
	decision := scheduler.New().Decide(task, scheduler.QuotaSnapshot{})
	if !decision.Accepted {
		return emit(map[string]any{
			"status":   "error",
			"task_id":  task.ID,
			"accepted": decision.Accepted,
			"reason":   decision.Reason,
		}, *asJSON, 1)
	}
	payload := map[string]any{
		"status":   "ok",
		"task_id":  task.ID,
		"accepted": decision.Accepted,
		"executor": decision.Assignment.Executor,
		"reason":   decision.Reason,
	}
	if *asJSON {
		return emit(payload, true, 0)
	}
	_, err := fmt.Fprintf(os.Stdout, "smoke_ok %s\n", decision.Assignment.Executor)
	return err
}

func runPytestHarness(args []string) error {
	flags := flag.NewFlagSet("pytest-harness", flag.ContinueOnError)
	projectRoot := flags.String("project-root", "..", "project root containing tests/ and src/")
	asJSON := flags.Bool("json", false, "json")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl pytest-harness [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}

	resolvedProjectRoot := absPath(*projectRoot)
	inventory, err := testharness.InventoryPytestAssetsAt(resolvedProjectRoot)
	if err != nil {
		return fmt.Errorf("inventory pytest harness assets: %w", err)
	}
	deleteStatus := inventory.ConftestDeletionStatus()

	if *asJSON {
		payload := map[string]any{
			"status":                   "ok",
			"project_root":             resolvedProjectRoot,
			"inventory_summary":        inventory.Summary(),
			"test_modules":             inventory.TestModules,
			"bigclaw_imports":          inventory.BigclawImportModules,
			"pytest_imports":           inventory.PytestImportModules,
			"conftest_path":            inventory.ConftestPath,
			"conftest_prepends_src":    inventory.ConftestPrependsSrc,
			"conftest_imports_pytest":  inventory.ConftestImportsPytest,
			"conftest_defines_fixture": inventory.ConftestDefinesFixture,
			"conftest_defines_hook":    inventory.ConftestDefinesHook,
			"conftest_delete_status":   structToMap(deleteStatus),
		}
		return emit(payload, true, 0)
	}

	_, err = fmt.Fprintf(
		os.Stdout,
		"project_root=%s\ninventory=%s\nconftest_path=%s\nconftest_prepends_src=%t\nconftest_imports_pytest=%t\nconftest_defines_fixture=%t\nconftest_defines_hook=%t\nconftest_delete_ready=%t\nconftest_delete_summary=%s\n",
		resolvedProjectRoot,
		inventory.Summary(),
		inventory.ConftestPath,
		inventory.ConftestPrependsSrc,
		inventory.ConftestImportsPytest,
		inventory.ConftestDefinesFixture,
		inventory.ConftestDefinesHook,
		deleteStatus.CanDelete,
		deleteStatus.Summary,
	)
	if err != nil {
		return err
	}
	for _, blocker := range deleteStatus.Blockers {
		if _, err := fmt.Fprintf(os.Stdout, "blocker=%s\n", blocker); err != nil {
			return err
		}
	}
	return nil
}

func runCreateIssues(args []string) error {
	flags := flag.NewFlagSet("create-issues", flag.ContinueOnError)
	_ = flags.String("repo", "..", "repo root")
	owner := flags.String("owner", "OpenAGIs", "repository owner")
	repoName := flags.String("repo-name", "BigClaw", "repository name")
	planName := flags.String("plan", envOrDefault("BIGCLAW_PLAN", "v1"), "issue plan")
	apiBase := flags.String("api-base", "https://api.github.com", "GitHub API base URL")
	token := flags.String("token", os.Getenv("GITHUB_TOKEN"), "GitHub token")
	asJSON := flags.Bool("json", false, "json")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl create-issues [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	issues, ok := createIssuePlans[*planName]
	if !ok {
		available := make([]string, 0, len(createIssuePlans))
		for name := range createIssuePlans {
			available = append(available, name)
		}
		return fmt.Errorf("unknown plan %q. available plans: %s", *planName, strings.Join(available, ", "))
	}
	if trim(*token) == "" {
		return errors.New("GITHUB_TOKEN is required")
	}
	client := &githubIssueClient{
		baseURL:    strings.TrimRight(trim(*apiBase), "/"),
		token:      trim(*token),
		httpClient: http.DefaultClient,
	}
	existing, err := client.listIssueTitles(trim(*owner), trim(*repoName))
	if err != nil {
		return err
	}
	created := make([]map[string]any, 0, len(issues))
	for _, issue := range issues {
		if _, found := existing[issue.Title]; found {
			continue
		}
		number, title, err := client.createIssue(trim(*owner), trim(*repoName), issue.Title, issue.Body, createIssueLabels[*planName])
		if err != nil {
			return err
		}
		created = append(created, map[string]any{"number": number, "title": title})
	}
	return emit(map[string]any{
		"status":        "ok",
		"owner":         trim(*owner),
		"repo":          trim(*repoName),
		"plan":          *planName,
		"created_count": len(created),
		"created":       created,
	}, *asJSON, 0)
}

func runSymphony(args []string) error {
	flags := flag.NewFlagSet("symphony", flag.ContinueOnError)
	repoRoot := flags.String("repo", "..", "repo root")
	workflowPath := flags.String("workflow", "", "workflow path")
	port := flags.Int("port", 4000, "dashboard port")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl symphony [flags] [args...]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	resolvedRepoRoot := absPath(*repoRoot)
	workflow := trim(*workflowPath)
	if workflow == "" {
		workflow = filepath.Join(resolvedRepoRoot, "workflow.md")
	}
	commandArgs := append([]string{
		"--i-understand-that-this-will-be-running-without-the-usual-guardrails",
		"--port", fmt.Sprintf("%d", *port),
		workflow,
	}, flags.Args()...)
	return runSymphonyCommand(resolvedRepoRoot, commandArgs)
}

func runIssue(args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "list", "create", "state", "comment":
			return runIssueLocalTrackerCommand(args)
		}
	}
	return runSymphonyWorkflowTool("issue", args)
}

func runPanel(args []string) error {
	return runSymphonyWorkflowTool("panel", args)
}

func runIssueLocalTrackerCommand(args []string) error {
	repoRoot := defaultRepoRoot()
	localIssues := filepath.Join(repoRoot, "local-issues.json")
	translated := append([]string{}, args...)
	switch args[0] {
	case "state":
		if !containsFlag(args, "--issue") && !containsFlag(args, "--state") && len(args) >= 3 {
			translated = append([]string{"set-state", "--issue", args[1], "--state", args[2]}, args[3:]...)
		}
	case "comment":
		if !containsFlag(args, "--issue") && !containsFlag(args, "--body") && !containsFlag(args, "--body-file") && len(args) >= 3 {
			translated = append([]string{"comment", "--issue", args[1], "--body", args[2]}, args[3:]...)
		}
	}
	hasLocalIssues := containsFlag(translated, "--local-issues")
	hasRepo := containsFlag(translated, "--repo")
	if !hasRepo {
		translated = append(translated, "--repo", repoRoot)
	}
	if !hasLocalIssues {
		translated = append(translated, "--local-issues", localIssues)
	}
	if len(translated) > 0 && translated[0] == "state" {
		translated[0] = "set-state"
	}
	return runLocalIssues(translated)
}

func runSymphonyWorkflowTool(tool string, args []string) error {
	flags := flag.NewFlagSet(tool, flag.ContinueOnError)
	repoRoot := flags.String("repo", "..", "repo root")
	workflowPath := flags.String("workflow", "", "workflow path")
	if helpText, err := parseFlagsWithHelp(flags, fmt.Sprintf("usage: bigclawctl %s [flags] [args...]", tool), args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	resolvedRepoRoot := absPath(*repoRoot)
	workflow := trim(*workflowPath)
	if workflow == "" {
		workflow = filepath.Join(resolvedRepoRoot, "workflow.md")
	}
	commandArgs := append([]string{tool, "--workflow", workflow}, flags.Args()...)
	return runSymphonyCommand(resolvedRepoRoot, commandArgs)
}

func runSymphonyCommand(repoRoot string, args []string) error {
	commandPath, err := resolveSymphonyBinary(repoRoot)
	if err != nil {
		return err
	}
	cmd := exec.Command(commandPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), symphonyBootstrapEnv(repoRoot)...)
	return cmd.Run()
}

func symphonyBootstrapEnv(repoRoot string) []string {
	branch := gitCurrentBranch(repoRoot)
	return []string{
		fmt.Sprintf("SYMPHONY_BOOTSTRAP_REPO_URL=%s", envOrDefault("SYMPHONY_BOOTSTRAP_REPO_URL", repoRoot)),
		fmt.Sprintf("SYMPHONY_BOOTSTRAP_DEFAULT_BRANCH=%s", envOrDefault("SYMPHONY_BOOTSTRAP_DEFAULT_BRANCH", branch)),
	}
}

func gitCurrentBranch(repoRoot string) string {
	cmd := exec.Command("git", "-C", repoRoot, "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil {
		return "main"
	}
	branch := trim(string(output))
	if branch == "" {
		return "main"
	}
	return branch
}

func resolveSymphonyBinary(repoRoot string) (string, error) {
	localPath := filepath.Clean(filepath.Join(repoRoot, "..", "elixir", "bin", "symphony"))
	if stat, err := os.Stat(localPath); err == nil && stat.Mode()&0o111 != 0 {
		return localPath, nil
	}
	commandPath, err := exec.LookPath("symphony")
	if err == nil {
		return commandPath, nil
	}
	return "", errors.New("symphony CLI not found. Install Symphony or run from a checkout that contains ../elixir/bin/symphony")
}

func defaultRepoRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return ".."
	}
	return wd
}

func containsFlag(args []string, name string) bool {
	for _, arg := range args {
		if arg == name || strings.HasPrefix(arg, name+"=") {
			return true
		}
	}
	return false
}

func envOrDefault(key string, fallback string) string {
	value := trim(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

type githubIssueClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func (c *githubIssueClient) do(req *http.Request, target any) error {
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("User-Agent", "bigclaw-bootstrap")
	response, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		return fmt.Errorf("GitHub HTTP %d", response.StatusCode)
	}
	if target == nil {
		return nil
	}
	return json.NewDecoder(response.Body).Decode(target)
}

func (c *githubIssueClient) listIssueTitles(owner string, repo string) (map[string]struct{}, error) {
	request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/repos/%s/%s/issues?state=all&per_page=100", c.baseURL, owner, repo), nil)
	if err != nil {
		return nil, err
	}
	var issues []struct {
		Title string `json:"title"`
	}
	if err := c.do(request, &issues); err != nil {
		return nil, err
	}
	result := make(map[string]struct{}, len(issues))
	for _, issue := range issues {
		result[issue.Title] = struct{}{}
	}
	return result, nil
}

func (c *githubIssueClient) createIssue(owner string, repo string, title string, body string, labels []string) (int, string, error) {
	payload := map[string]any{
		"title":  title,
		"body":   body,
		"labels": labels,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return 0, "", err
	}
	request, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/repos/%s/%s/issues", c.baseURL, owner, repo), strings.NewReader(string(encoded)))
	if err != nil {
		return 0, "", err
	}
	request.Header.Set("Content-Type", "application/json")
	var created struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
	}
	if err := c.do(request, &created); err != nil {
		return 0, "", err
	}
	return created.Number, created.Title, nil
}
