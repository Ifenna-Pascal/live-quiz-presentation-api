package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ifenna-pascal/interactive-presentation/pkg/token"
)

func init() { gin.SetMode(gin.TestMode) }

func TestRateLimiterBlocksAfterLimitAndResets(t *testing.T) {
	now := time.Unix(0, 0)
	r := NewRateLimiter(2, time.Minute)
	r.now = func() time.Time { return now }

	for i := 0; i < 2; i++ {
		if ok, _ := r.Allow("ip"); !ok {
			t.Fatalf("request %d should pass", i+1)
		}
	}
	if ok, retry := r.Allow("ip"); ok || retry <= 0 {
		t.Fatalf("third request should be blocked with a retry time, got ok=%v retry=%v", ok, retry)
	}
	if ok, _ := r.Allow("other-ip"); !ok {
		t.Fatal("other keys have their own limit")
	}
	now = now.Add(time.Minute)
	if ok, _ := r.Allow("ip"); !ok {
		t.Fatal("a new window should allow requests again")
	}
}

func TestRateLimitMiddlewareAnswers429(t *testing.T) {
	router := gin.New()
	router.GET("/", RateLimit(1, time.Minute), func(c *gin.Context) { c.Status(http.StatusOK) })

	codes := []int{}
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
		codes = append(codes, w.Code)
	}
	if codes[0] != http.StatusOK || codes[1] != http.StatusTooManyRequests {
		t.Fatalf("got %v, want [200 429]", codes)
	}
}

func TestContentTypeChecker(t *testing.T) {
	router := gin.New()
	router.POST("/", ContentTypeChecker(), func(c *gin.Context) { c.Status(http.StatusOK) })

	for contentType, want := range map[string]int{"application/json": 200, "text/plain": 415, "": 415} {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}"))
		req.Header.Set("Content-Type", contentType)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != want {
			t.Errorf("Content-Type %q: got %d, want %d", contentType, w.Code, want)
		}
	}
}

func TestAuthMiddlewares(t *testing.T) {
	maker, _ := token.NewJWTMaker("test-secret-key-that-is-long-enough-123")
	valid, _ := maker.CreateToken("7c9e6679-7425-40de-944b-e07fc1f90ae7", time.Minute)

	cases := []struct {
		name     string
		header   string
		required int
		optional int
	}{
		{"no header", "", 401, 200},
		{"malformed", "Token abc", 401, 401},
		{"invalid token", "Bearer abc", 401, 401},
		{"valid token", "Bearer " + valid, 200, 200},
	}
	for _, tc := range cases {
		for _, mw := range []struct {
			handler gin.HandlerFunc
			want    int
		}{{AuthMiddleware(maker), tc.required}, {OptionalAuthMiddleware(maker), tc.optional}} {
			router := gin.New()
			router.GET("/", mw.handler, func(c *gin.Context) { c.Status(http.StatusOK) })
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != mw.want {
				t.Errorf("%s: got %d, want %d", tc.name, w.Code, mw.want)
			}
		}
	}
}
