package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupCleanupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}
	model.DB = db
	model.LOG_DB = db

	if err := db.AutoMigrate(
		&model.User{},
		&model.MemberProfile{},
		&model.MemberSession{},
		&model.EmailLoginCodeStore{},
		&model.AllergyOrder{},
	); err != nil {
		t.Fatalf("failed to migrate cleanup models: %v", err)
	}

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func seedCleanupUser(t *testing.T, db *gorm.DB, username string, email string, role int, verified bool) *model.User {
	t.Helper()

	passwordHash, err := common.Password2Hash("member-password")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	user := &model.User{
		Username:    username,
		Password:    passwordHash,
		DisplayName: username,
		Role:        role,
		Status:      common.UserStatusEnabled,
		Email:       email,
		Group:       "default",
		AffCode:     strings.ToUpper(common.GetRandomString(4)),
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("failed to create cleanup user: %v", err)
	}

	profile := &model.MemberProfile{
		UserID: user.Id,
		Status: "active",
	}
	if verified {
		verifiedAt := time.Now()
		profile.EmailVerifiedAt = &verifiedAt
	}
	if err := db.Create(profile).Error; err != nil {
		t.Fatalf("failed to create cleanup profile: %v", err)
	}

	return user
}

func TestBuildCleanupPlanFiltersLegacyMembersSafely(t *testing.T) {
	db := setupCleanupTestDB(t)

	legacyUser := seedCleanupUser(t, db, "legacy_member", "legacy@example.com", common.RoleCommonUser, false)
	if _, _, err := model.CreateMemberSession(legacyUser.Id, model.AllergyMemberClientWeb, "test-agent", "127.0.0.1", time.Hour); err != nil {
		t.Fatalf("failed to create legacy session: %v", err)
	}
	if _, err := model.CreateEmailLoginCode("legacy@example.com", model.AllergyRegisterVerifyCodePurpose, "123456", "127.0.0.1", 5*time.Minute); err != nil {
		t.Fatalf("failed to create legacy code: %v", err)
	}

	adminUser := seedCleanupUser(t, db, "admin_member", "admin@example.com", common.RoleAdminUser, false)
	if adminUser.Id == 0 {
		t.Fatalf("expected admin user id to be populated")
	}

	orderedUser := seedCleanupUser(t, db, "ordered_member", "ordered@example.com", common.RoleCommonUser, false)
	if err := db.Create(&model.AllergyOrder{
		OrderNo:             "AO-CLEANUP-001",
		UserID:              orderedUser.Id,
		ServiceCode:         "allergy-test-basic",
		ServiceNameSnapshot: "埃勒吉居家过敏原检测服务",
		ServicePriceCents:   19900,
		Currency:            "CNY",
		PaymentStatus:       "paid",
		OrderStatus:         "paid",
		RecipientName:       "张三",
		RecipientPhone:      "13800000000",
		RecipientEmail:      orderedUser.Email,
		ShippingAddressJSON: `{"city":"上海"}`,
	}).Error; err != nil {
		t.Fatalf("failed to create order for protected user: %v", err)
	}

	_ = seedCleanupUser(t, db, "verified_member", "verified@example.com", common.RoleCommonUser, true)

	plan, err := buildCleanupPlan()
	if err != nil {
		t.Fatalf("failed to build cleanup plan: %v", err)
	}

	if len(plan.Candidates) != 1 {
		t.Fatalf("expected 1 deletable candidate, got %d", len(plan.Candidates))
	}
	if plan.Candidates[0].UserID != legacyUser.Id {
		t.Fatalf("unexpected cleanup candidate: %+v", plan.Candidates[0])
	}
	if len(plan.SkippedAdmins) != 1 {
		t.Fatalf("expected 1 protected admin, got %d", len(plan.SkippedAdmins))
	}
	if len(plan.SkippedWithOrders) != 1 {
		t.Fatalf("expected 1 skipped user with orders, got %d", len(plan.SkippedWithOrders))
	}
	if plan.TotalSessionCount != 1 || plan.TotalCodeCount != 1 {
		t.Fatalf("unexpected affected counts: sessions=%d codes=%d", plan.TotalSessionCount, plan.TotalCodeCount)
	}
}

func TestExecuteCleanupPlanDeletesOnlyDeletableLegacyMembers(t *testing.T) {
	db := setupCleanupTestDB(t)

	legacyUser := seedCleanupUser(t, db, "legacy_member", "legacy@example.com", common.RoleCommonUser, false)
	if _, _, err := model.CreateMemberSession(legacyUser.Id, model.AllergyMemberClientWeb, "test-agent", "127.0.0.1", time.Hour); err != nil {
		t.Fatalf("failed to create legacy session: %v", err)
	}
	if _, err := model.CreateEmailLoginCode("legacy@example.com", model.AllergyRegisterVerifyCodePurpose, "123456", "127.0.0.1", 5*time.Minute); err != nil {
		t.Fatalf("failed to create legacy code: %v", err)
	}

	protectedUser := seedCleanupUser(t, db, "ordered_member", "ordered@example.com", common.RoleCommonUser, false)
	if err := db.Create(&model.AllergyOrder{
		OrderNo:             "AO-CLEANUP-002",
		UserID:              protectedUser.Id,
		ServiceCode:         "allergy-test-basic",
		ServiceNameSnapshot: "埃勒吉居家过敏原检测服务",
		ServicePriceCents:   19900,
		Currency:            "CNY",
		PaymentStatus:       "paid",
		OrderStatus:         "paid",
		RecipientName:       "张三",
		RecipientPhone:      "13800000000",
		RecipientEmail:      protectedUser.Email,
		ShippingAddressJSON: `{"city":"上海"}`,
	}).Error; err != nil {
		t.Fatalf("failed to create protected order: %v", err)
	}

	plan, err := buildCleanupPlan()
	if err != nil {
		t.Fatalf("failed to build cleanup plan: %v", err)
	}
	if err := executeCleanupPlan(plan); err != nil {
		t.Fatalf("failed to execute cleanup plan: %v", err)
	}

	var legacyUserCount int64
	if err := db.Unscoped().Model(&model.User{}).Where("id = ?", legacyUser.Id).Count(&legacyUserCount).Error; err != nil {
		t.Fatalf("failed to count legacy user after cleanup: %v", err)
	}
	if legacyUserCount != 0 {
		t.Fatalf("expected legacy user to be deleted, got count=%d", legacyUserCount)
	}

	var protectedUserCount int64
	if err := db.Unscoped().Model(&model.User{}).Where("id = ?", protectedUser.Id).Count(&protectedUserCount).Error; err != nil {
		t.Fatalf("failed to count protected user after cleanup: %v", err)
	}
	if protectedUserCount != 1 {
		t.Fatalf("expected protected user to remain, got count=%d", protectedUserCount)
	}
}
