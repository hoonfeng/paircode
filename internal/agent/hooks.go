package agent

import (
	"context"
	"time"
)

// HookKind 钩子类型。
type HookKind string

const (
	HookBefore HookKind = "before"
	HookAfter  HookKind = "after"
	HookError  HookKind = "error"
)

// Hook 工具钩子：可注册多个，按优先级有序调用。
type Hook struct {
	Name     string   // 唯一标识
	Kind     HookKind // before/after/error
	Priority int      // 越小越先调用（默认 100）
	BeforeFn func(ctx context.Context, name string, args map[string]any) (proceed bool, override string, overrideErr error)
	AfterFn  func(ctx context.Context, name string, args map[string]any, result string, err error, duration time.Duration)
	ErrorFn  func(ctx context.Context, name string, args map[string]any, err error) (result string, replacedErr error)
}

// HookStore 钩子存储：支持 Add/Remove/Get。
type HookStore struct {
	hooks  []*Hook          // 按 Priority 排序
	byName map[string]*Hook // 唯一名索引
}

// NewHookStore 创建钩子存储。
func NewHookStore() *HookStore {
	return &HookStore{byName: map[string]*Hook{}}
}

// Add 添加钩子（同名覆盖）。自动按 Priority 排序。
func (hs *HookStore) Add(h *Hook) {
	if h.Priority == 0 {
		h.Priority = 100
	}
	if old, ok := hs.byName[h.Name]; ok {
		*old = *h
	} else {
		hs.hooks = append(hs.hooks, h)
		hs.byName[h.Name] = h
	}
	// 排序：Priority 升序 → Name 升序
	for i := 0; i < len(hs.hooks)-1; i++ {
		for j := i + 1; j < len(hs.hooks); j++ {
			if hs.hooks[i].Priority > hs.hooks[j].Priority ||
				(hs.hooks[i].Priority == hs.hooks[j].Priority && hs.hooks[i].Name > hs.hooks[j].Name) {
				hs.hooks[i], hs.hooks[j] = hs.hooks[j], hs.hooks[i]
			}
		}
	}
}

// Remove 移除钩子。
func (hs *HookStore) Remove(name string) {
	if _, ok := hs.byName[name]; !ok {
		return
	}
	delete(hs.byName, name)
	filtered := make([]*Hook, 0, len(hs.hooks))
	for _, h := range hs.hooks {
		if h.Name != name {
			filtered = append(filtered, h)
		}
	}
	hs.hooks = filtered
}

// ExecuteBefore 执行所有 before 类钩子。任一返回 proceed=false 则短路。
func (hs *HookStore) ExecuteBefore(ctx context.Context, name string, args map[string]any) (proceed bool, override string, overrideErr error) {
	for _, h := range hs.hooks {
		if h.Kind != HookBefore || h.BeforeFn == nil {
			continue
		}
		if p, ov, oe := h.BeforeFn(ctx, name, args); !p {
			return p, ov, oe
		}
	}
	return true, "", nil
}

// ExecuteAfter 执行所有 after 类钩子。
func (hs *HookStore) ExecuteAfter(ctx context.Context, name string, args map[string]any, result string, err error, duration time.Duration) {
	for _, h := range hs.hooks {
		if h.Kind != HookAfter || h.AfterFn == nil {
			continue
		}
		h.AfterFn(ctx, name, args, result, err, duration)
	}
}

// ExecuteError 执行所有 error 类钩子。返回第一个非 nil 的替换结果。
func (hs *HookStore) ExecuteError(ctx context.Context, name string, args map[string]any, err error) (result string, replacedErr error) {
	for _, h := range hs.hooks {
		if h.Kind != HookError || h.ErrorFn == nil {
			continue
		}
		if r, re := h.ErrorFn(ctx, name, args, err); r != "" || re != nil {
			return r, re
		}
	}
	return "", nil
}
