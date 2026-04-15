package model

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupAllergyModelTestDB(t *testing.T) *gorm.DB {
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
	DB = db
	LOG_DB = db

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func TestMigrateDBCreatesAllergyBusinessTables(t *testing.T) {
	db := setupAllergyModelTestDB(t)

	if err := migrateDB(); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	requiredTables := []any{
		&MemberProfile{},
		&EmailLoginCodeStore{},
		&MemberSession{},
		&AllergyOrder{},
		&SampleKit{},
		&LabSubmission{},
		&LabReport{},
		&ReportDeliveryLog{},
		&OrderTimelineEvent{},
	}
	for _, table := range requiredTables {
		if !db.Migrator().HasTable(table) {
			t.Fatalf("expected table for %T to exist after migration", table)
		}
	}
}

func TestAllergyBusinessModelsCanPersistCoreWorkflowRecords(t *testing.T) {
	db := setupAllergyModelTestDB(t)

	if err := db.AutoMigrate(
		&User{},
		&MemberProfile{},
		&EmailLoginCodeStore{},
		&MemberSession{},
		&AllergyOrder{},
		&SampleKit{},
		&LabSubmission{},
		&LabReport{},
		&ReportDeliveryLog{},
		&OrderTimelineEvent{},
	); err != nil {
		t.Fatalf("failed to migrate allergy business models: %v", err)
	}

	passwordHash, err := common.Password2Hash("member-password")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	user := User{
		Username:    "member-001",
		Password:    passwordHash,
		DisplayName: "Member 001",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Email:       "member@example.com",
		Group:       "default",
		AffCode:     "MEM1",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	profile := MemberProfile{
		UserID:                user.Id,
		Nickname:              "Town",
		Status:                "active",
		DefaultRecipientName:  "Town",
		DefaultRecipientPhone: "13800000000",
		DefaultAddressJSON:    `{"city":"Shanghai"}`,
	}
	if err := db.Create(&profile).Error; err != nil {
		t.Fatalf("failed to create member profile: %v", err)
	}

	order := AllergyOrder{
		OrderNo:                "AO-20260413-0001",
		UserID:                 user.Id,
		ServiceCode:            "allergy-test-basic",
		ServiceNameSnapshot:    "埃勒吉居家过敏原检测服务",
		ServicePriceCents:      19900,
		Currency:               "CNY",
		PaymentStatus:          "paid",
		OrderStatus:            "paid",
		RecipientName:          "Town",
		RecipientPhone:         "13800000000",
		RecipientEmail:         user.Email,
		ShippingAddressJSON:    `{"city":"Shanghai","district":"Pudong"}`,
		PaymentMethod:          "epay",
		PaymentRef:             "EPAY-REF-001",
		PaymentProviderOrderNo: "EPAY-ORDER-001",
		PaidAt:                 timePtr(time.Now()),
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatalf("failed to create allergy order: %v", err)
	}

	kit := SampleKit{
		OrderID:         order.ID,
		KitCode:         "KIT-0001",
		Status:          "shipped",
		TrackingCompany: "SF",
		TrackingNumber:  "SF1234567890",
		ShippedAt:       timePtr(time.Now()),
	}
	if err := db.Create(&kit).Error; err != nil {
		t.Fatalf("failed to create sample kit: %v", err)
	}

	submission := LabSubmission{
		OrderID:            order.ID,
		Status:             "received",
		ExternalSampleCode: "LAB-SAMPLE-001",
		ReceivedAt:         timePtr(time.Now()),
	}
	if err := db.Create(&submission).Error; err != nil {
		t.Fatalf("failed to create lab submission: %v", err)
	}

	report := LabReport{
		OrderID:       order.ID,
		Version:       1,
		Status:        "published",
		IsCurrent:     true,
		FileName:      "report.pdf",
		FilePath:      "/data/reports/report.pdf",
		FileURL:       "/reports/report.pdf",
		MimeType:      "application/pdf",
		FileSizeBytes: 2048,
		UploadedAt:    timePtr(time.Now()),
		PublishedAt:   timePtr(time.Now()),
	}
	if err := db.Create(&report).Error; err != nil {
		t.Fatalf("failed to create lab report: %v", err)
	}

	deliveryLog := ReportDeliveryLog{
		ReportID:          report.ID,
		OrderID:           order.ID,
		RecipientEmail:    user.Email,
		DeliveryType:      "manual_resend",
		DeliveryStatus:    "sent",
		TriggeredByUserID: user.Id,
	}
	if err := db.Create(&deliveryLog).Error; err != nil {
		t.Fatalf("failed to create delivery log: %v", err)
	}

	timelineEvent := OrderTimelineEvent{
		OrderID:          order.ID,
		EventType:        "payment_completed",
		EventTitle:       "支付完成",
		EventPayloadJSON: `{"payment_method":"epay"}`,
		OccurredAt:       time.Now(),
	}
	if err := db.Create(&timelineEvent).Error; err != nil {
		t.Fatalf("failed to create timeline event: %v", err)
	}

	var persistedOrder AllergyOrder
	if err := db.First(&persistedOrder, order.ID).Error; err != nil {
		t.Fatalf("failed to reload order: %v", err)
	}
	if persistedOrder.PaymentStatus != "paid" || persistedOrder.OrderStatus != "paid" {
		t.Fatalf("unexpected order status after reload: %+v", persistedOrder)
	}
}

func TestValidateAllergyMemberUserRejectsInvalidStates(t *testing.T) {
	db := setupAllergyModelTestDB(t)

	if err := db.AutoMigrate(&User{}, &MemberProfile{}); err != nil {
		t.Fatalf("failed to migrate auth models: %v", err)
	}

	passwordHash, err := common.Password2Hash("member-password")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	createUser := func(username string, email string, role int, status int) *User {
		t.Helper()
		user := &User{
			Username:    username,
			Password:    passwordHash,
			DisplayName: username,
			Role:        role,
			Status:      status,
			Email:       email,
			Group:       "default",
			AffCode:     strings.ToUpper(common.GetRandomString(4)),
		}
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("failed to create user: %v", err)
		}
		return user
	}

	verifiedAt := time.Now()
	activeUser := createUser("active_member", "active@example.com", common.RoleCommonUser, common.UserStatusEnabled)
	if err := db.Create(&MemberProfile{
		UserID:          activeUser.Id,
		Status:          memberProfileStatusActive,
		EmailVerifiedAt: &verifiedAt,
	}).Error; err != nil {
		t.Fatalf("failed to create active profile: %v", err)
	}

	profile, err := ValidateAllergyMemberUser(activeUser)
	if err != nil {
		t.Fatalf("expected active verified member to pass validation: %v", err)
	}
	if profile == nil || profile.UserID != activeUser.Id {
		t.Fatalf("unexpected validated profile: %+v", profile)
	}

	adminUser := createUser("admin_member", "admin@example.com", common.RoleAdminUser, common.UserStatusEnabled)
	if _, err := ValidateAllergyMemberUser(adminUser); !errors.Is(err, ErrAllergyAdminAccountNotAllowed) {
		t.Fatalf("expected admin account rejection, got %v", err)
	}

	noProfileUser := createUser("no_profile", "no-profile@example.com", common.RoleCommonUser, common.UserStatusEnabled)
	if _, err := ValidateAllergyMemberUser(noProfileUser); !errors.Is(err, ErrAllergyMemberProfileRequired) {
		t.Fatalf("expected no-profile rejection, got %v", err)
	}

	disabledUser := createUser("disabled_member", "disabled@example.com", common.RoleCommonUser, common.UserStatusDisabled)
	if err := db.Create(&MemberProfile{
		UserID:          disabledUser.Id,
		Status:          memberProfileStatusActive,
		EmailVerifiedAt: &verifiedAt,
	}).Error; err != nil {
		t.Fatalf("failed to create disabled user profile: %v", err)
	}
	if _, err := ValidateAllergyMemberUser(disabledUser); !errors.Is(err, ErrAllergyMemberDisabled) {
		t.Fatalf("expected disabled user rejection, got %v", err)
	}

	unverifiedUser := createUser("unverified_member", "unverified@example.com", common.RoleCommonUser, common.UserStatusEnabled)
	if err := db.Create(&MemberProfile{
		UserID: unverifiedUser.Id,
		Status: memberProfileStatusActive,
	}).Error; err != nil {
		t.Fatalf("failed to create unverified user profile: %v", err)
	}
	if _, err := ValidateAllergyMemberUser(unverifiedUser); !errors.Is(err, ErrAllergyEmailNotVerified) {
		t.Fatalf("expected unverified user rejection, got %v", err)
	}

	suspendedMember := createUser("suspended_member", "suspended@example.com", common.RoleCommonUser, common.UserStatusEnabled)
	if err := db.Create(&MemberProfile{
		UserID:          suspendedMember.Id,
		Status:          memberProfileStatusDisabled,
		EmailVerifiedAt: &verifiedAt,
	}).Error; err != nil {
		t.Fatalf("failed to create suspended profile: %v", err)
	}
	if _, err := ValidateAllergyMemberUser(suspendedMember); !errors.Is(err, ErrAllergyMemberDisabled) {
		t.Fatalf("expected disabled profile rejection, got %v", err)
	}
}

func TestAuthenticateMemberSessionRequiresVerifiedMemberProfile(t *testing.T) {
	db := setupAllergyModelTestDB(t)

	if err := db.AutoMigrate(&User{}, &MemberProfile{}, &MemberSession{}); err != nil {
		t.Fatalf("failed to migrate session auth models: %v", err)
	}

	passwordHash, err := common.Password2Hash("member-password")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	user := &User{
		Username:    "session_member",
		Password:    passwordHash,
		DisplayName: "session_member",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Email:       "session@example.com",
		Group:       "default",
		AffCode:     strings.ToUpper(common.GetRandomString(4)),
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("failed to create session user: %v", err)
	}

	verifiedAt := time.Now()
	profile := &MemberProfile{
		UserID:          user.Id,
		Status:          memberProfileStatusActive,
		EmailVerifiedAt: &verifiedAt,
	}
	if err := db.Create(profile).Error; err != nil {
		t.Fatalf("failed to create session profile: %v", err)
	}

	token, session, err := CreateMemberSession(user.Id, AllergyMemberClientWeb, "test-agent", "127.0.0.1", time.Hour)
	if err != nil {
		t.Fatalf("failed to create member session: %v", err)
	}

	authenticatedSession, authenticatedUser, err := AuthenticateMemberSession(token)
	if err != nil {
		t.Fatalf("expected session to authenticate, got %v", err)
	}
	if authenticatedSession.ID != session.ID || authenticatedUser.Id != user.Id {
		t.Fatalf("unexpected session auth result: session=%+v user=%+v", authenticatedSession, authenticatedUser)
	}

	if err := db.Model(profile).Update("email_verified_at", nil).Error; err != nil {
		t.Fatalf("failed to clear email verification: %v", err)
	}

	if _, _, err := AuthenticateMemberSession(token); !errors.Is(err, ErrAllergyUnauthorized) {
		t.Fatalf("expected session auth to fail after email verification cleared, got %v", err)
	}
}

func timePtr(v time.Time) *time.Time {
	return &v
}
