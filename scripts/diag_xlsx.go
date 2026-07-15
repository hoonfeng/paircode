//go:build ignore

package main

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: go run diag.go <xlsx路径>")
		os.Exit(1)
	}
	path := os.Args[1]
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("读取失败: %v\n", err)
		os.Exit(1)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		fmt.Printf("ZIP 打开失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("文件: %s (%d 字节, %d 个文件)\n\n", path, len(data), len(zr.File))
	for _, f := range zr.File {
		fmt.Printf("  %s (%d 字节, 压缩后 %d)\n", f.Name, f.UncompressedSize64, f.CompressedSize64)
		rc, _ := f.Open()
		content, _ := io.ReadAll(rc)
		rc.Close()
		if strings.HasSuffix(f.Name, ".xml") || strings.HasSuffix(f.Name, ".rels") {
			fmt.Printf("    --- 内容 ---\n    %s\n", string(content))
		}
	}

	// 解析 workbook
	fmt.Println("\n--- 解析工作簿 ---")
	for _, f := range zr.File {
		if f.Name == "xl/workbook.xml" {
			rc, _ := f.Open()
			data, _ := io.ReadAll(rc)
			rc.Close()
			var wb struct {
				Sheets []struct {
					Name  string `xml:"name,attr"`
					ID    string `xml:"sheetId,attr"`
					RID   string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
				} `xml:"sheets>sheet"`
			}
			xml.Unmarshal(data, &wb)
			fmt.Printf("Sheets: %+v\n", wb.Sheets)
		}
	}

	// 解析关系
	fmt.Println("\n--- 解析关系 ---")
	for _, f := range zr.File {
		if f.Name == "xl/_rels/workbook.xml.rels" {
			rc, _ := f.Open()
			data, _ := io.ReadAll(rc)
			rc.Close()
			var rels struct {
				Relationships []struct {
					ID     string `xml:"Id,attr"`
					Type   string `xml:"Type,attr"`
					Target string `xml:"Target,attr"`
				} `xml:"Relationship"`
			}
			xml.Unmarshal(data, &rels)
			for _, r := range rels.Relationships {
				fmt.Printf("  %s -> %s\n", r.ID, r.Target)
			}
		}
	}
}
