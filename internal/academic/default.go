package academic

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
)

// defaultStore 是进程级默认教务库。启动时由容器初始化（软失败），
// 工具层通过 Default() 取用；未初始化时为 nil，工具会返回明确错误。
var defaultStore atomic.Pointer[Store]

// InitFromEnv 从环境变量初始化默认教务库：
// WEKNORA_ACADEMIC_DB 指定文件路径，缺省 ./data/academic.db。
// 目录不存在则创建；库文件按需建表。
func InitFromEnv() error {
	path := os.Getenv("WEKNORA_ACADEMIC_DB")
	if path == "" {
		path = "./data/academic.db"
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	s, err := Open(path)
	if err != nil {
		return err
	}
	if err := s.Migrate(context.Background()); err != nil {
		s.Close()
		return err
	}
	defaultStore.Store(s)
	return nil
}

// Default 返回进程级默认教务库；未初始化时返回 nil。
func Default() *Store { return defaultStore.Load() }

// MustFromEnvName 返回默认库文件路径（展示用）。
func MustFromEnvName() string {
	path := os.Getenv("WEKNORA_ACADEMIC_DB")
	if path == "" {
		path = "./data/academic.db"
	}
	return filepath.Dir(path)
}

// QueryDescription 返回给工具描述用的物化表清单文本。
// 只列物化表名，物理表名不出现在任何面向模型的文本里。
func QueryDescription(id Identity) string {
	return fmt.Sprintf("%v", AvailableTables(id))
}
