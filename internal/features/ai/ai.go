package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"app/internal/config"
	"app/internal/response"

	"github.com/gin-gonic/gin"
)

type GenerateReq struct {
	Topic            string `json:"topic" binding:"required"`
	Category         string `json:"category"`
	IncludeWebSearch bool   `json:"include_web_search"`
}
type Source struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}
type Article struct {
	Title    string   `json:"title"`
	Category string   `json:"category"`
	Excerpt  string   `json:"excerpt"`
	Content  string   `json:"content"`
	Sources  []Source `json:"sources"`
}

type Service struct {
	Cfg  *config.Config
	HTTP *http.Client
}

func NewService(c *config.Config) *Service {
	return &Service{Cfg: c, HTTP: &http.Client{Timeout: 90 * time.Second}}
}

// Search: query Search Engine -> snippets & referensi web
func (s *Service) Search(q string) ([]Source, error) {
	res, err := s.HTTP.Get(s.Cfg.SearchEndpoint + "?q=" + url.QueryEscape(q))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	var raw struct {
		Data    json.RawMessage `json:"data"`
		Results []Source        `json:"results"`
	}
	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		return nil, err
	}
	if len(raw.Results) == 0 && len(raw.Data) > 0 {
		json.Unmarshal(raw.Data, &raw.Results)
	}
	return raw.Results, nil
}

// Chat: kirim prompt ke OpenAI Chat Completions
func (s *Service) Chat(prompt string) (string, error) {
	b, _ := json.Marshal(map[string]any{"model": s.Cfg.OpenAIModel, "messages": []map[string]string{
		{"role": "system", "content": "Kamu penulis blog. Balas HANYA JSON valid: {title,category,excerpt,content(markdown)}."},
		{"role": "user", "content": prompt}}})
	req, _ := http.NewRequest("POST", strings.TrimRight(s.Cfg.OpenAIEndpoint, "/")+"/chat/completions", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.Cfg.OpenAIKey)
	res, err := s.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	var out struct {
		Choices []struct {
			Message struct{ Content string } `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &out); err != nil || len(out.Choices) == 0 {
		return "", fmt.Errorf("respons AI tidak valid")
	}
	return out.Choices[0].Message.Content, nil
}

func (s *Service) Generate(q GenerateReq) (*Article, error) {
	prompt := fmt.Sprintf("Tulis artikel blog lengkap (Markdown, bahasa Indonesia) tentang: %s. Kategori: %s.", q.Topic, q.Category)
	var srcs []Source
	if q.IncludeWebSearch {
		var err error
		if srcs, err = s.Search(q.Topic); err != nil {
			return nil, err
		}
		prompt += "\nGunakan konteks hasil pencarian berikut:\n"
		for i, x := range srcs {
			prompt += fmt.Sprintf("[%d] %s (%s): %s\n", i+1, x.Title, x.URL, x.Snippet)
		}
	}
	txt, err := s.Chat(prompt)
	if err != nil {
		return nil, err
	}
	if i, j := strings.Index(txt, "{"), strings.LastIndex(txt, "}"); i >= 0 && j > i {
		txt = txt[i : j+1]
	}
	var a Article
	if err := json.Unmarshal([]byte(txt), &a); err != nil {
		return nil, fmt.Errorf("format JSON AI tidak valid")
	}
	if a.Category == "" {
		a.Category = q.Category
	}
	a.Sources = srcs
	return &a, nil
}

type Handler struct{ S *Service }

func (h *Handler) Generate(c *gin.Context) {
	var q GenerateReq
	if c.ShouldBindJSON(&q) != nil || strings.TrimSpace(q.Topic) == "" {
		response.Fail(c, http.StatusBadRequest, "Topik wajib diisi")
		return
	}
	a, err := h.S.Generate(q)
	if err != nil {
		response.Error(c, http.StatusBadGateway, err.Error())
		return
	}
	response.Success(c, 200, "Draf artikel berhasil dibuat", a)
}
