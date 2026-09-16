// instance.go — 单实例锁与本地端口工具。
//
// 单实例模型：
//   - 启动时读数据目录 bridge.lock，若场上有健康实例（/health 探活通过）→ 报告复用；
//   - 否则写锁继续运行；退出时仅删除属于本进程的锁（防误删新实例的锁）。
//
// 端口监听与自动避让由 localhttp.ListenLocal 提供（见 localhttp 包）。
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/hoonfeng/paircode/plugins-src/plugins/wechat-bridge/config"
	"github.com/hoonfeng/paircode/plugins-src/plugins/wechat-bridge/store"
)

// lockInfo 实例锁内容（bridge.lock）。
type lockInfo struct {
	Pid       int      `json:"pid"`
	Port      int      `json:"port"`
	StartedAt string   `json:"startedAt"`
	Accounts  []string `json:"accounts,omitempty"`
}

// probeExistingInstance 读实例锁并向 /health 探活。
// 返回 (锁信息, true) 表示已有健康实例在运行（本进程应退出并报告复用）。
func probeExistingInstance(cfg config.Config) (lockInfo, bool) {
	var li lockInfo
	if !store.ReadJSON(cfg.LockPath(), &li) || li.Port <= 0 {
		return lockInfo{}, false
	}
	client := &http.Client{Timeout: 800 * time.Millisecond}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/health", li.Port))
	if err != nil {
		return lockInfo{}, false // 端口无响应：陈旧锁，接管
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	var h struct {
		OK  bool `json:"ok"`
		Pid int  `json:"pid"`
	}
	if resp.StatusCode != 200 || json.Unmarshal(body, &h) != nil || !h.OK {
		return lockInfo{}, false
	}
	li.Pid = h.Pid
	return li, true
}

// writeLock 写本实例锁。
func writeLock(cfg config.Config) error {
	return store.WriteJSONAtomic(cfg.LockPath(), lockInfo{
		Pid:       os.Getpid(),
		Port:      cfg.Port,
		StartedAt: time.Now().UTC().Format(time.RFC3339),
	})
}

// updateLockAccounts 刷新锁中账号列表（账号增删后调用；失败静默）。
func updateLockAccounts(cfg config.Config, accounts []string) {
	var li lockInfo
	if !store.ReadJSON(cfg.LockPath(), &li) || li.Pid != os.Getpid() {
		return
	}
	li.Accounts = accounts
	_ = store.WriteJSONAtomic(cfg.LockPath(), li)
}

// removeLock 退出时删除本进程的锁（pid 不匹配则不动，防误删新实例）。
func removeLock(cfg config.Config) {
	var li lockInfo
	if store.ReadJSON(cfg.LockPath(), &li) && li.Pid != os.Getpid() {
		return
	}
	_ = os.Remove(cfg.LockPath())
}
