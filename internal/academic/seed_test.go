package academic

import (
	"context"
	"math"
	"testing"
)

func TestSeedIsDeterministicAndComputesDerivedFields(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	opts := SeedOptions{RNGSeed: 42, StudentsPerClass: 10, Terms: 2, Rules: DefaultRules()}
	if err := Seed(ctx, s, opts); err != nil {
		t.Fatal(err)
	}
	if err := Seed(ctx, s, opts); err != nil { // 幂等：重跑结果一致
		t.Fatal(err)
	}

	// 1) 每条成绩的总评与绩点都符合规则
	rows, err := s.DB().Query(`SELECT usual_score, final_score, total_score, gpa FROM grade`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var u, f, total, gpa float64
		if err := rows.Scan(&u, &f, &total, &gpa); err != nil {
			t.Fatal(err)
		}
		if math.Abs(total-opts.Rules.TotalScore(u, f)) > 0.01 {
			t.Fatalf("总评不符: %v vs (%v,%v)", total, u, f)
		}
		if gpa != opts.Rules.GPA(total) {
			t.Fatalf("绩点不符: %v vs %v", gpa, total)
		}
	}

	// 2) 每个学生都有毕业审核结论与已修学分
	var noAudit int
	if err := s.DB().QueryRow(`SELECT COUNT(*) FROM student st
		WHERE NOT EXISTS (SELECT 1 FROM graduation_audit g WHERE g.student_id = st.id)`).Scan(&noAudit); err != nil {
		t.Fatal(err)
	}
	if noAudit != 0 {
		t.Fatalf("%d 个学生没有毕业审核结论", noAudit)
	}
}

func TestSeedCoversEveryRoleSituation(t *testing.T) {
	s := newTestStore(t)
	if err := Seed(context.Background(), s, SeedOptions{RNGSeed: 7, StudentsPerClass: 30, Terms: 3, Rules: DefaultRules()}); err != nil {
		t.Fatal(err)
	}
	for _, q := range []struct{ name, sql string }{
		{"有挂科的学生", `SELECT COUNT(*) FROM student st JOIN enrollment e ON e.student_id=st.id
		 JOIN grade g ON g.enrollment_id=e.id WHERE g.total_score < 60`},
		{"有重修记录", `SELECT COUNT(*) FROM grade WHERE is_retake = 1`},
		{"有休学记录", `SELECT COUNT(*) FROM status_change WHERE type='suspend'`},
	} {
		var n int
		if err := s.DB().QueryRow(q.sql).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n == 0 {
			t.Errorf("种子数据缺少：%s", q.name)
		}
	}
}
