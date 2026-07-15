package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	
	_ "modernc.org/sqlite"
)

func main() {
	dbPath := filepath.Join(os.TempDir(), "test_sqlite.db")
	fmt.Println("DB path:", dbPath)
	
	// 测试1: Open
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		fmt.Printf("FAIL: Open 失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("PASS: Open 成功")

	db.SetMaxOpenConns(1)

	// 测试2: 建表
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS test (id INTEGER PRIMARY KEY, name TEXT)`)
	if err != nil {
		fmt.Printf("FAIL: 建表失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("PASS: 建表成功")

	// 测试3: 读写
	_, err = db.Exec(`INSERT INTO test (id, name) VALUES (1, 'hello')`)
	if err != nil {
		fmt.Printf("FAIL: 写入失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("PASS: 写入成功")

	var name string
	err = db.QueryRow(`SELECT name FROM test WHERE id = 1`).Scan(&name)
	if err != nil {
		fmt.Printf("FAIL: 读取失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("PASS: 读取成功, name=%s\n", name)

	db.Close()
	os.Remove(dbPath)
	fmt.Println("ALL PASS: modernc.org/sqlite 在纯 Go 环境下正常工作")
}
