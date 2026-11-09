package academic

import (
	"context"
	"errors"
	"testing"
)

// seedIdentityFixtures 铺最小的人员外键目标（院系/专业/班/学生/教师/辅导员），
// 让身份绑定测试可以用任意 id。
func seedIdentityFixtures(ctx context.Context, s *Store) error {
	stmts := []string{
		`INSERT INTO department(id, name) VALUES(1, '测试学院')`,
		`INSERT INTO major(id, name, department_id, years) VALUES(1, '测试专业', 1, 4)`,
		`INSERT INTO class(id, name, major_id, grade_year) VALUES(1, '测试2401班', 1, 2024)`,
		`INSERT INTO student(id, student_no, name, class_id, enroll_year) VALUES(7, '20240001', '张三', 1, 2024)`,
		`INSERT INTO teacher(id, teacher_no, name, department_id) VALUES(9, 'T90001', '李四', 1)`,
		`INSERT INTO counselor(id, staff_no, name, department_id) VALUES(8, 'C80001', '王五', 1)`,
	}
	for _, stmt := range stmts {
		if _, err := s.DB().ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

func TestResolveIdentity(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	// 外键约束要求被绑定的学生真实存在：先铺最小人员数据。
	if err := seedIdentityFixtures(ctx, s); err != nil {
		t.Fatal(err)
	}
	if err := s.BindIdentity(ctx, "user-1", Identity{Role: RoleStudent, StudentID: 7}); err != nil {
		t.Fatal(err)
	}
	got, err := s.ResolveIdentity(ctx, "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Role != RoleStudent || got.StudentID != 7 {
		t.Fatalf("got %+v", got)
	}
}

func TestResolveUnknownIdentityIsNotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.ResolveIdentity(context.Background(), "nobody")
	if !errors.Is(err, ErrIdentityNotFound) {
		t.Fatalf("want ErrIdentityNotFound, got %v", err)
	}
}
