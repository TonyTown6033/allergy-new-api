package controller

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type allergyAuthActionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

type allergyMemberUserResponse struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

type allergyMemberProfileResponse struct {
	ID              int64  `json:"id"`
	Username        string `json:"username"`
	Email           string `json:"email"`
	Nickname        string `json:"nickname"`
	Phone           string `json:"phone"`
	Status          string `json:"status"`
	EmailVerified   bool   `json:"emailVerified"`
	EmailVerifiedAt string `json:"emailVerifiedAt"`
}

type allergyMemberPayload struct {
	User    allergyMemberUserResponse    `json:"user"`
	Profile allergyMemberProfileResponse `json:"profile"`
}

type allergyAuthData struct {
	Token string `json:"token"`
	allergyMemberPayload
}

type allergyAuthResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Code    string          `json:"code"`
	Token   string          `json:"token"`
	Data    allergyAuthData `json:"data"`
}

type allergyMeResponse struct {
	Success bool                 `json:"success"`
	Message string               `json:"message"`
	Data    allergyMemberPayload `json:"data"`
}

type allergyHeroResponse struct {
	Image string `json:"image"`
}

type allergyProductResponse struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Image       string `json:"image"`
	CTAText     string `json:"ctaText"`
	Tag         string `json:"tag"`
}

func setupAllergyControllerTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false
	common.CryptoSecret = "allergy-controller-test-secret"
	common.SMTPServer = ""
	common.SMTPAccount = ""
	common.SMTPToken = ""
	common.SMTPFrom = ""

	common.OptionMapRWMutex.Lock()
	common.OptionMap = make(map[string]string)
	common.OptionMapRWMutex.Unlock()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}
	model.DB = db
	model.LOG_DB = db

	if err := db.AutoMigrate(
		&model.User{},
		&model.Option{},
		&model.MemberProfile{},
		&model.EmailLoginCodeStore{},
		&model.MemberSession{},
	); err != nil {
		t.Fatalf("failed to migrate allergy auth tables: %v", err)
	}

	engine := gin.New()
	engine.GET("/api/hero", GetAllergyHero)
	engine.GET("/api/testimonials", GetAllergyTestimonials)
	engine.GET("/api/articles", GetAllergyArticles)
	engine.GET("/api/products", GetAllergyProducts)

	authGroup := engine.Group("/api/auth")
	authGroup.POST("/register/send-code", SendAllergyRegisterCode)
	authGroup.POST("/register", RegisterAllergyMember)
	authGroup.POST("/login", LoginAllergyMember)
	authGroup.POST("/forgot-password/send-code", SendAllergyPasswordResetCode)
	authGroup.POST("/forgot-password/reset", ResetAllergyMemberPassword)
	authGroup.GET("/me", middleware.AllergyMemberAuth(), GetAllergyAuthMe)
	authGroup.POST("/logout", middleware.AllergyMemberAuth(), LogoutAllergyMember)
	authGroup.PATCH("/profile", middleware.AllergyMemberAuth(), UpdateAllergyProfile)

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db, engine
}

func performAllergyRequest(t *testing.T, engine *gin.Engine, method string, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	var bodyReader *bytes.Reader
	if body != nil {
		payload, err := common.Marshal(body)
		if err != nil {
			t.Fatalf("failed to marshal body: %v", err)
		}
		bodyReader = bytes.NewReader(payload)
	} else {
		bodyReader = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(method, path, bodyReader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)
	return recorder
}

func decodeAllergyJSON[T any](t *testing.T, recorder *httptest.ResponseRecorder) T {
	t.Helper()

	var result T
	if err := common.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to decode response: %v; body=%s", err, recorder.Body.String())
	}
	return result
}

func seedAllergyCode(t *testing.T, db *gorm.DB, email string, purpose string, rawCode string) {
	t.Helper()

	codeHash, err := common.Password2Hash(rawCode)
	if err != nil {
		t.Fatalf("failed to hash code: %v", err)
	}
	record := model.EmailLoginCodeStore{
		Email:     model.NormalizeEmail(email),
		Purpose:   purpose,
		CodeHash:  codeHash,
		ExpiresAt: time.Now().Add(5 * time.Minute),
		CreatedAt: time.Now(),
	}
	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("failed to seed code: %v", err)
	}
}

func seedAllergyMemberUser(t *testing.T, db *gorm.DB, username string, email string, password string, profileStatus string, emailVerified bool) *model.User {
	t.Helper()

	passwordHash, err := common.Password2Hash(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	user := &model.User{
		Username:    username,
		Password:    passwordHash,
		DisplayName: username,
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Email:       model.NormalizeEmail(email),
		Group:       "default",
		AffCode:     strings.ToUpper(common.GetRandomString(4)),
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	profile := &model.MemberProfile{
		UserID:   user.Id,
		Status:   profileStatus,
		Nickname: username,
	}
	if emailVerified {
		now := time.Now()
		profile.EmailVerifiedAt = &now
	}
	if err := db.Create(profile).Error; err != nil {
		t.Fatalf("failed to create member profile: %v", err)
	}
	return user
}

func seedAllergyAdminUser(t *testing.T, db *gorm.DB, username string, email string, password string) *model.User {
	t.Helper()

	passwordHash, err := common.Password2Hash(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	user := &model.User{
		Username:    username,
		Password:    passwordHash,
		DisplayName: username,
		Role:        common.RoleAdminUser,
		Status:      common.UserStatusEnabled,
		Email:       model.NormalizeEmail(email),
		Group:       "default",
		AffCode:     strings.ToUpper(common.GetRandomString(4)),
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("failed to create admin user: %v", err)
	}
	return user
}

func TestGetAllergyHeroFallsBackToDefaultContent(t *testing.T) {
	_, engine := setupAllergyControllerTest(t)

	recorder := performAllergyRequest(t, engine, http.MethodGet, "/api/hero", nil, nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	response := decodeAllergyJSON[allergyHeroResponse](t, recorder)
	if response.Image != "/images/hero1.png" {
		t.Fatalf("expected default hero image, got %q", response.Image)
	}
}

func TestGetAllergyProductsUsesOptionJSONWhenConfigured(t *testing.T) {
	_, engine := setupAllergyControllerTest(t)

	rawProducts := `[{"id":"allergy-test-basic","title":"埃勒吉居家过敏原检测服务","description":"单次检测服务","image":"https://cdn.example.com/product-1.jpg","ctaText":"立即购买","tag":"推荐"}]`
	if err := model.UpdateOption("AllergyProducts", rawProducts); err != nil {
		t.Fatalf("failed to seed option: %v", err)
	}

	recorder := performAllergyRequest(t, engine, http.MethodGet, "/api/products", nil, nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	response := decodeAllergyJSON[[]allergyProductResponse](t, recorder)
	if len(response) != 1 {
		t.Fatalf("expected 1 product, got %d", len(response))
	}
	if response[0].ID != "allergy-test-basic" || response[0].CTAText != "立即购买" {
		t.Fatalf("unexpected product payload: %+v", response[0])
	}
}

func TestSendAllergyRegisterCodeCreatesDatabaseRecord(t *testing.T) {
	db, engine := setupAllergyControllerTest(t)

	recorder := performAllergyRequest(t, engine, http.MethodPost, "/api/auth/register/send-code", map[string]any{
		"email": "member@example.com",
	}, nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	response := decodeAllergyJSON[allergyAuthActionResponse](t, recorder)
	if !response.Success {
		t.Fatalf("expected success response, got message: %s", response.Message)
	}

	var record model.EmailLoginCodeStore
	if err := db.Where("email = ? AND purpose = ?", "member@example.com", model.AllergyRegisterVerifyCodePurpose).First(&record).Error; err != nil {
		t.Fatalf("expected register code record to be created: %v", err)
	}
	if record.CodeHash == "" {
		t.Fatalf("expected code hash to be stored")
	}
	if record.UsedAt != nil {
		t.Fatalf("expected fresh login code to be unused")
	}
	if time.Until(record.ExpiresAt) <= 0 {
		t.Fatalf("expected login code to have a future expiration time")
	}
}

func TestAllergyMemberRegisterLifecycleCreatesSessionAndSupportsMeLogout(t *testing.T) {
	db, engine := setupAllergyControllerTest(t)

	seedAllergyCode(t, db, "member@example.com", model.AllergyRegisterVerifyCodePurpose, "123456")

	registerRecorder := performAllergyRequest(t, engine, http.MethodPost, "/api/auth/register", map[string]any{
		"email":           "member@example.com",
		"code":            "123456",
		"username":        "member_user",
		"password":        "Password123",
		"confirmPassword": "Password123",
	}, nil)
	if registerRecorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", registerRecorder.Code)
	}

	registerResponse := decodeAllergyJSON[allergyAuthResponse](t, registerRecorder)
	if !registerResponse.Success {
		t.Fatalf("expected success response, got body: %s", registerRecorder.Body.String())
	}
	if registerResponse.Token == "" || registerResponse.Data.Token == "" {
		t.Fatalf("expected session token to be returned")
	}
	if registerResponse.Token != registerResponse.Data.Token {
		t.Fatalf("expected top-level token and data token to match")
	}

	var user model.User
	if err := db.Where("email = ?", "member@example.com").First(&user).Error; err != nil {
		t.Fatalf("expected member user to be created: %v", err)
	}
	if user.Username != "member_user" {
		t.Fatalf("expected username member_user, got %q", user.Username)
	}
	if user.Role != common.RoleCommonUser {
		t.Fatalf("expected common user role, got %d", user.Role)
	}
	if user.Status != common.UserStatusEnabled {
		t.Fatalf("expected enabled user status, got %d", user.Status)
	}

	var profile model.MemberProfile
	if err := db.Where("user_id = ?", user.Id).First(&profile).Error; err != nil {
		t.Fatalf("expected member profile to be created: %v", err)
	}
	if profile.EmailVerifiedAt == nil {
		t.Fatalf("expected email_verified_at to be set")
	}

	var session model.MemberSession
	if err := db.Where("user_id = ?", user.Id).First(&session).Error; err != nil {
		t.Fatalf("expected member session to be created: %v", err)
	}
	if session.TokenHash == "" || session.TokenHash == registerResponse.Token {
		t.Fatalf("expected token hash to be stored instead of raw token")
	}
	if session.RevokedAt != nil {
		t.Fatalf("expected fresh session to be active")
	}

	var usedCode model.EmailLoginCodeStore
	if err := db.Where("email = ? AND purpose = ?", "member@example.com", model.AllergyRegisterVerifyCodePurpose).Order("id desc").First(&usedCode).Error; err != nil {
		t.Fatalf("failed to reload register code: %v", err)
	}
	if usedCode.UsedAt == nil {
		t.Fatalf("expected register code to be marked as used")
	}

	authHeader := map[string]string{"Authorization": "Bearer " + registerResponse.Token}
	meRecorder := performAllergyRequest(t, engine, http.MethodGet, "/api/auth/me", nil, authHeader)
	if meRecorder.Code != http.StatusOK {
		t.Fatalf("expected status 200 for me, got %d", meRecorder.Code)
	}

	meResponse := decodeAllergyJSON[allergyMeResponse](t, meRecorder)
	if !meResponse.Success {
		t.Fatalf("expected me to succeed, got body: %s", meRecorder.Body.String())
	}
	if meResponse.Data.User.ID != user.Id || meResponse.Data.User.Email != user.Email {
		t.Fatalf("unexpected me response payload: %+v", meResponse.Data.User)
	}
	if !meResponse.Data.Profile.EmailVerified {
		t.Fatalf("expected me response to expose verified email state")
	}

	patchRec := performAllergyRequest(t, engine, http.MethodPatch, "/api/auth/profile", allergyUpdateProfileRequest{
		Nickname: "小花",
		Phone:    "13900139000",
	}, authHeader)
	patchResp := decodeAllergyJSON[allergyAuthActionResponse](t, patchRec)
	if !patchResp.Success {
		t.Fatalf("expected profile update to succeed: %s", patchRec.Body.String())
	}

	logoutRecorder := performAllergyRequest(t, engine, http.MethodPost, "/api/auth/logout", nil, authHeader)
	if logoutRecorder.Code != http.StatusOK {
		t.Fatalf("expected status 200 for logout, got %d", logoutRecorder.Code)
	}
	logoutResponse := decodeAllergyJSON[allergyAuthActionResponse](t, logoutRecorder)
	if !logoutResponse.Success {
		t.Fatalf("expected logout to succeed, got body: %s", logoutRecorder.Body.String())
	}

	afterLogoutRecorder := performAllergyRequest(t, engine, http.MethodGet, "/api/auth/me", nil, authHeader)
	if afterLogoutRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected me to be unauthorized after logout, got %d body=%s", afterLogoutRecorder.Code, afterLogoutRecorder.Body.String())
	}
}

func TestAllergyPasswordLoginSupportsUsernameAndEmailAndRejectsInvalidMemberStates(t *testing.T) {
	db, engine := setupAllergyControllerTest(t)

	seedAllergyMemberUser(t, db, "member_user", "member@example.com", "Password123", "active", true)

	usernameLogin := performAllergyRequest(t, engine, http.MethodPost, "/api/auth/login", map[string]any{
		"identifier": "member_user",
		"password":   "Password123",
	}, nil)
	usernameResponse := decodeAllergyJSON[allergyAuthResponse](t, usernameLogin)
	if !usernameResponse.Success || usernameResponse.Token == "" {
		t.Fatalf("expected username login to succeed, got body=%s", usernameLogin.Body.String())
	}

	emailLogin := performAllergyRequest(t, engine, http.MethodPost, "/api/auth/login", map[string]any{
		"identifier": "member@example.com",
		"password":   "Password123",
	}, nil)
	emailResponse := decodeAllergyJSON[allergyAuthResponse](t, emailLogin)
	if !emailResponse.Success || emailResponse.Token == "" {
		t.Fatalf("expected email login to succeed, got body=%s", emailLogin.Body.String())
	}

	seedAllergyAdminUser(t, db, "admin_user", "admin@example.com", "Password123")
	adminLogin := performAllergyRequest(t, engine, http.MethodPost, "/api/auth/login", map[string]any{
		"identifier": "admin@example.com",
		"password":   "Password123",
	}, nil)
	adminResponse := decodeAllergyJSON[allergyAuthActionResponse](t, adminLogin)
	if adminResponse.Success {
		t.Fatalf("expected admin account to be rejected")
	}

	seedAllergyMemberUser(t, db, "disabled_user", "disabled@example.com", "Password123", "disabled", true)
	disabledLogin := performAllergyRequest(t, engine, http.MethodPost, "/api/auth/login", map[string]any{
		"identifier": "disabled_user",
		"password":   "Password123",
	}, nil)
	disabledResponse := decodeAllergyJSON[allergyAuthActionResponse](t, disabledLogin)
	if disabledResponse.Success {
		t.Fatalf("expected disabled member account to be rejected")
	}

	seedAllergyMemberUser(t, db, "unverified_user", "unverified@example.com", "Password123", "active", false)
	unverifiedLogin := performAllergyRequest(t, engine, http.MethodPost, "/api/auth/login", map[string]any{
		"identifier": "unverified_user",
		"password":   "Password123",
	}, nil)
	unverifiedResponse := decodeAllergyJSON[allergyAuthActionResponse](t, unverifiedLogin)
	if unverifiedResponse.Success {
		t.Fatalf("expected unverified member account to be rejected")
	}

	passwordHash, err := common.Password2Hash("Password123")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	noProfileUser := &model.User{
		Username:    "no_profile_user",
		Password:    passwordHash,
		DisplayName: "no_profile_user",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Email:       "no-profile@example.com",
		Group:       "default",
		AffCode:     strings.ToUpper(common.GetRandomString(4)),
	}
	if err := db.Create(noProfileUser).Error; err != nil {
		t.Fatalf("failed to create no-profile user: %v", err)
	}
	noProfileLogin := performAllergyRequest(t, engine, http.MethodPost, "/api/auth/login", map[string]any{
		"identifier": "no_profile_user",
		"password":   "Password123",
	}, nil)
	noProfileResponse := decodeAllergyJSON[allergyAuthActionResponse](t, noProfileLogin)
	if noProfileResponse.Success {
		t.Fatalf("expected user without member profile to be rejected")
	}
}

func TestAllergyPasswordResetLifecycleSendsCodeResetsPasswordAndRevokesSessions(t *testing.T) {
	db, engine := setupAllergyControllerTest(t)

	user := seedAllergyMemberUser(t, db, "reset_user", "reset@example.com", "Password123", "active", true)
	token, _, err := model.CreateMemberSession(user.Id, model.AllergyMemberClientWeb, "test-agent", "127.0.0.1", time.Hour)
	if err != nil {
		t.Fatalf("failed to create seed session: %v", err)
	}

	sendRecorder := performAllergyRequest(t, engine, http.MethodPost, "/api/auth/forgot-password/send-code", map[string]any{
		"email": "reset@example.com",
	}, nil)
	sendResponse := decodeAllergyJSON[allergyAuthActionResponse](t, sendRecorder)
	if !sendResponse.Success {
		t.Fatalf("expected send reset code to succeed, got body=%s", sendRecorder.Body.String())
	}

	var count int64
	if err := db.Model(&model.EmailLoginCodeStore{}).Where("email = ? AND purpose = ?", "reset@example.com", model.AllergyPasswordResetCodePurpose).Count(&count).Error; err != nil {
		t.Fatalf("failed to count reset codes: %v", err)
	}
	if count == 0 {
		t.Fatalf("expected password reset code record to be created")
	}

	seedAllergyCode(t, db, "reset@example.com", model.AllergyPasswordResetCodePurpose, "654321")
	resetRecorder := performAllergyRequest(t, engine, http.MethodPost, "/api/auth/forgot-password/reset", map[string]any{
		"email":           "reset@example.com",
		"code":            "654321",
		"password":        "NewPassword123",
		"confirmPassword": "NewPassword123",
	}, nil)
	resetResponse := decodeAllergyJSON[allergyAuthActionResponse](t, resetRecorder)
	if !resetResponse.Success {
		t.Fatalf("expected password reset to succeed, got body=%s", resetRecorder.Body.String())
	}

	oldLogin := performAllergyRequest(t, engine, http.MethodPost, "/api/auth/login", map[string]any{
		"identifier": "reset_user",
		"password":   "Password123",
	}, nil)
	oldLoginResponse := decodeAllergyJSON[allergyAuthActionResponse](t, oldLogin)
	if oldLoginResponse.Success {
		t.Fatalf("expected old password to be rejected after reset")
	}

	newLogin := performAllergyRequest(t, engine, http.MethodPost, "/api/auth/login", map[string]any{
		"identifier": "reset@example.com",
		"password":   "NewPassword123",
	}, nil)
	newLoginResponse := decodeAllergyJSON[allergyAuthResponse](t, newLogin)
	if !newLoginResponse.Success || newLoginResponse.Token == "" {
		t.Fatalf("expected new password login to succeed, got body=%s", newLogin.Body.String())
	}

	afterResetMe := performAllergyRequest(t, engine, http.MethodGet, "/api/auth/me", nil, map[string]string{
		"Authorization": "Bearer " + token,
	})
	if afterResetMe.Code != http.StatusUnauthorized {
		t.Fatalf("expected pre-reset session to be revoked, got %d body=%s", afterResetMe.Code, afterResetMe.Body.String())
	}
}
