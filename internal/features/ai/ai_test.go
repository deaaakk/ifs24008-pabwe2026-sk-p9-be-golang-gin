package ai

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"app/internal/config"

	"github.com/gin-gonic/gin"
)

const art = "```json\n{\"title\":\"T\",\"excerpt\":\"e\",\"content\":\"c\"}\n```"

func srv(body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(body)) }))
}
func chat(content string) string {
	return `{"choices":[{"message":{"content":` + quote(content) + `}}]}`
}
func quote(s string) string {
	return `"` + strings.NewReplacer("\\", "\\\\", "\"", "\\\"", "\n", "\\n").Replace(s) + `"`
}

func TestGenerate(t *testing.T) {
	sr := srv(`{"results":[{"title":"t","url":"u","snippet":"s"}]}`)
	sd := srv(`{"data":[{"title":"t","url":"u","snippet":"s"}]}`)
	sj := srv(`bukan json`)
	ok := srv(chat(art))
	withCat := srv(chat(`{"title":"T","category":"Bisnis"}`))
	none := srv(`{"choices":[]}`)
	badArt := srv(chat("bukan json"))
	for _, s := range []*httptest.Server{sr, sd, sj, ok, withCat, none, badArt} {
		defer s.Close()
	}
	dead := "http://127.0.0.1:1"
	mk := func(search, oa string) *Service {
		return NewService(&config.Config{SearchEndpoint: search, OpenAIEndpoint: oa, OpenAIModel: "m"})
	}
	if a, err := mk(sr.URL, ok.URL).Generate(GenerateReq{Topic: "x", Category: "Umum", IncludeWebSearch: true}); err != nil || len(a.Sources) != 1 || a.Category != "Umum" {
		t.Fatal("generate+search", err)
	}
	if a, err := mk(sd.URL, ok.URL).Generate(GenerateReq{Topic: "x", IncludeWebSearch: true}); err != nil || len(a.Sources) != 1 {
		t.Fatal("format data", err)
	}
	if a, err := mk("", withCat.URL).Generate(GenerateReq{Topic: "x"}); err != nil || a.Category != "Bisnis" {
		t.Fatal("tanpa search", err)
	}
	for name, s := range map[string]*Service{
		"search mati":  mk(dead, ok.URL),
		"search bukan json": mk(sj.URL, ok.URL),
		"ai mati":      mk(sr.URL, dead),
		"ai kosong":    mk(sr.URL, none.URL),
		"artikel rusak": mk(sr.URL, badArt.URL),
	} {
		if _, err := s.Generate(GenerateReq{Topic: "x", IncludeWebSearch: true}); err == nil {
			t.Fatal("harusnya error:", name)
		}
	}
}

func TestHandler(t *testing.T) {
	ok := srv(chat(art))
	defer ok.Close()
	r := gin.New()
	r.POST("/ok", (&Handler{S: NewService(&config.Config{OpenAIEndpoint: ok.URL})}).Generate)
	r.POST("/bad", (&Handler{S: NewService(&config.Config{OpenAIEndpoint: "http://127.0.0.1:1"})}).Generate)
	for _, c := range []struct{ p, b string; want int }{{"/ok", `{}`, 400}, {"/ok", `{"topic":"x"}`, 200}, {"/bad", `{"topic":"x"}`, 502}} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("POST", c.p, strings.NewReader(c.b)))
		if w.Code != c.want {
			t.Fatal(c.p, w.Code)
		}
	}
}
