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
}

type allergyLoginResponse struct {
	Success   bool `json:"success"`
	Token     string
	Email     string
	IsNewUser bool `json:"is_new_user"`
	User      struct {
		ID    int    `json:"id"`
		Email string `json:"email"`
	} `json:"user"`
}

type allergyMeResponse struct {
	Success bool   `json:"success"`
	Email   string `json:"email"`
	User    struct {
		ID       int    `json:"id"`
		Email    string `json:"email"`
		Nickname string `json:"nickname"`
		Phone    string `json:"phone"`
	} `json:"user"`
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
	authGroup.POST("/send-code", SendAllergyLoginCode)
	authGroup.POST("/login", LoginAllergyMember)
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

func TestSendAllergyLoginCodeCreatesDatabaseRecord(t *testing.T) {
	db, engine := setupAllergyControllerTest(t)

	recorder := performAllergyRequest(t, engine, http.MethodPost, "/api/auth/send-code", map[string]any{
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
	if err := db.Where("email = ? AND purpose = ?", "member@example.com", "login").First(&record).Error; err != nil {
		t.Fatalf("expected login code record to be created: %v", err)
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

func TestAllergyMemberLoginLifecycleCreatesSessionAndSupportsMeLogout(t *testing.T) {
	db, engine := setupAllergyControllerTest(t)

	codeHash, err := common.Password2Hash("123456")
	if err != nil {
		t.Fatalf("failed to hash login code: %v", err)
	}
	loginCode := model.EmailLoginCodeStore{
		Email:     "member@example.com",
		Purpose:   "login",
		CodeHash:  codeHash,
		ExpiresAt: time.Now().Add(5 * time.Minute),
		CreatedAt: time.Now(),
	}
	if err := db.Create(&loginCode).Error; err != nil {
		t.Fatalf("failed to seed login code: %v", err)
	}

	loginRecorder := performAllergyRequest(t, engine, http.MethodPost, "/api/auth/login", map[string]any{
		"email": "member@example.com",
		"code":  "123456",
	}, nil)
	if loginRecorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", loginRecorder.Code)
	}

	loginResponse := decodeAllergyJSON[allergyLoginResponse](t, loginRecorder)
	if !loginResponse.Success {
		t.Fatalf("expected success response, got body: %s", loginRecorder.Body.String())
	}
	if loginResponse.Token == "" {
		t.Fatalf("expected session token to be returned")
	}
	if loginResponse.Email != "member@example.com" {
		t.Fatalf("expected response email to match login email, got %q", loginResponse.Email)
	}

	var user model.User
	if err := db.Where("email = ?", "member@example.com").First(&user).Error; err != nil {
		t.Fatalf("expected member user to be created: %v", err)
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

	var session model.MemberSession
	if err := db.Where("user_id = ?", user.Id).First(&session).Error; err != nil {
		t.Fatalf("expected member session to be created: %v", err)
	}
	if session.TokenHash == "" || session.TokenHash == loginResponse.Token {
		t.Fatalf("expected token hash to be stored instead of raw token")
	}
	if session.RevokedAt != nil {
		t.Fatalf("expected fresh session to be active")
	}

	var usedCode model.EmailLoginCodeStore
	if err := db.First(&usedCode, loginCode.ID).Error; err != nil {
		t.Fatalf("failed to reload login code: %v", err)
	}
	if usedCode.UsedAt == nil {
		t.Fatalf("expected login code to be marked as used")
	}

	authHeader := map[string]string{"Authorization": "Bearer " + loginResponse.Token}
	meRecorder := performAllergyRequest(t, engine, http.MethodGet, "/api/auth/me", nil, authHeader)
	if meRecorder.Code != http.StatusOK {
		t.Fatalf("expected status 200 for me, got %d", meRecorder.Code)
	}

	meResponse := decodeAllergyJSON[allergyMeResponse](t, meRecorder)
	if !meResponse.Success {
		t.Fatalf("expected me to succeed, got body: %s", meRecorder.Body.String())
	}
	if meResponse.User.ID != user.Id || meResponse.User.Email != user.Email {
		t.Fatalf("unexpected me response payload: %+v", meResponse.User)
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

func TestAllergyLoginReturnsIsNewUserFlag(t *testing.T) {
	db, engine := setupAllergyControllerTest(t)

	// 新用户：邮箱不存在 → is_new_user=true
	codeHash, _ := common.Password2Hash("111111")
	db.Create(&model.EmailLoginCodeStore{
		Email: "newbie@example.com", Purpose: "login",
		CodeHash: codeHash, ExpiresAt: time.Now().Add(5 * time.Minute), CreatedAt: time.Now(),
	})

	rec := performAllergyRequest(t, engine, http.MethodPost, "/api/auth/login", map[string]any{
		"email": "newbie@example.com", "code": "111111",
	}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	resp := decodeAllergyJSON[allergyLoginResponse](t, rec)
	if !resp.Success {
		t.Fatalf("expected success, got %s", rec.Body.String())
	}
	if !resp.IsNewUser {
		t.Fatalf("expected is_new_user=true for first-time login, got false")
	}
	firstToken := resp.Token

	// 老用户：同邮箱再次登录 → is_new_user=false
	codeHash2, _ := common.Password2Hash("222222")
	db.Create(&model.EmailLoginCodeStore{
		Email: "newbie@example.com", Purpose: "login",
		CodeHash: codeHash2, ExpiresAt: time.Now().Add(5 * time.Minute), CreatedAt: time.Now(),
	})
	rec2 := performAllergyRequest(t, engine, http.MethodPost, "/api/auth/login", map[string]any{
		"email": "newbie@example.com", "code": "222222",
	}, nil)
	resp2 := decodeAllergyJSON[allergyLoginResponse](t, rec2)
	if resp2.IsNewUser {
		t.Fatalf("expected is_new_user=false for returning user")
	}
	if resp2.Token == firstToken {
		t.Fatalf("expected different session token on second login")
	}
}

func TestAllergyUpdateProfileSavesNicknameAndPhone(t *testing.T) {
	db, engine := setupAllergyControllerTest(t)

	// 先登录拿 token
	codeHash, _ := common.Password2Hash("999999")
	db.Create(&model.EmailLoginCodeStore{
		Email: "profile@example.com", Purpose: "login",
		CodeHash: codeHash, ExpiresAt: time.Now().Add(5 * time.Minute), CreatedAt: time.Now(),
	})
	loginRec := performAllergyRequest(t, engine, http.MethodPost, "/api/auth/login", map[string]any{
		"email": "profile@example.com", "code": "999999",
	}, nil)
	loginResp := decodeAllergyJSON[allergyLoginResponse](t, loginRec)
	if !loginResp.Success {
		t.Fatalf("login failed: %s", loginRec.Body.String())
	}
	authHeader := map[string]string{"Authorization": "Bearer " + loginResp.Token}

	// 更新 profile
	patchRec := performAllergyRequest(t, engine, http.MethodPatch, "/api/auth/profile", allergyUpdateProfileRequest{
		Nickname: "小花",
		Phone:    "13900139000",
	}, authHeader)
	if patchRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", patchRec.Code, patchRec.Body.String())
	}
	patchResp := decodeAllergyJSON[allergyAuthActionResponse](t, patchRec)
	if !patchResp.Success {
		t.Fatalf("expected profile update to succeed: %s", patchRec.Body.String())
	}

	// 验证 /me 返回更新后的资料
	meRec := performAllergyRequest(t, engine, http.MethodGet, "/api/auth/me", nil, authHeader)
	meResp := decodeAllergyJSON[allergyMeResponse](t, meRec)
	if meResp.User.Nickname != "小花" {
		t.Fatalf("expected nickname '小花', got %q", meResp.User.Nickname)
	}
	if meResp.User.Phone != "13900139000" {
		t.Fatalf("expected phone '13900139000', got %q", meResp.User.Phone)
	}

	// 验证数据库里的 member_profile 也更新了
	var dbProfile model.MemberProfile
	db.Where("user_id = ?", loginResp.User.ID).First(&dbProfile)
	if dbProfile.Nickname != "小花" || dbProfile.Phone != "13900139000" {
		t.Fatalf("expected profile in DB to be updated, got nickname=%q phone=%q",
			dbProfile.Nickname, dbProfile.Phone)
	}
}

func TestSendAllergyLoginCodeRejectsAdminEmail(t *testing.T) {
	db, engine := setupAllergyControllerTest(t)

	passwordHash, err := common.Password2Hash("admin-password")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	admin := model.User{
		Username:    "allergy-admin",
		Password:    passwordHash,
		DisplayName: "Allergy Admin",
		Role:        common.RoleAdminUser,
		Status:      common.UserStatusEnabled,
		Email:       "admin@example.com",
		Group:       "default",
		AffCode:     "ADM1",
	}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("failed to create admin user: %v", err)
	}

	recorder := performAllergyRequest(t, engine, http.MethodPost, "/api/auth/send-code", map[string]any{
		"email": "admin@example.com",
	}, nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	response := decodeAllergyJSON[allergyAuthActionResponse](t, recorder)
	if response.Success {
		t.Fatalf("expected admin email to be rejected")
	}

	var count int64
	if err := db.Model(&model.EmailLoginCodeStore{}).Where("email = ?", "admin@example.com").Count(&count).Error; err != nil {
		t.Fatalf("failed to count login codes: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no login code to be created for admin email, got %d", count)
	}
}
