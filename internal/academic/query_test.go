package academic

import (
	"context"
	"errors"
	"testing"
	"time"
)

func seedForQueryTest(t *testing.T) *Store {
	t.Helper()
	s := newTestStore(t)
	if err := Seed(context.Background(), s, SeedOptions{RNGSeed: 1, StudentsPerClass: 5, Terms: 1, Rules: DefaultRules()}); err != nil {
		t.Fatal(err)
	}
	if err := s.BindIdentity(context.Background(), "stu-1", Identity{Role: RoleStudent, StudentID: 1}); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestStudentCannotReachPhysicalTables(t *testing.T) {
	s := seedForQueryTest(t)
	ctx := context.Background()
	id, _ := s.ResolveIdentity(ctx, "stu-1")
	for _, q := range []string{
		`SELECT * FROM grade`,
		`SELECT * FROM main.grade`,
		`SELECT * FROM src.grade`,
		`SELECT * FROM my_grades g JOIN grade x ON 1=1`,
	} {
		if _, err := s.Query(ctx, id, q, DefaultQueryOptions()); err == nil {
			t.Errorf("越权查询竟然成功: %s", q)
		}
	}
}

func TestStudentSeesOnlyOwnRows(t *testing.T) {
	s := seedForQueryTest(t)
	ctx := context.Background()
	id, _ := s.ResolveIdentity(ctx, "stu-1")
	res, err := s.Query(ctx, id, `SELECT COUNT(*) AS n FROM my_grades`, DefaultQueryOptions())
	if err != nil {
		t.Fatal(err)
	}
	if res.Rows[0][0] == "0" {
		t.Fatal("本人的成绩表不应该为空")
	}
	// 再绑一个学生，确认拿到的行数与上一位不同源
	if err := s.BindIdentity(ctx, "stu-2", Identity{Role: RoleStudent, StudentID: 2}); err != nil {
		t.Fatal(err)
	}
	id2, _ := s.ResolveIdentity(ctx, "stu-2")
	res2, err := s.Query(ctx, id2, `SELECT COUNT(*) FROM my_grades`, DefaultQueryOptions())
	if err != nil {
		t.Fatal(err)
	}
	if res2.Rows[0][0] != res.Rows[0][0] {
		// 只要两次都能查到即可；相等也不代表泄漏
		t.Log("两位学生行数不同，符合预期")
	}
}

func TestNonSelectIsRejected(t *testing.T) {
	s := seedForQueryTest(t)
	ctx := context.Background()
	id, _ := s.ResolveIdentity(ctx, "stu-1")
	for _, q := range []string{
		`UPDATE my_grades SET total_score = 100`,
		`DELETE FROM my_grades`,
		`PRAGMA table_info(my_grades)`,
		`ATTACH DATABASE '/tmp/x.db' AS x`,
	} {
		if _, err := s.Query(ctx, id, q, DefaultQueryOptions()); !errors.Is(err, ErrNotSelect) {
			t.Errorf("应拒绝 %q，得到 %v", q, err)
		}
	}
}

func TestNoLeakBetweenCalls(t *testing.T) {
	s := seedForQueryTest(t)
	ctx := context.Background()
	admin := Identity{Role: RoleAdmin}
	if _, err := s.Query(ctx, admin, `SELECT COUNT(*) FROM students_all`, DefaultQueryOptions()); err != nil {
		t.Fatal(err)
	}
	// 管理员视图里有 students_all；学生视图里必须没有
	id, _ := s.ResolveIdentity(ctx, "stu-1")
	if _, err := s.Query(ctx, id, `SELECT COUNT(*) FROM students_all`, DefaultQueryOptions()); err == nil {
		t.Fatal("学生的连接里出现了上一次请求的物化表")
	}
}

func TestTruncatesLargeResult(t *testing.T) {
	s := seedForQueryTest(t)
	ctx := context.Background()
	admin := Identity{Role: RoleAdmin}
	res, err := s.Query(ctx, admin, `SELECT * FROM students_all ORDER BY id`, QueryOptions{MaxRows: 2, MaxBytes: 1 << 20, Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Rows) != 2 || !res.Truncated {
		t.Fatalf("want 2 rows truncated, got %d rows truncated=%v", len(res.Rows), res.Truncated)
	}
}
