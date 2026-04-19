package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Calcium-Ion/go-epay/epay"
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type allergyAPIResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type allergyOrderCreateData struct {
	OrderID       int64  `json:"order_id"`
	OrderNo       string `json:"order_no"`
	PaymentStatus string `json:"payment_status"`
	OrderStatus   string `json:"order_status"`
}

type allergyPayData struct {
	PaymentMethod string            `json:"payment_method"`
	TradeNo       string            `json:"trade_no"`
	RedirectURL   string            `json:"redirect_url"`
	FormData      map[string]string `json:"form_data"`
	PaymentStatus string            `json:"payment_status"`
}

type allergyPayStatusData struct {
	OrderID       int64  `json:"order_id"`
	PaymentStatus string `json:"payment_status"`
	OrderStatus   string `json:"order_status"`
	PaidAt        string `json:"paid_at"`
}

type allergyOrderListItem struct {
	OrderID       int64  `json:"order_id"`
	OrderNo       string `json:"order_no"`
	ServiceName   string `json:"service_name"`
	PaymentStatus string `json:"payment_status"`
	OrderStatus   string `json:"order_status"`
}

type allergyPaymentMethodDTO struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

type allergyOrderDetailData struct {
	OrderID                 int64                     `json:"order_id"`
	OrderNo                 string                    `json:"order_no"`
	ServiceName             string                    `json:"service_name"`
	ServicePriceCents       int                       `json:"service_price_cents"`
	Currency                string                    `json:"currency"`
	PaymentStatus           string                    `json:"payment_status"`
	OrderStatus             string                    `json:"order_status"`
	AvailablePaymentMethods []allergyPaymentMethodDTO `json:"available_payment_methods"`
	SampleKit               struct {
		KitCode            string `json:"kit_code"`
		KitStatus          string `json:"kit_status"`
		OutboundTrackingNo string `json:"outbound_tracking_no"`
	} `json:"sample_kit"`
}

type allergyTimelineItem struct {
	EventType   string `json:"event_type"`
	Title       string `json:"title"`
	Description string `json:"description"`
	OccurredAt  string `json:"occurred_at"`
}

type allergyOrderReportData struct {
	ReportID     int64  `json:"report_id"`
	ReportTitle  string `json:"report_title"`
	ReportStatus string `json:"report_status"`
	PreviewURL   string `json:"preview_url"`
	DownloadURL  string `json:"download_url"`
}

type allergyAdminReportUploadData struct {
	ReportID     int64  `json:"report_id"`
	ReportStatus string `json:"report_status"`
}

type allergyAdminSendEmailData struct {
	ReportID       int64  `json:"report_id"`
	TargetEmail    string `json:"target_email"`
	DeliveryStatus string `json:"delivery_status"`
}

type allergyAdminOrderListItem struct {
	OrderID        int64  `json:"order_id"`
	OrderNo        string `json:"order_no"`
	ServiceName    string `json:"service_name"`
	RecipientName  string `json:"recipient_name"`
	RecipientEmail string `json:"recipient_email"`
	PaymentStatus  string `json:"payment_status"`
	OrderStatus    string `json:"order_status"`
	PaidAt         string `json:"paid_at"`
}

type allergyAdminOrderListPage struct {
	Page     int                         `json:"page"`
	PageSize int                         `json:"page_size"`
	Total    int                         `json:"total"`
	Items    []allergyAdminOrderListItem `json:"items"`
}

type allergyAdminServiceProductItem struct {
	ID          int64  `json:"id"`
	ServiceCode string `json:"service_code"`
	Title       string `json:"title"`
	PriceCents  int    `json:"price_cents"`
	Currency    string `json:"currency"`
	Status      string `json:"status"`
	SortOrder   int    `json:"sort_order"`
}

type allergyAdminServiceProductPage struct {
	Page     int                              `json:"page"`
	PageSize int                              `json:"page_size"`
	Total    int                              `json:"total"`
	Items    []allergyAdminServiceProductItem `json:"items"`
}

type allergyAdminServiceProductDetail struct {
	ID          int64  `json:"id"`
	ServiceCode string `json:"service_code"`
	Title       string `json:"title"`
	Description string `json:"description"`
	ImageURL    string `json:"image_url"`
	CTAText     string `json:"cta_text"`
	Tag         string `json:"tag"`
	PriceCents  int    `json:"price_cents"`
	Currency    string `json:"currency"`
	SortOrder   int    `json:"sort_order"`
	Status      string `json:"status"`
}

type allergyAdminTimelineItem struct {
	EventType        string `json:"event_type"`
	Title            string `json:"title"`
	Description      string `json:"description"`
	VisibleToUser    bool   `json:"visible_to_user"`
	EventPayloadJSON string `json:"event_payload_json"`
	OccurredAt       string `json:"occurred_at"`
}

type allergyAdminReportItem struct {
	ReportID        int64  `json:"report_id"`
	ReportTitle     string `json:"report_title"`
	ReportStatus    string `json:"report_status"`
	Version         int    `json:"version"`
	IsCurrent       bool   `json:"is_current"`
	PreviewURL      string `json:"preview_url"`
	DownloadURL     string `json:"download_url"`
	EmailSentCount  int    `json:"email_sent_count"`
	LastEmailSentAt string `json:"last_email_sent_at"`
}

type allergyAdminOrderDetailData struct {
	OrderID                    int64  `json:"order_id"`
	OrderNo                    string `json:"order_no"`
	ServiceName                string `json:"service_name"`
	PaymentStatus              string `json:"payment_status"`
	OrderStatus                string `json:"order_status"`
	PaymentMethod              string `json:"payment_method"`
	PaymentRef                 string `json:"payment_ref"`
	PaymentProviderOrderNo     string `json:"payment_provider_order_no"`
	PaymentCallbackPayloadJSON string `json:"payment_callback_payload_json"`
	AdminRemark                string `json:"admin_remark"`
	SampleKit                  *struct {
		KitCode            string `json:"kit_code"`
		KitStatus          string `json:"kit_status"`
		OutboundTrackingNo string `json:"outbound_tracking_no"`
		ReturnTrackingNo   string `json:"return_tracking_no"`
		SampleReceivedAt   string `json:"sample_received_at"`
	} `json:"sample_kit"`
	LabSubmission *struct {
		Status           string `json:"status"`
		TrackingNumber   string `json:"tracking_number"`
		SubmittedAt      string `json:"submitted_at"`
		TestingStartedAt string `json:"testing_started_at"`
		CompletedAt      string `json:"completed_at"`
	} `json:"lab_submission"`
	CurrentReport *allergyAdminReportItem    `json:"current_report"`
	Reports       []allergyAdminReportItem   `json:"reports"`
	Timeline      []allergyAdminTimelineItem `json:"timeline"`
}

type allergyDeliveryLogItem struct {
	Target          string `json:"target"`
	Status          string `json:"status"`
	DeliveryChannel string `json:"delivery_channel"`
}

func setupAllergyFlowControllerTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false
	common.CryptoSecret = "allergy-flow-controller-test-secret"

	originalPayAddress := operation_setting.PayAddress
	originalCallback := operation_setting.CustomCallbackAddress
	originalEpayID := operation_setting.EpayId
	originalEpayKey := operation_setting.EpayKey
	originalPayMethods := operation_setting.PayMethods
	originalServerAddress := system_setting.ServerAddress
	originalStorageDir := os.Getenv("ALLERGY_REPORT_STORAGE_DIR")

	storageDir := t.TempDir()
	if err := os.Setenv("ALLERGY_REPORT_STORAGE_DIR", storageDir); err != nil {
		t.Fatalf("failed to set report storage dir: %v", err)
	}

	operation_setting.PayAddress = "https://pay.example.com"
	operation_setting.CustomCallbackAddress = ""
	operation_setting.EpayId = "epay-partner-id"
	operation_setting.EpayKey = "epay-secret"
	operation_setting.PayMethods = []map[string]string{
		{"name": "支付宝", "type": "alipay"},
		{"name": "微信", "type": "wxpay"},
	}
	system_setting.ServerAddress = "https://api.allergy.test"

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
		&model.AllergyOrder{},
		&model.SampleKit{},
		&model.LabSubmission{},
		&model.LabReport{},
		&model.ReportDeliveryLog{},
		&model.OrderTimelineEvent{},
		&model.AllergyServiceProduct{},
	); err != nil {
		t.Fatalf("failed to migrate allergy flow tables: %v", err)
	}
	if err := model.EnsureDefaultAllergyServiceProducts(); err != nil {
		t.Fatalf("failed to seed default allergy service products: %v", err)
	}

	engine := gin.New()
	engine.GET("/api/products", GetAllergyProducts)
	memberRoute := engine.Group("/api")
	memberRoute.Use(middleware.AllergyMemberAuth())
	{
		memberRoute.POST("/orders", CreateAllergyOrder)
		memberRoute.GET("/orders", ListAllergyOrders)
		memberRoute.GET("/orders/:id", GetAllergyOrderDetail)
		memberRoute.POST("/orders/:id/pay", RequestAllergyOrderEpay)
		memberRoute.GET("/orders/:id/pay-status", GetAllergyOrderPayStatus)
		memberRoute.GET("/orders/:id/timeline", GetAllergyOrderTimeline)
		memberRoute.GET("/orders/:id/report", GetAllergyOrderReport)
		memberRoute.GET("/reports/:id/preview", PreviewAllergyReport)
		memberRoute.GET("/reports/:id/download", DownloadAllergyReport)
	}

	engine.POST("/api/orders/epay/notify", AllergyOrderEpayNotify)
	engine.GET("/api/orders/epay/notify", AllergyOrderEpayNotify)
	engine.POST("/api/orders/epay/return", AllergyOrderEpayReturn)
	engine.GET("/api/orders/epay/return", AllergyOrderEpayReturn)

	adminRoute := engine.Group("/api/admin")
	adminRoute.Use(func(c *gin.Context) {
		adminID, err := strconv.Atoi(strings.TrimSpace(c.GetHeader("X-Test-Admin-Id")))
		if err != nil || adminID <= 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "missing test admin"})
			c.Abort()
			return
		}
		c.Set("id", adminID)
		c.Next()
	})
	{
		adminRoute.GET("/orders", ListAdminAllergyOrders)
		adminRoute.GET("/orders/:id", GetAdminAllergyOrderDetail)
		adminRoute.PATCH("/orders/:id/status", UpdateAdminAllergyOrderStatus)
		adminRoute.POST("/orders/:id/kit", UpsertAdminAllergyOrderKit)
		adminRoute.POST("/orders/:id/sample-sent-back", MarkAdminAllergySampleSentBack)
		adminRoute.POST("/orders/:id/sample-received", MarkAdminAllergySampleReceived)
		adminRoute.POST("/orders/:id/testing-started", StartAdminAllergyTesting)
		adminRoute.POST("/orders/:id/report", UploadAdminAllergyOrderReport)
		adminRoute.POST("/orders/:id/complete", CompleteAdminAllergyOrder)
		adminRoute.GET("/reports/:id/preview", PreviewAdminAllergyReport)
		adminRoute.GET("/reports/:id/download", DownloadAdminAllergyReport)
		adminRoute.POST("/reports/:id/publish", PublishAdminAllergyReport)
		adminRoute.POST("/reports/:id/send-email", SendAdminAllergyReportEmail)
		adminRoute.GET("/reports/:id/delivery-logs", ListAdminAllergyReportDeliveryLogs)
		adminRoute.GET("/service-products", ListAdminAllergyServiceProducts)
		adminRoute.GET("/service-products/:id", GetAdminAllergyServiceProduct)
		adminRoute.POST("/service-products", CreateAdminAllergyServiceProduct)
		adminRoute.PATCH("/service-products/:id", UpdateAdminAllergyServiceProduct)
		adminRoute.POST("/service-products/:id/publish", PublishAdminAllergyServiceProduct)
		adminRoute.POST("/service-products/:id/archive", ArchiveAdminAllergyServiceProduct)
	}

	t.Cleanup(func() {
		operation_setting.PayAddress = originalPayAddress
		operation_setting.CustomCallbackAddress = originalCallback
		operation_setting.EpayId = originalEpayID
		operation_setting.EpayKey = originalEpayKey
		operation_setting.PayMethods = originalPayMethods
		system_setting.ServerAddress = originalServerAddress
		if originalStorageDir == "" {
			_ = os.Unsetenv("ALLERGY_REPORT_STORAGE_DIR")
		} else {
			_ = os.Setenv("ALLERGY_REPORT_STORAGE_DIR", originalStorageDir)
		}
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db, engine
}

func performFlowRequest(t *testing.T, engine *gin.Engine, method string, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
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

func performMultipartFlowRequest(t *testing.T, engine *gin.Engine, path string, fields map[string]string, fileField string, fileName string, fileContent []byte, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("failed to write multipart field: %v", err)
		}
	}
	part, err := writer.CreateFormFile(fileField, fileName)
	if err != nil {
		t.Fatalf("failed to create multipart file part: %v", err)
	}
	if _, err := part.Write(fileContent); err != nil {
		t.Fatalf("failed to write multipart file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, path, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)
	return recorder
}

func decodeAllergyAPIResponse[T any](t *testing.T, recorder *httptest.ResponseRecorder) T {
	t.Helper()

	var envelope allergyAPIResponse
	if err := common.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("failed to decode api response: %v body=%s", err, recorder.Body.String())
	}
	if !envelope.Success {
		t.Fatalf("expected success response, got message: %s body=%s", envelope.Message, recorder.Body.String())
	}
	var result T
	if len(envelope.Data) > 0 && string(envelope.Data) != "null" {
		if err := common.Unmarshal(envelope.Data, &result); err != nil {
			t.Fatalf("failed to decode api data: %v body=%s", err, recorder.Body.String())
		}
	}
	return result
}

func seedAllergyMemberSession(t *testing.T, db *gorm.DB, email string) (*model.User, string) {
	t.Helper()

	passwordHash, err := common.Password2Hash("member-password")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	user := &model.User{
		Username:    "u_" + strings.ReplaceAll(strings.Split(email, "@")[0], ".", "_"),
		Password:    passwordHash,
		DisplayName: "Member",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Email:       email,
		Group:       "default",
		AffCode:     strings.ToUpper(common.GetRandomString(4)),
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	verifiedAt := time.Now()
	if err := db.Create(&model.MemberProfile{
		UserID:          user.Id,
		Status:          "active",
		EmailVerifiedAt: &verifiedAt,
	}).Error; err != nil {
		t.Fatalf("failed to create member profile: %v", err)
	}
	token, _, err := model.CreateMemberSession(user.Id, model.AllergyMemberClientWeb, "test-agent", "127.0.0.1", time.Hour)
	if err != nil {
		t.Fatalf("failed to create member session: %v", err)
	}
	return user, token
}

func seedAdminUser(t *testing.T, db *gorm.DB) *model.User {
	t.Helper()

	passwordHash, err := common.Password2Hash("admin-password")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	user := &model.User{
		Username:    "admin_user",
		Password:    passwordHash,
		DisplayName: "Admin",
		Role:        common.RoleAdminUser,
		Status:      common.UserStatusEnabled,
		Email:       "admin@example.com",
		Group:       "default",
		AffCode:     strings.ToUpper(common.GetRandomString(4)),
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("failed to create admin user: %v", err)
	}
	return user
}

func writeTestPDF(t *testing.T, dir string, name string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	content := []byte("%PDF-1.4\n1 0 obj\n<<>>\nendobj\ntrailer\n<<>>\n%%EOF")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("failed to write pdf: %v", err)
	}
	return path
}

func TestAllergyOrderCreatePayAndCallbackFlow(t *testing.T) {
	db, engine := setupAllergyFlowControllerTest(t)
	user, token := seedAllergyMemberSession(t, db, "member@example.com")
	headers := map[string]string{"Authorization": "Bearer " + token}

	createRecorder := performFlowRequest(t, engine, http.MethodPost, "/api/orders", map[string]any{
		"service_code":    "allergy-test-basic",
		"recipient_name":  "张三",
		"recipient_phone": "13800000000",
		"recipient_email": user.Email,
		"shipping_address": map[string]any{
			"province":     "上海市",
			"city":         "上海市",
			"district":     "浦东新区",
			"address_line": "世纪大道 100 号",
		},
	}, headers)
	createData := decodeAllergyAPIResponse[allergyOrderCreateData](t, createRecorder)
	if createData.PaymentStatus != "pending" || createData.OrderStatus != "pending_payment" {
		t.Fatalf("unexpected create response: %+v", createData)
	}

	payRecorder := performFlowRequest(t, engine, http.MethodPost, fmt.Sprintf("/api/orders/%d/pay", createData.OrderID), map[string]any{
		"payment_method": "alipay",
		"success_url":    "https://www.allergy.test/orders/1",
		"cancel_url":     "https://www.allergy.test/orders/1",
	}, headers)
	payData := decodeAllergyAPIResponse[allergyPayData](t, payRecorder)
	if payData.PaymentMethod != "alipay" || payData.TradeNo == "" {
		t.Fatalf("unexpected pay response: %+v", payData)
	}
	if !strings.Contains(payData.RedirectURL, "submit.php") {
		t.Fatalf("expected epay redirect url, got %q", payData.RedirectURL)
	}

	callbackParams := map[string]string{
		"pid":          operation_setting.EpayId,
		"type":         "alipay",
		"out_trade_no": payData.TradeNo,
		"trade_no":     "EPAY-THIRD-ORDER-001",
		"name":         "Allergy Order",
		"money":        "199.00",
		"trade_status": epay.StatusTradeSuccess,
		"sign_type":    "MD5",
	}
	signed := epay.GenerateParams(callbackParams, operation_setting.EpayKey)
	form := url.Values{}
	for key, value := range signed {
		form.Set(key, value)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/orders/epay/notify", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)
	if recorder.Body.String() != "success" {
		t.Fatalf("expected success callback body, got %q", recorder.Body.String())
	}

	payStatusRecorder := performFlowRequest(t, engine, http.MethodGet, fmt.Sprintf("/api/orders/%d/pay-status", createData.OrderID), nil, headers)
	payStatus := decodeAllergyAPIResponse[allergyPayStatusData](t, payStatusRecorder)
	if payStatus.PaymentStatus != "paid" || payStatus.OrderStatus != "paid" || payStatus.PaidAt == "" {
		t.Fatalf("unexpected pay status response: %+v", payStatus)
	}

	timelineRecorder := performFlowRequest(t, engine, http.MethodGet, fmt.Sprintf("/api/orders/%d/timeline", createData.OrderID), nil, headers)
	timeline := decodeAllergyAPIResponse[[]allergyTimelineItem](t, timelineRecorder)
	if len(timeline) < 2 {
		t.Fatalf("expected at least 2 timeline events, got %d", len(timeline))
	}
	if timeline[0].EventType != "order_created" {
		t.Fatalf("expected first timeline event to be order_created, got %+v", timeline[0])
	}
	foundPaymentCompleted := false
	for _, item := range timeline {
		if item.EventType == "payment_completed" {
			foundPaymentCompleted = true
			break
		}
	}
	if !foundPaymentCompleted {
		t.Fatalf("expected payment_completed timeline event, got %+v", timeline)
	}
}

func TestAllergyServiceProductAdminCatalogAndOrderSnapshot(t *testing.T) {
	db, engine := setupAllergyFlowControllerTest(t)
	user, token := seedAllergyMemberSession(t, db, "member@example.com")
	admin := seedAdminUser(t, db)
	memberHeaders := map[string]string{"Authorization": "Bearer " + token}
	adminHeaders := map[string]string{"X-Test-Admin-Id": strconv.Itoa(admin.Id)}

	createProductRecorder := performFlowRequest(t, engine, http.MethodPost, "/api/admin/service-products", map[string]any{
		"service_code": "children-panel",
		"title":        "儿童专项过敏原检测",
		"description":  "面向儿童常见过敏原的检测项目",
		"image_url":    "https://cdn.example.com/product-children.jpg",
		"cta_text":     "立即检测",
		"tag":          "新品",
		"price_cents":  29900,
		"sort_order":   1,
		"status":       "draft",
	}, adminHeaders)
	createdProduct := decodeAllergyAPIResponse[allergyAdminServiceProductDetail](t, createProductRecorder)
	if createdProduct.ID == 0 || createdProduct.Status != "draft" || createdProduct.Currency != "CNY" {
		t.Fatalf("unexpected created product: %+v", createdProduct)
	}

	initialPublicRecorder := performFlowRequest(t, engine, http.MethodGet, "/api/products", nil, nil)
	initialPublicProducts := decodeAllergyJSON[[]allergyProductResponse](t, initialPublicRecorder)
	for _, product := range initialPublicProducts {
		if product.ID == "children-panel" {
			t.Fatalf("draft product must not be publicly listed: %+v", initialPublicProducts)
		}
	}

	publishRecorder := performFlowRequest(t, engine, http.MethodPost, fmt.Sprintf("/api/admin/service-products/%d/publish", createdProduct.ID), nil, adminHeaders)
	publishedProduct := decodeAllergyAPIResponse[allergyAdminServiceProductDetail](t, publishRecorder)
	if publishedProduct.Status != "published" {
		t.Fatalf("expected published status, got %+v", publishedProduct)
	}

	publicRecorder := performFlowRequest(t, engine, http.MethodGet, "/api/products", nil, nil)
	publicProducts := decodeAllergyJSON[[]allergyProductResponse](t, publicRecorder)
	if len(publicProducts) == 0 || publicProducts[0].ID != "children-panel" || publicProducts[0].PriceCents != 29900 {
		t.Fatalf("expected published children product first, got %+v", publicProducts)
	}

	createOrderRecorder := performFlowRequest(t, engine, http.MethodPost, "/api/orders", map[string]any{
		"service_code":    "children-panel",
		"recipient_name":  "张三",
		"recipient_phone": "13800000000",
		"recipient_email": user.Email,
		"shipping_address": map[string]any{
			"province":     "上海市",
			"city":         "上海市",
			"district":     "浦东新区",
			"address_line": "世纪大道 100 号",
		},
	}, memberHeaders)
	createdOrder := decodeAllergyAPIResponse[allergyOrderCreateData](t, createOrderRecorder)

	patchRecorder := performFlowRequest(t, engine, http.MethodPatch, fmt.Sprintf("/api/admin/service-products/%d", createdProduct.ID), map[string]any{
		"title":       "儿童专项过敏原检测（升级版）",
		"description": "升级后的检测项目说明",
		"image_url":   "https://cdn.example.com/product-children-v2.jpg",
		"cta_text":    "立即预约",
		"tag":         "热门",
		"price_cents": 32900,
		"sort_order":  2,
		"status":      "published",
	}, adminHeaders)
	updatedProduct := decodeAllergyAPIResponse[allergyAdminServiceProductDetail](t, patchRecorder)
	if updatedProduct.ServiceCode != "children-panel" || updatedProduct.PriceCents != 32900 {
		t.Fatalf("unexpected updated product: %+v", updatedProduct)
	}

	var persistedOrder model.AllergyOrder
	if err := db.First(&persistedOrder, createdOrder.OrderID).Error; err != nil {
		t.Fatalf("failed to reload order: %v", err)
	}
	if persistedOrder.ServiceCode != "children-panel" ||
		persistedOrder.ServiceNameSnapshot != "儿童专项过敏原检测" ||
		persistedOrder.ServicePriceCents != 29900 {
		t.Fatalf("expected order snapshot to stay unchanged, got %+v", persistedOrder)
	}

	archiveRecorder := performFlowRequest(t, engine, http.MethodPost, fmt.Sprintf("/api/admin/service-products/%d/archive", createdProduct.ID), nil, adminHeaders)
	archivedProduct := decodeAllergyAPIResponse[allergyAdminServiceProductDetail](t, archiveRecorder)
	if archivedProduct.Status != "archived" {
		t.Fatalf("expected archived status, got %+v", archivedProduct)
	}

	archivedOrderRecorder := performFlowRequest(t, engine, http.MethodPost, "/api/orders", map[string]any{
		"service_code":    "children-panel",
		"recipient_name":  "李四",
		"recipient_phone": "13900000000",
		"recipient_email": user.Email,
		"shipping_address": map[string]any{
			"province":     "上海市",
			"city":         "上海市",
			"district":     "浦东新区",
			"address_line": "世纪大道 200 号",
		},
	}, memberHeaders)
	var envelope allergyAPIResponse
	if err := common.Unmarshal(archivedOrderRecorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("failed to decode archived order response: %v", err)
	}
	if envelope.Success || envelope.Message != "服务不存在" {
		t.Fatalf("expected archived product order to fail, got body=%s", archivedOrderRecorder.Body.String())
	}

	listRecorder := performFlowRequest(t, engine, http.MethodGet, "/api/admin/service-products", nil, adminHeaders)
	listData := decodeAllergyAPIResponse[allergyAdminServiceProductPage](t, listRecorder)
	if listData.Total < 2 {
		t.Fatalf("expected admin product list to include default and children products, got %+v", listData)
	}
}

func TestAllergyOrderEpayReturnRedirectsToRequestedFrontendURL(t *testing.T) {
	db, engine := setupAllergyFlowControllerTest(t)
	user, token := seedAllergyMemberSession(t, db, "member@example.com")
	headers := map[string]string{"Authorization": "Bearer " + token}

	createRecorder := performFlowRequest(t, engine, http.MethodPost, "/api/orders", map[string]any{
		"service_code":    "allergy-test-basic",
		"recipient_name":  "张三",
		"recipient_phone": "13800000000",
		"recipient_email": user.Email,
		"shipping_address": map[string]any{
			"province":     "上海市",
			"city":         "上海市",
			"district":     "浦东新区",
			"address_line": "世纪大道 100 号",
		},
	}, headers)
	createData := decodeAllergyAPIResponse[allergyOrderCreateData](t, createRecorder)

	successURL := fmt.Sprintf("https://www.allergy.test/orders/%d", createData.OrderID)
	cancelURL := fmt.Sprintf("https://www.allergy.test/orders/%d?pay=cancelled", createData.OrderID)

	payRecorder := performFlowRequest(t, engine, http.MethodPost, fmt.Sprintf("/api/orders/%d/pay", createData.OrderID), map[string]any{
		"payment_method": "alipay",
		"success_url":    successURL,
		"cancel_url":     cancelURL,
	}, headers)
	payData := decodeAllergyAPIResponse[allergyPayData](t, payRecorder)

	callbackParams := map[string]string{
		"pid":          operation_setting.EpayId,
		"type":         "alipay",
		"out_trade_no": payData.TradeNo,
		"trade_no":     "EPAY-RETURN-ORDER-001",
		"name":         "Allergy Order",
		"money":        "199.00",
		"trade_status": epay.StatusTradeSuccess,
		"sign_type":    "MD5",
	}
	signed := epay.GenerateParams(callbackParams, operation_setting.EpayKey)
	query := url.Values{}
	for key, value := range signed {
		query.Set(key, value)
	}
	query.Set("success", successURL)
	query.Set("cancel", cancelURL)

	req := httptest.NewRequest(http.MethodGet, "/api/orders/epay/return?"+query.Encode(), nil)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusFound {
		t.Fatalf("expected redirect response, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if location := recorder.Header().Get("Location"); location != successURL {
		t.Fatalf("expected redirect to success url, got %q", location)
	}

	payStatusRecorder := performFlowRequest(t, engine, http.MethodGet, fmt.Sprintf("/api/orders/%d/pay-status", createData.OrderID), nil, headers)
	payStatus := decodeAllergyAPIResponse[allergyPayStatusData](t, payStatusRecorder)
	if payStatus.PaymentStatus != "paid" || payStatus.OrderStatus != "paid" {
		t.Fatalf("expected order to be paid after return callback, got %+v", payStatus)
	}
}

func TestAllergyOrderQueriesAndReportPermissions(t *testing.T) {
	db, engine := setupAllergyFlowControllerTest(t)
	user, token := seedAllergyMemberSession(t, db, "member@example.com")
	_, otherToken := seedAllergyMemberSession(t, db, "other@example.com")

	order := model.AllergyOrder{
		OrderNo:             "AO-QUERY-001",
		UserID:              user.Id,
		ServiceCode:         "allergy-test-basic",
		ServiceNameSnapshot: "埃勒吉居家过敏原检测服务",
		ServicePriceCents:   19900,
		Currency:            "CNY",
		PaymentStatus:       "paid",
		OrderStatus:         "report_ready",
		RecipientName:       "张三",
		RecipientPhone:      "13800000000",
		RecipientEmail:      user.Email,
		ShippingAddressJSON: `{"province":"上海市","city":"上海市","district":"浦东新区","address_line":"世纪大道 100 号"}`,
		ReportReadyAt:       timePtr(time.Now()),
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatalf("failed to create order: %v", err)
	}
	kit := model.SampleKit{
		OrderID:        order.ID,
		KitCode:        "KIT-QUERY-001",
		Status:         "sample_received",
		TrackingNumber: "SF1234567890",
	}
	if err := db.Create(&kit).Error; err != nil {
		t.Fatalf("failed to create sample kit: %v", err)
	}
	pdfPath := writeTestPDF(t, t.TempDir(), "report.pdf")
	report := model.LabReport{
		OrderID:       order.ID,
		Version:       1,
		Status:        "published",
		IsCurrent:     true,
		ReportTitle:   "过敏原检测报告",
		FileName:      "report.pdf",
		FilePath:      pdfPath,
		MimeType:      "application/pdf",
		FileSizeBytes: 42,
		PublishedAt:   timePtr(time.Now()),
	}
	if err := db.Create(&report).Error; err != nil {
		t.Fatalf("failed to create report: %v", err)
	}

	headers := map[string]string{"Authorization": "Bearer " + token}
	listRecorder := performFlowRequest(t, engine, http.MethodGet, "/api/orders", nil, headers)
	orderList := decodeAllergyAPIResponse[[]allergyOrderListItem](t, listRecorder)
	if len(orderList) != 1 || orderList[0].OrderID != order.ID {
		t.Fatalf("unexpected order list: %+v", orderList)
	}

	detailRecorder := performFlowRequest(t, engine, http.MethodGet, fmt.Sprintf("/api/orders/%d", order.ID), nil, headers)
	detail := decodeAllergyAPIResponse[allergyOrderDetailData](t, detailRecorder)
	if detail.SampleKit.KitCode != "KIT-QUERY-001" || detail.SampleKit.OutboundTrackingNo != "SF1234567890" {
		t.Fatalf("unexpected detail response: %+v", detail)
	}

	reportRecorder := performFlowRequest(t, engine, http.MethodGet, fmt.Sprintf("/api/orders/%d/report", order.ID), nil, headers)
	reportData := decodeAllergyAPIResponse[allergyOrderReportData](t, reportRecorder)
	if reportData.ReportID != report.ID || reportData.PreviewURL == "" || reportData.DownloadURL == "" {
		t.Fatalf("unexpected report response: %+v", reportData)
	}

	previewRecorder := performFlowRequest(t, engine, http.MethodGet, fmt.Sprintf("/api/reports/%d/preview", report.ID), nil, headers)
	if previewRecorder.Code != http.StatusOK {
		t.Fatalf("expected preview 200, got %d", previewRecorder.Code)
	}
	if !strings.Contains(previewRecorder.Header().Get("Content-Disposition"), "inline") {
		t.Fatalf("expected inline disposition, got %q", previewRecorder.Header().Get("Content-Disposition"))
	}

	downloadRecorder := performFlowRequest(t, engine, http.MethodGet, fmt.Sprintf("/api/reports/%d/download", report.ID), nil, headers)
	if downloadRecorder.Code != http.StatusOK {
		t.Fatalf("expected download 200, got %d", downloadRecorder.Code)
	}
	if !strings.Contains(downloadRecorder.Header().Get("Content-Disposition"), "attachment") {
		t.Fatalf("expected attachment disposition, got %q", downloadRecorder.Header().Get("Content-Disposition"))
	}

	otherHeaders := map[string]string{"Authorization": "Bearer " + otherToken}
	forbiddenPreview := performFlowRequest(t, engine, http.MethodGet, fmt.Sprintf("/api/reports/%d/preview", report.ID), nil, otherHeaders)
	if forbiddenPreview.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden preview for other user, got %d body=%s", forbiddenPreview.Code, forbiddenPreview.Body.String())
	}
}

func TestAllergyOrderDetailReturnsPriceSnapshotAndDynamicPaymentMethods(t *testing.T) {
	db, engine := setupAllergyFlowControllerTest(t)
	user, token := seedAllergyMemberSession(t, db, "member@example.com")
	headers := map[string]string{"Authorization": "Bearer " + token}

	createRecorder := performFlowRequest(t, engine, http.MethodPost, "/api/orders", map[string]any{
		"service_code":    "allergy-test-basic",
		"recipient_name":  "张三",
		"recipient_phone": "13800000000",
		"recipient_email": user.Email,
		"shipping_address": map[string]any{
			"province":     "上海市",
			"city":         "上海市",
			"district":     "浦东新区",
			"address_line": "世纪大道 100 号",
		},
	}, headers)
	createData := decodeAllergyAPIResponse[allergyOrderCreateData](t, createRecorder)

	originalPayMethods := operation_setting.PayMethods
	operation_setting.PayMethods = []map[string]string{
		{"name": "银联支付", "type": "unionpay"},
		{"name": "微信支付", "type": "wxpay"},
	}
	t.Cleanup(func() {
		operation_setting.PayMethods = originalPayMethods
	})

	detailRecorder := performFlowRequest(t, engine, http.MethodGet, fmt.Sprintf("/api/orders/%d", createData.OrderID), nil, headers)
	detail := decodeAllergyAPIResponse[allergyOrderDetailData](t, detailRecorder)
	if detail.ServicePriceCents <= 0 || detail.Currency != "CNY" {
		t.Fatalf("expected price snapshot and currency, got %+v", detail)
	}
	if len(detail.AvailablePaymentMethods) != 2 {
		t.Fatalf("expected 2 payment methods, got %+v", detail.AvailablePaymentMethods)
	}
	if detail.AvailablePaymentMethods[0].Code != "unionpay" || detail.AvailablePaymentMethods[0].Label != "银联支付" {
		t.Fatalf("expected dynamic payment methods from setting, got %+v", detail.AvailablePaymentMethods)
	}
}

func TestAllergyAdminFulfillmentAndReportFlow(t *testing.T) {
	db, engine := setupAllergyFlowControllerTest(t)
	user, _ := seedAllergyMemberSession(t, db, "member@example.com")
	admin := seedAdminUser(t, db)
	paidAt := time.Now().Add(-2 * time.Hour)
	sentBackAt := paidAt.Add(40 * time.Minute)
	receivedAt := sentBackAt.Add(8 * time.Hour)
	testingStartedAt := receivedAt.Add(30 * time.Minute)
	completedAt := testingStartedAt.Add(24 * time.Hour)

	order := model.AllergyOrder{
		OrderNo:                    "AO-ADMIN-001",
		UserID:                     user.Id,
		ServiceCode:                "allergy-test-basic",
		ServiceNameSnapshot:        "埃勒吉居家过敏原检测服务",
		ServicePriceCents:          19900,
		Currency:                   "CNY",
		PaymentStatus:              "paid",
		OrderStatus:                "paid",
		RecipientName:              "张三",
		RecipientPhone:             "13800000000",
		RecipientEmail:             user.Email,
		ShippingAddressJSON:        `{"province":"上海市","city":"上海市","district":"浦东新区","address_line":"世纪大道 100 号"}`,
		PaymentMethod:              "alipay",
		PaymentRef:                 "AO_PAY_ADMIN_001",
		PaymentProviderOrderNo:     "EPAY-ADMIN-001",
		PaymentCallbackPayloadJSON: `{"trade_no":"EPAY-ADMIN-001","status":"success"}`,
		PaidAt:                     timePtr(paidAt),
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatalf("failed to create order: %v", err)
	}
	headers := map[string]string{"X-Test-Admin-Id": strconv.Itoa(admin.Id)}

	statusRecorder := performFlowRequest(t, engine, http.MethodPatch, fmt.Sprintf("/api/admin/orders/%d/status", order.ID), map[string]any{
		"order_status": "kit_preparing",
		"remark":       "已通知仓库备货",
	}, headers)
	_ = decodeAllergyAPIResponse[map[string]any](t, statusRecorder)

	kitRecorder := performFlowRequest(t, engine, http.MethodPost, fmt.Sprintf("/api/admin/orders/%d/kit", order.ID), map[string]any{
		"kit_code":             "KIT-ADMIN-001",
		"kit_status":           "shipped",
		"outbound_carrier":     "顺丰",
		"outbound_tracking_no": "SF1234567890",
		"outbound_shipped_at":  paidAt.Add(10 * time.Minute).Format(time.RFC3339),
	}, headers)
	_ = decodeAllergyAPIResponse[map[string]any](t, kitRecorder)

	sentBackRecorder := performFlowRequest(t, engine, http.MethodPost, fmt.Sprintf("/api/admin/orders/%d/sample-sent-back", order.ID), map[string]any{
		"sent_back_at":       sentBackAt.Format(time.RFC3339),
		"return_tracking_no": "SF-RETURN-001",
		"remark":             "用户已回寄样本",
	}, headers)
	_ = decodeAllergyAPIResponse[map[string]any](t, sentBackRecorder)

	sampleRecorder := performFlowRequest(t, engine, http.MethodPost, fmt.Sprintf("/api/admin/orders/%d/sample-received", order.ID), map[string]any{
		"received_at": receivedAt.Format(time.RFC3339),
		"remark":      "检测机构已签收",
	}, headers)
	_ = decodeAllergyAPIResponse[map[string]any](t, sampleRecorder)

	adminListRecorder := performFlowRequest(t, engine, http.MethodGet, "/api/admin/orders?payment_status=paid&order_status=sample_received", nil, headers)
	adminListData := decodeAllergyAPIResponse[allergyAdminOrderListPage](t, adminListRecorder)
	if adminListData.Total != 1 || len(adminListData.Items) != 1 {
		t.Fatalf("unexpected admin list: %+v", adminListData)
	}
	if adminListData.Items[0].ServiceName != "埃勒吉居家过敏原检测服务" || adminListData.Items[0].PaidAt == "" {
		t.Fatalf("unexpected admin list item: %+v", adminListData.Items[0])
	}

	testingRecorder := performFlowRequest(t, engine, http.MethodPost, fmt.Sprintf("/api/admin/orders/%d/testing-started", order.ID), map[string]any{
		"started_at": testingStartedAt.Format(time.RFC3339),
		"remark":     "实验室开始检测",
	}, headers)
	_ = decodeAllergyAPIResponse[map[string]any](t, testingRecorder)

	reportBytes := []byte("%PDF-1.4\n1 0 obj\n<<>>\nendobj\ntrailer\n<<>>\n%%EOF")
	uploadRecorder := performMultipartFlowRequest(t, engine, fmt.Sprintf("/api/admin/orders/%d/report", order.ID), map[string]string{
		"report_title": "过敏原检测报告",
	}, "file", "report.pdf", reportBytes, headers)
	uploadData := decodeAllergyAPIResponse[allergyAdminReportUploadData](t, uploadRecorder)
	if uploadData.ReportStatus != "uploaded" || uploadData.ReportID == 0 {
		t.Fatalf("unexpected upload response: %+v", uploadData)
	}

	publishRecorder := performFlowRequest(t, engine, http.MethodPost, fmt.Sprintf("/api/admin/reports/%d/publish", uploadData.ReportID), nil, headers)
	_ = decodeAllergyAPIResponse[map[string]any](t, publishRecorder)

	emailRecorder := performFlowRequest(t, engine, http.MethodPost, fmt.Sprintf("/api/admin/reports/%d/send-email", uploadData.ReportID), map[string]any{
		"target_email": user.Email,
	}, headers)
	emailData := decodeAllergyAPIResponse[allergyAdminSendEmailData](t, emailRecorder)
	if emailData.DeliveryStatus != "sent" || emailData.TargetEmail != user.Email {
		t.Fatalf("unexpected email send response: %+v", emailData)
	}

	logRecorder := performFlowRequest(t, engine, http.MethodGet, fmt.Sprintf("/api/admin/reports/%d/delivery-logs", uploadData.ReportID), nil, headers)
	logs := decodeAllergyAPIResponse[[]allergyDeliveryLogItem](t, logRecorder)
	if len(logs) != 1 || logs[0].Status != "sent" {
		t.Fatalf("unexpected delivery logs: %+v", logs)
	}

	completeRecorder := performFlowRequest(t, engine, http.MethodPost, fmt.Sprintf("/api/admin/orders/%d/complete", order.ID), map[string]any{
		"completed_at": completedAt.Format(time.RFC3339),
		"remark":       "人工确认履约完成",
	}, headers)
	_ = decodeAllergyAPIResponse[map[string]any](t, completeRecorder)

	detailRecorder := performFlowRequest(t, engine, http.MethodGet, fmt.Sprintf("/api/admin/orders/%d", order.ID), nil, headers)
	detail := decodeAllergyAPIResponse[allergyAdminOrderDetailData](t, detailRecorder)
	if detail.OrderStatus != "completed" || detail.PaymentMethod != "alipay" || detail.PaymentRef != "AO_PAY_ADMIN_001" {
		t.Fatalf("unexpected admin detail order fields: %+v", detail)
	}
	if !strings.Contains(detail.PaymentCallbackPayloadJSON, "EPAY-ADMIN-001") || detail.AdminRemark != "人工确认履约完成" {
		t.Fatalf("unexpected admin detail payment fields: %+v", detail)
	}
	if detail.SampleKit == nil || detail.SampleKit.ReturnTrackingNo != "SF-RETURN-001" || detail.SampleKit.SampleReceivedAt == "" {
		t.Fatalf("unexpected admin detail sample kit: %+v", detail.SampleKit)
	}
	if detail.LabSubmission == nil || detail.LabSubmission.Status != "completed" || detail.LabSubmission.TrackingNumber != "SF-RETURN-001" {
		t.Fatalf("unexpected admin detail lab submission: %+v", detail.LabSubmission)
	}
	if detail.CurrentReport == nil || detail.CurrentReport.ReportStatus != "published" || detail.CurrentReport.PreviewURL == "" || detail.CurrentReport.DownloadURL == "" {
		t.Fatalf("unexpected admin detail current report: %+v", detail.CurrentReport)
	}
	if len(detail.Reports) != 1 || detail.Reports[0].EmailSentCount != 1 || detail.Reports[0].LastEmailSentAt == "" {
		t.Fatalf("unexpected admin detail reports: %+v", detail.Reports)
	}

	adminPreviewRecorder := performFlowRequest(t, engine, http.MethodGet, detail.CurrentReport.PreviewURL, nil, headers)
	if adminPreviewRecorder.Code != http.StatusOK {
		t.Fatalf("expected admin preview 200, got %d body=%s", adminPreviewRecorder.Code, adminPreviewRecorder.Body.String())
	}
	adminDownloadRecorder := performFlowRequest(t, engine, http.MethodGet, detail.CurrentReport.DownloadURL, nil, headers)
	if adminDownloadRecorder.Code != http.StatusOK {
		t.Fatalf("expected admin download 200, got %d body=%s", adminDownloadRecorder.Code, adminDownloadRecorder.Body.String())
	}

	var persistedOrder model.AllergyOrder
	if err := db.First(&persistedOrder, order.ID).Error; err != nil {
		t.Fatalf("failed to reload order: %v", err)
	}
	if persistedOrder.OrderStatus != "completed" || persistedOrder.CompletedAt == nil {
		t.Fatalf("expected completed order, got %+v", persistedOrder)
	}

	var timeline []model.OrderTimelineEvent
	if err := db.Where("order_id = ?", order.ID).Order("id asc").Find(&timeline).Error; err != nil {
		t.Fatalf("failed to query timeline: %v", err)
	}
	expectedEvents := []string{"kit_preparing", "kit_shipped", "sample_sent_back", "sample_received", "in_testing", "report_uploaded", "report_published", "report_email_sent", "completed"}
	for _, eventType := range expectedEvents {
		found := false
		for _, item := range timeline {
			if item.EventType == eventType {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected timeline event %q, got %+v", eventType, timeline)
		}
	}
	if len(detail.Timeline) < len(expectedEvents) {
		t.Fatalf("expected admin detail timeline to include enriched events, got %+v", detail.Timeline)
	}
}

func timePtr(v time.Time) *time.Time {
	return &v
}
