//go:build windows

// Package ui 资源加载辅助。
// 所有面板通过本包加载 HTML 模板，统一资源目录路径解析。
package ui

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hoonfeng/gwui/dom"
	"github.com/hoonfeng/gwui/uixml"
)

// ResourceDir 返回资源目录的绝对路径。
// 按优先级尝试：CWD/resources → CWD/cmd/companion/resources → 源码相对路径。
func ResourceDir() string {
	candidates := []string{
		"resources",
		"cmd/companion/resources",
		"../cmd/companion/resources",
	}
	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && fi.IsDir() {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	abs, _ := filepath.Abs("cmd/companion/resources")
	return abs
}

// ResourcePath 返回资源目录下某个文件的完整路径。
func ResourcePath(rel string) string {
	return filepath.Join(ResourceDir(), rel)
}

// ReadResource 读取资源文件内容。
func ReadResource(rel string) ([]byte, error) {
	return os.ReadFile(ResourcePath(rel))
}

// ReadResourceString 读取资源文件为字符串。
func ReadResourceString(rel string) string {
	data, err := ReadResource(rel)
	if err != nil {
		return ""
	}
	return string(data)
}

// LoadPanelHTML 加载面板 HTML 模板到文档。
// htmlFile 是相对于 resources/html/ 的路径（如 "panels/search.html"）。
// 加载后元素通过 doc.GetElementByID 获取。
func LoadPanelHTML(doc *dom.Document, htmlFile string, reg *uixml.Registry) error {
	path := filepath.Join(ResourceDir(), "html", htmlFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("加载面板 HTML 失败 %s: %w", path, err)
	}
	return uixml.LoadInto(doc, data, reg)
}

// MustLoadPanelHTML 同 LoadPanelHTML，失败时 panic。
func MustLoadPanelHTML(doc *dom.Document, htmlFile string, reg *uixml.Registry) {
	if err := LoadPanelHTML(doc, htmlFile, reg); err != nil {
		panic(err)
	}
}

// DetachRoot 从临时父节点中分离根元素（uixml.LoadInto 会将根挂到 body 下）。
func DetachRoot(root *dom.Element) {
	if root == nil {
		return
	}
	if parent, ok := root.Parent().(*dom.Element); ok {
		parent.RemoveChild(root)
	}
}

// TransferComponents 递归地将 srcDoc 中的组件注册转移到 dstDoc。
// 从 search.go 抽取的公共函数，供所有面板复用。
func TransferComponents(srcDoc, dstDoc *dom.Document, el *dom.Element) {
	if comp := srcDoc.ComponentAtNode(el); comp != nil {
		dstDoc.RegisterComponent(el, comp)
	}
	for _, child := range el.Children() {
		if e, ok := child.(*dom.Element); ok {
			TransferComponents(srcDoc, dstDoc, e)
		}
	}
}

// ReplaceChildByID 用新元素替换指定 ID 的占位元素。
func ReplaceChildByID(doc *dom.Document, id string, newEl *dom.Element) bool {
	placeholder := doc.GetElementByID(id)
	if placeholder == nil {
		return false
	}
	if parent, ok := placeholder.Parent().(*dom.Element); ok {
		parent.ReplaceChild(newEl, placeholder)
		return true
	}
	return false
}

// AdoptBodyChildren 将文档 body 下的所有子元素迁移到 parent 下。
// 用于 HTML 模板含多个根元素的场景（如对话框内容：容器 + 关闭按钮）。
// 同时把组件注册从临时文档迁移到 dstDoc。
func AdoptBodyChildren(dstDoc *dom.Document, parent *dom.Element) {
	body := dstDoc.Body()
	for {
		c := body.FirstChild()
		if c == nil {
			break
		}
		body.RemoveChild(c)
		if el, ok := c.(*dom.Element); ok {
			TransferComponents(dstDoc, dstDoc, el)
		}
		parent.AppendChild(c)
	}
}
