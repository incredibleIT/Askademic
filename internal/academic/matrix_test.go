package academic

import (
	"context"
	"testing"
)

// TestPermissionMatrix 把设计文档 §6 的每一格写成断言（表驱动）。
// 若有格子不符，改物化 SQL，不改测试。
func TestPermissionMatrix(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	if err := Seed(ctx, s, SeedOptions{RNGSeed: 3, StudentsPerClass: 5, Terms: 2, Rules: DefaultRules()}); err != nil {
		t.Fatal(err)
	}
	// 每个角色预绑一个测试用户。
	binds := []struct {
		user string
		id   Identity
	}{
		{"stu", Identity{Role: RoleStudent, StudentID: 1}},
		{"tea", Identity{Role: RoleTeacher, TeacherID: 1}},
		{"cou", Identity{Role: RoleCounselor, CounselorID: 1}},
		{"adm", Identity{Role: RoleAdmin}},
	}
	for _, b := range binds {
		if err := s.BindIdentity(ctx, b.user, b.id); err != nil {
			t.Fatalf("绑定 %s: %v", b.user, err)
		}
	}

	matrix := []struct {
		name  string
		user  string
		query string
		allow bool
	}{
		{"学生看自己成绩", "stu", `SELECT * FROM my_grades`, true},
		{"学生看自己档案", "stu", `SELECT * FROM my_profile`, true},
		{"学生看自己学分", "stu", `SELECT * FROM my_credits`, true},
		{"学生看班级平均分", "stu", `SELECT AVG(total_score) FROM class_grades`, false},
		{"学生看他人档案", "stu", `SELECT * FROM students_all`, false},
		{"教师看所带教学班成绩", "tea", `SELECT * FROM class_grades`, true},
		{"教师看所带名单", "tea", `SELECT * FROM class_roster`, true},
		{"教师看所带统计", "tea", `SELECT * FROM class_stats`, true},
		{"教师看非所带班成绩", "tea", `SELECT * FROM students_all`, false},
		{"辅导员看所带班级", "cou", `SELECT * FROM counselor_classes`, true},
		{"辅导员看所带名单", "cou", `SELECT * FROM class_roster`, true},
		{"辅导员看所带统计", "cou", `SELECT * FROM class_stats`, true},
		{"辅导员看全校汇总", "cou", `SELECT * FROM students_all`, false},
		{"教务看全量学生", "adm", `SELECT * FROM students_all`, true},
		{"教务看全量成绩明细", "adm", `SELECT * FROM grades_all`, true},
		{"教务看班级统计", "adm", `SELECT * FROM class_stats`, true},
	}

	for _, cell := range matrix {
		id, err := s.ResolveIdentity(ctx, cell.user)
		if err != nil {
			t.Fatalf("%s: 解析身份失败: %v", cell.name, err)
		}
		res, err := s.Query(ctx, id, cell.query, DefaultQueryOptions())
		if cell.allow {
			if err != nil {
				t.Errorf("[%s] 应允许却失败: %v", cell.name, err)
			} else if res == nil {
				t.Errorf("[%s] 应允许却无结果", cell.name)
			}
		} else {
			if err == nil {
				t.Errorf("[%s] 应拒绝却成功: %q", cell.name, cell.query)
			}
		}
	}
}
