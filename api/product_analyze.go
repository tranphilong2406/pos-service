package api

import (
	"StoreServer/models"
	"StoreServer/utils/claude"
	"StoreServer/utils/logger"
	"StoreServer/utils/response"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/zap"
)

const (
	maxAnalyzeImages     = 3
	maxAnalyzeImageBytes = 5 << 20
	maxAnalyzeBodyBytes  = 16 << 20
	maxAnalyzeCategories = 200

	analyzeTimeout = 60 * time.Second
	// analyzeWriteDeadline replaces the server-wide WriteTimeout for this route only. It leaves
	// time to upload the images and to answer after analyzeTimeout has passed.
	analyzeWriteDeadline = 70 * time.Second

	// statusClientClosedRequest is the conventional status for a request the client gave up on.
	statusClientClosedRequest = 499
)

// ProductAnalyzer suggests product form values from photos.
type ProductAnalyzer interface {
	AnalyzeProduct(ctx context.Context, images []claude.Image, categories []claude.Category) (claude.Suggestion, error)
}

// CategoryLister returns the categories Claude may choose from.
type CategoryLister func() ([]claude.Category, error)

// ListActiveCategories reads the categories that have not been deleted.
func ListActiveCategories() ([]claude.Category, error) {
	res := models.CategoryDB.Query(bson.M{"deleted_time": nil}, 0, maxAnalyzeCategories)
	if res.Code != http.StatusOK {
		return nil, errors.New(res.Message)
	}
	stored, ok := res.Data.([]models.Category)
	if !ok {
		return nil, errors.New("unexpected category list type")
	}

	categories := make([]claude.Category, 0, len(stored))
	for _, category := range stored {
		categories = append(categories, claude.Category{ID: category.ID.Hex(), Name: category.Name})
	}
	return categories, nil
}

// AnalyzeProductImage suggests form values from product photos. It saves nothing: the
// client shows the suggestion and saves through the normal product routes.
func AnalyzeProductImage(analyzer ProductAnalyzer, listCategories CategoryLister) gin.HandlerFunc {
	return func(c *gin.Context) {
		if analyzer == nil {
			respondAnalyze(c, http.StatusServiceUnavailable, "Product image analysis is not configured")
			return
		}

		// Without this the server-wide WriteTimeout could cut the response while Claude works.
		deadline := time.Now().Add(analyzeWriteDeadline)
		if err := http.NewResponseController(c.Writer).SetWriteDeadline(deadline); err != nil {
			logger.GetLogger().Warn("could not extend the write deadline", zap.Error(err))
		}

		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxAnalyzeBodyBytes)
		images, status, message := readAnalyzeImages(c)
		if status != http.StatusOK {
			respondAnalyze(c, status, message)
			return
		}

		categories, err := listCategories()
		if err != nil {
			logger.GetLogger().Error("could not read categories for image analysis", zap.Error(err))
			respondAnalyze(c, http.StatusInternalServerError, "Could not read categories")
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), analyzeTimeout)
		defer cancel()

		suggestion, err := analyzer.AnalyzeProduct(ctx, images, categories)
		if err != nil {
			respondAnalyzeError(c, err)
			return
		}

		cleaned, ok := sanitizeSuggestion(suggestion, categories)
		if !ok {
			respondAnalyze(c, http.StatusUnprocessableEntity, "No product recognized in the images")
			return
		}
		response.MyResponse.Success(c, cleaned)
	}
}

func respondAnalyze(c *gin.Context, status int, message string) {
	c.JSON(status, response.Response{Code: status, Message: message})
}

// respondAnalyzeError keeps the cause in the log: it can hold details of the Claude account.
func respondAnalyzeError(c *gin.Context, err error) {
	if errors.Is(err, context.Canceled) {
		c.Status(statusClientClosedRequest)
		return
	}

	logger.GetLogger().Error("product image analysis failed", zap.Error(err))
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		respondAnalyze(c, http.StatusGatewayTimeout, "Product image analysis timed out")
	case errors.Is(err, claude.ErrRefused):
		respondAnalyze(c, http.StatusUnprocessableEntity, "These images cannot be analyzed")
	default:
		respondAnalyze(c, http.StatusBadGateway, "Product image analysis failed")
	}
}

// readAnalyzeImages returns the uploaded images, or the status and message to answer with.
func readAnalyzeImages(c *gin.Context) ([]claude.Image, int, string) {
	form, err := c.MultipartForm()
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return nil, http.StatusRequestEntityTooLarge, "Images are too large"
		}
		return nil, http.StatusBadRequest, "Send the images as multipart/form-data"
	}

	files := form.File["images"]
	if len(files) == 0 || len(files) > maxAnalyzeImages {
		return nil, http.StatusBadRequest, "Send 1 to 3 images in the images field"
	}

	images := make([]claude.Image, 0, len(files))
	for _, header := range files {
		if header.Size > maxAnalyzeImageBytes {
			return nil, http.StatusRequestEntityTooLarge, "Images are too large"
		}
		file, err := header.Open()
		if err != nil {
			return nil, http.StatusBadRequest, "Could not read the images"
		}
		data, err := io.ReadAll(file)
		_ = file.Close()
		if err != nil {
			return nil, http.StatusBadRequest, "Could not read the images"
		}

		// The filename and the part's Content-Type come from the client, so go by the content.
		mediaType := http.DetectContentType(data)
		switch mediaType {
		case "image/jpeg", "image/png", "image/webp":
		default:
			return nil, http.StatusBadRequest, "Images must be JPEG, PNG or WebP"
		}
		images = append(images, claude.Image{MediaType: mediaType, Data: data})
	}
	return images, http.StatusOK, ""
}

// sanitizeSuggestion cleans model output so that it passes Product.Validate once the owner
// has filled in the missing prices. It returns false when no product name is left.
func sanitizeSuggestion(s claude.Suggestion, categories []claude.Category) (claude.Suggestion, bool) {
	cleaned := claude.Suggestion{
		Name:            strings.TrimSpace(s.Name),
		Description:     strings.TrimSpace(s.Description),
		CategoryID:      knownCategoryID(s.CategoryID, categories),
		ReferencePrices: sanitizePriceLines(s.ReferencePrices),
		Notes:           []string{},
	}
	if cleaned.Name == "" {
		return claude.Suggestion{}, false
	}

	for _, note := range s.Notes {
		if note = strings.TrimSpace(note); note != "" {
			cleaned.Notes = append(cleaned.Notes, note)
		}
	}
	return cleaned, true
}

func knownCategoryID(id *string, categories []claude.Category) *string {
	if id == nil {
		return nil
	}
	for _, category := range categories {
		if category.ID == *id {
			return id
		}
	}
	return nil
}

func sanitizePriceLines(lines []claude.PriceSuggestion) []claude.PriceSuggestion {
	cleaned := make([]claude.PriceSuggestion, 0, len(lines))
	seen := make(map[string]struct{}, len(lines))
	for _, line := range lines {
		if len(cleaned) == models.MaxReferencePrices {
			break
		}
		unit := strings.TrimSpace(line.Unit)
		if unit == "" || utf8.RuneCountInString(unit) > models.MaxReferenceUnitLen {
			continue
		}
		key := strings.ToLower(unit)
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		seen[key] = struct{}{}

		price := line.Price
		if price != nil && *price <= 0 {
			price = nil
		}
		cleaned = append(cleaned, claude.PriceSuggestion{Unit: unit, Price: price})
	}
	return cleaned
}
