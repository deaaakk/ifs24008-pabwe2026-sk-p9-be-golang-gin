package post

import (
	"errors"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"app/internal/config"
	"app/internal/database"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func TestHelpers(t *testing.T) {
	if Slugify("Halo Dunia!") != "halo-dunia" || Slugify("!!!") != "post" {
		t.Fatal("slugify")
	}
	if ReadingTime("") != 1 || ReadingTime(strings.Repeat("a ", 401)) != 3 {
		t.Fatal("reading time")
	}
}

func routes(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := &Handler{S: &Service{Repo: &Repository{DB: db}}}
	r := gin.New()
	r.Use(func(c *gin.Context) {
		id, _ := strconv.Atoi(c.GetHeader("X-Uid"))
		c.Set("userID", uint(id))
	})
	r.GET("/cat", h.Categories)
	r.GET("/pub", h.PublicList)
	r.GET("/pub/:slug", h.PublicDetail)
	r.GET("/mine", h.Mine)
	r.POST("/mine", h.Create)
	r.GET("/stats", h.Stats)
	r.GET("/p/:id", h.Get)
	r.PUT("/p/:id", h.Update)
	r.PATCH("/p/:id/t", h.Toggle)
	r.DELETE("/p/:id", h.Delete)
	r.GET("/au", h.AdminUsers)
	r.GET("/ap", h.AdminPosts)
	r.GET("/ap/:id", h.AdminPosts)
	return r
}

type tc struct {
	m, p, body string
	want       int
}

func run(t *testing.T, r *gin.Engine, cases []tc) {
	t.Helper()
	for _, c := range cases {
		req := httptest.NewRequest(c.m, c.p, strings.NewReader(c.body))
		req.Header.Set("X-Uid", "2")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != c.want {
			t.Fatalf("%s %s: got %d want %d", c.m, c.p, w.Code, c.want)
		}
	}
}

func newDB() *gorm.DB {
	cfg := &config.Config{DBConn: "sqlite", AdminUser: "admin", AdminPass: "admin"}
	db, _ := database.Connect(cfg)
	database.Migrate(db, cfg)
	return db
}

func TestPostFlow(t *testing.T) {
	db := newDB()
	db.Create(&database.User{Username: "budi", Email: "b@x.id", Password: "x", Role: "member"})
	fail := false
	db.Callback().Update().Before("gorm:update").Register("t:fail", func(d *gorm.DB) {
		if fail {
			d.AddError(errors.New("x"))
		}
	})
	r := routes(db)
	run(t, r, []tc{
		{"GET", "/cat", "", 200},
		{"POST", "/mine", "{", 400},
		{"POST", "/mine", `{"title":"Halo","content":"isi","is_published":true}`, 201},
		{"POST", "/mine", `{"title":"Halo","content":"isi"}`, 201},
		{"POST", "/mine", `{"title":"Opini Saya","content":"isi","category":"Opini"}`, 201},
		{"GET", "/mine", "", 200},
		{"GET", "/stats", "", 200},
		{"GET", "/p/1", "", 200},
		{"GET", "/p/99", "", 404},
		{"PUT", "/p/1", "{", 400},
		{"PUT", "/p/99", `{"title":"a","content":"b"}`, 404},
		{"PUT", "/p/1", `{"title":"Judul Baru","content":"x","is_published":true}`, 200},
		{"PUT", "/p/1", `{"title":"Judul Baru","content":"x"}`, 200},
		{"PATCH", "/p/1/t", "", 200},
		{"PATCH", "/p/99/t", "", 404},
		{"GET", "/pub", "", 200},
		{"GET", "/pub?q=judul&category=Umum", "", 200},
		{"GET", "/pub/judul-baru", "", 200},
		{"GET", "/pub/tidak-ada", "", 404},
		{"GET", "/au", "", 200},
		{"GET", "/ap", "", 200},
		{"GET", "/ap/2", "", 200},
	})
	fail = true
	run(t, r, []tc{{"PUT", "/p/1", `{"title":"Judul Baru","content":"x"}`, 500}})
	fail = false
	run(t, r, []tc{{"DELETE", "/p/1", "", 200}, {"DELETE", "/p/1", "", 404}})
}

func TestDBError(t *testing.T) {
	db := newDB()
	s, _ := db.DB()
	s.Close()
	run(t, routes(db), []tc{
		{"GET", "/pub", "", 500},
		{"GET", "/mine", "", 500},
		{"POST", "/mine", `{"title":"a","content":"b"}`, 500},
		{"GET", "/au", "", 500},
	})
}
