// download.go — 资产下载：多端点降级、断点续传、流式 sha256、停滞检测、进度回调。
package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

const (
	// chunkSize 读缓冲 / 进度回调粒度
	chunkSize = 256 << 10
	// stallTimeout 无数据增长的停滞判定（超时即中止）
	stallTimeout = 60 * time.Second
	// progressInterval 进度回调最小间隔（防高频回调拖慢前端轮询）
	progressInterval = 300 * time.Millisecond
)

// DownloadResult 下载结果。
type DownloadResult struct {
	Path    string // 最终 zip 路径（<DownloadDir>/<asset.Name>）
	Bytes   int64  // 实际字节数
	SHA256  string // 实际 sha256（十六进制小写）
	Resumed bool   // 是否走了断点续传
	URL     string // 实际成功的下载端点
	Reused  bool   // 复用上次已下载文件（未重新下载）
}

// Download 下载并校验本平台更新包。
//
// 行为：
//   - 已存在且校验通过的成品 zip → 直接复用（Reused=true）；
//   - 存在 .part → 带 Range 续传（服务不支持 Range 则从头重下）；
//   - 多端点按 downloadURLs 顺序降级尝试；
//   - 完成后按 RequireSHA256 强校验，不通过则删除 .part 并报错；
//   - onProgress 每 ~256KB / 300ms 回调一次（可为 nil）。
func (c Config) Download(ctx context.Context, a AssetRef, onProgress func(downloaded int64)) (DownloadResult, error) {
	if strings.TrimSpace(a.Name) == "" {
		return DownloadResult{}, fmt.Errorf("资产名为空")
	}
	if err := os.MkdirAll(c.DownloadDir, 0o755); err != nil {
		return DownloadResult{}, fmt.Errorf("创建下载目录失败: %w", err)
	}
	final := filepath.Join(c.DownloadDir, filepath.Base(a.Name))
	part := final + ".part"

	// ① 复用：已存在且校验通过（或非强制校验且大小一致）
	if st, err := os.Stat(final); err == nil && st.Size() > 0 {
		sum, herr := fileSHA256(final)
		if herr == nil {
			if verr := c.verify(a, st.Size(), sum); verr == nil {
				return DownloadResult{Path: final, Bytes: st.Size(), SHA256: sum, URL: "(cached)", Reused: true}, nil
			}
		}
		// 复用失败（校验不过）→ 删除后重下
		_ = os.Remove(final)
	}

	urls := c.downloadURLs(a)
	if len(urls) == 0 {
		return DownloadResult{}, fmt.Errorf("资产无可下载地址（URL 为空）")
	}

	var lastErr error
	for _, u := range urls {
		res, err := c.downloadFrom(ctx, u, a, part, final, onProgress)
		if err == nil {
			return res, nil
		}
		lastErr = err
		if ctx.Err() != nil { // 用户取消 / 停滞中止：不再尝试其它端点
			break
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("下载失败")
	}
	return DownloadResult{}, fmt.Errorf("下载失败（已尝试 %d 个端点）: %w", len(urls), lastErr)
}

// downloadFrom 从单个端点下载（含续传）。
func (c Config) downloadFrom(ctx context.Context, u string, a AssetRef, part, final string, onProgress func(int64)) (DownloadResult, error) {
	offset := int64(0)
	if st, err := os.Stat(part); err == nil && st.Size() > 0 {
		offset = st.Size()
	}
	if a.Size > 0 && offset >= a.Size {
		// .part 已达完整大小：直接进入校验分支
		offset = 0
		_ = os.Remove(part)
	}

	dctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var stalled atomic.Bool
	lastByte := atomic.Int64{}
	lastByte.Store(time.Now().UnixNano())
	stopWatch := make(chan struct{})
	go func() { // 停滞看门狗：60s 无字节增长 → 中止
		t := time.NewTicker(5 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-stopWatch:
				return
			case <-dctx.Done():
				return
			case <-t.C:
				if time.Since(time.Unix(0, lastByte.Load())) > stallTimeout {
					stalled.Store(true)
					cancel()
					return
				}
			}
		}
	}()
	defer close(stopWatch)

	req, err := http.NewRequestWithContext(dctx, "GET", u, nil)
	if err != nil {
		return DownloadResult{}, fmt.Errorf("构造下载请求失败: %w", err)
	}
	c.decorate(req, true)
	if offset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return DownloadResult{}, fmt.Errorf("%s: %w", hostOf(u), err)
	}
	defer resp.Body.Close()

	resumed := false
	switch resp.StatusCode {
	case http.StatusOK:
		offset = 0 // 服务端不支持 Range → 从头写
		_ = os.Remove(part)
	case http.StatusPartialContent:
		resumed = offset > 0
		if !resumed {
			_ = os.Remove(part)
		}
	case http.StatusRequestedRangeNotSatisfiable: // 请求区间越界：清掉 .part 重来
		_ = os.Remove(part)
		return DownloadResult{}, fmt.Errorf("%s: 断点区间失效（HTTP 416），已清理续传文件", hostOf(u))
	default:
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return DownloadResult{}, fmt.Errorf("%s: HTTP %d %s", hostOf(u), resp.StatusCode, strings.TrimSpace(string(snippet)))
	}

	flags := os.O_CREATE | os.O_WRONLY
	mode := "从头下载"
	if resumed {
		flags |= os.O_APPEND
		mode = fmt.Sprintf("断点续传（已有 %d 字节）", offset)
	} else {
		flags |= os.O_TRUNC
	}
	f, err := os.OpenFile(part, flags, 0o644)
	if err != nil {
		return DownloadResult{}, fmt.Errorf("打开下载文件失败: %w", err)
	}

	hasher := sha256.New()
	if resumed { // 续传：把已有内容先喂给 hasher（否则摘要只覆盖新数据）
		if fh, herr := os.Open(part); herr == nil {
			_, _ = io.Copy(hasher, fh)
			_ = fh.Close()
		}
	}

	total := a.Size
	if resp.ContentLength > 0 && resp.StatusCode == http.StatusOK {
		total = resp.ContentLength
	}
	written, cerr := c.copyLoop(f, hasher, resp.Body, offset, total, &lastByte, onProgress)
	closeErr := f.Close()
	if cerr != nil {
		_ = os.Remove(part)
		if stalled.Load() {
			return DownloadResult{}, fmt.Errorf("下载停滞（%s 无数据增长）", stallTimeout)
		}
		if dctx.Err() != nil {
			return DownloadResult{}, fmt.Errorf("下载已取消")
		}
		return DownloadResult{}, fmt.Errorf("%s: %w", hostOf(u), cerr)
	}
	if closeErr != nil {
		_ = os.Remove(part)
		return DownloadResult{}, fmt.Errorf("写入下载文件失败: %w", closeErr)
	}

	sum := hex.EncodeToString(hasher.Sum(nil))
	// ★ 校验用「文件总大小」= 续传前的 offset + 本次写入量（Content-Length 只含剩余部分）
	totalBytes := offset + written
	if err := c.verify(a, totalBytes, sum); err != nil {
		_ = os.Remove(part)
		return DownloadResult{}, fmt.Errorf("%s（%s，%s）", err.Error(), mode, hostOf(u))
	}
	if err := os.Rename(part, final); err != nil { // 校验通过才落地成品名
		_ = os.Remove(part)
		return DownloadResult{}, fmt.Errorf("重命名下载文件失败: %w", err)
	}
	return DownloadResult{Path: final, Bytes: totalBytes, SHA256: sum, Resumed: resumed, URL: u}, nil
}

// copyLoop 读 → 写文件 + 喂 hasher，按节流回调进度。
func (c Config) copyLoop(dst *os.File, hasher hash.Hash, src io.Reader, offset, total int64, lastByte *atomic.Int64, onProgress func(int64)) (int64, error) {
	buf := make([]byte, chunkSize)
	var written int64
	lastCb := time.Now()
	for {
		n, rerr := src.Read(buf)
		if n > 0 {
			if _, werr := dst.Write(buf[:n]); werr != nil {
				return written, werr
			}
			hasher.Write(buf[:n])
			written += int64(n)
			lastByte.Store(time.Now().UnixNano())
			if onProgress != nil && time.Since(lastCb) >= progressInterval {
				lastCb = time.Now()
				onProgress(offset + written)
			}
		}
		if rerr == io.EOF {
			if onProgress != nil {
				onProgress(offset + written)
			}
			return written, nil
		}
		if rerr != nil {
			return written, rerr
		}
	}
}

// verify 大小 + sha256 校验（按 Config.RequireSHA256 决定严格度）。
func (c Config) verify(a AssetRef, size int64, sum string) error {
	if a.Size > 0 && size != a.Size {
		return fmt.Errorf("大小不符：期望 %d 字节，实际 %d 字节", a.Size, size)
	}
	want := normalizeDigest(a.SHA256)
	if want == "" {
		if c.RequireSHA256 {
			return errors.New("清单未提供 sha256，已拒绝安装（可在更新设置中关闭强制校验）")
		}
		return nil
	}
	if !strings.EqualFold(want, strings.ToLower(sum)) {
		return fmt.Errorf("sha256 校验失败：期望 %s，实际 %s", want, sum)
	}
	return nil
}

// fileSHA256 计算文件摘要（用于复用已下载文件 / dry-run 校验）。
func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// hostOf 取 URL 主机名（错误信息里比全 URL 可读）。
func hostOf(u string) string {
	if i := strings.Index(u, "://"); i >= 0 {
		rest := u[i+3:]
		if j := strings.IndexAny(rest, "/?#"); j >= 0 {
			return rest[:j]
		}
		return rest
	}
	return u
}
