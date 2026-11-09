package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Tencent/WeKnora/internal/academic"
	"github.com/Tencent/WeKnora/internal/types"
)

// ToolAcademicDB 是教务数据查询工具的注册名。
const ToolAcademicDB = "academic_db"

// academicDBParameters 是 academic_db 的 JSON Schema：
// 只有一个参数 sql——要跑在物化视图上的只读查询。
func academicDBParameters() json.RawMessage {
	return json.RawMessage(`{
  "type": "object",
  "properties": {
    "sql": {
      "type": "string",
      "description": "只读 SQL（SELECT/WITH），只能查询本工具列出的物化表，物理表不可见"
    }
  },
  "required": ["sql"]
}`)
}

// AcademicDBTool 把教务库的按角色物化查询包装成 Agent 工具。
// 它只做三件事：取调用者 → 解析教务身份 → 跑物化查询；
// 任何失败都转成“教模型怎么改”的错误文案。
type AcademicDBTool struct {
	store *academic.Store
}

// NewAcademicDBTool 用给定教务库构造工具。
func NewAcademicDBTool(store *academic.Store) *AcademicDBTool {
	return &AcademicDBTool{store: store}
}

// Name 实现 types.Tool。
func (t *AcademicDBTool) Name() string { return ToolAcademicDB }

// Description 面向模型：只描述物化表与能力，不出现物理表名。
func (t *AcademicDBTool) Description() string {
	return "查询教务数据（成绩、学分、班级、学籍）。可用表按当前用户角色而定：" +
		"学生可用 my_profile(本人档案)、my_grades(本人成绩)、my_enrollments(本人选课)、my_credits(学分进度)；" +
		"教师可用 my_teaching_classes(所带教学班)、class_grades(所带班成绩)、class_roster(名单)、class_stats(统计)；" +
		"辅导员可用 counselor_classes(所带班级)、class_roster、class_stats；" +
		"教务管理员可用 students_all(全校学生)等全量表。只支持 SELECT/WITH 只读查询。"
}

// Parameters 实现 types.Tool。
func (t *AcademicDBTool) Parameters() json.RawMessage { return academicDBParameters() }

// Execute 实现 types.Tool：取 Caller.UserID → ResolveIdentity → Query。
// 错误一律转成 ToolResult{Success:false, Error: ...}，返回值 error 恒为 nil，
// 让模型能读到“怎么改”而不是中断会话。
func (t *AcademicDBTool) Execute(ctx context.Context, args json.RawMessage) (*types.ToolResult, error) {
	if t.store == nil {
		return &types.ToolResult{Success: false,
			Error: "教务库未初始化：请先运行教务种子脚本或设置 WEKNORA_ACADEMIC_DB 后重启"}, nil
	}
	var in struct {
		SQL string `json:"sql"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return &types.ToolResult{Success: false,
			Error: fmt.Sprintf("参数解析失败：sql 字段必须是要执行的只读 SQL 字符串 (%v)", err)}, nil
	}
	if strings.TrimSpace(in.SQL) == "" {
		return &types.ToolResult{Success: false,
			Error: "缺少必填参数 sql：请提供要执行的 SELECT 查询"}, nil
	}

	caller := types.CallerFromContext(ctx)
	if caller.UserID == "" {
		return &types.ToolResult{Success: false,
			Error: "未绑定教务身份：当前会话没有可识别的用户，无法确定数据权限"}, nil
	}
	identity, err := t.store.ResolveIdentity(ctx, caller.UserID)
	if err != nil {
		if err == academic.ErrIdentityNotFound {
			return &types.ToolResult{Success: false,
				Error: "未绑定教务身份：当前账号在教务库中没有对应的学生/教师/辅导员记录，无法查询学业数据。请提示用户先完成教务身份绑定。"}, nil
		}
		return &types.ToolResult{Success: false,
			Error: fmt.Sprintf("教务身份解析失败：%v", err)}, nil
	}

	res, err := t.store.Query(ctx, identity, in.SQL, academic.DefaultQueryOptions())
	if err != nil {
		if strings.Contains(err.Error(), academic.ErrNotSelect.Error()) {
			return &types.ToolResult{Success: false,
				Error: fmt.Sprintf("仅支持只读查询：%v。请改用 SELECT/WITH 语句。", err)}, nil
		}
		avail := academic.AvailableTables(identity)
		return &types.ToolResult{Success: false,
			Error: fmt.Sprintf("%v。当前角色可用的表：%s", err, strings.Join(avail, ", "))}, nil
	}

	var b strings.Builder
	b.WriteString("列: ")
	b.WriteString(strings.Join(res.Columns, " | "))
	b.WriteString("\n")
	for _, row := range res.Rows {
		b.WriteString(strings.Join(row, " | "))
		b.WriteString("\n")
	}
	if res.Truncated {
		fmt.Fprintf(&b, "\n[结果已截断：超过单次查询上限(行数/字节数)，请加 WHERE 或 LIMIT 缩小范围]\n")
	}
	fmt.Fprintf(&b, "[耗时 %s]", res.Elapsed)
	return &types.ToolResult{Success: true, Output: b.String()}, nil
}
