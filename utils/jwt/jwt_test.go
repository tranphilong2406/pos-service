package jwt_test

import (
	"StoreServer/config"
	"StoreServer/utils/jwt"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestGenerateAndParseToken(t *testing.T) {
	cfg := config.GetConfig()
	cfg.JWTSecret = "test_secret_key_12345"
	cfg.JWTExpiryHours = 24

	userID := bson.NewObjectID()
	username := "testuser"
	role := "admin"

	token, err := jwt.GenerateToken(userID, username, role)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	claims, err := jwt.ParseToken(token)
	if err != nil {
		t.Fatalf("Failed to parse token: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("Expected UserID %v, got %v", userID, claims.UserID)
	}
	if claims.Username != username {
		t.Errorf("Expected Username %s, got %s", username, claims.Username)
	}
	if claims.Role != role {
		t.Errorf("Expected Role %s, got %s", role, claims.Role)
	}
	if claims.ExpiresAt == nil || claims.ExpiresAt.Time.Before(time.Now()) {
		t.Errorf("Expected valid future expiration date, got %v", claims.ExpiresAt)
	}

	valid, err := jwt.ValidateToken(token)
	if err != nil || !valid {
		t.Errorf("Expected token to be valid, got valid=%v, err=%v", valid, err)
	}
}

func TestParseInvalidToken(t *testing.T) {
	cfg := config.GetConfig()
	cfg.JWTSecret = "test_secret_key_12345"

	_, err := jwt.ParseToken("invalid.token.string")
	if err == nil {
		t.Errorf("Expected error when parsing invalid token, got nil")
	}
}
