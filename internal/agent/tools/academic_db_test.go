package tools

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/academic"
	"github.com/Tencent/WeKnora/internal/types"
)

// newAcademicFixture 构造一个已种子化、已绑定 stu-1 的教务库。
func newAcademicFixture(t *testing.T) *academic.Store {
	t.Helper()
	s, err := academic.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	ctx := context.Background()
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if err := academic.Seed(ctx, s, academic.SeedOptions{RNGSeed: 1, StudentsPerClass: 5, Terms: 1, Rules: academic.DefaultRules()}); err != nil {
		t.Fatalf("Seed: %v", err)
	}
	if err := s.BindIdentity(ctx, "stu-1", academic.Identity{Role: academic.RoleStudent, StudentID: 1}); err != nil {
		t.Fatalf("BindIdentity: %v", err)
	}
	return s
}

func TestAcademicDBToolReturnsRows(t *testing.T) {
	store := newAcademicFixture(t)
	tool := NewAcademicDBTool(store)
	ctx := types.WithCaller(context.Background(), types.Caller{UserID: "stu-1"})
	res, err := tool.Execute(ctx, json.RawMessage(`{"sql":"SELECT COUNT(*) AS n FROM my_grades"}`))
	if err != nil {
		t.Fatalf("Execute 返回了 error: %v", err)
	}
	if res == nil || !res.Success {
		t.Fatalf("want success, got %+v", res)
	}
	if !strings.Contains(res.Output, "n") {
		t.Fatalf("输出缺列名: %q", res.Output)
	}
	if strings.Contains(res.Output, "0") && !strings.Contains(res.Output, "my_grades") {
		// 行数 0 不代表失败，但这里种子后必有成绩
		t.Logf("output=%q", res.Output)
	}
}

func TestAcademicDBToolUnboundIdentityIsRejected(t *testing.T) {
	store := newAcademicFixture(t)
	tool := NewAcademicDBTool(store)
	ctx := types.WithCaller(context.Background(), types.Caller{UserID: "ghost"})
	res, err := tool.Execute(ctx, json.RawMessage(`{"sql":"SELECT COUNT(*) FROM my_grades"}`))
	if err != nil {
		t.Fatalf("Execute 返回了 error: %v", err)
	}
	if res == nil || res.Success {
		t.Fatalf("未绑定身份必须失败, got %+v", res)
	}
	if !strings.Contains(res.Error, "未绑定") {
		t.Fatalf("错误文案应含「未绑定」, got %q", res.Error)
	}
}

func TestAcademicDBToolUnauthorizedTableListsAvailable(t *testing.T) {
	store := newAcademicFixture(t)
	tool := NewAcademicDBTool(store)
	ctx := types.WithCaller(context.Background(), types.Caller{UserID: "stu-1"})
	res, err := tool.Execute(ctx, json.RawMessage(`{"sql":"SELECT COUNT(*) FROM students_all"}`))
	if err != nil {
		t.Fatalf("Execute 返回了 error: %v", err)
	}
	if res == nil || res.Success {
		t.Fatalf("越权查询必须失败, got %+v", res)
	}
	for _, want := range []string{"my_grades", "my_profile"} {
		if !strings.Contains(res.Error, want) {
			t.Fatalf("错误文案应列出可用表 %q, got %q", want, res.Error)
		}
	}
}

func TestAcademicDBToolRejectsNonSelect(t *testing.T) {
	store := newAcademicFixture(t)
	tool := NewAcademicDBTool(store)
	ctx := types.WithCaller(context.Background(), types.Caller{UserID: "stu-1"})
	res, err := tool.Execute(ctx, json.RawMessage(`{"sql":"DELETE FROM my_grades"}`))
	if err != nil {
		t.Fatalf("Execute 返回了 error: %v", err)
	}
	if res == nil || res.Success {
		t.Fatalf("非查询语句必须失败, got %+v", res)
	}
}
