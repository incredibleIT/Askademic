package academic

// Rules 是“规则只在写数据时生效一次”（设计文档 §7.6）的唯一落点：
// 及格线、总评权重、GPA 分段、毕业绩点下限全部集中在这里，
// 由种子脚本在写入时算好 total_score / gpa / weighted_gpa / earned_credits，
// 查询路径上没有任何策略判断。
type Rules struct {
	PassScore        float64 // 及格线
	UsualWeight      float64 // 总评中平时成绩权重
	FinalWeight      float64 // 总评中期末成绩权重
	GraduationMinGPA float64 // 毕业审核的加权绩点下限
}

// DefaultRules 返回本系统的默认教务规则（4.0 制分段表，见 GPA）。
func DefaultRules() Rules {
	return Rules{
		PassScore:        60,
		UsualWeight:      0.3,
		FinalWeight:      0.7,
		GraduationMinGPA: 2.0,
	}
}

// TotalScore 按权重合成总评成绩。
func (r Rules) TotalScore(usual, final float64) float64 {
	return usual*r.UsualWeight + final*r.FinalWeight
}

// GPA 按 4.0 制分段表换算：
// 90+→4.0；85→3.7；82→3.3；78→3.0；75→2.7；72→2.3；68→2.0；64→1.5；60→1.0；<60→0。
func (r Rules) GPA(score float64) float64 {
	switch {
	case score >= 90:
		return 4.0
	case score >= 85:
		return 3.7
	case score >= 82:
		return 3.3
	case score >= 78:
		return 3.0
	case score >= 75:
		return 2.7
	case score >= 72:
		return 2.3
	case score >= 68:
		return 2.0
	case score >= 64:
		return 1.5
	case score >= 60:
		return 1.0
	default:
		return 0
	}
}
