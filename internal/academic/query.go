package academic

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// 物化表名常量：这些名字对模型可见，物理表名永远不出现。
const (
	TblMyProfile         = "my_profile"
	TblMyGrades          = "my_grades"
	TblMyEnrollments     = "my_enrollments"
	TblMyCredits         = "my_credits"
	TblMyTeachingClasses = "my_teaching_classes"
	TblClassGrades       = "class_grades"
	TblClassRoster       = "class_roster"
	TblClassStats        = "class_stats"
	TblCounselorClasses  = "counselor_classes"
	TblStudentsAll       = "students_all"
)

// QueryOptions 控制单次查询的资源上限。
type QueryOptions struct {
	MaxRows  int
	MaxBytes int
	Timeout  time.Duration
}

// DefaultQueryOptions 返回默认上限：200 行 / 32KB / 5 秒。
func DefaultQueryOptions() QueryOptions {
	return QueryOptions{MaxRows: 200, MaxBytes: 32 * 1024, Timeout: 5 * time.Second}
}

// QueryResult 是模型可见的查询结果（全部字符串化）。
type QueryResult struct {
	Columns   []string
	Rows      [][]string
	Truncated bool
	Elapsed   time.Duration
}

// ErrNotSelect 表示语句不是只读查询。这是“能否执行”的开关，不是权限判断。
var ErrNotSelect = errors.New("仅支持 SELECT/WITH 查询语句")

// AvailableTables 返回该角色本次请求里可见的物化表名（报错提示用）。
func AvailableTables(id Identity) []string {
	switch id.Role {
	case RoleStudent:
		return []string{TblMyProfile, TblMyGrades, TblMyEnrollments, TblMyCredits}
	case RoleTeacher:
		return []string{TblMyTeachingClasses, TblClassGrades, TblClassRoster, TblClassStats}
	case RoleCounselor:
		return []string{TblCounselorClasses, TblClassRoster, TblClassStats}
	case RoleAdmin:
		return []string{TblStudentsAll, TblClassRoster, TblClassStats}
	default:
		return nil
	}
}

// Query 在按角色物化的内存视图上执行一条只读 SQL。
// 每次调用都新开一个 :memory: 连接并在结束时销毁——这是“请求间不泄漏”的唯一保证。
// 执行顺序（顺序不能变）：
//  1. 语句校验（只认 SELECT/WITH 开头）
//  2. 新开 :memory: 连接
//  3. ATTACH 源库 → 按角色物化 → DETACH
//  4. 带超时执行模型 SQL，逐行字符串化，超行/超字节截断
func (s *Store) Query(ctx context.Context, id Identity, sqlText string, opt QueryOptions) (*QueryResult, error) {
	// 1) 语句校验：只读开关。
	head := strings.TrimSpace(sqlText)
	if len(head) > 0 && head[0] == ';' {
		head = strings.TrimSpace(head[1:])
	}
	upper := strings.ToUpper(head)
	if !strings.HasPrefix(upper, "SELECT") && !strings.HasPrefix(upper, "WITH") {
		return nil, fmt.Errorf("%w: 查询必须以 SELECT 或 WITH 开头", ErrNotSelect)
	}

	// 2) 全新内存连接：每次调用必须新开。
	mem, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("打开物化内存库失败: %w", err)
	}
	mem.SetMaxOpenConns(1)
	defer mem.Close()
	if _, err := mem.ExecContext(ctx, `ATTACH DATABASE ? AS src`, s.path); err != nil {
		return nil, fmt.Errorf("挂载教务库失败: %w", err)
	}

	// 3) 按角色物化（权限在此落地：模型只见到这些视图）。
	if err := materialize(ctx, mem, id); err != nil {
		return nil, err
	}
	if _, err := mem.ExecContext(ctx, `DETACH DATABASE src`); err != nil {
		return nil, fmt.Errorf("卸载教务库失败: %w", err)
	}

	// 4) 执行模型 SQL。
	if opt.MaxRows <= 0 {
		opt.MaxRows = 200
	}
	if opt.MaxBytes <= 0 {
		opt.MaxBytes = 32 * 1024
	}
	if opt.Timeout <= 0 {
		opt.Timeout = 5 * time.Second
	}
	runCtx, cancel := context.WithTimeout(ctx, opt.Timeout)
	defer cancel()

	start := time.Now()
	rows, err := mem.QueryContext(runCtx, sqlText)
	if err != nil {
		return nil, fmt.Errorf("查询失败(可用表: %s): %w", strings.Join(AvailableTables(id), ", "), err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("读取列名失败: %w", err)
	}
	res := &QueryResult{Columns: cols}
	totalBytes := 0
	for rows.Next() {
		if len(res.Rows) >= opt.MaxRows {
			res.Truncated = true
			break
		}
		raw := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range raw {
			ptrs[i] = &raw[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, fmt.Errorf("读取行失败: %w", err)
		}
		row := make([]string, len(cols))
		rowBytes := 0
		for i, v := range raw {
			row[i] = stringify(v)
			rowBytes += len(row[i])
		}
		if totalBytes+rowBytes > opt.MaxBytes {
			res.Truncated = true
			break
		}
		totalBytes += rowBytes
		res.Rows = append(res.Rows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历结果失败: %w", err)
	}
	res.Elapsed = time.Since(start)
	return res, nil
}

// materialize 在 mem 连接里按角色建物化表。每条 CREATE TABLE ... AS SELECT
// 都从 src.<物理表> 取数，WHERE 条件就是数据行权限。
func materialize(ctx context.Context, mem *sql.DB, id Identity) error {
	type stmt struct{ sql string }
	var stmts []stmt

	switch id.Role {
	case RoleStudent:
		cond := "st.id = ?"
		args := []any{id.StudentID}
		_ = cond
		_ = args
		stmts = []stmt{
			{`CREATE TABLE my_profile AS
			  SELECT st.id, st.student_no, st.name, cl.name AS class_name, mj.name AS major_name,
			         cl.grade_year, st.enroll_year, st.status, st.weighted_gpa, st.earned_credits
			  FROM src.student st
			  JOIN src.class cl ON cl.id = st.class_id
			  JOIN src.major mj ON mj.id = cl.major_id
			  WHERE st.id = ` + fmt.Sprint(id.StudentID)},
			{`CREATE TABLE my_grades AS
			  SELECT co.name AS course_name, co.credits, tc.term, g.usual_score, g.final_score,
			         g.total_score, g.gpa, g.is_retake
			  FROM src.grade g
			  JOIN src.enrollment e ON e.id = g.enrollment_id
			  JOIN src.teaching_class tc ON tc.id = e.teaching_class_id
			  JOIN src.course co ON co.id = tc.course_id
			  WHERE e.student_id = ` + fmt.Sprint(id.StudentID)},
			{`CREATE TABLE my_enrollments AS
			  SELECT co.name AS course_name, tc.term, tc.weekday, tc.periods, tc.location, e.status
			  FROM src.enrollment e
			  JOIN src.teaching_class tc ON tc.id = e.teaching_class_id
			  JOIN src.course co ON co.id = tc.course_id
			  WHERE e.student_id = ` + fmt.Sprint(id.StudentID)},
			{`CREATE TABLE my_credits AS
			  SELECT pr.category, COALESCE(SUM(CASE WHEN g.gpa > 0 THEN co.credits ELSE 0 END), 0) AS earned_credits,
			         pr.required_credits
			  FROM src.program_requirement pr
			  LEFT JOIN src.student st ON st.class_id IN (SELECT id FROM src.class WHERE major_id = pr.major_id AND st.enroll_year = pr.enroll_year)
			  LEFT JOIN src.enrollment e ON e.student_id = st.id
			  LEFT JOIN src.grade g ON g.enrollment_id = e.id
			  LEFT JOIN src.teaching_class tc ON tc.id = e.teaching_class_id
			  LEFT JOIN src.course co ON co.id = tc.course_id AND co.category = pr.category
			  WHERE pr.major_id = (SELECT major_id FROM src.class WHERE id = (SELECT class_id FROM src.student WHERE id = ` + fmt.Sprint(id.StudentID) + `))
			    AND pr.enroll_year = (SELECT enroll_year FROM src.student WHERE id = ` + fmt.Sprint(id.StudentID) + `)
			  GROUP BY pr.category, pr.required_credits`},
		}
	case RoleTeacher:
		cond := fmt.Sprintf(`tc.id IN (SELECT teaching_class_id FROM src.teaching_class_teacher WHERE teacher_id = %d)`, id.TeacherID)
		stmts = []stmt{
			{`CREATE TABLE my_teaching_classes AS
			  SELECT tc.id, co.name AS course_name, co.credits, tc.term, tc.weekday, tc.periods, tc.location, tc.capacity,
			         (SELECT COUNT(*) FROM src.enrollment e WHERE e.teaching_class_id = tc.id) AS enrolled_count
			  FROM src.teaching_class tc
			  JOIN src.course co ON co.id = tc.course_id
			  WHERE ` + cond},
			{`CREATE TABLE class_grades AS
			  SELECT tc.id AS teaching_class_id, co.name AS course_name, tc.term,
			         st.student_no, st.name AS student_name, g.usual_score, g.final_score, g.total_score, g.gpa, g.is_retake
			  FROM src.grade g
			  JOIN src.enrollment e ON e.id = g.enrollment_id
			  JOIN src.student st ON st.id = e.student_id
			  JOIN src.teaching_class tc ON tc.id = e.teaching_class_id
			  JOIN src.course co ON co.id = tc.course_id
			  WHERE ` + cond},
			{`CREATE TABLE class_roster AS
			  SELECT tc.id AS teaching_class_id, co.name AS course_name, tc.term,
			         st.student_no, st.name AS student_name, st.status, e.status AS enroll_status
			  FROM src.enrollment e
			  JOIN src.student st ON st.id = e.student_id
			  JOIN src.teaching_class tc ON tc.id = e.teaching_class_id
			  JOIN src.course co ON co.id = tc.course_id
			  WHERE ` + cond},
			{`CREATE TABLE class_stats AS
			  SELECT tc.id AS teaching_class_id, co.name AS course_name, tc.term,
			         COUNT(g.id) AS graded_count,
			         ROUND(AVG(g.total_score), 2) AS avg_score,
			         ROUND(100.0 * SUM(CASE WHEN g.total_score >= 60 THEN 1 ELSE 0 END) / COUNT(g.id), 1) AS pass_rate
			  FROM src.grade g
			  JOIN src.enrollment e ON e.id = g.enrollment_id
			  JOIN src.teaching_class tc ON tc.id = e.teaching_class_id
			  JOIN src.course co ON co.id = tc.course_id
			  WHERE ` + cond + ` GROUP BY tc.id, co.name, tc.term`},
		}
	case RoleCounselor:
		cond := fmt.Sprintf(`st.class_id IN (SELECT id FROM src.class WHERE counselor_id = %d)`, id.CounselorID)
		stmts = []stmt{
			{`CREATE TABLE counselor_classes AS
			  SELECT cl.id AS class_id, cl.name AS class_name, cl.grade_year, mj.name AS major_name,
			         (SELECT COUNT(*) FROM src.student st WHERE st.class_id = cl.id) AS student_count
			  FROM src.class cl
			  JOIN src.major mj ON mj.id = cl.major_id
			  WHERE cl.counselor_id = ` + fmt.Sprint(id.CounselorID)},
			{`CREATE TABLE class_roster AS
			  SELECT cl.name AS class_name, st.student_no, st.name AS student_name, st.status,
			         st.weighted_gpa, st.earned_credits
			  FROM src.student st
			  JOIN src.class cl ON cl.id = st.class_id
			  WHERE ` + cond},
			{`CREATE TABLE class_stats AS
			  SELECT cl.name AS class_name,
			         COUNT(st.id) AS student_count,
			         ROUND(AVG(st.weighted_gpa), 2) AS avg_gpa,
			         SUM(CASE WHEN st.status = 'suspend' THEN 1 ELSE 0 END) AS suspended_count
			  FROM src.student st
			  JOIN src.class cl ON cl.id = st.class_id
			  WHERE ` + cond + ` GROUP BY cl.id, cl.name`},
		}
	case RoleAdmin:
		stmts = []stmt{
			{`CREATE TABLE students_all AS SELECT * FROM src.student`},
			{`CREATE TABLE class_roster AS
			  SELECT cl.name AS class_name, st.student_no, st.name AS student_name, st.status,
			         st.weighted_gpa, st.earned_credits
			  FROM src.student st JOIN src.class cl ON cl.id = st.class_id`},
			{`CREATE TABLE class_stats AS
			  SELECT cl.name AS class_name,
			         COUNT(st.id) AS student_count,
			         ROUND(AVG(st.weighted_gpa), 2) AS avg_gpa,
			         SUM(CASE WHEN st.status = 'suspend' THEN 1 ELSE 0 END) AS suspended_count
			  FROM src.student st JOIN src.class cl ON cl.id = st.class_id
			  GROUP BY cl.id, cl.name`},
		}
	default:
		return fmt.Errorf("未知教务角色 %q，请先绑定教务身份", string(id.Role))
	}

	for _, s := range stmts {
		if _, err := mem.ExecContext(ctx, s.sql); err != nil {
			return fmt.Errorf("物化失败(%s…): %w", firstWords(s.sql, 40), err)
		}
	}
	return nil
}

// stringify 把任意 SQLite 值转成模型可读的字符串。
func stringify(v any) string {
	switch x := v.(type) {
	case nil:
		return "NULL"
	case []byte:
		return string(x)
	default:
		return fmt.Sprintf("%v", x)
	}
}

// firstWords 截取 SQL 前 n 个字符用于错误定位。
func firstWords(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) > n {
		r = r[:n]
	}
	return string(r)
}
