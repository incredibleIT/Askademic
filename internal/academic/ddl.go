package academic

// 教务库全部表名常量。除本包外，任何代码不得拼接这些标识符。
const (
	TableDepartment           = "department"
	TableMajor                = "major"
	TableCounselor            = "counselor"
	TableClass                = "class"
	TableTeacher              = "teacher"
	TableCourse               = "course"
	TableTeachingClass        = "teaching_class"
	TableTeachingClassTeacher = "teaching_class_teacher"
	TableStudent              = "student"
	TableEnrollment           = "enrollment"
	TableGrade                = "grade"
	TableProgramRequirement   = "program_requirement"
	TableGraduationAudit      = "graduation_audit"
	TableStatusChange         = "status_change"
	TableAcademicIdentity     = "academic_identity"
)

// 表结构对齐设计文档 §7.1（teaching_class_teacher 为 §7.4 补充的合上关联表；
// academic_identity 额外带 counselor_id 以承载辅导员身份绑定）。
// 全部语句幂等，可重复执行。
var ddl = []string{
	// 院系
	`CREATE TABLE IF NOT EXISTS department (
	  id   INTEGER PRIMARY KEY,
	  name TEXT NOT NULL UNIQUE
	)`,

	// 专业：挂在院系下，years 为学制年数
	`CREATE TABLE IF NOT EXISTS major (
	  id            INTEGER PRIMARY KEY,
	  name          TEXT NOT NULL UNIQUE,
	  department_id INTEGER NOT NULL REFERENCES department(id),
	  years         INTEGER NOT NULL DEFAULT 4
	)`,

	// 辅导员（先于 class 建出，class.counselor_id 指向它）
	`CREATE TABLE IF NOT EXISTS counselor (
	  id           INTEGER PRIMARY KEY,
	  staff_no     TEXT NOT NULL UNIQUE,
	  name         TEXT NOT NULL,
	  department_id INTEGER NOT NULL REFERENCES department(id)
	)`,

	// 行政班：一个班对应一个班级空间；counselor_id 为带班辅导员
	`CREATE TABLE IF NOT EXISTS class (
	  id           INTEGER PRIMARY KEY,
	  name         TEXT NOT NULL UNIQUE,
	  major_id     INTEGER NOT NULL REFERENCES major(id),
	  grade_year   INTEGER NOT NULL,
	  counselor_id INTEGER REFERENCES counselor(id)
	)`,

	// 教师
	`CREATE TABLE IF NOT EXISTS teacher (
	  id            INTEGER PRIMARY KEY,
	  teacher_no    TEXT NOT NULL UNIQUE,
	  name          TEXT NOT NULL,
	  department_id INTEGER NOT NULL REFERENCES department(id)
	)`,

	// 课程：category ∈ 必修/选修/通识/实践
	`CREATE TABLE IF NOT EXISTS course (
	  id        INTEGER PRIMARY KEY,
	  course_no TEXT NOT NULL UNIQUE,
	  name      TEXT NOT NULL,
	  credits   REAL NOT NULL,
	  hours     INTEGER NOT NULL DEFAULT 0,
	  category  TEXT NOT NULL DEFAULT '必修'
	)`,

	// 教学班：某学期的一次开课；teacher_id 为主讲（合上见关联表）
	`CREATE TABLE IF NOT EXISTS teaching_class (
	  id        INTEGER PRIMARY KEY,
	  course_id INTEGER NOT NULL REFERENCES course(id),
	  term      TEXT NOT NULL,
	  teacher_id INTEGER REFERENCES teacher(id),
	  weekday   INTEGER NOT NULL DEFAULT 1,
	  periods   INTEGER NOT NULL DEFAULT 1,
	  location  TEXT NOT NULL DEFAULT '',
	  capacity  INTEGER NOT NULL DEFAULT 0
	)`,

	// 合上关联表：“所带教学班”以本表为准（§7.4）
	`CREATE TABLE IF NOT EXISTS teaching_class_teacher (
	  teaching_class_id INTEGER NOT NULL REFERENCES teaching_class(id),
	  teacher_id        INTEGER NOT NULL REFERENCES teacher(id),
	  PRIMARY KEY (teaching_class_id, teacher_id)
	)`,

	// 学生：绩点与已修学分在写入时算好存这里（§7.6）
	`CREATE TABLE IF NOT EXISTS student (
	  id             INTEGER PRIMARY KEY,
	  student_no     TEXT NOT NULL UNIQUE,
	  name           TEXT NOT NULL,
	  class_id       INTEGER NOT NULL REFERENCES class(id),
	  enroll_year    INTEGER NOT NULL,
	  status         TEXT NOT NULL DEFAULT 'active',
	  weighted_gpa   REAL NOT NULL DEFAULT 0,
	  earned_credits REAL NOT NULL DEFAULT 0
	)`,

	// 选课记录
	`CREATE TABLE IF NOT EXISTS enrollment (
	  id               INTEGER PRIMARY KEY,
	  student_id       INTEGER NOT NULL REFERENCES student(id),
	  teaching_class_id INTEGER NOT NULL REFERENCES teaching_class(id),
	  status           TEXT NOT NULL DEFAULT 'enrolled'
	)`,

	// 成绩：total_score 与 gpa 在写入时算好；is_retake 标记重修
	`CREATE TABLE IF NOT EXISTS grade (
	  id            INTEGER PRIMARY KEY,
	  enrollment_id INTEGER NOT NULL REFERENCES enrollment(id),
	  usual_score   REAL NOT NULL DEFAULT 0,
	  final_score   REAL NOT NULL DEFAULT 0,
	  total_score   REAL NOT NULL DEFAULT 0,
	  gpa           REAL NOT NULL DEFAULT 0,
	  is_retake     INTEGER NOT NULL DEFAULT 0
	)`,

	// 培养方案：某专业某入学年份某类课要求多少学分
	`CREATE TABLE IF NOT EXISTS program_requirement (
	  id               INTEGER PRIMARY KEY,
	  major_id         INTEGER NOT NULL REFERENCES major(id),
	  enroll_year      INTEGER NOT NULL,
	  category         TEXT NOT NULL,
	  required_credits REAL NOT NULL
	)`,

	// 毕业审核结论：批量预计算，一个学生一行
	`CREATE TABLE IF NOT EXISTS graduation_audit (
	  id           INTEGER PRIMARY KEY,
	  student_id   INTEGER NOT NULL UNIQUE REFERENCES student(id),
	  conclusion   TEXT NOT NULL,
	  missing_json TEXT NOT NULL DEFAULT '[]',
	  computed_at  TEXT NOT NULL
	)`,

	// 学籍异动：休学/复学/转专业
	`CREATE TABLE IF NOT EXISTS status_change (
	  id           INTEGER PRIMARY KEY,
	  student_id   INTEGER NOT NULL REFERENCES student(id),
	  type         TEXT NOT NULL,
	  effective_at TEXT NOT NULL,
	  reason       TEXT NOT NULL DEFAULT ''
	)`,

	// WeKnora 用户 ↔ 教务身份绑定；查不到即 ErrIdentityNotFound，绝不回落管理员
	`CREATE TABLE IF NOT EXISTS academic_identity (
	  id              INTEGER PRIMARY KEY,
	  weknora_user_id TEXT NOT NULL UNIQUE,
	  role            TEXT NOT NULL,
	  student_id      INTEGER REFERENCES student(id),
	  teacher_id      INTEGER REFERENCES teacher(id),
	  counselor_id    INTEGER REFERENCES counselor(id)
	)`,
}
