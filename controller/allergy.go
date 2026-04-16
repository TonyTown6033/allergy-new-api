package controller

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	allergyHeroOptionKey         = "AllergyHero"
	allergyTestimonialsOptionKey = "AllergyTestimonials"
	allergyArticlesOptionKey     = "AllergyArticles"
)

type allergyHeroContent struct {
	Image string `json:"image"`
}

type allergyTestimonialContent struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Role  string `json:"role"`
	Quote string `json:"quote"`
	Image string `json:"image"`
}

type allergyArticleContent struct {
	ID       string `json:"id"`
	Category string `json:"category"`
	Title    string `json:"title"`
	Summary  string `json:"summary"`
	Image    string `json:"image"`
}

type allergyProductContent struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Image       string `json:"image"`
	CTAText     string `json:"ctaText"`
	Tag         string `json:"tag"`
	PriceCents  int    `json:"price_cents,omitempty"`
	Currency    string `json:"currency,omitempty"`
}

type allergySendCodeRequest struct {
	Email string `json:"email"`
}

type allergyRegisterRequest struct {
	Email           string `json:"email"`
	Code            string `json:"code"`
	Username        string `json:"username"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirmPassword"`
}

type allergyLoginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type allergyPasswordResetRequest struct {
	Email           string `json:"email"`
	Code            string `json:"code"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirmPassword"`
}

const (
	allergyErrorInvalidRequest      = "INVALID_REQUEST"
	allergyErrorInvalidEmail        = "INVALID_EMAIL"
	allergyErrorEmailAlreadyExists  = "EMAIL_ALREADY_EXISTS"
	allergyErrorUsernameExists      = "USERNAME_ALREADY_EXISTS"
	allergyErrorCodeInvalid         = "CODE_INVALID"
	allergyErrorCodeExpired         = "CODE_EXPIRED"
	allergyErrorAccountNotFound     = "ACCOUNT_NOT_FOUND"
	allergyErrorPasswordIncorrect   = "PASSWORD_INCORRECT"
	allergyErrorAccountDisabled     = "ACCOUNT_DISABLED"
	allergyErrorEmailNotVerified    = "EMAIL_NOT_VERIFIED"
	allergyErrorMemberProfileNeeded = "MEMBER_PROFILE_REQUIRED"
	allergyErrorUnauthorized        = "UNAUTHORIZED"
	allergyErrorServer              = "SERVER_ERROR"
)

var defaultAllergyHero = allergyHeroContent{Image: "/images/hero1.png"}

var defaultAllergyTestimonials = []allergyTestimonialContent{
	{ID: "1", Name: "李妈妈", Role: "3岁过敏儿妈妈", Quote: "\"终于睡了个整觉。使用埃勒吉食谱两周后，宝宝的湿疹明显消退了。\"", Image: "/images/family1.png"},
	{ID: "2", Name: "张女士", Role: "二胎宝妈", Quote: "\"不用再盲目忌口了。测试报告非常精准，现在我们知道该给孩子吃什么。\"", Image: "/images/family2.png"},
	{ID: "3", Name: "王医生", Role: "儿科专家推荐", Quote: "\"埃勒吉将检测与日常管理结合的模式，是目前最科学的过敏管理方案。\"", Image: "/images/hero2.png"},
}

var defaultAllergyArticles = []allergyArticleContent{
	{ID: "1", Category: "过敏科普", Title: "为什么现在的孩子过敏越来越多？", Summary: "环境卫生假说与现代饮食结构的改变，如何影响了这一代儿童的免疫系统。", Image: "https://picsum.photos/400/300?random=10"},
	{ID: "2", Category: "最新研究", Title: "益生菌对特应性皮炎真的有效吗？", Summary: "埃勒吉首席科学家解析：不同菌株对皮肤屏障修复的临床数据对比。", Image: "https://picsum.photos/400/300?random=11"},
	{ID: "3", Category: "专家访谈", Title: "关于燕麦与麸质的真相", Summary: "燕麦到底含不含麸质？乳糜泻儿童如何安全食用谷物？", Image: "https://picsum.photos/400/300?random=12"},
}

func GetAllergyHero(c *gin.Context) {
	respondAllergyContent(c, allergyHeroOptionKey, defaultAllergyHero)
}

func GetAllergyTestimonials(c *gin.Context) {
	respondAllergyContent(c, allergyTestimonialsOptionKey, defaultAllergyTestimonials)
}

func GetAllergyArticles(c *gin.Context) {
	respondAllergyContent(c, allergyArticlesOptionKey, defaultAllergyArticles)
}

func GetAllergyProducts(c *gin.Context) {
	products, err := model.ListPublishedAllergyServiceProducts()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	response := make([]allergyProductContent, 0, len(products))
	for _, product := range products {
		response = append(response, allergyProductContent{
			ID:          product.ServiceCode,
			Title:       product.Title,
			Description: product.Description,
			Image:       product.ImageURL,
			CTAText:     product.CTAText,
			Tag:         product.Tag,
			PriceCents:  product.PriceCents,
			Currency:    product.Currency,
		})
	}
	c.JSON(http.StatusOK, response)
}

type allergyUpdateProfileRequest struct {
	Nickname string `json:"nickname"`
	Phone    string `json:"phone"`
}

func SendAllergyRegisterCode(c *gin.Context) {
	var req allergySendCodeRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		respondAllergyAuthError(c, "无效的参数", allergyErrorInvalidRequest)
		return
	}
	email, err := validateAllergyEmail(req.Email)
	if err != nil {
		respondAllergyAuthError(c, err.Error(), allergyErrorInvalidEmail)
		return
	}
	exists, err := model.UserExistsByEmail(email)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if exists {
		respondAllergyAuthError(c, "邮箱已注册", allergyErrorEmailAlreadyExists)
		return
	}

	code, err := generateAllergyLoginCode()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if _, err := model.CreateEmailLoginCode(email, model.AllergyRegisterVerifyCodePurpose, code, c.ClientIP(), 5*time.Minute); err != nil {
		common.ApiError(c, err)
		return
	}

	if strings.TrimSpace(common.SMTPServer) != "" || strings.TrimSpace(common.SMTPAccount) != "" {
		subject := fmt.Sprintf("%s 注册验证码", common.SystemName)
		content := fmt.Sprintf("<p>您好，您正在注册 %s。</p><p>验证码为：<strong>%s</strong></p><p>验证码 5 分钟内有效。</p>", common.SystemName, code)
		if err := common.SendEmail(subject, email, content); err != nil {
			common.ApiError(c, err)
			return
		}
	} else {
		common.SysLog(fmt.Sprintf("[allergy-auth] register code for %s: %s", common.MaskEmail(email), code))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "验证码已发送",
		"data": gin.H{
			"email":            email,
			"purpose":          model.AllergyRegisterVerifyCodePurpose,
			"expiresInSeconds": 300,
		},
	})
}

func RegisterAllergyMember(c *gin.Context) {
	var req allergyRegisterRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		respondAllergyAuthError(c, "无效的参数", allergyErrorInvalidRequest)
		return
	}
	email, err := validateAllergyEmail(req.Email)
	if err != nil {
		respondAllergyAuthError(c, err.Error(), allergyErrorInvalidEmail)
		return
	}
	username, err := validateAllergyUsername(req.Username)
	if err != nil {
		respondAllergyAuthError(c, err.Error(), allergyErrorInvalidRequest)
		return
	}
	password, err := validateAllergyPasswordPair(req.Password, req.ConfirmPassword)
	if err != nil {
		respondAllergyAuthError(c, err.Error(), allergyErrorInvalidRequest)
		return
	}
	if strings.TrimSpace(req.Code) == "" {
		respondAllergyAuthError(c, "验证码不能为空", allergyErrorInvalidRequest)
		return
	}

	emailExists, err := model.UserExistsByEmail(email)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if emailExists {
		respondAllergyAuthError(c, "邮箱已注册", allergyErrorEmailAlreadyExists)
		return
	}
	usernameExists, err := model.UserExistsByUsername(username)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if usernameExists {
		respondAllergyAuthError(c, "用户名已存在", allergyErrorUsernameExists)
		return
	}
	if _, err := model.ConsumeEmailLoginCode(email, model.AllergyRegisterVerifyCodePurpose, req.Code); err != nil {
		respondAllergyMappedError(c, err)
		return
	}
	verifiedAt := time.Now()
	user, profile, err := model.CreateAllergyMemberAccount(username, email, password, verifiedAt)
	if err != nil {
		if duplicateMessage, duplicateCode := mapAllergyDuplicateError(err); duplicateCode != "" {
			respondAllergyAuthError(c, duplicateMessage, duplicateCode)
			return
		}
		common.ApiError(c, err)
		return
	}
	token, _, err := model.CreateMemberSession(user.Id, model.AllergyMemberClientWeb, c.Request.UserAgent(), c.ClientIP(), model.AllergyMemberSessionTTL)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	respondAllergyAuthSuccess(c, "注册成功", token, user, profile)
}

func LoginAllergyMember(c *gin.Context) {
	var req allergyLoginRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		respondAllergyAuthError(c, "无效的参数", allergyErrorInvalidRequest)
		return
	}
	identifier := strings.TrimSpace(req.Identifier)
	password := strings.TrimSpace(req.Password)
	if identifier == "" || password == "" {
		respondAllergyAuthError(c, "用户名/邮箱和密码不能为空", allergyErrorInvalidRequest)
		return
	}

	user, err := model.GetUserByIdentifier(identifier)
	if err != nil {
		respondAllergyMappedError(c, err)
		return
	}
	if !common.ValidatePasswordAndHash(password, user.Password) {
		respondAllergyAuthError(c, "用户名、邮箱或密码错误", allergyErrorPasswordIncorrect)
		return
	}
	profile, err := model.ValidateAllergyMemberUser(user)
	if err != nil {
		respondAllergyMappedError(c, err)
		return
	}
	token, _, err := model.CreateMemberSession(user.Id, model.AllergyMemberClientWeb, c.Request.UserAgent(), c.ClientIP(), model.AllergyMemberSessionTTL)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	respondAllergyAuthSuccess(c, "登录成功", token, user, profile)
}

func SendAllergyPasswordResetCode(c *gin.Context) {
	var req allergySendCodeRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		respondAllergyAuthError(c, "无效的参数", allergyErrorInvalidRequest)
		return
	}
	email, err := validateAllergyEmail(req.Email)
	if err != nil {
		respondAllergyAuthError(c, err.Error(), allergyErrorInvalidEmail)
		return
	}
	user, err := model.GetUserByEmail(email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondAllergyAuthError(c, "账号不存在", allergyErrorAccountNotFound)
			return
		}
		common.ApiError(c, err)
		return
	}
	if _, err := model.ValidateAllergyMemberUser(user); err != nil {
		respondAllergyMappedError(c, err)
		return
	}
	code, err := generateAllergyLoginCode()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if _, err := model.CreateEmailLoginCode(email, model.AllergyPasswordResetCodePurpose, code, c.ClientIP(), 5*time.Minute); err != nil {
		common.ApiError(c, err)
		return
	}
	if strings.TrimSpace(common.SMTPServer) != "" || strings.TrimSpace(common.SMTPAccount) != "" {
		subject := fmt.Sprintf("%s 找回密码验证码", common.SystemName)
		content := fmt.Sprintf("<p>您好，您正在重置 %s 的登录密码。</p><p>验证码为：<strong>%s</strong></p><p>验证码 5 分钟内有效。</p>", common.SystemName, code)
		if err := common.SendEmail(subject, email, content); err != nil {
			common.ApiError(c, err)
			return
		}
	} else {
		common.SysLog(fmt.Sprintf("[allergy-auth] password reset code for %s: %s", common.MaskEmail(email), code))
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "验证码已发送",
		"data": gin.H{
			"email":            email,
			"purpose":          model.AllergyPasswordResetCodePurpose,
			"expiresInSeconds": 300,
		},
	})
}

func ResetAllergyMemberPassword(c *gin.Context) {
	var req allergyPasswordResetRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		respondAllergyAuthError(c, "无效的参数", allergyErrorInvalidRequest)
		return
	}
	email, err := validateAllergyEmail(req.Email)
	if err != nil {
		respondAllergyAuthError(c, err.Error(), allergyErrorInvalidEmail)
		return
	}
	password, err := validateAllergyPasswordPair(req.Password, req.ConfirmPassword)
	if err != nil {
		respondAllergyAuthError(c, err.Error(), allergyErrorInvalidRequest)
		return
	}
	if strings.TrimSpace(req.Code) == "" {
		respondAllergyAuthError(c, "验证码不能为空", allergyErrorInvalidRequest)
		return
	}
	user, err := model.GetUserByEmail(email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondAllergyAuthError(c, "账号不存在", allergyErrorAccountNotFound)
			return
		}
		common.ApiError(c, err)
		return
	}
	if _, err := model.ValidateAllergyMemberUser(user); err != nil {
		respondAllergyMappedError(c, err)
		return
	}
	if _, err := model.ConsumeEmailLoginCode(email, model.AllergyPasswordResetCodePurpose, req.Code); err != nil {
		respondAllergyMappedError(c, err)
		return
	}
	if err := model.ResetUserPasswordByEmail(email, password); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.RevokeMemberSessionsByUserID(user.Id); err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "密码已重置",
		"data": gin.H{
			"reset":           true,
			"sessionsRevoked": true,
		},
	})
}

func UpdateAllergyProfile(c *gin.Context) {
	var req allergyUpdateProfileRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		respondAllergyAuthError(c, "参数错误", allergyErrorInvalidRequest)
		return
	}
	userID := c.GetInt("id")
	if err := model.UpdateMemberProfile(userID, req.Nickname, req.Phone); err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "资料已更新",
		"data": gin.H{
			"updated": true,
		},
	})
}

func GetAllergyAuthMe(c *gin.Context) {
	userValue, exists := c.Get("allergy_member_user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "登录状态无效或已过期",
			"code":    allergyErrorUnauthorized,
		})
		return
	}
	user := userValue.(*model.User)

	profile, err := model.ValidateAllergyMemberUser(user)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "登录状态无效或已过期",
			"code":    allergyErrorUnauthorized,
		})
		return
	}
	common.ApiSuccess(c, buildAllergyMemberPayload(user, profile))
}

func LogoutAllergyMember(c *gin.Context) {
	sessionValue, exists := c.Get("allergy_member_session")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "登录状态无效或已过期",
			"code":    allergyErrorUnauthorized,
		})
		return
	}
	session := sessionValue.(*model.MemberSession)
	if err := model.RevokeMemberSession(session.ID); err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "已退出登录",
		"data": gin.H{
			"revoked": true,
		},
	})
}

func respondAllergyAuthError(c *gin.Context, message string, code string) {
	c.JSON(http.StatusOK, gin.H{
		"success": false,
		"message": message,
		"code":    code,
	})
}

func respondAllergyMappedError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, model.ErrAllergyCodeInvalid):
		respondAllergyAuthError(c, err.Error(), allergyErrorCodeInvalid)
	case errors.Is(err, model.ErrAllergyCodeExpired):
		respondAllergyAuthError(c, err.Error(), allergyErrorCodeExpired)
	case errors.Is(err, model.ErrAllergyAccountNotFound):
		respondAllergyAuthError(c, "账号不存在", allergyErrorAccountNotFound)
	case errors.Is(err, model.ErrAllergyPasswordIncorrect):
		respondAllergyAuthError(c, err.Error(), allergyErrorPasswordIncorrect)
	case errors.Is(err, model.ErrAllergyMemberDisabled):
		respondAllergyAuthError(c, err.Error(), allergyErrorAccountDisabled)
	case errors.Is(err, model.ErrAllergyEmailNotVerified):
		respondAllergyAuthError(c, err.Error(), allergyErrorEmailNotVerified)
	case errors.Is(err, model.ErrAllergyMemberProfileRequired), errors.Is(err, model.ErrAllergyAdminAccountNotAllowed):
		respondAllergyAuthError(c, err.Error(), allergyErrorMemberProfileNeeded)
	case errors.Is(err, model.ErrAllergyUnauthorized):
		respondAllergyAuthError(c, err.Error(), allergyErrorUnauthorized)
	default:
		common.ApiError(c, err)
	}
}

func respondAllergyAuthSuccess(c *gin.Context, message string, token string, user *model.User, profile *model.MemberProfile) {
	payload := buildAllergyMemberPayload(user, profile)
	data := gin.H{
		"token":   token,
		"user":    payload["user"],
		"profile": payload["profile"],
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": message,
		"token":   token,
		"data":    data,
	})
}

func buildAllergyMemberPayload(user *model.User, profile *model.MemberProfile) gin.H {
	emailVerifiedAt := ""
	createdAt := ""
	profileID := int64(0)
	nickname := ""
	phone := ""
	status := ""
	emailVerified := false
	if profile != nil {
		profileID = profile.ID
		nickname = profile.Nickname
		phone = profile.Phone
		status = profile.Status
		emailVerified = profile.EmailVerifiedAt != nil
		if profile.EmailVerifiedAt != nil {
			emailVerifiedAt = profile.EmailVerifiedAt.Format(time.RFC3339)
		}
		if !profile.CreatedAt.IsZero() {
			createdAt = profile.CreatedAt.Format(time.RFC3339)
		}
	}
	return gin.H{
		"user": gin.H{
			"id":       user.Id,
			"username": user.Username,
			"email":    user.Email,
		},
		"profile": gin.H{
			"id":              profileID,
			"username":        user.Username,
			"email":           user.Email,
			"nickname":        nickname,
			"phone":           phone,
			"status":          status,
			"emailVerified":   emailVerified,
			"emailVerifiedAt": emailVerifiedAt,
			"createdAt":       createdAt,
		},
	}
}

func validateAllergyUsername(username string) (string, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return "", fmt.Errorf("用户名不能为空")
	}
	if len([]rune(username)) > model.UserNameMaxLength {
		return "", fmt.Errorf("用户名不能超过 %d 个字符", model.UserNameMaxLength)
	}
	return username, nil
}

func validateAllergyPasswordPair(password string, confirmPassword string) (string, error) {
	password = strings.TrimSpace(password)
	confirmPassword = strings.TrimSpace(confirmPassword)
	if password == "" || confirmPassword == "" {
		return "", fmt.Errorf("密码和确认密码不能为空")
	}
	if password != confirmPassword {
		return "", fmt.Errorf("两次输入的密码不一致")
	}
	if len([]rune(password)) < 8 {
		return "", fmt.Errorf("密码至少需要 8 位")
	}
	return password, nil
}

func mapAllergyDuplicateError(err error) (string, string) {
	if err == nil {
		return "", ""
	}
	lower := strings.ToLower(err.Error())
	switch {
	case strings.Contains(lower, "username"):
		return "用户名已存在", allergyErrorUsernameExists
	case strings.Contains(lower, "email"):
		return "邮箱已注册", allergyErrorEmailAlreadyExists
	default:
		return "", ""
	}
}

func respondAllergyContent[T any](c *gin.Context, optionKey string, fallback T) {
	common.OptionMapRWMutex.RLock()
	raw := strings.TrimSpace(common.OptionMap[optionKey])
	common.OptionMapRWMutex.RUnlock()
	if raw == "" {
		c.JSON(http.StatusOK, fallback)
		return
	}

	var payload T
	if err := common.UnmarshalJsonStr(raw, &payload); err != nil {
		common.ApiError(c, fmt.Errorf("解析 %s 配置失败: %w", optionKey, err))
		return
	}
	c.JSON(http.StatusOK, payload)
}

func validateAllergyEmail(email string) (string, error) {
	email = model.NormalizeEmail(email)
	if err := common.Validate.Var(email, "required,email"); err != nil {
		return "", fmt.Errorf("请输入正确的邮箱地址")
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "", fmt.Errorf("请输入正确的邮箱地址")
	}
	localPart := parts[0]
	domainPart := parts[1]

	if common.EmailDomainRestrictionEnabled {
		allowed := false
		for _, domain := range common.EmailDomainWhitelist {
			if domainPart == domain {
				allowed = true
				break
			}
		}
		if !allowed {
			return "", fmt.Errorf("该邮箱域名不被允许")
		}
	}
	if common.EmailAliasRestrictionEnabled && (strings.Contains(localPart, "+") || strings.Contains(localPart, ".")) {
		return "", fmt.Errorf("该邮箱地址不被允许")
	}
	return email, nil
}

func generateAllergyLoginCode() (string, error) {
	var builder strings.Builder
	builder.Grow(6)
	max := big.NewInt(10)
	for i := 0; i < 6; i++ {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		builder.WriteByte(byte('0' + n.Int64()))
	}
	return builder.String(), nil
}
