package claude

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/anthropics/anthropic-sdk-go/v2"
	"github.com/anthropics/anthropic-sdk-go/v2/option"
)

const testKey = "secret-key"

// base64 of the bytes in testImage.
const testImageBase64 = "AQI="

var testImage = Image{MediaType: "image/jpeg", Data: []byte{1, 2}}

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return New(testKey, "claude-haiku-5-5", option.WithBaseURL(srv.URL), option.WithMaxRetries(0))
}

func messageResponse(t *testing.T, stopReason, text string) http.HandlerFunc {
	t.Helper()
	quoted, err := json.Marshal(text)
	if err != nil {
		t.Fatal(err)
	}
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"id":"m","type":"message","role":"assistant","model":"x","content":[{"type":"thinking","thinking":"","signature":"s"},{"type":"text","text":%s}],"stop_reason":%q,"usage":{"input_tokens":3,"output_tokens":4}}`, quoted, stopReason)
	}
}

const validSuggestion = `{"name":"Bút bi","description":"Mực xanh","category_id":null,"reference_prices":[{"unit":"Hộp 20 cây","price":null},{"unit":"1 cây","price":5000}],"notes":["Không đọc rõ mã"]}`

func TestAnalyzeProductRequestShape(t *testing.T) {
	var request map[string]any
	reply := messageResponse(t, "end_turn", validSuggestion)
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &request); err != nil {
			t.Errorf("Expected a JSON request, got %v", err)
		}
		reply(w, r)
	})

	_, err := client.AnalyzeProduct(context.Background(), []Image{testImage}, []Category{{ID: "c1", Name: "Bút viết"}})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if request["model"] != "claude-haiku-5-5" {
		t.Errorf("Expected the configured model, got %v", request["model"])
	}
	if request["max_tokens"] != float64(16000) {
		t.Errorf("Expected max_tokens 16000, got %v", request["max_tokens"])
	}
	if system, ok := request["system"].([]any); !ok || len(system) == 0 {
		t.Errorf("Expected a system prompt, got %v", request["system"])
	}

	content := request["messages"].([]any)[0].(map[string]any)["content"].([]any)
	first := content[0].(map[string]any)
	source, _ := first["source"].(map[string]any)
	if first["type"] != "image" || source["media_type"] != "image/jpeg" || source["data"] != testImageBase64 {
		t.Errorf("Expected the image first, got %v", first)
	}
	last := content[len(content)-1].(map[string]any)
	text, _ := last["text"].(string)
	if last["type"] != "text" || !strings.Contains(text, "c1") || !strings.Contains(text, "Bút viết") {
		t.Errorf("Expected a final text block with the categories, got %v", last)
	}

	output, _ := request["output_config"].(map[string]any)
	if output["effort"] != "medium" {
		t.Errorf("Expected effort medium, got %v", output["effort"])
	}
	format, _ := output["format"].(map[string]any)
	if format["type"] != "json_schema" {
		t.Errorf("Expected a JSON schema format, got %v", format)
	}
	schema, _ := format["schema"].(map[string]any)
	required, _ := schema["required"].([]any)
	for _, field := range []string{"name", "description", "category_id", "reference_prices", "notes"} {
		found := false
		for _, r := range required {
			found = found || r == field
		}
		if !found {
			t.Errorf("Expected %s to be required, got %v", field, required)
		}
	}

	for _, key := range []string{"thinking", "fallbacks", "cache_control"} {
		if _, ok := request[key]; ok {
			t.Errorf("Expected no %s in the request", key)
		}
	}
}

func TestAnalyzeProductParsesSuggestion(t *testing.T) {
	client := newTestClient(t, messageResponse(t, "end_turn", validSuggestion))

	got, err := client.AnalyzeProduct(context.Background(), []Image{testImage}, nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if got.Name != "Bút bi" || got.Description != "Mực xanh" || got.CategoryID != nil {
		t.Errorf("Unexpected fields: %+v", got)
	}
	if len(got.ReferencePrices) != 2 || got.ReferencePrices[0].Unit != "Hộp 20 cây" || got.ReferencePrices[0].Price != nil {
		t.Fatalf("Expected a unit without a price first, got %+v", got.ReferencePrices)
	}
	if got.ReferencePrices[1].Price == nil || *got.ReferencePrices[1].Price != 5000 {
		t.Errorf("Expected price 5000, got %+v", got.ReferencePrices[1])
	}
	if len(got.Notes) != 1 || got.Notes[0] != "Không đọc rõ mã" {
		t.Errorf("Expected the note, got %v", got.Notes)
	}
}

func TestAnalyzeProductRefusal(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"m","type":"message","role":"assistant","model":"x","content":[],"stop_reason":"refusal","stop_details":{"type":"refusal","category":null,"explanation":null},"usage":{"input_tokens":3,"output_tokens":0}}`)
	})

	_, err := client.AnalyzeProduct(context.Background(), []Image{testImage}, nil)
	if !errors.Is(err, ErrRefused) {
		t.Errorf("Expected ErrRefused, got %v", err)
	}
}

func TestAnalyzeProductTruncated(t *testing.T) {
	// Valid JSON on purpose: a cut-off answer is not trusted even when it happens to parse.
	client := newTestClient(t, messageResponse(t, "max_tokens", validSuggestion))

	_, err := client.AnalyzeProduct(context.Background(), []Image{testImage}, nil)
	if !errors.Is(err, ErrBadOutput) {
		t.Errorf("Expected ErrBadOutput, got %v", err)
	}
}

func TestAnalyzeProductInvalidJSON(t *testing.T) {
	client := newTestClient(t, messageResponse(t, "end_turn", "{"))

	_, err := client.AnalyzeProduct(context.Background(), []Image{testImage}, nil)
	if !errors.Is(err, ErrBadOutput) {
		t.Errorf("Expected ErrBadOutput, got %v", err)
	}
}

func TestAnalyzeProductAPIError(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"}}`)
	})

	_, err := client.AnalyzeProduct(context.Background(), []Image{testImage}, nil)

	var apiErr *anthropic.Error
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected a 401 API error, got %v", err)
	}
	if strings.Contains(err.Error(), testKey) || strings.Contains(err.Error(), testImageBase64) {
		t.Errorf("Expected the error to hold neither the key nor the image, got %q", err.Error())
	}
}

func TestAnalyzeProductDeadline(t *testing.T) {
	client := newTestClient(t, func(_ http.ResponseWriter, r *http.Request) {
		// The server only notices a closed connection once the body has been read.
		_, _ = io.Copy(io.Discard, r.Body)
		<-r.Context().Done()
	})
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := client.AnalyzeProduct(ctx, []Image{testImage}, nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected context.DeadlineExceeded, got %v", err)
	}
}
