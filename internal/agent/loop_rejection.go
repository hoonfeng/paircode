package agent

import (
	"errors"
	"fmt"
)

// rejectTrack 工具驳回追踪器：连续驳回保护。
type rejectTrack struct {
	count int    // 连续驳回次数
	last  string // 上次被驳回的工具名
}

// record 记录一次驳回，返回 true 表示已达连续驳回上限（3 次）。
func (rt *rejectTrack) record(name string) (stop bool, reason string) {
	if name == rt.last {
		rt.count++
	} else {
		rt.count = 1
		rt.last = name
	}
	if rt.count >= 3 {
		return true, fmt.Sprintf("操作 %s 已被连续驳回 %d 次，自动停止", name, rt.count)
	}
	return false, ""
}

// resetIf 如果本次通过的工具正是上次被驳回的，重置计数。
func (rt *rejectTrack) resetIf(name string) {
	if name == rt.last {
		rt.count = 0
		rt.last = ""
	}
}

// ErrMaxRejections 连续驳回上限错误。
var ErrMaxRejections = errors.New("连续驳回 3 次，自动停止")
