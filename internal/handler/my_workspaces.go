package handler

import (
	"fmt"
	"net/http"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

// tenantMemberItem 是 my-workspaces 响应里的一个空间条目。
type tenantMemberItem struct {
	ID   uint64 `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

// MyWorkspacesHandler 返回调用者是 active 成员的全部空间：
//
//	GET /api/v1/auth/my-workspaces → {"data":{"items":[{id,name,role}]}}
//
// 与 /tenants/search 的区别：那个端点被 RequireCrossTenantAccess 保护，
// 只有跨空间超管可用；普通用户（例如带多个班级的教师）需要一个
// “我自己在哪些空间”的列表来切换空间。本 handler 只读调用者自己的
// 成员关系（ListByUser + active 过滤），不泄露任何非本人数据。
type MyWorkspacesHandler struct {
	memberService interfaces.TenantMemberService
	tenantService interfaces.TenantService
}

// NewMyWorkspacesHandler 构造 handler。
func NewMyWorkspacesHandler(
	memberService interfaces.TenantMemberService,
	tenantService interfaces.TenantService,
) *MyWorkspacesHandler {
	return &MyWorkspacesHandler{memberService: memberService, tenantService: tenantService}
}

// ListMyWorkspaces 处理 GET /auth/my-workspaces。
func (h *MyWorkspacesHandler) ListMyWorkspaces(c *gin.Context) {
	// 与仓库其余 handler 一致：用户身份从 request context 读
	//（auth 中间件经 types.UserIDFromContext 注入）。
	userID, _ := types.UserIDFromContext(c.Request.Context())
	if userID == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "未认证：缺少用户身份",
		})
		return
	}

	ctx := c.Request.Context()
	members, err := h.memberService.ListByUser(ctx, userID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("查询空间列表失败: %v", err),
		})
		return
	}

	// active 过滤（ListByUser 返回的是全量行，含 suspended）。
	active := make([]*types.TenantMember, 0, len(members))
	ids := make([]uint64, 0, len(members))
	roleByTenant := make(map[uint64]string, len(members))
	for _, m := range members {
		if m == nil || m.Status != types.TenantMemberStatusActive {
			continue
		}
		active = append(active, m)
		ids = append(ids, m.TenantID)
		roleByTenant[m.TenantID] = string(m.Role)
	}
	if len(active) == 0 {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"items": []tenantMemberItem{}}})
		return
	}

	tenants, err := h.tenantService.GetTenantsByIDs(ctx, ids)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("查询空间信息失败: %v", err),
		})
		return
	}

	items := make([]tenantMemberItem, 0, len(active))
	for _, m := range active {
		t, ok := tenants[m.TenantID]
		if !ok || t == nil {
			continue
		}
		items = append(items, tenantMemberItem{ID: t.ID, Name: t.Name, Role: roleByTenant[m.TenantID]})
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"items": items}})
}
