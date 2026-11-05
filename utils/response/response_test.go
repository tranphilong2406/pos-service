package response_test

import (
	myerr "StoreServer/utils/error"
	"StoreServer/utils/response"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func keysOf(t *testing.T, write func(c *gin.Context)) (int, string) {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	write(c)

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	keys := make([]string, 0, len(body))
	for k := range body {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return w.Code, strings.Join(keys, ",")
}

func TestAllHelpersWriteTheSameShape(t *testing.T) {
	const want = "code,data,message,page,page_size,total"

	cases := map[string]func(c *gin.Context){
		"Success": func(c *gin.Context) { response.MyResponse.Success(c, gin.H{"token": "x"}) },
		"Created": func(c *gin.Context) { response.MyResponse.Created(c, gin.H{"id": "x"}) },
		"Error": func(c *gin.Context) {
			response.MyResponse.Error(c, myerr.AnyError(http.StatusBadRequest, errors.New("bad")))
		},
	}
	for name, write := range cases {
		if _, keys := keysOf(t, write); keys != want {
			t.Errorf("%s wrote keys %q, want %q", name, keys, want)
		}
	}
}

func TestSuccessKeepsStatusCode(t *testing.T) {
	code, _ := keysOf(t, func(c *gin.Context) { response.MyResponse.Created(c, nil) })
	if code != http.StatusCreated {
		t.Errorf("Expected 201, got %d", code)
	}
}
