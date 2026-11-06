package api

import (
	"StoreServer/models"
	"StoreServer/utils/claude"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

const (
	jpegMagic = "\xFF\xD8\xFF\xE0"
	webpMagic = "RIFF\x00\x00\x00\x00WEBPVP"
	gifMagic  = "GIF89a"
)

type fakeAnalyzer struct {
	suggestion claude.Suggestion
	err        error
	delay      time.Duration
	calls      int
	images     []claude.Image
	categories []claude.Category
	ctx        context.Context
}

func (f *fakeAnalyzer) AnalyzeProduct(ctx context.Context, images []claude.Image, categories []claude.Category) (claude.Suggestion, error) {
	f.calls++
	f.images, f.categories, f.ctx = images, categories, ctx
	time.Sleep(f.delay)
	return f.suggestion, f.err
}

func validAnalyzer() *fakeAnalyzer {
	return &fakeAnalyzer{suggestion: claude.Suggestion{Name: "Bút bi"}}
}

func noCategories() ([]claude.Category, error) {
	return nil, nil
}

type upload struct {
	contentType string
	data        []byte
}

func jpeg(size int) upload {
	data := make([]byte, size)
	copy(data, jpegMagic)
	return upload{contentType: "image/jpeg", data: data}
}

func imagesBody(t *testing.T, files ...upload) (*bytes.Buffer, string) {
	t.Helper()
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	for i, file := range files {
		header := textproto.MIMEHeader{}
		header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="images"; filename="a%d.jpg"`, i))
		header.Set("Content-Type", file.contentType)
		part, err := form.CreatePart(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write(file.data); err != nil {
			t.Fatal(err)
		}
	}
	if err := form.Close(); err != nil {
		t.Fatal(err)
	}
	return &body, form.FormDataContentType()
}

func analyzeRouter(analyzer ProductAnalyzer, list CategoryLister) *gin.Engine {
	r := gin.New()
	r.POST("/", AnalyzeProductImage(analyzer, list))
	return r
}

func postImages(t *testing.T, analyzer ProductAnalyzer, list CategoryLister, files ...upload) *httptest.ResponseRecorder {
	t.Helper()
	body, contentType := imagesBody(t, files...)
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	analyzeRouter(analyzer, list).ServeHTTP(w, req)
	return w
}

func TestAnalyzeWithoutAnalyzerIs503(t *testing.T) {
	listed := false
	list := func() ([]claude.Category, error) {
		listed = true
		return nil, nil
	}

	w := postImages(t, nil, list, jpeg(16))

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected 503, got %d", w.Code)
	}
	if listed {
		t.Error("Expected categories not to be read")
	}
}

func TestAnalyzeRejectsNonMultipart(t *testing.T) {
	analyzer := validAnalyzer()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	analyzeRouter(analyzer, noCategories).ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest || analyzer.calls != 0 {
		t.Errorf("Expected 400 without calling Claude, got %d and %d calls", w.Code, analyzer.calls)
	}
}

func TestAnalyzeRejectsNoImages(t *testing.T) {
	analyzer := validAnalyzer()

	w := postImages(t, analyzer, noCategories)

	if w.Code != http.StatusBadRequest || analyzer.calls != 0 {
		t.Errorf("Expected 400 without calling Claude, got %d and %d calls", w.Code, analyzer.calls)
	}
}

func TestAnalyzeRejectsFourImages(t *testing.T) {
	analyzer := validAnalyzer()

	w := postImages(t, analyzer, noCategories, jpeg(16), jpeg(16), jpeg(16), jpeg(16))

	if w.Code != http.StatusBadRequest || analyzer.calls != 0 {
		t.Errorf("Expected 400 without calling Claude, got %d and %d calls", w.Code, analyzer.calls)
	}
}

// The filename and the header both claim a JPEG, so only the content can expose it.
func TestAnalyzeRejectsSpoofedImage(t *testing.T) {
	analyzer := validAnalyzer()

	w := postImages(t, analyzer, noCategories, upload{contentType: "image/jpeg", data: []byte("<html><body>hi</body></html>")})

	if w.Code != http.StatusBadRequest || analyzer.calls != 0 {
		t.Errorf("Expected 400 without spending credit, got %d and %d calls", w.Code, analyzer.calls)
	}
}

func TestAnalyzeRejectsGIF(t *testing.T) {
	analyzer := validAnalyzer()

	w := postImages(t, analyzer, noCategories, upload{contentType: "image/gif", data: []byte(gifMagic + "rest")})

	if w.Code != http.StatusBadRequest || analyzer.calls != 0 {
		t.Errorf("Expected 400 without calling Claude, got %d and %d calls", w.Code, analyzer.calls)
	}
}

func TestAnalyzeRejectsOversizedImage(t *testing.T) {
	analyzer := validAnalyzer()

	w := postImages(t, analyzer, noCategories, jpeg(5<<20+1))

	if w.Code != http.StatusRequestEntityTooLarge || analyzer.calls != 0 {
		t.Errorf("Expected 413 without calling Claude, got %d and %d calls", w.Code, analyzer.calls)
	}
}

func TestAnalyzeRejectsOversizedBody(t *testing.T) {
	analyzer := validAnalyzer()

	w := postImages(t, analyzer, noCategories, jpeg(6<<20), jpeg(6<<20), jpeg(6<<20))

	if w.Code != http.StatusRequestEntityTooLarge || analyzer.calls != 0 {
		t.Errorf("Expected 413 without calling Claude, got %d and %d calls", w.Code, analyzer.calls)
	}
}

// Three images at the per-image limit must also fit the body limit.
func TestAnalyzeAcceptsImagesAtLimit(t *testing.T) {
	analyzer := validAnalyzer()

	w := postImages(t, analyzer, noCategories, jpeg(5<<20), jpeg(5<<20), jpeg(5<<20))

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if len(analyzer.images) != 3 || len(analyzer.images[0].Data) != 5<<20 {
		t.Errorf("Expected 3 full images, got %d", len(analyzer.images))
	}
}

func TestAnalyzePassesSniffedMediaType(t *testing.T) {
	analyzer := validAnalyzer()

	w := postImages(t, analyzer, noCategories, upload{contentType: "image/jpeg", data: []byte(webpMagic + "rest")})

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if analyzer.images[0].MediaType != "image/webp" {
		t.Errorf("Expected the sniffed type, got %q", analyzer.images[0].MediaType)
	}
}

func intPtr(v int) *int       { return &v }
func strPtr(v string) *string { return &v }

func TestSanitizeSuggestionRequiresName(t *testing.T) {
	if _, ok := sanitizeSuggestion(claude.Suggestion{Name: "  "}, nil); ok {
		t.Error("Expected a blank name to be rejected")
	}
}

func TestSanitizeSuggestionTrimsText(t *testing.T) {
	got, ok := sanitizeSuggestion(claude.Suggestion{
		Name:            "  Bút bi ",
		Description:     " Mực xanh ",
		ReferencePrices: []claude.PriceSuggestion{{Unit: " Lốc "}},
		Notes:           []string{" Kiểm tra mã ", "   "},
	}, nil)

	if !ok || got.Name != "Bút bi" || got.Description != "Mực xanh" {
		t.Errorf("Expected trimmed text, got %+v", got)
	}
	if got.ReferencePrices[0].Unit != "Lốc" {
		t.Errorf("Expected a trimmed unit, got %q", got.ReferencePrices[0].Unit)
	}
	if len(got.Notes) != 1 || got.Notes[0] != "Kiểm tra mã" {
		t.Errorf("Expected the blank note to be dropped, got %q", got.Notes)
	}
}

func TestSanitizeSuggestionChecksCategory(t *testing.T) {
	categories := []claude.Category{{ID: "c1", Name: "Bút viết"}}

	got, _ := sanitizeSuggestion(claude.Suggestion{Name: "But", CategoryID: strPtr("c1")}, categories)
	if got.CategoryID == nil || *got.CategoryID != "c1" {
		t.Errorf("Expected a known category to stay, got %v", got.CategoryID)
	}

	got, _ = sanitizeSuggestion(claude.Suggestion{Name: "But", CategoryID: strPtr("khong-co")}, categories)
	if got.CategoryID != nil {
		t.Errorf("Expected an unknown category to become nil, got %v", *got.CategoryID)
	}
}

func TestSanitizeSuggestionClearsBadPrices(t *testing.T) {
	got, _ := sanitizeSuggestion(claude.Suggestion{Name: "But", ReferencePrices: []claude.PriceSuggestion{
		{Unit: "Zero", Price: intPtr(0)},
		{Unit: "Negative", Price: intPtr(-5)},
		{Unit: "Good", Price: intPtr(5000)},
		{Unit: "Missing"},
	}}, nil)

	if len(got.ReferencePrices) != 4 {
		t.Fatalf("Expected the units to stay, got %+v", got.ReferencePrices)
	}
	if got.ReferencePrices[0].Price != nil || got.ReferencePrices[1].Price != nil || got.ReferencePrices[3].Price != nil {
		t.Errorf("Expected bad prices to become nil, got %+v", got.ReferencePrices)
	}
	if got.ReferencePrices[2].Price == nil || *got.ReferencePrices[2].Price != 5000 {
		t.Errorf("Expected 5000 to stay, got %+v", got.ReferencePrices[2])
	}
}

func TestSanitizeSuggestionDropsBadUnits(t *testing.T) {
	got, _ := sanitizeSuggestion(claude.Suggestion{Name: "But", ReferencePrices: []claude.PriceSuggestion{
		{Unit: "   "},
		{Unit: strings.Repeat("ố", 51)},
		{Unit: strings.Repeat("ố", 50)},
		{Unit: "Lốc", Price: intPtr(1)},
		{Unit: "lốc", Price: intPtr(2)},
	}}, nil)

	if len(got.ReferencePrices) != 2 {
		t.Fatalf("Expected 2 lines to stay, got %+v", got.ReferencePrices)
	}
	if got.ReferencePrices[1].Unit != "Lốc" || *got.ReferencePrices[1].Price != 1 {
		t.Errorf("Expected the first of the duplicates to stay, got %+v", got.ReferencePrices[1])
	}
}

func manyPriceSuggestions(n int) []claude.PriceSuggestion {
	lines := make([]claude.PriceSuggestion, n)
	for i := range lines {
		lines[i] = claude.PriceSuggestion{Unit: fmt.Sprintf("Unit %d", i)}
	}
	return lines
}

func TestSanitizeSuggestionKeepsFirstTenLines(t *testing.T) {
	got, _ := sanitizeSuggestion(claude.Suggestion{Name: "But", ReferencePrices: manyPriceSuggestions(12)}, nil)

	if len(got.ReferencePrices) != 10 || got.ReferencePrices[9].Unit != "Unit 9" {
		t.Errorf("Expected the first 10 lines, got %+v", got.ReferencePrices)
	}
}

func TestSanitizeSuggestionAlwaysReturnsLists(t *testing.T) {
	got, _ := sanitizeSuggestion(claude.Suggestion{Name: "But"}, nil)

	if got.ReferencePrices == nil || got.Notes == nil {
		t.Errorf("Expected empty lists, got %#v", got)
	}
}

func TestSanitizedSuggestionPassesProductValidate(t *testing.T) {
	lines := manyPriceSuggestions(12)
	lines[1].Unit = "unit 0"
	lines[2].Unit = strings.Repeat("ố", 51)
	lines[3].Unit = "  "

	got, ok := sanitizeSuggestion(claude.Suggestion{Name: "  But  ", ReferencePrices: lines}, nil)
	if !ok {
		t.Fatal("Expected the suggestion to be usable")
	}

	product := models.Product{Name: got.Name}
	for _, line := range got.ReferencePrices {
		product.ReferencePrices = append(product.ReferencePrices, models.ReferencePrice{Unit: line.Unit, Price: 1000})
	}
	if res := product.Validate(); res.Code != http.StatusOK {
		t.Errorf("Expected the owner to be able to save it after filling in prices, got %d: %s", res.Code, res.Message)
	}
}

func TestAnalyzeSuccessResponse(t *testing.T) {
	analyzer := &fakeAnalyzer{suggestion: claude.Suggestion{
		Name:            " Bút bi ",
		ReferencePrices: []claude.PriceSuggestion{{Unit: "Hộp 20 cây"}},
	}}
	categories := func() ([]claude.Category, error) {
		return []claude.Category{{ID: "c1", Name: "Bút viết"}}, nil
	}

	w := postImages(t, analyzer, categories, jpeg(16))

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var res struct {
		Message string `json:"message"`
		Data    map[string]any
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if res.Message != "ok" || res.Data["name"] != "Bút bi" {
		t.Errorf("Unexpected response: %s", w.Body.String())
	}
	if value, ok := res.Data["category_id"]; !ok || value != nil {
		t.Errorf("Expected category_id to be null, got %v", value)
	}
	lines, ok := res.Data["reference_prices"].([]any)
	if !ok || len(lines) != 1 {
		t.Fatalf("Expected one reference price, got %v", res.Data["reference_prices"])
	}
	if value, ok := lines[0].(map[string]any)["price"]; !ok || value != nil {
		t.Errorf("Expected price to be null, got %v", value)
	}
	if _, ok := res.Data["notes"].([]any); !ok {
		t.Errorf("Expected notes to be a list, got %v", res.Data["notes"])
	}
	if len(analyzer.categories) != 1 || analyzer.categories[0].ID != "c1" {
		t.Errorf("Expected the categories to reach Claude, got %v", analyzer.categories)
	}
}

func TestAnalyzeErrorMapping(t *testing.T) {
	cases := map[string]struct {
		analyzer *fakeAnalyzer
		want     int
	}{
		"refused":    {&fakeAnalyzer{err: fmt.Errorf("wrapped: %w", claude.ErrRefused)}, http.StatusUnprocessableEntity},
		"no product": {&fakeAnalyzer{suggestion: claude.Suggestion{Name: " "}}, http.StatusUnprocessableEntity},
		"deadline":   {&fakeAnalyzer{err: fmt.Errorf("claude: %w", context.DeadlineExceeded)}, http.StatusGatewayTimeout},
		"bad output": {&fakeAnalyzer{err: claude.ErrBadOutput}, http.StatusBadGateway},
		"other":      {&fakeAnalyzer{err: errors.New("secret detail")}, http.StatusBadGateway},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			w := postImages(t, tc.analyzer, noCategories, jpeg(16))

			if w.Code != tc.want {
				t.Errorf("Expected %d, got %d", tc.want, w.Code)
			}
			if strings.Contains(w.Body.String(), "secret detail") {
				t.Errorf("Expected the cause to stay out of the response, got %s", w.Body.String())
			}
		})
	}
}

func TestAnalyzeClientCancelIs499(t *testing.T) {
	analyzer := &fakeAnalyzer{err: fmt.Errorf("claude: %w", context.Canceled)}

	w := postImages(t, analyzer, noCategories, jpeg(16))

	if w.Code != 499 {
		t.Errorf("Expected 499, got %d", w.Code)
	}
	if w.Body.Len() != 0 {
		t.Errorf("Expected no body, got %s", w.Body.String())
	}
}

func TestAnalyzeCategoryErrorIs500(t *testing.T) {
	analyzer := validAnalyzer()
	failing := func() ([]claude.Category, error) {
		return nil, errors.New("secret db detail")
	}

	w := postImages(t, analyzer, failing, jpeg(16))

	if w.Code != http.StatusInternalServerError || analyzer.calls != 0 {
		t.Errorf("Expected 500 without calling Claude, got %d and %d calls", w.Code, analyzer.calls)
	}
	if strings.Contains(w.Body.String(), "secret db detail") {
		t.Errorf("Expected the cause to stay out of the response, got %s", w.Body.String())
	}
}

func TestAnalyzeContextHasDeadline(t *testing.T) {
	analyzer := validAnalyzer()

	postImages(t, analyzer, noCategories, jpeg(16))

	deadline, ok := analyzer.ctx.Deadline()
	if !ok {
		t.Fatal("Expected a deadline on the context")
	}
	if remaining := time.Until(deadline); remaining > 60*time.Second {
		t.Errorf("Expected at most 60 seconds, got %v", remaining)
	}
}

// Claude can take longer than the server-wide WriteTimeout, which would otherwise cut the response.
func TestAnalyzeOutlivesServerWriteTimeout(t *testing.T) {
	analyzer := &fakeAnalyzer{suggestion: claude.Suggestion{Name: "Bút bi"}, delay: 300 * time.Millisecond}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: analyzeRouter(analyzer, noCategories), WriteTimeout: 100 * time.Millisecond}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })

	body, contentType := imagesBody(t, jpeg(16))
	res, err := http.Post("http://"+listener.Addr().String()+"/", contentType, body)
	if err != nil {
		t.Fatalf("Expected a response after the write timeout, got %v", err)
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK || !strings.Contains(string(data), "Bút bi") {
		t.Errorf("Expected the full response, got %d: %s", res.StatusCode, data)
	}
}
