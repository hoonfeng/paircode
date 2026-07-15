package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	_ "modernc.org/sqlite"
)

func main() {
	wd, _ := os.Getwd()
	dbPath := filepath.Join(wd, ".pair", "pair.db")
	fmt.Println("DB path:", dbPath)

	db, err := sql.Open("sqlite", dbPath)
	if err != nil { fmt.Println("Open error:", err); os.Exit(1) }
	defer db.Close()

	rows, err := db.Query("SELECT id, title, created_at, msg_count FROM conversations ORDER BY created_at DESC LIMIT 15")
	if err != nil { fmt.Println("Query error:", err); os.Exit(1) }
	defer rows.Close()
	for rows.Next() {
		var id, title, createdAt string
		var cnt int
		rows.Scan(&id, &title, &createdAt, &cnt)
		fmt.Printf("ID=%s | 标题=%s | 创建=%s | %d条\n", id, title, createdAt, cnt)
	}
}
