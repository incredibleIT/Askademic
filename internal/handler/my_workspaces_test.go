package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

// fakeMemberService / fakeTenantService 是最小测试替身。
type fakeMemberService struct {
	members []*types.TenantMember
}

func (f *fakeMemberService) ListByUser(ctx context.Context, userID string) ([]*types.TenantMember, error) {
	return f.members, nil
}

func (f *fakeMemberService) AddMember(ctx context.Context, userID string, tenantID uint64, role types.TenantRole, invitedBy *string) (*types.TenantMember, error) {
	return nil, nil
}
func (f *fakeMemberService) EnsureOwner(ctx context.Context, userID string, tenantID uint64) (*types.TenantMember, error) {
	return nil, nil
}
func (f *fakeMemberService) GetMembership(ctx context.Context, userID string, tenantID uint64) (*types.TenantMember, error) {
	return nil, nil
}
func (f *fakeMemberService) ListByTenant(ctx context.Context, tenantID uint64) ([]*types.TenantMember, error) {
	return nil, nil
}
func (f *fakeMemberService) ListMembersPage(ctx context.Context, tenantID uint64, query string, page, pageSize int) ([]*types.TenantMember, int64, error) {
	return nil, 0, nil
}
func (f *fakeMemberService) HasAnyMembers(ctx context.Context, tenantID uint64) (bool, error) {
	return false, nil
}
func (f *fakeMemberService) UpdateRole(ctx context.Context, userID string, tenantID uint64, newRole types.TenantRole) error {
	return nil
}
func (f *fakeMemberService) RemoveMember(ctx context.Context, userID string, tenantID uint64) error {
	return nil
}

type fakeTenantService struct {
	interfaces.TenantService // 嵌入(nil)：本测试只会调 GetTenantsByIDs，其余方法不可达
	tenants                  map[uint64]*types.Tenant
}

func (f *fakeTenantService) GetTenantsByIDs(ctx context.Context, ids []uint64) (map[uint64]*types.Tenant, error) {
	out := make(map[uint64]*types.Tenant, len(ids))
	for _, id := range ids {
		if t, ok := f.tenants[id]; ok {
			out[id] = t
		}
	}
	return out, nil
}

func newMyWorkspacesFixture(t *testing.T, members []*types.TenantMember) (*MyWorkspacesHandler, *fakeTenantService) {
	t.Helper()
	tenants := map[uint64]*types.Tenant{
		100: {ID: 100, Name: "空间A"},
		200: {ID: 200, Name: "空间B"},
		300: {ID: 300, Name: "空间C"},
	}
	fts := &fakeTenantService{tenants: tenants}
	fms := &fakeMemberService{members: members}
	return NewMyWorkspacesHandler(fms, fts), fts
}

func serveMyWorkspaces(h *MyWorkspacesHandler, userID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/my-workspaces", func(c *gin.Context) {
		if userID != "" {
			ctx := context.WithValue(c.Request.Context(), types.UserIDContextKey, userID)
			c.Request = c.Request.WithContext(ctx)
		}
		c.Next()
	}, h.ListMyWorkspaces)
	return r
}

func member(uid string, tid uint64, status types.TenantMemberStatus, role types.TenantRole) *types.TenantMember {
	return &types.TenantMember{UserID: uid, TenantID: tid, Status: status, Role: role}
}

func TestListMyWorkspacesReturnsActiveMembershipsOnly(t *testing.T) {
	h, _ := newMyWorkspacesFixture(t, []*types.TenantMember{
		member("u-alice", 100, types.TenantMemberStatusActive, types.TenantRoleOwner),
		member("u-alice", 200, types.TenantMemberStatusActive, types.TenantRoleContributor),
		member("u-alice", 300, types.TenantMemberStatusSuspended, types.TenantRoleViewer),
	})
	r := serveMyWorkspaces(h, "u-alice")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/my-workspaces", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var body struct {
		Data struct {
			Items []tenantMemberItem `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	if len(body.Data.Items) != 2 {
		t.Fatalf("want 2 active memberships (A,B), got %+v", body.Data.Items)
	}
	ids := map[uint64]bool{}
	for _, it := range body.Data.Items {
		ids[it.ID] = true
	}
	if !ids[100] || !ids[200] || ids[300] {
		t.Fatalf("suspended 的 C 不应出现, got %+v", body.Data.Items)
	}
	if body.Data.Items[0].Name == "" || body.Data.Items[0].Role == "" {
		t.Fatalf("条目应含 name/role, got %+v", body.Data.Items)
	}
}

func TestListMyWorkspacesScopedToCaller(t *testing.T) {
	h, _ := newMyWorkspacesFixture(t, []*types.TenantMember{
		member("u-bob", 100, types.TenantMemberStatusActive, types.TenantRoleViewer),
	})
	r := serveMyWorkspaces(h, "u-bob")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/my-workspaces", nil))
	var body struct {
		Data struct {
			Items []tenantMemberItem `json:"items"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if len(body.Data.Items) != 1 || body.Data.Items[0].ID != 100 {
		t.Fatalf("u-bob 只应有空间A, got %+v", body.Data.Items)
	}
}

func TestListMyWorkspacesAnonymousIs401(t *testing.T) {
	h, _ := newMyWorkspacesFixture(t, nil)
	r := serveMyWorkspaces(h, "")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/my-workspaces", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", w.Code)
	}
}
