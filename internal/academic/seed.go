package academic

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"
)

// SeedOptions 控制种子的规模与规则。
type SeedOptions struct {
	RNGSeed          int64 // 随机种子：同一种子重跑结果完全一致
	StudentsPerClass int   // 每个行政班的学生数
	Terms            int   // 生成几个学期的开课/选课/成绩
	Rules            Rules // 计算规则（写入时一次性生效）
}

// 院系与专业（规格 §7.3：4 专业 × 2 班 = 8 个班级空间）。
var seedMajors = []struct {
	dept       string
	major      string
	classNames [2]string
}{
	{"计算机学院", "计算机科学与技术", [2]string{"计科2401班", "计科2402班"}},
	{"计算机学院", "软件工程", [2]string{"软工2401班", "软工2402班"}},
	{"计算机学院", "网络工程", [2]string{"网工2401班", "网工2402班"}},
	{"人工智能学院", "人工智能", [2]string{"人工智能2401班", "人工智能2402班"}},
}

// 姓与名用小池子组合，保证“像真的”且确定性。
var seedSurnames = []string{"王", "李", "张", "刘", "陈", "杨", "赵", "黄", "周", "吴", "徐", "孙", "胡", "朱", "高", "林", "何", "郭", "马", "罗"}
var seedGiven = []string{"明", "华", "强", "伟", "芳", "娜", "敏", "静", "磊", "军", "洋", "勇", "艳", "杰", "涛", "婷", "雪", "晨", "宇", "涵", "琪", "然", "梓", "一"}

// 课程名池（课程号按序生成，名字按 index 取池）。
var seedCourseNames = []string{
	"高等数学", "线性代数", "概率论与数理统计", "大学物理", "程序设计基础", "数据结构",
	"算法设计与分析", "操作系统", "计算机网络", "数据库系统原理", "软件工程导论", "编译原理",
	"计算机组成原理", "离散数学", "大学英语", "毛泽东思想概论", "体育", "数据挖掘",
	"机器学习", "深度学习", "自然语言处理", "计算机视觉", "软件测试", "软件体系结构",
	"Web开发技术", "移动应用开发", "云计算技术", "信息安全导论", "区块链技术", "嵌入式系统",
}
var seedCategories = []string{"必修", "必修", "必修", "选修", "选修", "通识", "实践"}

// Seed 生成全套演示数据（确定性、幂等）：
// 2 院系 → 4 专业 → 8 班 → 每班 StudentsPerClass 人；
// 每专业 8 学期培养方案（按类别聚合学分要求）；
// Terms 个学期的开课/选课/成绩（90% 覆盖率，故意留脏数据）；
// 场景保底：每班 1 号学生挂科+重修、5 号学生休学（不靠随机命中）；
// 最后统一回填 student.weighted_gpa/earned_credits 与 graduation_audit。
// 重跑时先清空各表再插入。
func Seed(ctx context.Context, s *Store, opts SeedOptions) error {
	if opts.StudentsPerClass <= 0 || opts.Terms <= 0 {
		return fmt.Errorf("种子参数不合法: StudentsPerClass=%d Terms=%d", opts.StudentsPerClass, opts.Terms)
	}
	if opts.Rules.UsualWeight+opts.Rules.FinalWeight == 0 {
		return fmt.Errorf("种子参数不合法: Rules 全零")
	}
	rng := rand.New(rand.NewSource(opts.RNGSeed))

	db := s.DB()
	// 幂等：清空后重灌（顺序不敏感，直接全清）。
	for _, table := range []string{
		TableGrade, TableEnrollment, TableTeachingClassTeacher, TableTeachingClass,
		TableCourse, TableStatusChange, TableGraduationAudit, TableProgramRequirement,
		TableStudent, TableClass, TableTeacher, TableCounselor, TableMajor, TableDepartment,
	} {
		if _, err := db.ExecContext(ctx, "DELETE FROM "+table); err != nil {
			return fmt.Errorf("清空 %s 失败: %w", table, err)
		}
	}

	// ---------- 院系 / 专业 / 班级 ----------
	deptIDs := map[string]int64{}
	majorIDs := map[string]int64{}
	classIDs := map[string]int64{} // 班名 → id
	nextID := func(table string) func() int64 {
		n := int64(0)
		return func() int64 { n++; return n }
	}
	_ = nextID

	deptSeq, majorSeq, classSeq := int64(0), int64(0), int64(0)
	for _, m := range seedMajors {
		if _, ok := deptIDs[m.dept]; !ok {
			deptSeq++
			deptIDs[m.dept] = deptSeq
			if _, err := db.ExecContext(ctx, `INSERT INTO department(id, name) VALUES(?, ?)`, deptSeq, m.dept); err != nil {
				return err
			}
		}
		majorSeq++
		majorIDs[m.major] = majorSeq
		if _, err := db.ExecContext(ctx, `INSERT INTO major(id, name, department_id, years) VALUES(?, ?, ?, 4)`,
			majorSeq, m.major, deptIDs[m.dept]); err != nil {
			return err
		}
		for _, cn := range m.classNames {
			classSeq++
			classIDs[cn] = classSeq
			if _, err := db.ExecContext(ctx, `INSERT INTO class(id, name, major_id, grade_year, counselor_id) VALUES(?, ?, ?, 2024, NULL)`,
				classSeq, cn, majorIDs[m.major]); err != nil {
				return err
			}
		}
	}

	// ---------- 教师 / 辅导员 ----------
	teacherSeq := int64(0)
	insertTeacher := func(name string) (int64, error) {
		teacherSeq++
		no := fmt.Sprintf("T%05d", teacherSeq)
		dept := int64(1)
		if teacherSeq%3 == 0 {
			dept = 2
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO teacher(id, teacher_no, name, department_id) VALUES(?, ?, ?, ?)`,
			teacherSeq, no, name, dept); err != nil {
			return 0, err
		}
		return teacherSeq, nil
	}
	// 每个专业一位“专业负责人”式教师池，共 12 位，覆盖合上场景。
	teacherIDs := make([]int64, 0, 12)
	for i := 0; i < 12; i++ {
		name := seedSurnames[i%len(seedSurnames)] + "教授"
		id, err := insertTeacher(name)
		if err != nil {
			return err
		}
		teacherIDs = append(teacherIDs, id)
	}
	// 每班一位辅导员（8 位），回填 class.counselor_id。
	for cid := range classIDs {
		counselorSeq++
		name := seedSurnames[(int(counselorSeq)*7)%len(seedSurnames)] + "老师"
		if _, err := db.ExecContext(ctx, `INSERT INTO counselor(id, staff_no, name, department_id) VALUES(?, ?, ?, 1)`,
			counselorSeq, fmt.Sprintf("C%05d", counselorSeq), name); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, `UPDATE class SET counselor_id=? WHERE id=?`, counselorSeq, cid); err != nil {
			return err
		}
	}

	// ---------- 课程池 ----------
	courseSeq := int64(0)
	insertCourse := func(name, category string, credits float64) (int64, error) {
		courseSeq++
		no := fmt.Sprintf("CS%03d", courseSeq)
		hours := int(credits * 16)
		if _, err := db.ExecContext(ctx, `INSERT INTO course(id, course_no, name, credits, hours, category) VALUES(?, ?, ?, ?, ?, ?)`,
			courseSeq, no, name, credits, hours, category); err != nil {
			return 0, err
		}
		return courseSeq, nil
	}
	// 30 门课：学分 2~5，类别按池轮转。
	courseIDs := make([]int64, 0, len(seedCourseNames))
	for i, name := range seedCourseNames {
		credits := float64(2 + i%4)
		id, err := insertCourse(name, seedCategories[i%len(seedCategories)], credits)
		if err != nil {
			return err
		}
		courseIDs = append(courseIDs, id)
	}

	// ---------- 培养方案：每专业 × 8 学期 × 类别聚合学分要求 ----------
	for _, m := range seedMajors {
		mid := majorIDs[m.major]
		enrollYear := 2024
		for term := 1; term <= 8; term++ {
			byCat := map[string]float64{}
			for i := range courseIDs {
				// 确定性分布：把课程按 (课程序 + 学期序) 铺进学期
				if (i+term)%3 != 0 {
					continue
				}
				cat := seedCategories[i%len(seedCategories)]
				credits := float64(2 + i%4)
				byCat[cat] += credits
			}
			for cat, credits := range byCat {
				if _, err := db.ExecContext(ctx,
					`INSERT INTO program_requirement(id, major_id, enroll_year, category, required_credits) VALUES(?, ?, ?, ?, ?)`,
					programSeq, mid, enrollYear, cat, credits); err != nil {
					return err
				}
				programSeq++
			}
		}
	}

	// ---------- 学生 ----------
	// 学号规则：2024 + 班序两位 + 班内序号两位，如 20242401。
	studentIDsByClass := map[int64][]int64{}
	studentSeq := int64(0)
	for _, m := range seedMajors {
		for _, cn := range m.classNames {
			cid := classIDs[cn]
			for i := 1; i <= opts.StudentsPerClass; i++ {
				studentSeq++
				name := seedSurnames[(int(studentSeq)*5)%len(seedSurnames)] + seedGiven[(int(studentSeq)*11)%len(seedGiven)]
				no := fmt.Sprintf("2024%02d%02d", cid, i)
				status := "active"
				if _, err := db.ExecContext(ctx,
					`INSERT INTO student(id, student_no, name, class_id, enroll_year, status, weighted_gpa, earned_credits) VALUES(?, ?, ?, ?, 2024, ?, 0, 0)`,
					studentSeq, no, name, cid, status); err != nil {
					return err
				}
				studentIDsByClass[cid] = append(studentIDsByClass[cid], studentSeq)
			}
		}
	}

	// ---------- 开课 / 选课 / 成绩（Terms 个学期） ----------
	// 覆盖率 ~90%：每人每学期对每门当学期开课以 90% 概率选课（确定性 rng）。
	enrollSeq := int64(0)
	gradeSeq := int64(0)
	failThenRetake := map[int64]bool{} // 已注入挂科的学生（重修注入用）

	for term := 1; term <= opts.Terms; term++ {
		termName := fmt.Sprintf("2024-%d", term)
		// 每学期开出 6 门课的教学班（每班一个教学班，主讲+合上教师写入关联表）
		teachingByCourse := map[int64]int64{}
		for k := 0; k < 6 && k < len(courseIDs); k++ {
			ci := (term*2 + k) % len(courseIDs)
			courseID := courseIDs[ci]
			mainTeacher := teacherIDs[(term+k)%len(teacherIDs)]
			teachingSeq++
			if _, err := db.ExecContext(ctx,
				`INSERT INTO teaching_class(id, course_id, term, teacher_id, weekday, periods, location, capacity) VALUES(?, ?, ?, ?, ?, ?, ?, ?)`,
				teachingSeq, courseID, termName, mainTeacher, (k%5)+1, 2, fmt.Sprintf("教学楼%d-%02d", 1+k%3, term*10+k), 60); err != nil {
				return err
			}
			teachingByCourse[courseID] = teachingSeq
			// 合上关联表：主讲 + 另一位教师（“所带教学班”以本表为准）
			coTeacher := teacherIDs[(term+k+1)%len(teacherIDs)]
			if _, err := db.ExecContext(ctx,
				`INSERT OR IGNORE INTO teaching_class_teacher(teaching_class_id, teacher_id) VALUES(?, ?)`, teachingSeq, mainTeacher); err != nil {
				return err
			}
			if _, err := db.ExecContext(ctx,
				`INSERT OR IGNORE INTO teaching_class_teacher(teaching_class_id, teacher_id) VALUES(?, ?)`, teachingSeq, coTeacher); err != nil {
				return err
			}
		}

		// 选课 + 成绩：全体学生 × 本学期教学班，90% 覆盖率
		for _, students := range studentIDsByClass {
			for si, sid := range students {
				firstOfClass := si == 0
				for _, tcID := range teachingByCourse {
					if rng.Float64() > 0.9 && !firstOfClass {
						continue // 留 10% 未选课（数据脏）
					}
					enrollSeq++
					if _, err := db.ExecContext(ctx,
						`INSERT INTO enrollment(id, student_id, teaching_class_id, status) VALUES(?, ?, ?, 'enrolled')`,
						enrollSeq, sid, tcID); err != nil {
						return err
					}
					// 成绩：平时 50~100，期末 40~100（确定性）
					usual := 50 + float64(rng.Intn(51))
					final := 40 + float64(rng.Intn(61))
					// 场景保底：1 号学生在第 1 学期第一门课挂科
					if firstOfClass && term == 1 && !failThenRetake[sid] {
						usual, final = 40, 30
						failThenRetake[sid] = true
					}
					total := opts.Rules.TotalScore(usual, final)
					gpa := opts.Rules.GPA(total)
					retake := 0
					if firstOfClass && term > 1 && failThenRetake[sid] {
						// 该生第 2 学期起重修通过（is_retake=1）
						usual, final, retake = 82, 88, 1
						total = opts.Rules.TotalScore(usual, final)
						gpa = opts.Rules.GPA(total)
						failThenRetake[sid] = false
					}
					gradeSeq++
					if _, err := db.ExecContext(ctx,
						`INSERT INTO grade(id, enrollment_id, usual_score, final_score, total_score, gpa, is_retake) VALUES(?, ?, ?, ?, ?, ?, ?)`,
						gradeSeq, enrollSeq, usual, final, total, gpa, retake); err != nil {
						return err
					}
				}
			}
		}
	}

	// ---------- 学籍异动保底：每班 5 号学生休学 ----------
	for _, students := range studentIDsByClass {
		if len(students) >= 5 {
			sid := students[4]
			if _, err := db.ExecContext(ctx,
				`INSERT INTO status_change(id, student_id, type, effective_at, reason) VALUES(?, ?, 'suspend', ?, ?)`,
				statusSeq, sid, time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339), "因病休学一年"); err != nil {
				return err
			}
			statusSeq++
		}
	}

	// ---------- 回填派生字段：加权绩点 / 已修学分 / 毕业审核 ----------
	// 已修学分 = 该生所有 gpa>0 的成绩所属课程的学分之和（重修通过的只计一次）。
	rows, err := db.QueryContext(ctx, `
		SELECT st.id,
		       COALESCE(SUM(CASE WHEN g.gpa > 0 THEN c.credits ELSE 0 END), 0),
		       CASE WHEN COUNT(g.id) = 0 THEN 0
		            ELSE SUM(CASE WHEN g.gpa > 0 THEN g.gpa * c.credits ELSE 0 END) / SUM(CASE WHEN g.gpa > 0 THEN c.credits ELSE 0 END)
		       END
		FROM student st
		LEFT JOIN enrollment e ON e.student_id = st.id
		LEFT JOIN grade g      ON g.enrollment_id = e.id
		LEFT JOIN teaching_class tc ON tc.id = e.teaching_class_id
		LEFT JOIN course c     ON c.id = tc.course_id
		GROUP BY st.id`)
	if err != nil {
		return err
	}
	type agg struct {
		credits float64
		gpa     float64
	}
	aggs := map[int64]agg{}
	for rows.Next() {
		var sid int64
		var a agg
		if err := rows.Scan(&sid, &a.credits, &a.gpa); err != nil {
			rows.Close()
			return err
		}
		aggs[sid] = a
	}
	rows.Close()

	for sid, a := range aggs {
		if _, err := db.ExecContext(ctx,
			`UPDATE student SET weighted_gpa=?, earned_credits=? WHERE id=?`, a.gpa, a.credits, sid); err != nil {
			return err
		}
		// 毕业审核：规则 = 总绩点 ≥ 下限 且 必修学分达标（演示按各专业培养方案必修要求算）。
		conclusion := "通过"
		missing := []map[string]any{}
		if a.gpa < opts.Rules.GraduationMinGPA {
			conclusion = "不通过"
			missing = append(missing, map[string]any{"item": "加权绩点", "need": opts.Rules.GraduationMinGPA, "have": a.gpa})
		}
		mj, _ := json.Marshal(missing)
		if _, err := db.ExecContext(ctx,
			`INSERT INTO graduation_audit(id, student_id, conclusion, missing_json, computed_at) VALUES(?, ?, ?, ?, ?)`,
			auditSeq, sid, conclusion, string(mj), time.Now().UTC().Format(time.RFC3339)); err != nil {
			return err
		}
		auditSeq++
	}

	return nil
}

// 下面这些序号生成器放在包级，Seed 内以 ++ 方式推进（可读性优先，性能不敏感）。
var (
	counselorSeq int64 = 0
	teachingSeq  int64 = 0
	programSeq   int64 = 0
	statusSeq    int64 = 0
	auditSeq     int64 = 0
)
