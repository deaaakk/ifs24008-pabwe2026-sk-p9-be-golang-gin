package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"app/internal/config"
	"app/internal/database"

	"github.com/gin-gonic/gin"
)

type M = map[string]any

func call(r *gin.Engine, m, p, tok string, body any) (int, M) {
	var rd *strings.Reader
	switch v := body.(type) {
	case nil:
		rd = strings.NewReader("")
	case string:
		rd = strings.NewReader(v)
	default:
		b, _ := json.Marshal(v)
		rd = strings.NewReader(string(b))
	}
	req := httptest.NewRequest(m, p, rd)
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var out M
	json.NewDecoder(bytes.NewReader(w.Body.Bytes())).Decode(&out)
	return w.Code, out
}

func TestAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{DBConn: "sqlite", Key: "k", AdminUser: "admin", AdminPass: "admin"}
	db, _ := database.Connect(cfg)
	database.Migrate(db, cfg)
	r := New(db, cfg)
	eq := func(got, want int, msg string) {
		t.Helper()
		if got != want {
			t.Fatalf("%s: got %d want %d", msg, got, want)
		}
	}
	login := func(u string) string {
		c, o := call(r, "POST", "/api/auth/login", "", M{"username": u, "password": map[string]string{"admin": "admin"}[u] + map[string]string{"budi": "rahasia"}[u]})
		eq(c, 200, "login "+u)
		return o["data"].(M)["token"].(string)
	}
	// auth
	c, o := call(r, "POST", "/api/auth/register", "", M{"username": "budi", "email": "b@x.id", "password": "rahasia"})
	eq(c, 201, "register")
	if d, ok := o["data"].(M); !ok || d["token"] == nil || d["token"] == "" {
		t.Fatal("register harus mengembalikan token")
	}
	c, _ = call(r, "POST", "/api/auth/register", "", M{"username": "budi", "email": "b@x.id", "password": "rahasia"})
	eq(c, 400, "register dup")
	c, _ = call(r, "POST", "/api/auth/register", "", M{"username": "x1", "password": "rahasia"})
	eq(c, 400, "register tanpa email")
	c, _ = call(r, "POST", "/api/auth/register", "", M{"username": "x1", "email": "x@x.id", "password": "12"})
	eq(c, 400, "register password pendek")
	c, _ = call(r, "POST", "/api/auth/register", "", M{"username": "x1", "email": "salah", "password": "rahasia"})
	eq(c, 400, "register email salah")
	c, _ = call(r, "POST", "/api/auth/login", "", M{"username": "b@x.id", "password": "rahasia"})
	eq(c, 200, "login pakai email")
	c, _ = call(r, "POST", "/api/auth/register", "", "{")
	eq(c, 400, "register bad")
	c, _ = call(r, "POST", "/api/auth/login", "", "{")
	eq(c, 400, "login bad")
	c, _ = call(r, "POST", "/api/auth/login", "", M{"username": "budi", "password": "salah"})
	eq(c, 401, "login salah")
	tb, ta := login("budi"), login("admin")
	c, _ = call(r, "GET", "/api/auth/profile", tb, nil)
	eq(c, 200, "profile")
	c, _ = call(r, "PUT", "/api/auth/profile", tb, M{"username": "budi", "email": "b@x.id", "password": "baru123"})
	eq(c, 200, "profile update")
	c, _ = call(r, "PUT", "/api/auth/profile", tb, "{")
	eq(c, 400, "profile bad")
	c, _ = call(r, "PUT", "/api/auth/profile", tb, M{"username": "admin", "email": "b@x.id"})
	eq(c, 400, "profile conflict")
	c, _ = call(r, "POST", "/api/auth/logout", tb, nil)
	eq(c, 200, "logout")
	c, _ = call(r, "GET", "/api/posts", "", nil)
	eq(c, 401, "tanpa token")
	// posts
	c, _ = call(r, "POST", "/api/posts", tb, M{})
	eq(c, 400, "post bad")
	c, o = call(r, "POST", "/api/posts", tb, M{"title": "Halo Dunia", "content": "isi", "is_published": true})
	eq(c, 201, "post create")
	id := "/api/posts/" + json.Number(strings.TrimSuffix(strings.TrimSpace(
		string(mustJSON(o["data"].(M)["id"]))), ".0")).String()
	c, _ = call(r, "POST", "/api/posts", tb, M{"title": "Halo Dunia", "content": "isi", "category": "Opini"})
	eq(c, 201, "slug unik")
	c, _ = call(r, "GET", "/api/posts", tb, nil)
	eq(c, 200, "mine")
	c, o = call(r, "GET", "/api/posts/stats", tb, nil)
	eq(c, 200, "stats")
	if d, _ := o["data"].(M); d["total_articles"] != float64(2) || d["total_selesai"] != float64(1) || d["belum_selesai"] != float64(1) {
		t.Fatal("stats salah:", o["data"])
	}
	c, _ = call(r, "GET", "/api/posts/abc", tb, nil)
	eq(c, 400, "id bukan angka")
	c, _ = call(r, "GET", "/api/posts/1", "", nil)
	eq(c, 401, "get tanpa token")
	c, o2 := call(r, "POST", "/api/posts", ta, M{"title": "Milik Admin", "content": "isi"})
	eq(c, 201, "post admin")
	ida := "/api/posts/" + strconv.Itoa(int(o2["data"].(M)["id"].(float64)))
	c, _ = call(r, "GET", ida, tb, nil)
	eq(c, 403, "get milik orang lain")
	c, _ = call(r, "PUT", ida, tb, M{"title": "x", "content": "y"})
	eq(c, 403, "put milik orang lain")
	c, _ = call(r, "PATCH", ida, tb, M{"title": "x"})
	eq(c, 403, "patch milik orang lain")
	c, _ = call(r, "DELETE", ida, tb, nil)
	eq(c, 403, "delete milik orang lain")
	c, _ = call(r, "GET", id, tb, nil)
	eq(c, 200, "get")
	c, _ = call(r, "GET", "/api/posts/999", tb, nil)
	eq(c, 404, "get 404")
	c, _ = call(r, "PUT", id, tb, M{"title": "Judul Baru", "content": "x", "is_published": true})
	eq(c, 200, "update")
	c, _ = call(r, "PUT", id, tb, M{"title": "Judul Baru", "content": "x", "is_published": true})
	eq(c, 200, "update sama")
	c, _ = call(r, "PUT", id, tb, "{")
	eq(c, 400, "update bad")
	c, _ = call(r, "PATCH", id+"/toggle-publish", tb, nil)
	eq(c, 200, "toggle")
	c, _ = call(r, "PATCH", id+"/toggle-publish", tb, nil)
	eq(c, 200, "toggle lagi")
	c, _ = call(r, "PATCH", "/api/posts/999/toggle-publish", tb, nil)
	eq(c, 404, "toggle 404")
	// publik
	c, _ = call(r, "GET", "/api/public/categories", "", nil)
	eq(c, 200, "categories")
	c, _ = call(r, "GET", "/api/public/posts?q=halo&category=Umum", "", nil)
	eq(c, 200, "public list")
	c, o = call(r, "GET", "/api/public/posts", "", nil)
	eq(c, 200, "public list semua")
	arr, _ := o["data"].([]any)
	if len(arr) == 0 {
		t.Fatal("public list harus array berisi")
	}
	for _, k := range []string{"user_id", "is_published"} {
		if _, ada := arr[0].(M)[k]; ada {
			t.Fatal("data internal bocor:", k)
		}
	}
	if au, _ := arr[0].(M)["author"].(M); au["email"] != nil || au["role"] != nil {
		t.Fatal("email/role penulis bocor")
	}
	c, _ = call(r, "GET", "/api/public/posts/judul-baru", "", nil)
	eq(c, 200, "public detail")
	c, _ = call(r, "GET", "/api/public/posts/tidak-ada", "", nil)
	eq(c, 404, "public 404")
	// admin
	c, _ = call(r, "GET", "/api/admin/users", tb, nil)
	eq(c, 403, "admin member")
	for _, p := range []string{"/api/admin/users", "/api/admin/posts", "/api/admin/users/2/posts"} {
		c, _ = call(r, "GET", p, ta, nil)
		eq(c, 200, p)
	}
	c, o = call(r, "GET", "/api/admin/users/9999/posts", ta, nil)
	eq(c, 200, "admin posts user tidak ada")
	if l, ok := o["data"].([]any); !ok || len(l) != 0 {
		t.Fatal("harus daftar kosong []")
	}
	c, _ = call(r, "GET", "/api/admin/users/abc/posts", ta, nil)
	eq(c, 400, "admin id bukan angka")
	c, _ = call(r, "GET", "/api/admin/posts", "", nil)
	eq(c, 401, "admin tanpa token")
	// ai tanpa topik, swagger, 404, cors
	c, _ = call(r, "POST", "/api/posts/ai-generate", tb, M{})
	eq(c, 400, "ai bad")
	c, _ = call(r, "GET", "/api.json", "", nil)
	eq(c, 200, "api.json")
	c, _ = call(r, "GET", "/swagger/index.html", "", nil)
	eq(c, 200, "swagger")
	c, _ = call(r, "GET", "/nope", "", nil)
	eq(c, 404, "noroute")
	c, _ = call(r, http.MethodOptions, "/api/posts", "", nil)
	eq(c, 204, "cors")
	c, _ = call(r, "DELETE", id, tb, nil)
	eq(c, 200, "delete")
	c, _ = call(r, "DELETE", id, tb, nil)
	eq(c, 404, "delete 404")
	c, o = call(r, "GET", "/api/posts/stats", tb, nil)
	if d, _ := o["data"].(M); d["total_articles"] != float64(1) {
		t.Fatal("stats harus berkurang setelah delete:", o["data"])
	}
}

func mustJSON(v any) []byte { b, _ := json.Marshal(v); return b }

