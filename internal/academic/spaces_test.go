package academic

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newSpaceTestDB 用内存 SQLite + AutoMigrate 造一个最小 WeKnora 库。
func newSpaceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&types.User{}, &types.Tenant{}, &types.TenantMember{},
		&types.Organization{}, &types.OrganizationTenantMember{}, &types.KnowledgeBaseShare{},
	); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	return db
}

func TestSeedSpacesIdempotentReconcile(t *testing.T) {
	db := newSpaceTestDB(t)
	ctx := context.Background()

	// 第一次：从零调和 → 9 空间 + 组织 + 9 成员关系
	m1, err := SeedSpaces(ctx, db, SpaceOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(m1.Classes) != 8 || m1.Dean == 0 {
		t.Fatalf("want 8 classes + dean, got %+v", m1)
	}
	var tenants int64
	db.Model(&types.Tenant{}).Count(&tenants)
	if tenants != 9 {
		t.Fatalf("want 9 tenants, got %d", tenants)
	}
	var orgMembers int64
	db.Model(&types.OrganizationTenantMember{}).Where("organization_id = ?", m1.OrgID).Count(&orgMembers)
	if orgMembers != 9 {
		t.Fatalf("org members want 9, got %d", orgMembers)
	}

	// 第二次：已存在 → 幂等跳过，不再翻倍
	m2, err := SeedSpaces(ctx, db, SpaceOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if m2.Dean != m1.Dean || m2.OrgID != m1.OrgID {
		t.Fatalf("重跑不得新建: %+v vs %+v", m1, m2)
	}
	db.Model(&types.Tenant{}).Count(&tenants)
	if tenants != 9 {
		t.Fatalf("重跑后 want 9 tenants, got %d", tenants)
	}
}

func TestSeedAccountsCreatesReadableNamesAndMemberships(t *testing.T) {
	db := newSpaceTestDB(t)
	ctx := context.Background()
	m, err := SeedSpaces(ctx, db, SpaceOptions{})
	if err != nil {
		t.Fatal(err)
	}

	// 建账号（密码不落明文——bcrypt 后进库）
	created, err := SeedAccounts(ctx, db, m, AccountOptions{})
	if err != nil {
		t.Fatal(err)
	}
	// 现有 4 个保留账号 + 5 个新建
	if len(created.NewAccounts) != 5 {
		t.Fatalf("want 5 new accounts, got %v", created.NewAccounts)
	}
	for _, want := range []string{"student-001", "student-002", "student-003", "teacher-001", "counselor-001"} {
		if !contains(created.NewAccounts, want) {
			t.Fatalf("缺可读账号 %s, got %v", want, created.NewAccounts)
		}
	}

	// student-001 在计科2402 空间；teacher-001 在软工2401；counselor-001 在计科2401
	assertMember := func(email string, tenantID uint64) {
		t.Helper()
		var u types.User
		if err := db.Where("email = ?", email).First(&u).Error; err != nil {
			t.Fatalf("账号不存在 %s: %v", email, err)
		}
		var n int64
		db.Model(&types.TenantMember{}).Where("user_id = ? AND tenant_id = ? AND status = 'active'", u.ID, tenantID).Count(&n)
		if n != 1 {
			t.Fatalf("%s 应是空间 %d 的成员, got %d", email, tenantID, n)
		}
	}
	assertMember("student-001@askademic.local", m.Classes["计科2402班"])
	assertMember("student-002@askademic.local", m.Classes["软工2401班"])
	assertMember("student-003@askademic.local", m.Classes["网工2401班"])
	assertMember("teacher-001@askademic.local", m.Classes["软工2401班"])

	// 密码不含 Weknora 字样（bcrypt 哈希本身也不应包含）
	var u types.User
	db.Where("email = ?", "student-001@askademic.local").First(&u)
	if strings_CaseInsensitiveContains(u.PasswordHash, "weknora") {
		t.Fatal("密码哈希不应包含 weknora 字样")
	}
	// bcrypt 哈希可校验回原密码
	if !CheckPassword(u.PasswordHash, "Askademic2026") {
		t.Fatal("Askademic2026 应能通过校验")
	}
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func strings_CaseInsensitiveContains(haystack, needle string) bool {
	if len(needle) == 0 {
		return true
	}
	h, n := []rune(haystack), []rune(needle)
	for i := 0; i+len(n) <= len(h); i++ {
		match := true
		for j := range n {
			lh, ln := lower(h[i+j]), lower(n[j])
			if lh != ln {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func lower(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return r + 32
	}
	return r
}
