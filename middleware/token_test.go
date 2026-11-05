package middleware

import (
	"StoreServer/config"
	"StoreServer/models"
	"StoreServer/utils/jwt"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func init() {
	gin.SetMode(gin.TestMode)
	config.GetConfig().JWTSecret = "test_secret_key_12345"
}

func tokenFor(t *testing.T, id bson.ObjectID, role string) string {
	t.Helper()
	token, err := jwt.GenerateToken(id, "tester", role)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	return token
}

// serve runs one request through checkLogin (and CheckRole when roles are given).
func serve(lookup userLookup, token string, roles ...string) int {
	handlers := []gin.HandlerFunc{checkLogin(lookup)}
	if len(roles) > 0 {
		handlers = append(handlers, CheckRole(roles...))
	}
	handlers = append(handlers, func(c *gin.Context) { c.Status(http.StatusOK) })

	r := gin.New()
	r.GET("/", handlers...)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	r.ServeHTTP(w, req)
	return w.Code
}

func TestCheckLoginRejectsDeletedUser(t *testing.T) {
	notFound := func(bson.ObjectID) (*models.User, bool, error) { return nil, false, nil }

	if code := serve(notFound, tokenFor(t, bson.NewObjectID(), "admin")); code != http.StatusUnauthorized {
		t.Errorf("Expected 401 for a deleted user, got %d", code)
	}
}

func TestCheckLoginUsesRoleFromDatabase(t *testing.T) {
	demoted := func(id bson.ObjectID) (*models.User, bool, error) {
		return &models.User{ID: id, Role: models.CashierRole}, true, nil
	}

	if code := serve(demoted, tokenFor(t, bson.NewObjectID(), "admin"), "admin"); code != http.StatusForbidden {
		t.Errorf("Expected 403 when the database role is cashier, got %d", code)
	}
}

func TestCheckLoginAllowsActiveUser(t *testing.T) {
	id := bson.NewObjectID()
	var looked bson.ObjectID
	active := func(got bson.ObjectID) (*models.User, bool, error) {
		looked = got
		return &models.User{ID: got, Role: models.AdminRole}, true, nil
	}

	if code := serve(active, tokenFor(t, id, "admin"), "admin"); code != http.StatusOK {
		t.Errorf("Expected 200 for an active admin, got %d", code)
	}
	if looked != id {
		t.Errorf("Expected lookup of user %s, got %s", id.Hex(), looked.Hex())
	}
}

func TestCheckLoginLookupErrorIs500(t *testing.T) {
	failing := func(bson.ObjectID) (*models.User, bool, error) { return nil, false, errors.New("db down") }

	if code := serve(failing, tokenFor(t, bson.NewObjectID(), "admin")); code != http.StatusInternalServerError {
		t.Errorf("Expected 500 when the user lookup fails, got %d", code)
	}
}

func TestCheckLoginWithoutTokenSkipsLookup(t *testing.T) {
	called := false
	lookup := func(bson.ObjectID) (*models.User, bool, error) { called = true; return nil, false, nil }

	if code := serve(lookup, ""); code != http.StatusUnauthorized {
		t.Errorf("Expected 401 without token, got %d", code)
	}
	if called {
		t.Error("Lookup must not run without a valid token")
	}
}
