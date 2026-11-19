package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func run(h gin.HandlerFunc, m, auth string) int {
	r := gin.New()
	r.Use(h)
	r.Any("/", func(c *gin.Context) { c.Status(200) })
	req := httptest.NewRequest(m, "/", nil)
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code
}

func tok(key, role string) string {
	s, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": 1, "role": role}).SignedString([]byte(key))
	return s
}

func TestCORS(t *testing.T) {
	if run(CORS(), http.MethodOptions, "") != 204 || run(CORS(), "GET", "") != 200 {
		t.Fatal("cors")
	}
}

func TestAuthAndRole(t *testing.T) {
	a := Auth("k")
	if run(a, "GET", "") != 401 || run(a, "GET", "Bearer xxx") != 401 || run(a, "GET", "Bearer "+tok("lain", "admin")) != 401 {
		t.Fatal("auth harus 401")
	}
	if run(a, "GET", "Bearer "+tok("k", "admin")) != 200 {
		t.Fatal("auth valid")
	}
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("role", "member") })
	r.GET("/a", RequireRole("admin"), func(c *gin.Context) {})
	r.GET("/m", RequireRole("member"), func(c *gin.Context) {})
	for p, want := range map[string]int{"/a": 403, "/m": 200} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", p, nil))
		if w.Code != want {
			t.Fatal(p, w.Code)
		}
	}
}

func TestRateLimiter(t *testing.T) {
	h := RateLimiter(1, 30*time.Millisecond)
	r := gin.New()
	r.Use(h)
	r.GET("/", func(c *gin.Context) {})
	do := func() int {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
		return w.Code
	}
	if do() != 200 || do() != 429 {
		t.Fatal("limit")
	}
	time.Sleep(50 * time.Millisecond)
	if do() != 200 {
		t.Fatal("reset")
	}
}
