package auth

import (
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"app/internal/config"
	"app/internal/database"

	"github.com/gin-gonic/gin"
)

func TestAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{DBConn: "sqlite", AdminUser: "admin", AdminPass: "admin"}
	db, _ := database.Connect(cfg)
	database.Migrate(db, cfg)
	h := &Handler{S: &Service{Repo: &Repository{DB: db}, Key: "k"}}
	r := gin.New()
	r.Use(func(c *gin.Context) {
		id, _ := strconv.Atoi(c.GetHeader("X-Uid"))
		c.Set("userID", uint(id))
	})
	r.POST("/reg", h.Register)
	r.POST("/login", h.Login)
	r.POST("/logout", h.Logout)
	r.GET("/profile", h.Profile)
	r.PUT("/profile", h.UpdateProfile)

	good := `{"username":"budi","email":"b@x.id","password":"rahasia"}`
	for _, c := range []struct {
		m, p, uid, body string
		want            int
	}{
		{"POST", "/reg", "", good, 201},
		{"POST", "/reg", "", good, 400},
		{"POST", "/reg", "", `{"username":"ani","password":"rahasia"}`, 400},
		{"POST", "/reg", "", `{"username":"ani","email":"ani@x.id","password":"123"}`, 400},
		{"POST", "/reg", "", `{"username":"ani","email":"bukan-email","password":"rahasia"}`, 400},
		{"POST", "/reg", "", "{", 400},
		{"POST", "/login", "", "{", 400},
		{"POST", "/login", "", `{"username":"budi","password":"salah"}`, 401},
		{"POST", "/login", "", `{"username":"nouser","password":"x"}`, 401},
		{"POST", "/login", "", `{"username":"budi","password":"rahasia"}`, 200},
		{"POST", "/login", "", `{"username":"b@x.id","password":"rahasia"}`, 200},
		{"POST", "/login", "", `{"email":"b@x.id","password":"rahasia"}`, 200},
		{"POST", "/login", "", `{"password":"rahasia"}`, 400},
		{"POST", "/logout", "", "", 200},
		{"GET", "/profile", "2", "", 200},
		{"GET", "/profile", "999", "", 404},
		{"PUT", "/profile", "2", "{", 400},
		{"PUT", "/profile", "999", good, 404},
		{"PUT", "/profile", "2", `{"username":"admin"}`, 400},
		{"PUT", "/profile", "2", `{"email":"baru@x.id","password":"passbaru1"}`, 200},
		{"POST", "/login", "", `{"username":"baru@x.id","password":"passbaru1"}`, 200},
		{"PUT", "/profile", "2", `{"username":"budi","email":"b@x.id","password":"baru123"}`, 200},
		{"PUT", "/profile", "2", `{"username":"budi","email":"b@x.id"}`, 200},
	} {
		req := httptest.NewRequest(c.m, c.p, strings.NewReader(c.body))
		req.Header.Set("X-Uid", c.uid)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != c.want {
			t.Fatalf("%s %s uid=%s: got %d want %d", c.m, c.p, c.uid, w.Code, c.want)
		}
	}
}
