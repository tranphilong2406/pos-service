package config_test

import (
	"StoreServer/config"
	"strings"
	"testing"
)

func TestWarningsFlagShortJWTSecret(t *testing.T) {
	cfg := config.Config{JWTSecret: "only-19-characters!"}
	if w := cfg.Warnings(); len(w) != 1 || !strings.Contains(w[0], "JWT_SECRET") {
		t.Errorf("Expected one JWT_SECRET warning, got %v", w)
	}
}

func TestWarningsAcceptLongJWTSecret(t *testing.T) {
	cfg := config.Config{JWTSecret: strings.Repeat("x", 32)}
	if w := cfg.Warnings(); len(w) != 0 {
		t.Errorf("Expected no warnings, got %v", w)
	}
}
