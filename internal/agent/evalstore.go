// 评分记录存储与聚合分析 —— 独立于对话历史的持久化评分数据库。
// 存储位置：<工作区根>/.Pair/evals/records.json
// 每条记录含：时间戳、任务摘要、4维评分、优缺点、工具调用统计、执行配置快照。
// 聚合分析：按时间/模型统计平均分、识别高频弱点、趋势数据。
package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// ─── 常量 ──────────────────────────────────────────────────

const evalsDirName = ".Pair/evals"

// ─── 数据结构 ──────────────────────────────────────────────

// EvalRecord 一条完整的评分记录（存档在 .Pair/evals/records.json）。
type EvalRecord struct {
	ID            string    `json:"id"`            // 唯一标识 "eval_{seq}"
	Timestamp     time.Time `json:"timestamp"`     // 评分时间
	Task          string    `json:"task"`           // 原始任务（前 200 字符摘要）
	AgentModel    string    `json:"agentModel"`    // 执行模型名
	JudgeModel    string    `json:"judgeModel"`    // 评测模型名
	Scores        EvalScores `json:"scores"`       // 4 维度分
	Total         int       `json:"total"`         // 总分 0-100
	Strengths     []string  `json:"strengths"`     // 优点列表
	Weaknesses    []string  `json:"weaknesses"`    // 不足列表
	Feedback      string    `json:"feedback"`      // 一句话总评
	ToolCalls     int       `json:"toolCalls"`     // 工具调用次数
	ToolErrors    int       `json:"toolErrors"`    // 工具错误次数
	MaxIterations int       `json:"maxIterations"` // 执行时迭代上限
	Autonomous    bool      `json:"autonomous"`    // 是否自主模式
	Duration      string    `json:"duration"`      // 执行耗时（秒）
	Iteration     int       `json:"iteration"`     // 自动迭代轮次 0=首轮 1=第一轮改进 2=第二轮
}

// recordsFile 所有记录的持久化数据结构（JSON 文件格式）。
type recordsFile struct {
	Version int           `json:"version"`
	Seq     int           `json:"seq"`
	Records []*EvalRecord `json:"records"`
}

// ─── 记录存储 ──────────────────────────────────────────────

// EvalRecordStore 评分记录存储，线程安全。
type EvalRecordStore struct {
	mu      sync.Mutex
	seq     int
	records []*EvalRecord
	root    string // 工作区根
	loaded  bool
}

// globalEvalStore 全局单例。
var globalEvalStore *EvalRecordStore
var evalStoreOnce sync.Once

// ResetEvalStoreForTest 重置全局单例（仅测试用），让下一个 GetEvalStore 用新 root 重新加载。
func ResetEvalStoreForTest() {
	evalStoreOnce = sync.Once{}
	globalEvalStore = nil
}

// GetEvalStore 获取全局评分记录存储单例（懒加载）。
func GetEvalStore(root string) *EvalRecordStore {
	evalStoreOnce.Do(func() {
		globalEvalStore = &EvalRecordStore{}
		globalEvalStore.load(root)
	})
	if root != "" && globalEvalStore.root != root {
		globalEvalStore.mu.Lock()
		globalEvalStore.root = root
		globalEvalStore.reload()
		globalEvalStore.mu.Unlock()
	}
	return globalEvalStore
}

// evalsFilePath 返回评分记录 JSON 文件全路径。
func evalsFilePath(root string) string {
	return filepath.Join(root, evalsDirName, "records.json")
}

// load 从磁盘加载已有记录。
func (s *EvalRecordStore) load(root string) {
	s.root = root
	s.records = nil
	s.seq = 0
	if root == "" {
		s.loaded = true
		return
	}
	b, err := os.ReadFile(evalsFilePath(root))
	if err != nil {
		s.loaded = true
		return // 首次使用，无文件
	}
	var data recordsFile
	if err := json.Unmarshal(b, &data); err != nil || data.Version != 1 {
		s.loaded = true
		return
	}
	s.seq = data.Seq
	s.records = data.Records
	if s.records == nil {
		s.records = make([]*EvalRecord, 0)
	}
	s.loaded = true
}

// reload 重新从磁盘加载（感知外部修改）。
func (s *EvalRecordStore) reload() {
	if s.root == "" {
		return
	}
	b, err := os.ReadFile(evalsFilePath(s.root))
	if err != nil {
		return
	}
	var data recordsFile
	if err := json.Unmarshal(b, &data); err != nil || data.Version != 1 {
		return
	}
	s.seq = data.Seq
	s.records = data.Records
	if s.records == nil {
		s.records = make([]*EvalRecord, 0)
	}
}

// save 持久化到磁盘。
func (s *EvalRecordStore) save() {
	if s.root == "" {
		return
	}
	dir := filepath.Join(s.root, evalsDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	data := recordsFile{
		Version: 1,
		Seq:     s.seq,
		Records: s.records,
	}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(evalsFilePath(s.root), b, 0o644)
}

// ─── 写入 ──────────────────────────────────────────────────

// Append 追加一条评分记录并持久化。返回记录 ID。
func (s *EvalRecordStore) Append(rec *EvalRecord) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	rec.ID = fmt.Sprintf("eval_%d", s.seq)
	rec.Timestamp = time.Now()
	// 截断任务摘要（保留前 200 字符）
	if r := []rune(rec.Task); len(r) > 200 {
		rec.Task = string(r[:200]) + "…"
	}
	s.records = append(s.records, rec)
	s.save()
	return rec.ID
}

// ─── 查询 ──────────────────────────────────────────────────

// All 返回所有评分记录的副本（按时间倒序）。
func (s *EvalRecordStore) All() []*EvalRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*EvalRecord, len(s.records))
	copy(out, s.records)
	// 按时间倒序
	sort.Slice(out, func(i, j int) bool {
		return out[i].Timestamp.After(out[j].Timestamp)
	})
	return out
}

// Recent 返回最近 N 条记录（按时间倒序）。
func (s *EvalRecordStore) Recent(n int) []*EvalRecord {
	if n <= 0 {
		n = 10
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	total := len(s.records)
	if total == 0 {
		return nil
	}
	// 已有 records 按存储顺序（时间升序）
	start := total - n
	if start < 0 {
		start = 0
	}
	out := make([]*EvalRecord, 0, n)
	for i := total - 1; i >= start; i-- {
		out = append(out, s.records[i])
	}
	return out
}

// Count 返回记录总数。
func (s *EvalRecordStore) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.records)
}

// MarkIteration 标记最后一条记录的迭代轮次（autoIterate 使用）。
func (s *EvalRecordStore) MarkIteration(iteration int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.records) == 0 {
		return
	}
	s.records[len(s.records)-1].Iteration = iteration
	s.save()
}

// LastRecord 返回最后一条记录（最近一次评分）。
func (s *EvalRecordStore) LastRecord() *EvalRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.records) == 0 {
		return nil
	}
	return s.records[len(s.records)-1]
}

// ─── 聚合分析 ──────────────────────────────────────────────

// EvalStats 聚合统计结果。
type EvalStats struct {
	TotalRecords      int            `json:"totalRecords"`
	AvgTotal          float64        `json:"avgTotal"`
	AvgCompletion     float64        `json:"avgCompletion"`
	AvgCorrectness    float64        `json:"avgCorrectness"`
	AvgDepth          float64        `json:"avgDepth"`
	AvgEfficiency     float64        `json:"avgEfficiency"`
	TopWeaknesses     []*FreqItem    `json:"topWeaknesses"`   // 高频不足
	TopStrengths      []*FreqItem    `json:"topStrengths"`    // 高频优点
	RecentTrend       []*TrendPoint  `json:"recentTrend"`     // 最近 N 条趋势
	ModelStats        []*ModelStat   `json:"modelStats"`       // 按模型统计
	WeakDimension     string         `json:"weakDimension"`    // 最弱维度名
	WeakDimensionAvg  float64        `json:"weakDimensionAvg"` // 最弱维度平均分
}

// FreqItem 频次统计项。
type FreqItem struct {
	Text  string `json:"text"`
	Count int    `json:"count"`
}

// TrendPoint 趋势数据点。
type TrendPoint struct {
	Index int     `json:"index"`
	Total float64 `json:"total"`
}

// ModelStat 按模型聚合的统计数据。
type ModelStat struct {
	Model    string  `json:"model"`
	Count    int     `json:"count"`
	AvgTotal float64 `json:"avgTotal"`
}

// Stats 对全部记录做聚合统计。
func (s *EvalRecordStore) Stats() *EvalStats {
	s.mu.Lock()
	defer s.mu.Unlock()

	n := len(s.records)
	if n == 0 {
		return &EvalStats{}
	}

	st := &EvalStats{TotalRecords: n}

	var sumTotal, sumComp, sumCorr, sumDepth, sumEff float64
	weakFreq := make(map[string]int)
	strengthFreq := make(map[string]int)
	modelMap := make(map[string]*ModelStat)

	for _, r := range s.records {
		sumTotal += float64(r.Total)
		sumComp += float64(r.Scores.Completion)
		sumCorr += float64(r.Scores.Correctness)
		sumDepth += float64(r.Scores.Depth)
		sumEff += float64(r.Scores.Efficiency)

		for _, w := range r.Weaknesses {
			weakFreq[w]++
		}
		for _, st := range r.Strengths {
			strengthFreq[st]++
		}

		// 按模型统计
		model := r.AgentModel
		if model == "" {
			model = "(未知)"
		}
		if ms, ok := modelMap[model]; ok {
			ms.Count++
			ms.AvgTotal += float64(r.Total)
		} else {
			modelMap[model] = &ModelStat{Model: model, Count: 1, AvgTotal: float64(r.Total)}
		}
	}

	nf := float64(n)
	st.AvgTotal = sumTotal / nf
	st.AvgCompletion = sumComp / nf
	st.AvgCorrectness = sumCorr / nf
	st.AvgDepth = sumDepth / nf
	st.AvgEfficiency = sumEff / nf

	// 高频不足（按频次排序取前 10）
	st.TopWeaknesses = topFreqItems(weakFreq, 10)
	st.TopStrengths = topFreqItems(strengthFreq, 10)

	// 按模型统计（计算平均分）
	for _, ms := range modelMap {
		if ms.Count > 0 {
			ms.AvgTotal /= float64(ms.Count)
		}
	}
	st.ModelStats = make([]*ModelStat, 0, len(modelMap))
	for _, ms := range modelMap {
		st.ModelStats = append(st.ModelStats, ms)
	}
	sort.Slice(st.ModelStats, func(i, j int) bool {
		return st.ModelStats[i].AvgTotal > st.ModelStats[j].AvgTotal
	})

	// 最近 20 条趋势
	trendN := 20
	if trendN > n {
		trendN = n
	}
	st.RecentTrend = make([]*TrendPoint, 0, trendN)
	for i := n - trendN; i < n; i++ {
		st.RecentTrend = append(st.RecentTrend, &TrendPoint{
			Index: i,
			Total: float64(s.records[i].Total),
		})
	}

	// 最弱维度
	dims := []struct {
		name string
		avg  float64
	}{
		{"完成度", st.AvgCompletion / 40 * 100},
		{"正确性", st.AvgCorrectness / 30 * 100},
		{"深度", st.AvgDepth / 20 * 100},
		{"效率", st.AvgEfficiency / 10 * 100},
	}
	weakest := dims[0]
	for _, d := range dims[1:] {
		if d.avg < weakest.avg {
			weakest = d
		}
	}
	st.WeakDimension = weakest.name
	st.WeakDimensionAvg = weakest.avg

	return st
}

// ─── 优化建议 ──────────────────────────────────────────────

// OptimizationHint 一条优化建议。
type OptimizationHint struct {
	Dimension string `json:"dimension"` // 建议涉及的维度
	Severity  string `json:"severity"`  // high / medium / low
	Suggestion string `json:"suggestion"` // 建议内容
	Reason    string `json:"reason"`    // 依据（聚合数据说明）
}

// AnalyzeOptimization 基于历史评分记录给出配置优化建议。
func (s *EvalRecordStore) AnalyzeOptimization() []*OptimizationHint {
	st := s.Stats()
	if st.TotalRecords < 2 {
		return nil // 数据不足，无法分析
	}

	hints := make([]*OptimizationHint, 0)

	// 1. 最弱维度建议
	if st.WeakDimensionAvg < 60 {
		var suggestion, reason string
		switch st.WeakDimension {
		case "完成度":
			suggestion = "考虑增加 MaxIterations（当前值可在设置面板增大「最大迭代步数」），或开启自主模式让 Agent 一气呵成完成多步任务"
			reason = fmt.Sprintf("完成度平均 %.0f%%（偏低），可能迭代不足或任务被提前截断", st.WeakDimensionAvg)
		case "正确性":
			suggestion = "考虑降低模型 temperature（当前可在设置面板调整），或启用 AI 审核模式在写操作前由审核模型把关"
			reason = fmt.Sprintf("正确性平均 %.0f%%（偏低），可能输出不够严谨", st.WeakDimensionAvg)
		case "深度":
			suggestion = "考虑在系统指令中增加「深入分析架构与权衡」要求，或手动调整 Agent 的思考深度"
			reason = fmt.Sprintf("深度平均 %.0f%%（偏低），分析可能流于表面", st.WeakDimensionAvg)
		case "效率":
			suggestion = "考虑在系统指令中增加「保持输出简洁，避免冗余」要求"
			reason = fmt.Sprintf("效率平均 %.0f%%（偏低），输出可能冗余", st.WeakDimensionAvg)
		}
		hints = append(hints, &OptimizationHint{
			Dimension:  st.WeakDimension,
			Severity:   "high",
			Suggestion: suggestion,
			Reason:     reason,
		})
	}

	// 2. 高频弱点模式
	if len(st.TopWeaknesses) > 0 {
		top := st.TopWeaknesses[0]
		if top.Count >= st.TotalRecords/3 { // 超过 1/3 的记录都提到同一问题
			hints = append(hints, &OptimizationHint{
				Dimension:  "通用",
				Severity:   "medium",
				Suggestion: "Agent 似乎频繁出现类似不足（\"" + top.Text + "\"），建议在系统指令或设置中的系统级指令中针对性补充约束",
				Reason:     fmt.Sprintf("在 %d 条记录中出现 %d 次（%.0f%%）", st.TotalRecords, top.Count, float64(top.Count)/float64(st.TotalRecords)*100),
			})
		}
	}

	// 3. 模型选择建议
	if len(st.ModelStats) >= 2 {
		best := st.ModelStats[0]
		if best.AvgTotal > st.AvgTotal+5 && best.Count >= 2 {
			hints = append(hints, &OptimizationHint{
				Dimension:  "通用",
				Severity:   "medium",
				Suggestion: fmt.Sprintf("模型「%s」平均分 %.0f，高于整体平均 %.0f，可考虑换用此模型", best.Model, best.AvgTotal, st.AvgTotal),
				Reason:     fmt.Sprintf("对比 %d 个模型，%s 评分最高（%d 条记录）", len(st.ModelStats), best.Model, best.Count),
			})
		}
	}

	// 4. 趋势下降警告
	if len(st.RecentTrend) >= 5 {
		half := len(st.RecentTrend) / 2
		var firstHalf, secondHalf float64
		for i, p := range st.RecentTrend {
			if i < half {
				firstHalf += p.Total
			} else {
				secondHalf += p.Total
			}
		}
		firstHalf /= float64(half)
		secondHalf /= float64(len(st.RecentTrend) - half)
		if secondHalf < firstHalf-10 {
			hints = append(hints, &OptimizationHint{
				Dimension:  "通用",
				Severity:   "high",
				Suggestion: "近期评分呈持续下降趋势，建议检查：模型调参变化、任务复杂度上升、或项目上下文膨胀",
				Reason:     fmt.Sprintf("前 %d 条平均 %.0f，后 %d 条平均 %.0f（下降 %.0f 分）", half, firstHalf, len(st.RecentTrend)-half, secondHalf, firstHalf-secondHalf),
			})
		}
	}

	return hints
}

// ─── 辅助函数 ──────────────────────────────────────────────

// topFreqItems 从频率 map 取前 N 个高频项（按频次降序）。
func topFreqItems(freq map[string]int, n int) []*FreqItem {
	if len(freq) == 0 {
		return nil
	}
	items := make([]*FreqItem, 0, len(freq))
	for text, count := range freq {
		items = append(items, &FreqItem{Text: text, Count: count})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Count > items[j].Count
	})
	if len(items) > n {
		items = items[:n]
	}
	return items
}

// TaskPreview 从完整任务文本提取摘要（前 N 字符 + 截断 3 行）。
func TaskPreview(task string) string {
	lines := strings.SplitN(strings.TrimSpace(task), "\n", 4)
	if len(lines) > 3 {
		lines = lines[:3]
	}
	joined := strings.Join(lines, " | ")
	if r := []rune(joined); len(r) > 200 {
		joined = string(r[:200]) + "…"
	}
	return joined
}
