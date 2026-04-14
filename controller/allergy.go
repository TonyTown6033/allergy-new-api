package controller

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

const (
	allergyHeroOptionKey         = "AllergyHero"
	allergyTestimonialsOptionKey = "AllergyTestimonials"
	allergyArticlesOptionKey     = "AllergyArticles"
	allergyProductsOptionKey     = "AllergyProducts"
	allergyDevLoginEmail         = "member@example.com"
	allergyDevLoginCode          = "123456"
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
}

type allergySendCodeRequest struct {
	Email string `json:"email"`
}

type allergyLoginRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

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

var defaultAllergyProducts = []allergyProductContent{
	{ID: "allergy-test-basic", Title: "埃勒吉 APP + 居家过敏原检测包", Description: "通过一滴指尖血，精准检测100+种过敏原。APP根据结果生成个性化回避建议与营养补充方案。", Image: "https://picsum.photos/600/400?random=5", CTAText: "立即订购套装", Tag: "最受欢迎"},
	{ID: "allergy-test-plus", Title: "专家定制食谱 + 营养补充剂", Description: "不仅仅是不吃什么，更重要的是吃什么。由营养学家设计的30天轮替食谱，帮助修复肠道屏障。", Image: "https://picsum.photos/500/500?random=7", CTAText: "定制我的食谱", Tag: "订阅制服务"},
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
	respondAllergyContent(c, allergyProductsOptionKey, defaultAllergyProducts)
}

type allergyUpdateProfileRequest struct {
	Nickname string `json:"nickname"`
	Phone    string `json:"phone"`
}

func SendAllergyLoginCode(c *gin.Context) {
	var req allergySendCodeRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorMsg(c, "无效的参数")
		return
	}
	email, err := validateAllergyEmail(req.Email)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	if err := model.AllergyMemberLoginAllowed(email); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	code, err := generateAllergyLoginCode()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if _, err := model.CreateEmailLoginCode(email, model.AllergyLoginCodePurpose, code, c.ClientIP(), 10*time.Minute); err != nil {
		common.ApiError(c, err)
		return
	}

	if strings.TrimSpace(common.SMTPServer) != "" || strings.TrimSpace(common.SMTPAccount) != "" {
		subject := fmt.Sprintf("%s 登录验证码", common.SystemName)
		content := fmt.Sprintf("<p>您好，您正在登录 %s。</p><p>验证码为：<strong>%s</strong></p><p>验证码 10 分钟内有效。</p>", common.SystemName, code)
		if err := common.SendEmail(subject, email, content); err != nil {
			common.ApiError(c, err)
			return
		}
	} else {
		common.SysLog(fmt.Sprintf("[allergy-auth] login code for %s: %s", common.MaskEmail(email), code))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "验证码已发送",
	})
}

func LoginAllergyMember(c *gin.Context) {
	var req allergyLoginRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorMsg(c, "无效的参数")
		return
	}
	email, err := validateAllergyEmail(req.Email)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	if strings.TrimSpace(req.Code) == "" {
		common.ApiErrorMsg(c, "验证码不能为空")
		return
	}
	if !allowAllergyDevLogin(c, email, req.Code) {
		if _, err := model.ConsumeEmailLoginCode(email, model.AllergyLoginCodePurpose, req.Code); err != nil {
			common.ApiErrorMsg(c, err.Error())
			return
		}
	} else {
		common.SysLog(fmt.Sprintf("[allergy-auth] using local development login shortcut for %s", common.MaskEmail(email)))
	}

	isNewUser := false
	if _, err := model.GetUserByEmail(email); err != nil {
		isNewUser = true
	}

	user, err := model.FindOrCreateAllergyMemberByEmail(email)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	token, _, err := model.CreateMemberSession(user.Id, model.AllergyMemberClientWeb, c.Request.UserAgent(), c.ClientIP(), model.AllergyMemberSessionTTL)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"token":       token,
		"email":       user.Email,
		"is_new_user": isNewUser,
		"user": gin.H{
			"id":    user.Id,
			"email": user.Email,
		},
	})
}

func allowAllergyDevLogin(c *gin.Context, email string, code string) bool {
	if model.NormalizeEmail(email) != allergyDevLoginEmail || strings.TrimSpace(code) != allergyDevLoginCode {
		return false
	}
	if common.DebugEnabled {
		return true
	}
	host := c.Request.Host
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	}
	switch strings.Trim(host, "[]") {
	case "localhost", "127.0.0.1", "::1":
		return true
	default:
		return false
	}
}

func UpdateAllergyProfile(c *gin.Context) {
	var req allergyUpdateProfileRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	userID := c.GetInt("id")
	if err := model.UpdateMemberProfile(userID, req.Nickname, req.Phone); err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "资料已更新"})
}

func GetAllergyAuthMe(c *gin.Context) {
	userValue, exists := c.Get("allergy_member_user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "登录状态无效或已过期",
		})
		return
	}
	user := userValue.(*model.User)

	profile, err := model.GetMemberProfileByUserID(user.Id)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	nickname := ""
	phone := ""
	if profile != nil {
		nickname = profile.Nickname
		phone = profile.Phone
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"email":   user.Email,
		"user": gin.H{
			"id":       user.Id,
			"email":    user.Email,
			"nickname": nickname,
			"phone":    phone,
		},
	})
}

func LogoutAllergyMember(c *gin.Context) {
	sessionValue, exists := c.Get("allergy_member_session")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "登录状态无效或已过期",
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
		"message": "",
	})
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
