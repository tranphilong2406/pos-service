package cmd

import (
	"StoreServer/config"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// A typed nil here would make the route call a nil client instead of answering 503.
func TestNewProductAnalyzerWithoutKeyIsNil(t *testing.T) {
	if analyzer := newProductAnalyzer(&config.Config{}); analyzer != nil {
		t.Errorf("Expected a nil interface, got %#v", analyzer)
	}
}

func TestNewProductAnalyzerWithKey(t *testing.T) {
	if analyzer := newProductAnalyzer(&config.Config{AnthropicAPIKey: "k", ClaudeModel: "m"}); analyzer == nil {
		t.Error("Expected an analyzer when a key is set")
	}
}

func TestAnalyzeRouteIsBehindLogin(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/products/analyze-image", nil)

	SetHandler().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 without a token, got %d", w.Code)
	}
}
