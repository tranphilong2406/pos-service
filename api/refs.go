package api

import (
	"StoreServer/job"
	"StoreServer/models"
	"StoreServer/utils/response"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// checkRef answers 400 when id is set but is not an existing, non-deleted document of db.
// It returns false when it has already written the response.
func checkRef(c *gin.Context, db *job.DB, field, id string) bool {
	if id == "" {
		return true
	}

	if objID, err := bson.ObjectIDFromHex(id); err == nil {
		res := db.QueryOne(bson.M{"_id": objID, "deleted_time": nil})
		if res.Code == http.StatusOK {
			return true
		}
		if res.Code != http.StatusNotFound {
			c.JSON(res.Code, res)
			return false
		}
	}

	c.JSON(http.StatusBadRequest, response.Response{
		Code:    http.StatusBadRequest,
		Message: field + " not found: " + id,
	})
	return false
}

// checkProducts answers 400 naming the first order/purchase line whose product does not exist or was deleted.
// It returns false when it has already written the response.
func checkProducts(c *gin.Context, lines []models.ProductOrder) bool {
	for _, line := range lines {
		if !checkRef(c, &models.ProductDB, "product", line.ID.Hex()) {
			return false
		}
	}
	return true
}
