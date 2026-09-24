// source.go — 更新清单获取（GitHub Releases API / 自定义 feed）与下载端点选择。
package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	githubAPI = "https://api.github.com"
	// maxManifestBytes 清单响应上限（防超大响应体）
	maxManifestBytes = 4 << 20
	// maxNotesBytes 发布说明截断上限（前端 markdown 渲染安全余量）
	maxNotesBytes = 64 << 10
)

// httpClient 返回 HTTP 客户端：注入桩优先，否则自建（连接层超时，不设整体超时——
// 下载 85MB 包需长连接；清单类小请求各自用 context 限时）。
func (c Config) httpClient() HTTPDoer {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			ResponseHeaderTimeout: 30 * time.Second,
			TLSHandshakeTimeout:   15 * time.Second,
			IdleConnTimeout:       90 * time.Second,
			MaxIdleConns:          8,
			MaxIdleConnsPerHost:   4,
		},
	}
}

func (c Config) userAgent() string {
	if strings.TrimSpace(c.UserAgent) != "" {
		return c.UserAgent
	}
	v := NormalizeVersion(c.CurrentVersion)
	if v == "" {
		v = "dev"
	}
	return "PairCode-Updater/" + v
}

// manifestURL 返回清单地址：stable → /releases/latest；prerelease → /releases 列表。
func (c Config) manifestURL() string {
	if strings.EqualFold(c.FeedType, "custom") {
		return strings.TrimSpace(c.FeedURL)
	}
	repo := strings.TrimSpace(c.Repo)
	if strings.EqualFold(c.Channel, "prerelease") {
		return fmt.Sprintf("%s/repos/%s/releases?per_page=30", githubAPI, repo)
	}
	return fmt.Sprintf("%s/repos/%s/releases/latest", githubAPI, repo)
}

// downloadURLs 返回某资产的下载端点候选（按优先级）：
//  1. 镜像前缀 + 直链（配了 mirrorPrefix——github.com 不可达而镜像可达时的通路）
//  2. GitHub 资产 API 端点（api.github.com 稳定，实测 302→206 可用）
//  3. 直链 browser_download_url（最后回退）
func (c Config) downloadURLs(a AssetRef) []string {
	out := make([]string, 0, 3)
	prefix := strings.TrimSpace(c.MirrorPrefix)
	if prefix != "" && a.URL != "" {
		out = append(out, joinURL(prefix, a.URL))
	}
	if a.APIURL != "" {
		out = append(out, a.APIURL)
	}
	if a.URL != "" {
		out = append(out, a.URL)
	}
	return out
}

// joinURL 拼接镜像前缀与原始 URL（前缀无 scheme 时按 https:// 处理，末尾斜杠归一）。
func joinURL(prefix, raw string) string {
	p := strings.TrimSpace(prefix)
	if p != "" && !strings.Contains(p, "://") {
		p = "https://" + p
	}
	p = strings.TrimSuffix(p, "/")
	return p + "/" + raw
}

// FetchManifest 拉取并解析更新清单（GitHub 或自定义 feed）。
func (c Config) FetchManifest(ctx context.Context) (Manifest, error) {
	target := c.manifestURL()
	if target == "" {
		return Manifest{}, fmt.Errorf("更新源未配置（repo / feedURL 均为空）")
	}
	raw, err := c.fetchRaw(ctx, target, 30*time.Second)
	if err != nil {
		return Manifest{}, err
	}
	if strings.EqualFold(c.FeedType, "custom") {
		return c.parseCustomFeed(raw)
	}
	return c.parseGitHubRelease(raw)
}

// fetchRaw 读取 URL 内容：支持 http(s):// 与 file:// / 本地绝对路径（离线源与测试用）。
func (c Config) fetchRaw(ctx context.Context, target string, timeout time.Duration) ([]byte, error) {
	low := strings.ToLower(target)
	if strings.HasPrefix(low, "file://") {
		p := target[len("file://"):]
		if unescaped, err := url.PathUnescape(p); err == nil {
			p = unescaped
		}
		// Windows 形如 /F:/path → 去掉前导斜杠
		if len(p) > 2 && p[0] == '/' && p[2] == ':' {
			p = p[1:]
		}
		return readLimitedFile(p, maxManifestBytes)
	}
	if !strings.Contains(target, "://") {
		if st, err := os.Stat(target); err == nil && !st.IsDir() {
			return readLimitedFile(target, maxManifestBytes)
		}
		return nil, fmt.Errorf("无效的更新源地址: %s", target)
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", target, nil)
	if err != nil {
		return nil, err
	}
	c.decorate(req, false)
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求更新源失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("更新源返回 HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(snippet)))
	}
	return readLimited(resp.Body, maxManifestBytes)
}

// decorate 给请求加统一头；asset=true 时用于资产二进制下载。
func (c Config) decorate(req *http.Request, asset bool) {
	req.Header.Set("User-Agent", c.userAgent())
	if asset {
		req.Header.Set("Accept", "application/octet-stream")
	} else {
		req.Header.Set("Accept", "application/vnd.github+json, application/json")
		if strings.TrimSpace(c.Token) != "" {
			req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(c.Token))
		}
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	}
}

// ─── GitHub 解析 ─────────────────────────────────────────────

type ghAsset struct {
	ID                 int64  `json:"id"`
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	Digest             string `json:"digest"` // "sha256:<hex>"（GitHub 2025+ 返回）
	BrowserDownloadURL string `json:"browser_download_url"`
}

type ghRelease struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Body        string    `json:"body"`
	PublishedAt string    `json:"published_at"`
	Draft       bool      `json:"draft"`
	Prerelease  bool      `json:"prerelease"`
	Assets      []ghAsset `json:"assets"`
	Message     string    `json:"message"` // API 错误体
}

// parseGitHubRelease 解析 /releases/latest（对象）或 /releases（数组）响应。
func (c Config) parseGitHubRelease(raw []byte) (Manifest, error) {
	trimmed := strings.TrimSpace(string(raw))
	if strings.HasPrefix(trimmed, "[") {
		var list []ghRelease
		if err := json.Unmarshal(raw, &list); err != nil {
			return Manifest{}, fmt.Errorf("解析 release 列表失败: %w", err)
		}
		for i := range list {
			rel := list[i]
			if rel.Draft {
				continue
			}
			return c.buildManifest(rel)
		}
		return Manifest{}, fmt.Errorf("未找到可用发布（均为 draft）")
	}

	var rel ghRelease
	if err := json.Unmarshal(raw, &rel); err != nil {
		return Manifest{}, fmt.Errorf("解析 release 失败: %w", err)
	}
	if rel.TagName == "" && rel.Message != "" {
		return Manifest{}, fmt.Errorf("GitHub API 错误: %s", rel.Message)
	}
	if rel.Draft {
		return Manifest{}, fmt.Errorf("最新发布为 draft（未公开）")
	}
	return c.buildManifest(rel)
}

// buildManifest 从 GitHub release 构造归一化清单（含平台资产匹配）。
func (c Config) buildManifest(rel ghRelease) (Manifest, error) {
	ver := NormalizeVersion(rel.TagName)
	if ver == "" {
		return Manifest{}, fmt.Errorf("发布缺少 tag_name")
	}
	names := make([]string, 0, len(rel.Assets))
	byName := make(map[string]ghAsset, len(rel.Assets))
	for _, a := range rel.Assets {
		names = append(names, a.Name)
		byName[a.Name] = a
	}
	m := Manifest{
		Version:     ver,
		Tag:         rel.TagName,
		Notes:       truncate(rel.Body, maxNotesBytes),
		PublishedAt: rel.PublishedAt,
		Source:      "github:" + strings.TrimSpace(c.Repo),
	}
	picked := MatchAsset(names, ver, c.GOOS)
	if picked == "" {
		return m, fmt.Errorf("发布 %s 无本平台（%s）可用更新包", rel.TagName, c.GOOS)
	}
	a := byName[picked]
	m.Asset = AssetRef{
		ID:     a.ID,
		Name:   a.Name,
		Size:   a.Size,
		SHA256: normalizeDigest(a.Digest),
		URL:    a.BrowserDownloadURL,
		OS:     strings.ToLower(c.GOOS),
	}
	if a.ID > 0 {
		m.Asset.APIURL = fmt.Sprintf("%s/repos/%s/releases/assets/%d", githubAPI, strings.TrimSpace(c.Repo), a.ID)
	}
	return m, nil
}

// ─── 自定义 feed 解析 ────────────────────────────────────────

// customFeed 自定义清单格式（自建 CDN / 内网源 / 离线测试）。
//
//	{"version":"1.7.0","notes":"…","publishedAt":"…",
//	 "assets":{"windows":{"name":"…","url":"…","size":0,"sha256":"…"}, "linux":{…}, "darwin":{…}}}
type customFeed struct {
	Version     string                    `json:"version"`
	Tag         string                    `json:"tag"`
	Notes       string                    `json:"notes"`
	PublishedAt string                    `json:"publishedAt"`
	Assets      map[string]customAssetRef `json:"assets"`
}

type customAssetRef struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

func (c Config) parseCustomFeed(raw []byte) (Manifest, error) {
	var f customFeed
	if err := json.Unmarshal(raw, &f); err != nil {
		return Manifest{}, fmt.Errorf("解析自定义清单失败: %w", err)
	}
	ver := NormalizeVersion(f.Version)
	if ver == "" {
		return Manifest{}, fmt.Errorf("自定义清单缺少 version")
	}
	m := Manifest{
		Version:     ver,
		Tag:         firstNonEmpty(f.Tag, "v"+ver),
		Notes:       truncate(f.Notes, maxNotesBytes),
		PublishedAt: f.PublishedAt,
		Source:      "custom:" + c.manifestURL(),
	}
	key := strings.ToLower(c.GOOS)
	if a, ok := f.Assets[key]; ok {
		m.Asset = AssetRef{
			Name:   firstNonEmpty(a.Name, AssetNameForPlatform(ver, c.GOOS)),
			Size:   a.Size,
			SHA256: normalizeDigest(a.SHA256),
			URL:    a.URL,
			OS:     key,
		}
	}
	if m.Asset.URL == "" && m.Asset.Name == "" {
		return m, fmt.Errorf("自定义清单无本平台（%s）资产", c.GOOS)
	}
	return m, nil
}

// ─── 小工具 ─────────────────────────────────────────────────

// normalizeDigest 归一化摘要：去 `sha256:` 前缀、去空白、转小写。
func normalizeDigest(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.TrimPrefix(s, "sha256:")
	s = strings.TrimPrefix(s, "sha256=")
	return strings.TrimSpace(s)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n\n…（发布说明过长已截断）"
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// readLimited 读取上限内的全部内容（超限报错，不静默截断）。
func readLimited(r io.Reader, max int64) ([]byte, error) {
	buf, err := io.ReadAll(io.LimitReader(r, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(buf)) > max {
		return nil, fmt.Errorf("响应体超过 %d 字节上限", max)
	}
	return buf, nil
}

// readLimitedFile 读取本地文件（自定义/离线源）。
func readLimitedFile(path string, max int64) ([]byte, error) {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return readLimited(f, max)
}
