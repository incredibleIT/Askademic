package academic

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// 演示账号统一密码（不出现 WeKnora 字样；经 bcrypt 落库）。
const DemoPassword = "Askademic2026"

// SpaceOptions 控制空间调和行为；零值即全默认。
type SpaceOptions struct {
	OrgName string // 组织名，缺省 "XX大学"
}

// SpaceMap 是调和结果：教务处 tenant id + 班名 → tenant id + 组织 id。
type SpaceMap struct {
	Dean    uint64
	Classes map[string]uint64
	OrgID   string
}

// AccountOptions 控制账号调和行为。
type AccountOptions struct {
	Password string // 缺省 DemoPassword
}

// AccountReport 汇报账号调和结果。
type AccountReport struct {
	NewAccounts []string // 本次新建的账号（不含已存在的）
	Skipped     []string // 已存在而跳过的
}

// demoClasses 是 8 个班级空间（顺序即 tenant id 顺序，从 Dean+1 起）。
var demoClasses = []string{
	"计科2401班", "计科2402班",
	"软工2401班", "软工2402班",
	"网工2401班", "网工2402班",
	"人工智能2401班", "人工智能2402班",
}

// DefaultClasses 返回班级清单（供调用方对齐空间名）。
func DefaultClasses() []string { return demoClasses }

// SeedSpaces 幂等调和班级空间与组织共享：
//  1. 教务处空间（第一个班级 id-1 的位置约定为 10000，但实现上不写死数字，
//     而是按“教务处 + 8 班”的名字找/建）；
//  2. 8 个班级空间，缺哪个建哪个；
//  3. 组织（缺省 "XX大学"）+ 把全部 9 个空间收进组织；
//  4. 共享：制度文库所属空间（教务处）把自己的知识库共享给组织（viewer）。
//
// 对线上已有演示态（空间/组织/共享已建）完全幂等：缺的补、有的跳。
func SeedSpaces(ctx context.Context, db *gorm.DB, opts SpaceOptions) (SpaceMap, error) {
	if opts.OrgName == "" {
		opts.OrgName = "XX大学"
	}
	out := SpaceMap{Classes: map[string]uint64{}}

	// 1) 教务处
	dean, err := ensureTenant(ctx, db, "教务处", "全校教务管理与制度发布")
	if err != nil {
		return out, err
	}
	out.Dean = dean

	// 2) 8 个班级空间
	for _, name := range demoClasses {
		id, err := ensureTenant(ctx, db, name, "行政班空间：制度文库、班级资料与培养方案")
		if err != nil {
			return out, err
		}
		out.Classes[name] = id
	}

	// 3) 组织 + 成员关系（9 个空间全收）
	org, err := ensureOrganization(ctx, db, opts.OrgName, dean)
	if err != nil {
		return out, err
	}
	out.OrgID = org
	tenantIDs := []uint64{dean}
	for _, name := range demoClasses {
		tenantIDs = append(tenantIDs, out.Classes[name])
	}
	if err := ensureOrgMembers(ctx, db, org, tenantIDs, dean); err != nil {
		return out, err
	}

	// 4) 制度文库共享：教务处空间里的知识库共享给组织（viewer）。
	//    已有共享（任意条）则跳过——共享由教务处日常管理，seeder 只兜底。
	var shareCount int64
	if err := db.WithContext(ctx).Model(&types.KnowledgeBaseShare{}).
		Where("organization_id = ?", org).Count(&shareCount).Error; err != nil {
		return out, err
	}
	if shareCount == 0 {
		if err := shareDeanKBs(ctx, db, dean, org); err != nil {
			return out, err
		}
	}
	return out, nil
}

// ensureTenant 按名找空间，缺则建。
func ensureTenant(ctx context.Context, db *gorm.DB, name, description string) (uint64, error) {
	var t types.Tenant
	err := db.WithContext(ctx).Where("name = ?", name).First(&t).Error
	if err == nil {
		return t.ID, nil
	}
	if err != gorm.ErrRecordNotFound {
		return 0, err
	}
	t = types.Tenant{Name: name, Description: description, Status: "active"}
	if err := db.WithContext(ctx).Create(&t).Error; err != nil {
		return 0, fmt.Errorf("创建空间 %s 失败: %w", name, err)
	}
	return t.ID, nil
}

// ensureOrganization 按名找组织，缺则建（owner 挂教务处）。
func ensureOrganization(ctx context.Context, db *gorm.DB, name string, deanTenant uint64) (string, error) {
	var o types.Organization
	err := db.WithContext(ctx).Where("name = ?", name).First(&o).Error
	if err == nil {
		return o.ID, nil
	}
	if err != gorm.ErrRecordNotFound {
		return "", err
	}
	owner, err := deanOwnerUser(ctx, db, deanTenant)
	if err != nil {
		return "", err
	}
	o = types.Organization{
		ID:                     newUUID(),
		Name:                   name,
		Description:            "Askademic 演示组织：教务处与全部班级空间的共享层",
		OwnerID:                owner,
		OwnerTenantID:          deanTenant,
		InviteCode:             newInviteCode(),
		InviteCodeValidityDays: 0,
		RequireApproval:        false,
		Searchable:             false,
		MemberLimit:            0,
	}
	if err := db.WithContext(ctx).Create(&o).Error; err != nil {
		return "", fmt.Errorf("创建组织 %s 失败: %w", name, err)
	}
	return o.ID, nil
}

// ensureOrgMembers 保证 (org, tenant) 成员关系存在（角色 owner 的空间用 admin）。
func ensureOrgMembers(ctx context.Context, db *gorm.DB, org string, tenantIDs []uint64, deanTenant uint64) error {
	owner, err := deanOwnerUser(ctx, db, deanTenant)
	if err != nil {
		return err
	}
	for _, tid := range tenantIDs {
		var n int64
		if err := db.WithContext(ctx).Model(&types.OrganizationTenantMember{}).
			Where("organization_id = ? AND tenant_id = ?", org, tid).Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			continue
		}
		role := types.OrgRoleViewer
		if tid == deanTenant {
			role = types.OrgRoleAdmin
		}
		member := types.OrganizationTenantMember{
			ID:                   newUUID(),
			OrganizationID:       org,
			TenantID:             tid,
			Role:                 role,
			RepresentativeUserID: owner,
			JoinedAt:             nowPtr(),
		}
		if err := db.WithContext(ctx).Create(&member).Error; err != nil {
			return fmt.Errorf("组织成员关系(%d)创建失败: %w", tid, err)
		}
	}
	return nil
}

// shareDeanKBs 把教务处空间的知识库共享给组织（viewer）。
func shareDeanKBs(ctx context.Context, db *gorm.DB, dean uint64, org string) error {
	var kbs []types.KnowledgeBase
	if err := db.WithContext(ctx).Where("tenant_id = ?", dean).Find(&kbs).Error; err != nil {
		return err
	}
	if len(kbs) == 0 {
		return nil // 教务处还没有知识库，跳过（等教务处建库后可由管理员手动共享）
	}
	owner, err := deanOwnerUser(ctx, db, dean)
	if err != nil {
		return err
	}
	for _, kb := range kbs {
		var n int64
		if err := db.WithContext(ctx).Model(&types.KnowledgeBaseShare{}).
			Where("knowledge_base_id = ? AND organization_id = ?", kb.ID, org).Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			continue
		}
		share := types.KnowledgeBaseShare{
			ID:              newUUID(),
			KnowledgeBaseID: kb.ID,
			OrganizationID:  org,
			SharedByUserID:  owner,
			SourceTenantID:  dean,
			Permission:      types.OrgRoleViewer,
		}
		if err := db.WithContext(ctx).Create(&share).Error; err != nil {
			return fmt.Errorf("共享知识库 %s 失败: %w", kb.ID, err)
		}
	}
	return nil
}

// deanOwnerUser 返回教务处空间的第一个 active 成员（owner 展示用），没有则返回空串。
func deanOwnerUser(ctx context.Context, db *gorm.DB, dean uint64) (string, error) {
	var m types.TenantMember
	err := db.WithContext(ctx).Where("tenant_id = ?", dean).Order("id").First(&m).Error
	if err == gorm.ErrRecordNotFound {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return m.UserID, nil
}

// SeedAccounts 幂等调和演示账号（方案 A）：
//   - 现有账号（s001/t001/jw01/demo @askademic.local）保留不动；
//   - 新建可读名账号：student-001/002/003、teacher-001、counselor-001，
//     密码统一（bcrypt 落库），并写入 tenant_members；
//   - 调用方随后用 BindIdentity 把这些账号绑到教务身份。
//
// 返回新建与跳过的账号名单，便于命令行汇报。
func SeedAccounts(ctx context.Context, db *gorm.DB, spaces SpaceMap, opts AccountOptions) (AccountReport, error) {
	report := AccountReport{}
	if opts.Password == "" {
		opts.Password = DemoPassword
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(opts.Password), bcrypt.DefaultCost)
	if err != nil {
		return report, err
	}

	// 计划的账号矩阵（方案 A，可读名）：
	//   student-001 计科2402 / student-002 软工2401 / student-003 网工2401 (viewer)
	//   teacher-001 软工2401 (contributor)
	//   counselor-001 计科2401 (contributor)
	// 教务处 owner 由现有 jw01/demo 承担，不新建。
	plans := []struct {
		name     string
		tenant   string // 班名
		role     types.TenantRole
		classKey string // academic_identity 绑定用：班名
	}{
		{"student-001", "计科2402班", types.TenantRoleViewer, "计科2402班"},
		{"student-002", "软工2401班", types.TenantRoleViewer, "软工2401班"},
		{"student-003", "网工2401班", types.TenantRoleViewer, "网工2401班"},
		{"teacher-001", "软工2401班", types.TenantRoleContributor, "软工2401班"},
		{"counselor-001", "计科2401班", types.TenantRoleContributor, "计科2401班"},
	}

	for _, p := range plans {
		tenantID, ok := spaces.Classes[p.tenant]
		if !ok {
			return report, fmt.Errorf("空间映射缺班级 %s", p.tenant)
		}
		email := p.name + "@askademic.local"
		var u types.User
		err := db.WithContext(ctx).Where("email = ?", email).First(&u).Error
		if err == nil {
			report.Skipped = append(report.Skipped, p.name)
		} else if err == gorm.ErrRecordNotFound {
			u = types.User{
				ID:           newUUID(),
				Username:     p.name,
				Email:        email,
				PasswordHash: string(hash),
				TenantID:     tenantID,
				IsActive:     true,
			}
			if err := db.WithContext(ctx).Create(&u).Error; err != nil {
				return report, fmt.Errorf("创建账号 %s 失败: %w", email, err)
			}
			report.NewAccounts = append(report.NewAccounts, p.name)
		} else {
			return report, err
		}

		// 空间成员关系：有则跳过
		var n int64
		if err := db.WithContext(ctx).Model(&types.TenantMember{}).
			Where("user_id = ? AND tenant_id = ?", u.ID, tenantID).Count(&n).Error; err != nil {
			return report, err
		}
		if n == 0 {
			member := types.TenantMember{
				UserID:   u.ID,
				TenantID: tenantID,
				Role:     p.role,
				Status:   types.TenantMemberStatusActive,
			}
			if err := db.WithContext(ctx).Create(&member).Error; err != nil {
				return report, fmt.Errorf("成员关系(%s→%d)创建失败: %w", p.name, tenantID, err)
			}
		}
	}
	return report, nil
}

// CheckPassword 校验 bcrypt 哈希与明文是否匹配（测试与 CLI 汇报用）。
func CheckPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

// newUUID 生成 RFC4122 v4 字符串（对齐仓库 uuid.NewString 惯例）。
func newUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err) // crypto/rand 失败属于系统级故障
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// newInviteCode 生成 16 位十六进制邀请码。
func newInviteCode() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}

// nowPtr 返回当前时间指针。
func nowPtr() *time.Time {
	t := time.Now()
	return &t
}

// ensure compile-time use of strings (规范化账号名时用)。
var _ = strings.TrimSpace
