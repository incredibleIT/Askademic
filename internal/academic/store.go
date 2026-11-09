// Package academic 承载校园教务库：一个独立的 SQLite 文件，
// 与 WeKnora 的应用库（PostgreSQL）完全没有关系。
// 查询隔离采用“ATTACH → 物化 → DETACH”：模型 SQL 只允许跑在
// 按角色物化出来的内存视图上，物理表永不出现在查询面。
package academic

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// Store 是教务库的句柄。教务库是一个独立的 SQLite 文件，
// 与 WeKnora 的应用库完全没有关系。
type Store struct {
	db   *sql.DB
	path string
}

// Open 打开（必要时创建）指定路径的教务库文件。
// DSN 上开启外键约束并设置忙等超时；单连接让 :memory: 的行为可预期。
func Open(path string) (*Store, error) {
	dsn := fmt.Sprintf("file:%s?_busy_timeout=5000&_fk=1", path)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("打开教务库失败: %w", err)
	}
	// 单连接：SQLite 单文件写入更安全，也让 :memory: 库的行为可预期。
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("连接教务库失败: %w", err)
	}
	return &Store{db: db, path: path}, nil
}

// Close 关闭底层连接。
func (s *Store) Close() error { return s.db.Close() }

// DB 暴露底层连接（测试与内部物化使用；业务侧一律走 Query）。
func (s *Store) DB() *sql.DB { return s.db }

// Path 返回教务库文件路径（物化查询 ATTACH 时需要）。
func (s *Store) Path() string { return s.path }

// Migrate 依次执行全部建表语句；语句全部为 IF NOT EXISTS，可重复执行。
func (s *Store) Migrate(ctx context.Context) error {
	for _, stmt := range ddl {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("建表失败: %w", err)
		}
	}
	return nil
}
