package agent

// AssetStore — 通用资产持久化层
// 融合自定义工具和进化引擎的存储基础：
// 统一路径管理、JSON 读写、列表、删除。
// 所有资产统一存放在:
//   安装目录/.pair/assets/{scope}/{type}/   (全局)
//   工作区/.pair/assets/{scope}/{type}/    (项目级)
//
// scope: "global" | "project"
// assetType: "tools" | "capsules" | "genes"

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hoonfeng/paircode/cmd/companion/core"
)

// AssetStore 通用资产存储层。
type AssetStore struct {
	installDir string // 安装目录（全局资产）
	projectDir string // 工作区根目录（项目级资产）
}

// NewAssetStore 创建通用资产存储层。
// projectRoot: 当前工作区根目录。
func NewAssetStore(projectRoot string) *AssetStore {
	return &AssetStore{
		installDir: core.InstallDir(),
		projectDir: projectRoot,
	}
}

// ── 目录路径 ──────────────────────────────────────────────

// GlobalDir 返回全局资产的目录路径：安装目录/.pair/assets/{assetType}/
func (as *AssetStore) GlobalDir(assetType string) string {
	return filepath.Join(as.installDir, ".pair", "assets", "global", assetType)
}

// ProjectDir 返回项目级资产的目录路径：工作区/.pair/assets/{assetType}/
func (as *AssetStore) ProjectDir(assetType string) string {
	return filepath.Join(as.projectDir, ".pair", "assets", "project", assetType)
}

// Dir 根据作用域和资产类型返回对应目录。
// scope: "global" | "project"
func (as *AssetStore) Dir(scope, assetType string) string {
	if scope == "project" {
		return as.ProjectDir(assetType)
	}
	return as.GlobalDir(assetType)
}

// EnsureDir 确保目录存在（创建目录树）。
func (as *AssetStore) EnsureDir(scope, assetType string) string {
	dir := as.Dir(scope, assetType)
	os.MkdirAll(dir, 0o755)
	return dir
}

// ── 文件路径 ──────────────────────────────────────────────

// JSONPath 返回 .json 资产文件的完整路径。
func (as *AssetStore) JSONPath(scope, assetType, id string) string {
	base := id
	if !strings.HasSuffix(base, ".json") {
		base += ".json"
	}
	return filepath.Join(as.Dir(scope, assetType), base)
}

// ── 读写操作 ──────────────────────────────────────────────

// ReadJSON 从指定作用域和资产类型读取 JSON 文件到 out。
// id 不含 .json 后缀（自动补全）。
func (as *AssetStore) ReadJSON(scope, assetType, id string, out any) error {
	path := as.JSONPath(scope, assetType, id)
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取资产 %s/%s/%s 失败: %w", scope, assetType, id, err)
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("解析资产 %s/%s/%s 失败: %w", scope, assetType, id, err)
	}
	return nil
}

// WriteJSON 将数据写入指定作用域和资产类型的 JSON 文件。
// id 不含 .json 后缀（自动补全）。
func (as *AssetStore) WriteJSON(scope, assetType, id string, data any) error {
	as.EnsureDir(scope, assetType)
	path := as.JSONPath(scope, assetType, id)
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化资产 %s/%s/%s 失败: %w", scope, assetType, id, err)
	}
	return os.WriteFile(path, raw, 0o644)
}

// WriteJSONFile 将数据写入指定路径（不自动添加 .json 后缀）。
// 用于兼容单个文件存储多工具的场景（如 tools.json）。
func (as *AssetStore) WriteJSONFile(scope, assetType, filename string, data any) error {
	as.EnsureDir(scope, assetType)
	path := filepath.Join(as.Dir(scope, assetType), filename)
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化资产文件 %s 失败: %w", filename, err)
	}
	return os.WriteFile(path, raw, 0o644)
}

// ReadJSONFile 从指定路径读取 JSON 文件（不自动添加 .json 后缀）。
func (as *AssetStore) ReadJSONFile(scope, assetType, filename string, out any) error {
	path := filepath.Join(as.Dir(scope, assetType), filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取资产文件 %s 失败: %w", filename, err)
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("解析资产文件 %s 失败: %w", filename, err)
	}
	return nil
}

// ── 列表与搜索 ────────────────────────────────────────────

// ListFiles 列出指定作用域和资产类型下的所有文件，返回文件名列表。
func (as *AssetStore) ListFiles(scope, assetType string) ([]string, error) {
	dir := as.Dir(scope, assetType)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

// ListBySuffix 列出指定作用域和资产类型下符合后缀的文件名列表。
func (as *AssetStore) ListBySuffix(scope, assetType, suffix string) ([]string, error) {
	all, err := as.ListFiles(scope, assetType)
	if err != nil {
		return nil, err
	}
	var matched []string
	for _, name := range all {
		if strings.HasSuffix(name, suffix) {
			matched = append(matched, name)
		}
	}
	return matched, nil
}

// DeleteFile 删除指定作用域和资产类型下的文件。
func (as *AssetStore) DeleteFile(scope, assetType, filename string) error {
	path := filepath.Join(as.Dir(scope, assetType), filename)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("删除资产 %s/%s/%s 失败: %w", scope, assetType, filename, err)
	}
	return nil
}

// ForEachJSON 遍历指定目录下所有 .json 文件，对每个文件调用 fn。
func (as *AssetStore) ForEachJSON(scope, assetType string, fn func(filename string, data []byte) error) error {
	files, err := as.ListBySuffix(scope, assetType, ".json")
	if err != nil {
		return err
	}
	for _, f := range files {
		path := filepath.Join(as.Dir(scope, assetType), f)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if err := fn(f, data); err != nil {
			return err
		}
	}
	return nil
}
