package engine

import "fmt"

// Status 是单个检查项的状态。
type Status int

const (
	StatusOK      Status = iota // 已就绪,无需动作
	StatusFixable               // 缺失/不正确,但可在用户态自动修复
	StatusFailed                // 修复或校验失败
)

func (s Status) String() string {
	switch s {
	case StatusOK:
		return "OK"
	case StatusFixable:
		return "需修复"
	default:
		return "失败"
	}
}

// Check 是「环境医生」的核心抽象:每个开发依赖/权限/配置都是一个 Check。
// 三段式:Detect(诊断) → Fix(用户态修复) → Verify(校验)。
// 实现必须「幂等」:重复运行、上次中断后再跑,都能从当前状态收敛到 OK。
type Check interface {
	ID() string
	Name() string
	NeedsAdmin() bool // 几乎所有 Check 都应为 false;true 表示需要一次系统提权
	Detect(ctx *Context) (Status, string)
	Fix(ctx *Context) error
	Verify(ctx *Context) error
}

// Result 记录一个 Check 的最终结果。
type Result struct {
	ID      string
	Name    string
	Status  Status
	Message string
	Err     error
}

// DefaultChecks 返回标准体检清单。CLI 与 GUI 共用同一份,保证两端行为一致。
// 顺序很重要:先装 Node,再配 npm/PATH。
// 注意:settings.json(用哪个模型/Key)由用户「导入配置」单独处理,不在体检流程里。
func DefaultChecks() []Check {
	return []Check{
		NodeCheck{}, // Version 留空 = 自动跟随 npmmirror 最新 LTS(解析失败回退兜底版本)
		ClaudeCheck{},
		NpmPrefixCheck{},
		PathCheck{},
	}
}

// Doctor 按顺序执行一组 Check。
type Doctor struct {
	Checks []Check
	// Log 是可选的进度回调:CLI 留空时输出到 stdout,GUI 可注入回调把每行进度推到前端。
	// 仅改变「输出去向」,不影响任何检测/修复判定。
	Log func(line string)
}

// logf 把一行进度交给回调;未设置回调时退回 stdout(保持 CLI 原行为)。
func (d *Doctor) logf(format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	if d.Log != nil {
		d.Log(msg)
		return
	}
	fmt.Print(msg)
}

// Run 执行全部检查并返回结果,过程中实时打印进度。
func (d *Doctor) Run(ctx *Context) []Result {
	var results []Result
	for _, ch := range d.Checks {
		r := Result{ID: ch.ID(), Name: ch.Name()}

		st, msg := ch.Detect(ctx)
		if st == StatusOK {
			r.Status, r.Message = StatusOK, msg
			d.logf("  [就绪] %s — %s\n", ch.Name(), msg)
			results = append(results, r)
			continue
		}

		d.logf("  [修复] %s — %s\n", ch.Name(), msg)
		if err := ch.Fix(ctx); err != nil {
			r.Status, r.Err, r.Message = StatusFailed, err, "修复失败: "+err.Error()
			d.logf("  [失败] %s — %v\n", ch.Name(), err)
			results = append(results, r)
			continue
		}

		if err := ch.Verify(ctx); err != nil {
			r.Status, r.Err, r.Message = StatusFailed, err, "校验失败: "+err.Error()
			d.logf("  [失败] %s 校验未通过 — %v\n", ch.Name(), err)
		} else {
			r.Status, r.Message = StatusOK, "已修复"
			d.logf("  [完成] %s 已修复\n", ch.Name())
		}
		results = append(results, r)
	}
	return results
}
