package middleware

import (
	"StoreServer/models"
	"StoreServer/utils/jwt"
	"StoreServer/utils/response"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// userLookup returns the user with the given ID unless it does not exist or was deleted.
type userLookup func(id bson.ObjectID) (user *models.User, found bool, err error)

func CheckLogin() gin.HandlerFunc {
	return checkLogin(findActiveUser)
}

func checkLogin(lookup userLookup) gin.HandlerFunc {
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

		claims, err := jwt.ParseToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.Response{
				Code:    http.StatusUnauthorized,
				Message: "Unauthorized",
			})
			return
		}

		// The token alone is not enough: the user may have been deleted or had their role changed since it was issued.
		user, found, err := lookup(claims.UserID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, response.Response{
				Code:    http.StatusInternalServerError,
				Message: "Internal server error",
			})
			return
		}
		if !found {
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.Response{
				Code:    http.StatusUnauthorized,
				Message: "Unauthorized",
			})
			return
		}

		c.Set("user", claims)
		c.Set("role", string(user.Role))
		c.Next()
	}
}

func findActiveUser(id bson.ObjectID) (*models.User, bool, error) {
	res := models.AuthDB.QueryOne(bson.M{"_id": id, "deleted_time": nil})
	switch res.Code {
	case http.StatusOK:
		return res.Data.(*models.User), true, nil
	case http.StatusNotFound:
		return nil, false, nil
	default:
		return nil, false, errors.New(res.Message)
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

