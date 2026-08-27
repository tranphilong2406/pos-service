package middleware

import (
	"StoreServer/utils/jwt"
	"StoreServer/utils/response"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func CheckLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			authHeader = c.GetHeader("Token")
		}

		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.Response{
				Code:    http.StatusUnauthorized,
				Message: "Unauthorized",
			})
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")

		user, err := jwt.ParseToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.Response{
				Code:    http.StatusUnauthorized,
				Message: "Unauthorized",
			})
			return
		}

		c.Set("user", user)
		c.Set("role", user.Role)
		c.Next()
	}
}

func CheckRole(allowedRoles ...string) gin.HandlerFunc {
	roleMap := make(map[string]struct{}, len(allowedRoles))
	for _, role := range allowedRoles {
		roleMap[role] = struct{}{}
	}

	return func(c *gin.Context) {
		roleValue, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.Response{
				Code:    http.StatusUnauthorized,
				Message: "Unauthorized",
			})
			return
		}

		userRole, ok := roleValue.(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, response.Response{
				Code:    http.StatusInternalServerError,
				Message: "Role parsing failed",
			})
			return
		}

		if _, allowed := roleMap[userRole]; allowed {
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusForbidden, response.Response{
			Code:    http.StatusForbidden,
			Message: "Permission denied",
		})
	}
}

