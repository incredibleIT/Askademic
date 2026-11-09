package academic

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Role 是教务角色。与 WeKnora 的空间角色（owner/contributor/viewer）正交：
// 空间角色管知识库协作，教务角色管数据行权限。
type Role string

const (
	RoleStudent   Role = "student"
	RoleTeacher   Role = "teacher"
	RoleCounselor Role = "counselor"
	RoleAdmin     Role = "admin"
)

// Identity 是一次查询的教务身份上下文。
// StudentID/TeacherID/CounselorID 只在对应 Role 下有意义，其余为 0。
type Identity struct {
	Role        Role
	StudentID   int64
	TeacherID   int64
	CounselorID int64
}

// ErrIdentityNotFound 表示该 WeKnora 用户未绑定教务身份。
// 调用方必须把它当成明确的失败，绝不回落到管理员视图。
var ErrIdentityNotFound = errors.New("未绑定教务身份")

// BindIdentity 把 WeKnora 用户绑定到教务身份（upsert：同一用户重复绑定即覆盖）。
func (s *Store) BindIdentity(ctx context.Context, weknoraUserID string, id Identity) error {
	if weknoraUserID == "" {
		return fmt.Errorf("绑定教务身份失败: weknora_user_id 为空")
	}
	switch id.Role {
	case RoleStudent:
		if id.StudentID <= 0 {
			return fmt.Errorf("绑定学生身份失败: student_id 非法")
		}
	case RoleTeacher:
		if id.TeacherID <= 0 {
			return fmt.Errorf("绑定教师身份失败: teacher_id 非法")
		}
	case RoleCounselor:
		if id.CounselorID <= 0 {
			return fmt.Errorf("绑定辅导员身份失败: counselor_id 非法")
		}
	case RoleAdmin:
		// 教务管理员不需要指向具体人员
	default:
		return fmt.Errorf("绑定教务身份失败: 未知角色 %q", id.Role)
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO academic_identity(weknora_user_id, role, student_id, teacher_id, counselor_id)
		VALUES(?, ?, ?, ?, ?)
		ON CONFLICT(weknora_user_id) DO UPDATE SET
		  role = excluded.role,
		  student_id = excluded.student_id,
		  teacher_id = excluded.teacher_id,
		  counselor_id = excluded.counselor_id`,
		weknoraUserID, string(id.Role),
		nullID(id.StudentID), nullID(id.TeacherID), nullID(id.CounselorID))
	if err != nil {
		return fmt.Errorf("绑定教务身份失败: %w", err)
	}
	return nil
}

// ResolveIdentity 解析 WeKnora 用户的教务身份。
// 未绑定时返回 ErrIdentityNotFound（不返回管理员，不做任何回落）。
func (s *Store) ResolveIdentity(ctx context.Context, weknoraUserID string) (Identity, error) {
	var id Identity
	var role string
	var studentID, teacherID, counselorID any
	err := s.db.QueryRowContext(ctx, `
		SELECT role, student_id, teacher_id, counselor_id
		FROM academic_identity WHERE weknora_user_id = ?`, weknoraUserID).
		Scan(&role, &studentID, &teacherID, &counselorID)
	if errors.Is(err, sql.ErrNoRows) {
		return Identity{}, ErrIdentityNotFound
	}
	if err != nil {
		return Identity{}, fmt.Errorf("解析教务身份失败: %w", err)
	}
	id.Role = Role(role)
	id.StudentID, _ = studentID.(int64)
	id.TeacherID, _ = teacherID.(int64)
	id.CounselorID, _ = counselorID.(int64)
	return id, nil
}

// nullID 把 0 归一成 SQL NULL（身份表里非本角色的外键列应为 NULL）。
func nullID(v int64) any {
	if v <= 0 {
		return nil
	}
	return v
}
