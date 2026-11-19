package post

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"app/internal/database"
	"app/internal/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var Categories = []string{"Teknologi", "Pendidikan", "Gaya Hidup", "Bisnis", "Opini", "Umum"}

type PostReq struct {
	Title       string `json:"title" binding:"required"`
	Category    string `json:"category"`
	Excerpt     string `json:"excerpt"`
	Content     string `json:"content" binding:"required"`
	CoverImage  string `json:"cover_image_path"`
	IsPublished bool   `json:"is_published"`
}
type Stats struct {
	Total     int64 `json:"total"`
	Published int64 `json:"published"`
	Draft     int64 `json:"draft"`
}

type Repository struct{ DB *gorm.DB }

func (r *Repository) Public(q, cat string) (p []database.Post, err error) {
	t := r.DB.Preload("User").Where("is_published = ?", true)
	if q != "" {
		l := "%" + strings.ToLower(q) + "%"
		t = t.Where("LOWER(title) LIKE ? OR LOWER(excerpt) LIKE ? OR LOWER(content) LIKE ?", l, l, l)
	}
	if cat != "" {
		t = t.Where("category = ?", cat)
	}
	return p, t.Order("published_at DESC").Find(&p).Error
}
func (r *Repository) BySlug(s string) (p database.Post, err error) {
	return p, r.DB.Preload("User").Where("slug = ? AND is_published = ?", s, true).First(&p).Error
}
func (r *Repository) Mine(uid uint) (p []database.Post, err error) {
	return p, r.DB.Where("user_id = ?", uid).Order("created_at DESC").Find(&p).Error
}
func (r *Repository) Own(id, uid uint) (p database.Post, err error) {
	return p, r.DB.Where("id = ? AND user_id = ?", id, uid).First(&p).Error
}
func (r *Repository) All(uid string) (p []database.Post, err error) {
	t := r.DB.Preload("User")
	if uid != "" {
		t = t.Where("user_id = ?", uid)
	}
	return p, t.Order("created_at DESC").Find(&p).Error
}
func (r *Repository) Users() (u []map[string]any, err error) {
	return u, r.DB.Raw(`SELECT u.id,u.username,u.email,u.role,(SELECT COUNT(*) FROM posts p WHERE p.user_id=u.id) AS total_posts,
	(SELECT COUNT(*) FROM posts p WHERE p.user_id=u.id AND p.is_published=true) AS total_published FROM users u ORDER BY u.id`).Scan(&u).Error
}
func (r *Repository) SlugExists(s string) bool {
	var n int64
	r.DB.Model(&database.Post{}).Where("slug = ?", s).Count(&n)
	return n > 0
}
func (r *Repository) Stats(uid uint) (s Stats) {
	m := r.DB.Model(&database.Post{}).Where("user_id = ?", uid)
	m.Count(&s.Total)
	r.DB.Model(&database.Post{}).Where("user_id = ? AND is_published = ?", uid, true).Count(&s.Published)
	s.Draft = s.Total - s.Published
	return
}

type Service struct{ Repo *Repository }

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

func Slugify(t string) string {
	s := strings.Trim(nonSlug.ReplaceAllString(strings.ToLower(t), "-"), "-")
	if s == "" {
		return "post"
	}
	return s
}
func ReadingTime(content string) int {
	m := (len(strings.Fields(content)) + 199) / 200
	if m < 1 {
		return 1
	}
	return m
}
func (s *Service) UniqueSlug(title string) string {
	base := Slugify(title)
	slug := base
	for i := 2; s.Repo.SlugExists(slug); i++ {
		slug = fmt.Sprintf("%s-%d", base, i)
	}
	return slug
}
func fill(p *database.Post) { p.ReadingTime = ReadingTime(p.Content) }
func (s *Service) Create(uid uint, q PostReq) (*database.Post, error) {
	p := &database.Post{UserID: uid, Title: q.Title, Slug: s.UniqueSlug(q.Title), Category: orGeneral(q.Category),
		Excerpt: q.Excerpt, Content: q.Content, CoverImage: q.CoverImage}
	setPublish(p, q.IsPublished)
	err := s.Repo.DB.Create(p).Error
	fill(p)
	return p, err
}
func orGeneral(c string) string {
	if c == "" {
		return "Umum"
	}
	return c
}
func setPublish(p *database.Post, v bool) {
	p.IsPublished = v
	if v && p.PublishedAt == nil {
		n := time.Now()
		p.PublishedAt = &n
	}
	if !v {
		p.PublishedAt = nil
	}
}

type Handler struct{ S *Service }

func (h *Handler) Categories(c *gin.Context) { response.Success(c, 200, "OK", Categories) }
func (h *Handler) PublicList(c *gin.Context) {
	p, err := h.S.Repo.Public(c.Query("q"), c.Query("category"))
	if err != nil {
		response.Error(c, 500, err.Error())
		return
	}
	for i := range p {
		fill(&p[i])
	}
	response.Success(c, 200, "OK", p)
}
func (h *Handler) PublicDetail(c *gin.Context) {
	p, err := h.S.Repo.BySlug(c.Param("slug"))
	if err != nil {
		response.Fail(c, 404, "Artikel tidak ditemukan")
		return
	}
	fill(&p)
	response.Success(c, 200, "OK", p)
}
func (h *Handler) Mine(c *gin.Context) {
	p, err := h.S.Repo.Mine(c.GetUint("userID"))
	if err != nil {
		response.Error(c, 500, err.Error())
		return
	}
	for i := range p {
		fill(&p[i])
	}
	response.Success(c, 200, "OK", p)
}
func (h *Handler) Stats(c *gin.Context) {
	response.Success(c, 200, "OK", h.S.Repo.Stats(c.GetUint("userID")))
}
func (h *Handler) own(c *gin.Context) (database.Post, bool) {
	id, _ := strconv.Atoi(c.Param("id"))
	p, err := h.S.Repo.Own(uint(id), c.GetUint("userID"))
	if err != nil {
		response.Fail(c, 404, "Artikel tidak ditemukan")
		return p, false
	}
	return p, true
}
func (h *Handler) Get(c *gin.Context) {
	if p, ok := h.own(c); ok {
		fill(&p)
		response.Success(c, 200, "OK", p)
	}
}
func (h *Handler) Create(c *gin.Context) {
	var q PostReq
	if c.ShouldBindJSON(&q) != nil {
		response.Fail(c, http.StatusBadRequest, "Judul dan konten wajib diisi")
		return
	}
	p, err := h.S.Create(c.GetUint("userID"), q)
	if err != nil {
		response.Error(c, 500, err.Error())
		return
	}
	response.Success(c, 201, "Artikel dibuat", p)
}
func (h *Handler) Update(c *gin.Context) {
	p, ok := h.own(c)
	if !ok {
		return
	}
	var q PostReq
	if c.ShouldBindJSON(&q) != nil {
		response.Fail(c, 400, "Data tidak valid")
		return
	}
	if q.Title != p.Title {
		p.Slug = h.S.UniqueSlug(q.Title)
	}
	p.Title, p.Category, p.Excerpt, p.Content, p.CoverImage = q.Title, orGeneral(q.Category), q.Excerpt, q.Content, q.CoverImage
	setPublish(&p, q.IsPublished)
	if err := h.S.Repo.DB.Save(&p).Error; err != nil {
		response.Error(c, 500, err.Error())
		return
	}
	fill(&p)
	response.Success(c, 200, "Artikel diperbarui", p)
}
func (h *Handler) Toggle(c *gin.Context) {
	p, ok := h.own(c)
	if !ok {
		return
	}
	setPublish(&p, !p.IsPublished)
	h.S.Repo.DB.Save(&p)
	response.Success(c, 200, "Status publikasi diubah", p)
}
func (h *Handler) Delete(c *gin.Context) {
	if p, ok := h.own(c); ok {
		h.S.Repo.DB.Delete(&p)
		response.Success(c, 200, "Artikel dihapus", nil)
	}
}
func (h *Handler) AdminUsers(c *gin.Context) {
	u, err := h.S.Repo.Users()
	if err != nil {
		response.Error(c, 500, err.Error())
		return
	}
	response.Success(c, 200, "OK", u)
}
func (h *Handler) AdminPosts(c *gin.Context) {
	p, _ := h.S.Repo.All(c.Param("id"))
	for i := range p {
		fill(&p[i])
	}
	response.Success(c, 200, "OK", p)
}
