// academic-seed 一条命令铺好教务演示环境：
//
//  1. 教务库（独立 SQLite）：建表 + 灌入确定性种子；
//  2. WeKnora 主库（PostgreSQL）：班级空间/组织/共享 幂等调和；
//  3. WeKnora 主库：演示账号幂等调和（现有保留，缺的按可读名补建）；
//  4. academic_identity：把每个演示账号绑定到教务库中对应班级的学生。
//
// 用法：
//
//	go run ./cmd/academic-seed -db ./data/academic.db -seed 42 -per-class 30 -terms 3
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/Tencent/WeKnora/internal/academic"
	"github.com/Tencent/WeKnora/internal/types"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	dbPath := flag.String("db", "./data/academic.db", "教务库 SQLite 文件路径")
	dsn := flag.String("dsn", os.Getenv("DB_DSN"), "WeKnora 主库 DSN（缺省读 DB_DSN 环境变量）")
	rng := flag.Int64("seed", 42, "种子随机数（相同种子重跑结果一致）")
	perClass := flag.Int("per-class", 30, "每班学生数")
	terms := flag.Int("terms", 3, "生成几个学期的开课/选课/成绩")
	password := flag.String("password", academic.DemoPassword, "演示账号统一密码")
	flag.Parse()

	ctx := context.Background()
	if err := run(ctx, *dbPath, *dsn, *rng, *perClass, *terms, *password); err != nil {
		fmt.Fprintln(os.Stderr, "academic-seed:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, dbPath, dsn string, rngSeed int64, perClass, terms int, password string) error {
	if dsn == "" {
		return fmt.Errorf("缺少主库 DSN：传 -dsn 或设置 DB_DSN")
	}

	// ---------- 1) 教务库 ----------
	store, err := academic.Open(dbPath)
	if err != nil {
		return err
	}
	defer store.Close()
	if err := store.Migrate(ctx); err != nil {
		return err
	}
	if err := academic.Seed(ctx, store, academic.SeedOptions{
		RNGSeed:          rngSeed,
		StudentsPerClass: perClass,
		Terms:            terms,
		Rules:            academic.DefaultRules(),
	}); err != nil {
		return fmt.Errorf("教务库种子失败: %w", err)
	}
	fmt.Printf("✔ 教务库就绪: %s\n", dbPath)

	// ---------- 2) WeKnora 主库：空间与组织 ----------
	gdb, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("连接主库失败: %w", err)
	}
	spaces, err := academic.SeedSpaces(ctx, gdb, academic.SpaceOptions{})
	if err != nil {
		return fmt.Errorf("空间调和失败: %w", err)
	}
	fmt.Printf("✔ 空间就绪: 教务处=%d, 班级=%d 个, 组织=%s\n", spaces.Dean, len(spaces.Classes), spaces.OrgID)

	// ---------- 3) 演示账号 ----------
	report, err := academic.SeedAccounts(ctx, gdb, spaces, academic.AccountOptions{Password: password})
	if err != nil {
		return fmt.Errorf("账号调和失败: %w", err)
	}
	fmt.Printf("✔ 账号就绪: 新建 %v, 跳过(已存在) %v\n", report.NewAccounts, report.Skipped)

	// ---------- 4) academic_identity 绑定 ----------
	bound, err := bindDemoIdentities(ctx, store, gdb, spaces)
	if err != nil {
		return fmt.Errorf("身份绑定失败: %w", err)
	}
	fmt.Printf("✔ 身份绑定: %d 条\n", bound)

	fmt.Println("\n演示账号（密码统一 " + password + "）:")
	fmt.Println("  s001@askademic.local        计科2401 学生(现有)")
	fmt.Println("  student-001@askademic.local 计科2402 学生")
	fmt.Println("  student-002@askademic.local 软工2401 学生")
	fmt.Println("  student-003@askademic.local 网工2401 学生")
	fmt.Println("  t001@askademic.local        计科2401+2402 教师(现有)")
	fmt.Println("  teacher-001@askademic.local 软工2401 教师")
	fmt.Println("  counselor-001@askademic.local 计科2401 辅导员")
	fmt.Println("  jw01@askademic.local        教务处 管理员(现有)")
	fmt.Println("\n提示: 要关闭自助注册(仅管理员/邀请建号), 在 .env 中设置 DISABLE_REGISTRATION=true 后重启。")
	return nil
}

// bindDemoIdentities 把演示账号绑到教务库对应班级的第一个学生：
//   - 学生账号（s001→计科2401, student-001→计科2402, …）绑该班 1 号学生；
//   - 教师账号（t001/teacher-001）绑教师表里的一名教师；
//   - 辅导员账号（counselor-001）绑辅导员表里带计科2401 的辅导员；
//   - jw01/demo 绑教务管理员（全量视图）。
//
// 幂等：重复绑定即覆盖。
func bindDemoIdentities(ctx context.Context, store *academic.Store, gdb *gorm.DB, spaces academic.SpaceMap) (int, error) {
	db := store.DB()

	// 找某班第一个学生的 id。
	firstStudent := func(className string) (int64, error) {
		var id int64
		err := db.QueryRowContext(ctx, `
			SELECT st.id FROM student st JOIN class cl ON cl.id = st.class_id
			WHERE cl.name = ? ORDER BY st.id LIMIT 1`, className).Scan(&id)
		return id, err
	}
	// 找 WeKnora 用户 id。
	userID := func(email string) (string, error) {
		var u types.User
		if err := gdb.WithContext(ctx).Where("email = ?", email).First(&u).Error; err != nil {
			return "", err
		}
		return u.ID, nil
	}

	bind := func(email string, id academic.Identity) error {
		uid, err := userID(email)
		if err != nil {
			return fmt.Errorf("账号 %s 不存在(先跑账号调和): %w", email, err)
		}
		return store.BindIdentity(ctx, uid, id)
	}

	// 学生：s001 + student-001~003
	studentPlan := []struct {
		email, className string
	}{
		{"s001@askademic.local", "计科2401班"},
		{"student-001@askademic.local", "计科2402班"},
		{"student-002@askademic.local", "软工2401班"},
		{"student-003@askademic.local", "网工2401班"},
	}
	count := 0
	for _, p := range studentPlan {
		sid, err := firstStudent(p.className)
		if err != nil {
			return count, fmt.Errorf("教务库缺班级 %s 的学生: %w", p.className, err)
		}
		if err := bind(p.email, academic.Identity{Role: academic.RoleStudent, StudentID: sid}); err != nil {
			return count, err
		}
		count++
	}

	// 教师：t001(计科2401+2402 的任课教师)、teacher-001(软工2401)——
	// 各绑教师表第一位出现在对应班任课记录里的教师。
	firstTeacherOf := func(className string) (int64, error) {
		var id int64
		err := db.QueryRowContext(ctx, `
			SELECT DISTINCT tct.teacher_id
			FROM teaching_class_teacher tct
			JOIN teaching_class tc ON tc.id = tct.teaching_class_id
			JOIN enrollment e ON e.teaching_class_id = tc.id
			JOIN student st ON st.id = e.student_id
			JOIN class cl ON cl.id = st.class_id
			WHERE cl.name = ?
			ORDER BY tct.teacher_id LIMIT 1`, className).Scan(&id)
		return id, err
	}
	teacherPlan := []struct {
		email, className string
	}{
		{"t001@askademic.local", "计科2401班"},
		{"teacher-001@askademic.local", "软工2401班"},
	}
	for _, p := range teacherPlan {
		tid, err := firstTeacherOf(p.className)
		if err != nil {
			return count, fmt.Errorf("教务库缺班级 %s 的任课教师: %w", p.className, err)
		}
		if err := bind(p.email, academic.Identity{Role: academic.RoleTeacher, TeacherID: tid}); err != nil {
			return count, err
		}
		count++
	}

	// 辅导员：counselor-001 绑带计科2401 的辅导员。
	var counselorID int64
	if err := db.QueryRowContext(ctx, `
		SELECT cl.counselor_id FROM class cl WHERE cl.name = '计科2401班' AND cl.counselor_id IS NOT NULL LIMIT 1`).
		Scan(&counselorID); err != nil {
		return count, fmt.Errorf("教务库缺计科2401班辅导员: %w", err)
	}
	if err := bind("counselor-001@askademic.local", academic.Identity{Role: academic.RoleCounselor, CounselorID: counselorID}); err != nil {
		return count, err
	}
	count++

	// 教务管理员：jw01 + demo。
	for _, email := range []string{"jw01@askademic.local", "demo@askademic.local"} {
		if err := bind(email, academic.Identity{Role: academic.RoleAdmin}); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}
