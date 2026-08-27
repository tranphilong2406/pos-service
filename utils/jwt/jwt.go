package jwt

import (
	"StoreServer/config"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Claims struct {
	UserID   bson.ObjectID `json:"user_id"`
	Username string        `json:"username"`
	Role     string        `json:"role"`
	jwt.RegisteredClaims
}

func GenerateToken(userID bson.ObjectID, username, role string) (string, error) {
	cfg := config.GetConfig()
	secret := cfg.JWTSecret
	if secret == "" {
		return "", errors.New("JWT secret is not configured")
	}

	expiryHours := cfg.JWTExpiryHours
	if expiryHours <= 0 {
		expiryHours = 24
	}

	claims := Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expiryHours) * time.Hour)),
			Issuer:    "stationery_store",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

func ValidateToken(tokenString string) (bool, error) {
	cfg := config.GetConfig()
	secret := cfg.JWTSecret
	if secret == "" {
		return false, errors.New("JWT secret is not configured")
	}

	_, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil {
		return false, err
	}

	return true, nil
}

func ParseToken(tokenString string) (*Claims, error) {
	cfg := config.GetConfig()
	secret := cfg.JWTSecret
	if secret == "" {
		return nil, errors.New("JWT secret is not configured")
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

