package model

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	AllergyRegisterVerifyCodePurpose = "register_verify"
	AllergyPasswordResetCodePurpose  = "password_reset"
	AllergyMemberClientWeb           = "web"
	AllergyMemberSessionTTL          = 7 * 24 * time.Hour
	memberProfileStatusActive        = "active"
	memberProfileStatusDisabled      = "disabled"
)

var (
	ErrAllergyCodeInvalid            = errors.New("验证码错误")
	ErrAllergyCodeExpired            = errors.New("验证码无效或已过期，请重新获取")
	ErrAllergyAccountNotFound        = errors.New("账号不存在")
	ErrAllergyPasswordIncorrect      = errors.New("用户名、邮箱或密码错误")
	ErrAllergyMemberDisabled         = errors.New("会员账号已禁用")
	ErrAllergyMemberProfileRequired  = errors.New("当前账号不是有效会员")
	ErrAllergyEmailNotVerified       = errors.New("邮箱未验证")
	ErrAllergyAdminAccountNotAllowed = errors.New("公共会员登录入口不可用于管理员账号")
	ErrAllergyUnauthorized           = errors.New("登录状态无效或已过期")
)

type MemberProfile struct {
	ID                    int64      `json:"id"`
	UserID                int        `json:"user_id" gorm:"uniqueIndex"`
	Phone                 string     `json:"phone" gorm:"type:varchar(32);default:''"`
	Nickname              string     `json:"nickname" gorm:"type:varchar(64);default:''"`
	AvatarURL             string     `json:"avatar_url" gorm:"type:varchar(512);default:''"`
	RealName              string     `json:"real_name" gorm:"type:varchar(64);default:''"`
	DefaultRecipientName  string     `json:"default_recipient_name" gorm:"type:varchar(64);default:''"`
	DefaultRecipientPhone string     `json:"default_recipient_phone" gorm:"type:varchar(32);default:''"`
	DefaultAddressJSON    string     `json:"default_address_json" gorm:"type:text"`
	Status                string     `json:"status" gorm:"type:varchar(32);default:'active'"`
	EmailVerifiedAt       *time.Time `json:"email_verified_at" gorm:"index"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

func (MemberProfile) TableName() string {
	return "member_profile"
}

type EmailLoginCodeStore struct {
	ID           int64      `json:"id"`
	Email        string     `json:"email" gorm:"type:varchar(255);index:idx_email_login_code_lookup"`
	Purpose      string     `json:"purpose" gorm:"type:varchar(32);index:idx_email_login_code_lookup"`
	CodeHash     string     `json:"code_hash" gorm:"type:varchar(255);not null"`
	ExpiresAt    time.Time  `json:"expires_at" gorm:"index"`
	UsedAt       *time.Time `json:"used_at" gorm:"index"`
	SendIP       string     `json:"send_ip" gorm:"type:varchar(64);default:''"`
	AttemptCount int        `json:"attempt_count" gorm:"default:0"`
	CreatedAt    time.Time  `json:"created_at"`
}

func (EmailLoginCodeStore) TableName() string {
	return "email_login_code_store"
}

type MemberSession struct {
	ID         int64      `json:"id"`
	UserID     int        `json:"user_id" gorm:"index"`
	TokenHash  string     `json:"token_hash" gorm:"type:char(64);uniqueIndex"`
	ClientType string     `json:"client_type" gorm:"type:varchar(32);default:'web'"`
	UserAgent  string     `json:"user_agent" gorm:"type:varchar(1024);default:''"`
	LoginIP    string     `json:"login_ip" gorm:"type:varchar(64);default:''"`
	ExpiresAt  time.Time  `json:"expires_at" gorm:"index"`
	LastSeenAt *time.Time `json:"last_seen_at"`
	RevokedAt  *time.Time `json:"revoked_at" gorm:"index"`
	CreatedAt  time.Time  `json:"created_at"`
}

func (MemberSession) TableName() string {
	return "member_session"
}

type AllergyOrder struct {
	ID                         int64      `json:"id"`
	OrderNo                    string     `json:"order_no" gorm:"type:varchar(64);uniqueIndex"`
	UserID                     int        `json:"user_id" gorm:"index"`
	ServiceCode                string     `json:"service_code" gorm:"type:varchar(64)"`
	ServiceNameSnapshot        string     `json:"service_name_snapshot" gorm:"type:varchar(255)"`
	ServicePriceCents          int        `json:"service_price_cents"`
	Currency                   string     `json:"currency" gorm:"type:varchar(16);default:'CNY'"`
	PaymentStatus              string     `json:"payment_status" gorm:"type:varchar(32);index"`
	OrderStatus                string     `json:"order_status" gorm:"type:varchar(32);index"`
	RecipientName              string     `json:"recipient_name" gorm:"type:varchar(64)"`
	RecipientPhone             string     `json:"recipient_phone" gorm:"type:varchar(32)"`
	RecipientEmail             string     `json:"recipient_email" gorm:"type:varchar(255)"`
	ShippingAddressJSON        string     `json:"shipping_address_json" gorm:"type:text"`
	PaymentMethod              string     `json:"payment_method" gorm:"type:varchar(32);default:''"`
	PaymentRef                 string     `json:"payment_ref" gorm:"type:varchar(128);default:''"`
	PaymentProviderOrderNo     string     `json:"payment_provider_order_no" gorm:"type:varchar(128);default:''"`
	PaymentCallbackPayloadJSON string     `json:"payment_callback_payload_json" gorm:"type:text"`
	PaidAt                     *time.Time `json:"paid_at"`
	ReportReadyAt              *time.Time `json:"report_ready_at"`
	CompletedAt                *time.Time `json:"completed_at"`
	CancelledAt                *time.Time `json:"cancelled_at"`
	AdminRemark                string     `json:"admin_remark" gorm:"type:text"`
	CreatedAt                  time.Time  `json:"created_at"`
	UpdatedAt                  time.Time  `json:"updated_at"`
}

func (AllergyOrder) TableName() string {
	return "allergy_order"
}

type SampleKit struct {
	ID               int64      `json:"id"`
	OrderID          int64      `json:"order_id" gorm:"uniqueIndex"`
	KitCode          string     `json:"kit_code" gorm:"type:varchar(64);uniqueIndex"`
	Status           string     `json:"status" gorm:"type:varchar(32);index"`
	TrackingCompany  string     `json:"tracking_company" gorm:"type:varchar(64);default:''"`
	TrackingNumber   string     `json:"tracking_number" gorm:"type:varchar(128);default:''"`
	ReturnTrackingNo string     `json:"return_tracking_no" gorm:"type:varchar(128);default:''"`
	ShippedAt        *time.Time `json:"shipped_at"`
	DeliveredAt      *time.Time `json:"delivered_at"`
	SampleReceivedAt *time.Time `json:"sample_received_at"`
	Remark           string     `json:"remark" gorm:"type:text"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func (SampleKit) TableName() string {
	return "sample_kit"
}

type LabSubmission struct {
	ID                 int64      `json:"id"`
	OrderID            int64      `json:"order_id" gorm:"uniqueIndex"`
	SampleKitID        int64      `json:"sample_kit_id"`
	LabName            string     `json:"lab_name" gorm:"type:varchar(128);default:''"`
	SubmissionNo       string     `json:"submission_no" gorm:"type:varchar(128);default:''"`
	Status             string     `json:"status" gorm:"type:varchar(32);index"`
	ExternalSampleCode string     `json:"external_sample_code" gorm:"type:varchar(128);default:''"`
	TrackingNumber     string     `json:"tracking_number" gorm:"type:varchar(128);default:''"`
	ReceivedAt         *time.Time `json:"received_at"`
	SubmittedAt        *time.Time `json:"submitted_at"`
	TestingStartedAt   *time.Time `json:"testing_started_at"`
	CompletedAt        *time.Time `json:"completed_at"`
	RawPayloadJSON     string     `json:"raw_payload_json" gorm:"type:text"`
	Remark             string     `json:"remark" gorm:"type:text"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

func (LabSubmission) TableName() string {
	return "lab_submission"
}

type LabReport struct {
	ID                    int64      `json:"id"`
	OrderID               int64      `json:"order_id" gorm:"index"`
	SampleKitID           int64      `json:"sample_kit_id"`
	LabSubmissionID       int64      `json:"lab_submission_id"`
	Version               int        `json:"version" gorm:"default:1"`
	Status                string     `json:"status" gorm:"type:varchar(32);index"`
	IsCurrent             bool       `json:"is_current" gorm:"index"`
	ReportNo              string     `json:"report_no" gorm:"type:varchar(128);default:''"`
	ReportTitle           string     `json:"report_title" gorm:"type:varchar(255);default:''"`
	PDFStorageType        string     `json:"pdf_storage_type" gorm:"type:varchar(32);default:'local'"`
	FileName              string     `json:"file_name" gorm:"type:varchar(255);default:''"`
	FilePath              string     `json:"file_path" gorm:"type:varchar(1024);default:''"`
	FileURL               string     `json:"file_url" gorm:"type:varchar(1024);default:''"`
	MimeType              string     `json:"mime_type" gorm:"type:varchar(128);default:'application/pdf'"`
	FileSizeBytes         int64      `json:"file_size_bytes"`
	GeneratedAt           *time.Time `json:"generated_at"`
	UploadedAt            *time.Time `json:"uploaded_at"`
	PublishedAt           *time.Time `json:"published_at"`
	LastEmailSentAt       *time.Time `json:"last_email_sent_at"`
	EmailSentCount        int        `json:"email_sent_count" gorm:"default:0"`
	LastSentByAdminUserID int        `json:"last_sent_by_admin_user_id"`
	SummaryJSON           string     `json:"summary_json" gorm:"type:text"`
	Remark                string     `json:"remark" gorm:"type:text"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

func (LabReport) TableName() string {
	return "lab_report"
}

type ReportDeliveryLog struct {
	ID                int64      `json:"id"`
	ReportID          int64      `json:"report_id" gorm:"index"`
	OrderID           int64      `json:"order_id" gorm:"index"`
	RecipientEmail    string     `json:"recipient_email" gorm:"type:varchar(255)"`
	DeliveryChannel   string     `json:"delivery_channel" gorm:"type:varchar(32);default:'email'"`
	DeliveryType      string     `json:"delivery_type" gorm:"type:varchar(32)"`
	DeliveryStatus    string     `json:"delivery_status" gorm:"type:varchar(32);index"`
	TriggeredByUserID int        `json:"triggered_by_user_id"`
	MessageID         string     `json:"message_id" gorm:"type:varchar(255);default:''"`
	ErrorMessage      string     `json:"error_message" gorm:"type:text"`
	SentAt            *time.Time `json:"sent_at"`
	CreatedAt         time.Time  `json:"created_at"`
}

func (ReportDeliveryLog) TableName() string {
	return "report_delivery_log"
}

type OrderTimelineEvent struct {
	ID               int64     `json:"id"`
	OrderID          int64     `json:"order_id" gorm:"index"`
	EventType        string    `json:"event_type" gorm:"type:varchar(64);index"`
	EventTitle       string    `json:"event_title" gorm:"type:varchar(255);default:''"`
	EventDesc        string    `json:"event_desc" gorm:"type:text"`
	VisibleToUser    bool      `json:"visible_to_user" gorm:"default:true"`
	OperatorUserID   int       `json:"operator_user_id"`
	EventPayloadJSON string    `json:"event_payload_json" gorm:"type:text"`
	OccurredAt       time.Time `json:"occurred_at" gorm:"index"`
	CreatedAt        time.Time `json:"created_at"`
}

func (OrderTimelineEvent) TableName() string {
	return "order_timeline_event"
}

func UpdateMemberProfile(userID int, nickname string, phone string) error {
	updates := map[string]any{
		"nickname":   strings.TrimSpace(nickname),
		"phone":      strings.TrimSpace(phone),
		"updated_at": time.Now(),
	}
	return DB.Model(&MemberProfile{}).Where("user_id = ?", userID).Updates(updates).Error
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func GetUserByEmail(email string) (*User, error) {
	email = NormalizeEmail(email)
	if email == "" {
		return nil, errors.New("邮箱地址不能为空")
	}
	var user User
	err := DB.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func GetUserByIdentifier(identifier string) (*User, error) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return nil, ErrAllergyAccountNotFound
	}
	var user User
	err := DB.Where("username = ? OR email = ?", identifier, NormalizeEmail(identifier)).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrAllergyAccountNotFound
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func UserExistsByUsername(username string) (bool, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return false, nil
	}
	var count int64
	if err := DB.Unscoped().Model(&User{}).Where("username = ?", username).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func UserExistsByEmail(email string) (bool, error) {
	email = NormalizeEmail(email)
	if email == "" {
		return false, nil
	}
	var count int64
	if err := DB.Unscoped().Model(&User{}).Where("email = ?", email).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func GetMemberProfileByUserID(userID int) (*MemberProfile, error) {
	var profile MemberProfile
	err := DB.Where("user_id = ?", userID).First(&profile).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

func EnsureMemberProfile(userID int) (*MemberProfile, error) {
	profile, err := GetMemberProfileByUserID(userID)
	if err != nil {
		return nil, err
	}
	if profile != nil {
		return profile, nil
	}
	profile = &MemberProfile{
		UserID: userID,
		Status: memberProfileStatusActive,
	}
	if err := DB.Create(profile).Error; err != nil {
		return nil, err
	}
	return profile, nil
}

func ValidateAllergyMemberUser(user *User) (*MemberProfile, error) {
	if user == nil {
		return nil, ErrAllergyAccountNotFound
	}
	if user.Role >= common.RoleAdminUser {
		return nil, ErrAllergyAdminAccountNotAllowed
	}
	if user.Status != common.UserStatusEnabled {
		return nil, ErrAllergyMemberDisabled
	}
	profile, err := GetMemberProfileByUserID(user.Id)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, ErrAllergyMemberProfileRequired
	}
	if profile.Status != memberProfileStatusActive {
		return nil, ErrAllergyMemberDisabled
	}
	if profile.EmailVerifiedAt == nil {
		return nil, ErrAllergyEmailNotVerified
	}
	return profile, nil
}

func CreateAllergyMemberAccount(username string, email string, password string, verifiedAt time.Time) (*User, *MemberProfile, error) {
	username = strings.TrimSpace(username)
	email = NormalizeEmail(email)
	password = strings.TrimSpace(password)

	hashedPassword, err := common.Password2Hash(password)
	if err != nil {
		return nil, nil, err
	}

	displayName := username
	if displayName == "" {
		displayName = strings.TrimSpace(strings.Split(email, "@")[0])
	}
	if len(displayName) > 20 {
		displayName = displayName[:20]
	}
	user := &User{
		Username:    username,
		Password:    hashedPassword,
		DisplayName: displayName,
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Email:       email,
		Group:       "default",
		AffCode:     strings.ToUpper(common.GetRandomString(4)),
	}
	profile := &MemberProfile{
		Phone:           "",
		Nickname:        "",
		Status:          memberProfileStatusActive,
		EmailVerifiedAt: &verifiedAt,
	}

	if err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			return err
		}
		profile.UserID = user.Id
		if err := tx.Create(profile).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, nil, err
	}
	return user, profile, nil
}

func CreateEmailLoginCode(email string, purpose string, rawCode string, sendIP string, ttl time.Duration) (*EmailLoginCodeStore, error) {
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	codeHash, err := common.Password2Hash(rawCode)
	if err != nil {
		return nil, err
	}
	record := &EmailLoginCodeStore{
		Email:     NormalizeEmail(email),
		Purpose:   purpose,
		CodeHash:  codeHash,
		ExpiresAt: time.Now().Add(ttl),
		SendIP:    sendIP,
		CreatedAt: time.Now(),
	}
	if err := DB.Create(record).Error; err != nil {
		return nil, err
	}
	return record, nil
}

func ConsumeEmailLoginCode(email string, purpose string, rawCode string) (*EmailLoginCodeStore, error) {
	record := &EmailLoginCodeStore{}
	err := DB.Where("email = ? AND purpose = ?", NormalizeEmail(email), purpose).
		Order("created_at desc").
		First(record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrAllergyCodeExpired
	}
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if record.UsedAt != nil || now.After(record.ExpiresAt) {
		return nil, ErrAllergyCodeExpired
	}
	if !common.ValidatePasswordAndHash(strings.TrimSpace(rawCode), record.CodeHash) {
		record.AttemptCount++
		if updateErr := DB.Model(record).Update("attempt_count", record.AttemptCount).Error; updateErr != nil {
			return nil, updateErr
		}
		return nil, ErrAllergyCodeInvalid
	}
	record.UsedAt = &now
	if err := DB.Model(record).Updates(map[string]any{
		"used_at": record.UsedAt,
	}).Error; err != nil {
		return nil, err
	}
	return record, nil
}

func CreateMemberSession(userID int, clientType string, userAgent string, loginIP string, ttl time.Duration) (string, *MemberSession, error) {
	if clientType == "" {
		clientType = AllergyMemberClientWeb
	}
	if ttl <= 0 {
		ttl = AllergyMemberSessionTTL
	}
	rawToken := "ams_" + common.GetUUID()
	session := &MemberSession{
		UserID:     userID,
		TokenHash:  common.GenerateHMAC(rawToken),
		ClientType: clientType,
		UserAgent:  userAgent,
		LoginIP:    loginIP,
		ExpiresAt:  time.Now().Add(ttl),
		CreatedAt:  time.Now(),
	}
	if err := DB.Create(session).Error; err != nil {
		return "", nil, err
	}
	return rawToken, session, nil
}

func AuthenticateMemberSession(token string) (*MemberSession, *User, error) {
	token = strings.TrimSpace(token)
	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		token = strings.TrimSpace(token[7:])
	}
	if token == "" {
		return nil, nil, ErrAllergyUnauthorized
	}
	var session MemberSession
	err := DB.Where("token_hash = ?", common.GenerateHMAC(token)).First(&session).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, ErrAllergyUnauthorized
	}
	if err != nil {
		return nil, nil, err
	}
	now := time.Now()
	if session.RevokedAt != nil || now.After(session.ExpiresAt) {
		return nil, nil, ErrAllergyUnauthorized
	}
	user, err := GetUserById(session.UserID, false)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrAllergyUnauthorized
		}
		return nil, nil, err
	}
	if _, err := ValidateAllergyMemberUser(user); err != nil {
		return nil, nil, ErrAllergyUnauthorized
	}
	if touchErr := DB.Model(&session).Update("last_seen_at", now).Error; touchErr == nil {
		session.LastSeenAt = &now
	}
	return &session, user, nil
}

func RevokeMemberSession(sessionID int64) error {
	now := time.Now()
	return DB.Model(&MemberSession{}).
		Where("id = ? AND revoked_at IS NULL", sessionID).
		Update("revoked_at", &now).Error
}

func RevokeMemberSessionsByUserID(userID int) error {
	now := time.Now()
	return DB.Model(&MemberSession{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", &now).Error
}

func (session *MemberSession) DebugString() string {
	return fmt.Sprintf("member_session<id=%d user_id=%d expires_at=%s>", session.ID, session.UserID, session.ExpiresAt.Format(time.RFC3339))
}
