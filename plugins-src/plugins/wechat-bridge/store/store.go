// Package store 提供 JSON 文件的原子读写。
//
// 原子写 = 先写 <file>.tmp 再 rename 替换；Windows 下目标存在时 rename 可能失败，
// 按 M1 store.mjs 的策略降级：删目标后重试 rename。
package store

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// WriteJSONAtomic 原子写 JSON（自动创建父目录，缩进 2 空格）。
func WriteJSONAtomic(file string, v any) error {
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := file + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, file); err != nil {
		// Windows：目标存在时 rename 失败 → 删目标重试（M1 同款兜底）
		_ = os.Remove(file)
		if err := os.Rename(tmp, file); err != nil {
			_ = os.Remove(tmp)
			return err
		}
	}
	return nil
}

// ReadJSON 读文件并反序列化到 v；文件不存在/损坏返回 false（与 M1 readJson 语义一致）。
func ReadJSON(file string, v any) bool {
	data, err := os.ReadFile(file)
	if err != nil {
		return false
	}
	return json.Unmarshal(data, v) == nil
}
