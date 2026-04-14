package middleware

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func AllergyMemberAuth() func(c *gin.Context) {
	return func(c *gin.Context) {
		authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "未提供登录凭证",
			})
			c.Abort()
			return
		}

		session, user, err := model.AuthenticateMemberSession(authHeader)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": err.Error(),
			})
			c.Abort()
			return
		}

		common.SetContextKey(c, constant.ContextKeyUserId, user.Id)
		user.ToBaseUser().WriteContext(c)
		c.Set("allergy_member_session", session)
		c.Set("allergy_member_user", user)
		c.Next()
	}
}
