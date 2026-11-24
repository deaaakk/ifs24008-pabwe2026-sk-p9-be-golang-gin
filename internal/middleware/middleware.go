package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"app/internal/response"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("Access-Control-Allow-Origin", "*")
		h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func Auth(key string) gin.HandlerFunc {
	return func(c *gin.Context) {
		hd := c.GetHeader("Authorization")
		t := strings.TrimPrefix(hd, "Bearer ")
		if t == "" || t == hd {
			response.Fail(c, http.StatusUnauthorized, "Token tidak ditemukan")
			c.Abort()
			return
		}
		claims := jwt.MapClaims{}
		tok, err := jwt.ParseWithClaims(t, claims, func(*jwt.Token) (any, error) { return []byte(key), nil },
			jwt.WithValidMethods([]string{"HS256"}))
		sub, ok := claims["sub"].(float64)
		if err != nil || !tok.Valid || !ok {
			response.Fail(c, http.StatusUnauthorized, "Token tidak valid")
			c.Abort()
			return
		}
		c.Set("userID", uint(sub))
		c.Set("role", claims["role"])
		c.Next()
	}
}

func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		r, _ := c.Get("role")
		for _, x := range roles {
			if r == x {
				c.Next()
				return
			}
		}
		response.Fail(c, http.StatusForbidden, "Akses ditolak")
		c.Abort()
	}
}

func RateLimiter(max int, per time.Duration) gin.HandlerFunc {
	var mu sync.Mutex
	hits := map[string][]time.Time{}
	return func(c *gin.Context) {
		mu.Lock()
		now, ip := time.Now(), c.ClientIP()
		var keep []time.Time
		for _, t := range hits[ip] {
			if now.Sub(t) < per {
				keep = append(keep, t)
			}
		}
		if len(keep) >= max {
			hits[ip] = keep
			mu.Unlock()
			response.Fail(c, http.StatusTooManyRequests, "Terlalu banyak permintaan")
			c.Abort()
			return
		}
		hits[ip] = append(keep, now)
		mu.Unlock()
		c.Next()
	}
}
