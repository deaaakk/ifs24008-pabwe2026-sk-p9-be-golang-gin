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

// UpdateReq dipakai PUT/PATCH: hanya field yang dikirim yang diubah.
type UpdateReq struct {
	Title       *string `json:"title"`
	Category    *string `json:"category"`
	Excerpt     *string `json:"excerpt"`
	Content     *string `json:"content"`
	CoverImage  *string `json:"cover_image_path"`
	IsPublished *bool   `json:"is_published"`
}

// Stats statistik artikel milik penulis.
// total_selesai = artikel terbit, belum_selesai = artikel draf.
type Stats struct {
	TotalArticles int64 `json:"total_articles"`
	TotalSelesai  int64 `json:"total_selesai"`
	BelumSelesai  int64 `json:"belum_selesai"`
	Total         int64 `json:"total"`
	Published     int64 `json:"published"`
	Draft         int64 `json:"draft"`
}

// PublicAuthor & PublicPost: representasi publik tanpa data internal
// (tanpa email, role, user_id, status publikasi).
type PublicAuthor struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
}
type PublicPost struct {
	ID          uint         `json:"id"`
	Title       string       `json:"title"`
	Slug        string       `json:"slug"`
	Category    string       `json:"category"`
	Excerpt     string       `json:"excerpt"`
	Content     string       `json:"content"`
	CoverImage  string       `json:"cover_image_path"`
	PublishedAt *time.Time   `json:"published_at"`
	CreatedAt   time.Time    `json:"created_at"`
	ReadingTime int          `json:"reading_time"`
	AuthorName  string       `json:"author_name"`
	Author      PublicAuthor `json:"author"`
}

// UserRow: pengguna + statistik artikelnya (untuk panel admin).
type UserRow struct {
	ID             uint      `json:"id"`
	Username       string    `json:"username"`
	Email          string    `json:"email"`
	Role           string    `json:"role"`
	CreatedAt      time.Time `json:"created_at"`
	TotalArticles  int64     `json:"total_articles"`
	TotalSelesai   int64     `json:"total_selesai"`
	BelumSelesai   int64     `json:"belum_selesai"`
	TotalPosts     int64     `json:"total_posts"`
	TotalPublished int64     `json:"total_published"`
}

type userCount struct {
	UserID    uint
	Total     int64
	Published int64
}

type Repository struct{ DB *gorm.DB }

func (r *Repository) Public(q, cat string) ([]database.Post, error) {
	p := []database.Post{}
	t := r.DB.Preload("User").Where("is_published = ?", true)
	if q != "" {
		l := "%" + strings.ToLower(q) + "%"
		t = t.Where("LOWER(title) LIKE ? OR LOWER(excerpt) LIKE ? OR LOWER(content) LIKE ?", l, l, l)
	}
	if cat != "" {
		t = t.Where("category = ?", cat)
	}
	err := t.Order("published_at DESC").Find(&p).Error
	return p, err
}
func (r *Repository) BySlug(s string) (database.Post, error) {
	var p database.Post
	err := r.DB.Preload("User").Where("slug = ? AND is_published = ?", s, true).First(&p).Error
	return p, err
}
func (r *Repository) Mine(uid uint) ([]database.Post, error) {
	p := []database.Post{}
	err := r.DB.Where("user_id = ?", uid).Order("created_at DESC").Find(&p).Error
	return p, err
}
func (r *Repository) ByID(id uint) (database.Post, error) {
	var p database.Post
	err := r.DB.First(&p, id).Error
	return p, err
}

// All: semua artikel; bila uid != nil hanya milik user tersebut.
func (r *Repository) All(uid *uint) ([]database.Post, error) {
	p := []database.Post{}
	t := r.DB.Preload("User")
	if uid != nil {
		t = t.Where("user_id = ?", *uid)
	}
	err := t.Order("created_at DESC").Find(&p).Error
	return p, err
}
func (r *Repository) Users() ([]UserRow, error) {
	var us []database.User
	if err := r.DB.Order("id").Find(&us).Error; err != nil {
		return nil, err
	}
	var cs []userCount
	if err := r.DB.Model(&database.Post{}).
		Select("user_id, COUNT(*) AS total, COUNT(CASE WHEN is_published = ? THEN 1 END) AS published", true).
		Group("user_id").Scan(&cs).Error; err != nil {
		return nil, err
	}
	type cnt struct{ total, published int64 }
	m := map[uint]cnt{}
	for _, c := range cs {
		m[c.UserID] = cnt{c.Total, c.Published}
	}
	rows := make([]UserRow, 0, len(us))
	for _, u := range us {
		c := m[u.ID]
		rows = append(rows, UserRow{
			ID: u.ID, Username: u.Username, Email: u.Email, Role: u.Role, CreatedAt: u.CreatedAt,
			TotalArticles: c.total, TotalSelesai: c.published, BelumSelesai: c.total - c.published,
			TotalPosts: c.total, TotalPublished: c.published,
		})
	}
	return rows, nil
}
func (r *Repository) SlugExists(s string) bool {
	var n int64
	r.DB.Model(&database.Post{}).Where("slug = ?", s).Count(&n)
	return n > 0
}
func (r *Repository) Stats(uid uint) Stats {
	var total, pub int64
	r.DB.Model(&database.Post{}).Where("user_id = ?", uid).Count(&total)
	r.DB.Model(&database.Post{}).Where("user_id = ? AND is_published = ?", uid, true).Count(&pub)
	return Stats{
		TotalArticles: total, TotalSelesai: pub, BelumSelesai: total - pub,
		Total: total, Published: pub, Draft: total - pub,
	}
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
func fill(p *database.Post) {
	p.ReadingTime = ReadingTime(p.Content)
	p.AuthorName = p.User.Username
}
func toPublic(p database.Post) PublicPost {
	fill(&p)
	return PublicPost{
		ID: p.ID, Title: p.Title, Slug: p.Slug, Category: p.Category, Excerpt: p.Excerpt,
		Content: p.Content, CoverImage: p.CoverImage, PublishedAt: p.PublishedAt, CreatedAt: p.CreatedAt,
		ReadingTime: p.ReadingTime, AuthorName: p.User.Username,
		Author: PublicAuthor{ID: p.User.ID, Username: p.User.Username},
	}
}
func (s *Service) Create(uid uint, q PostReq) (*database.Post, error) {
	p := &database.Post{UserID: uid, Title: q.Title, Slug: s.UniqueSlug(q.Title), Category: orGeneral(q.Category),
		Excerpt: q.Excerpt, Content: q.Content, CoverImage: q.CoverImage}
	setPublish(p, q.IsPublished)
	err := s.Repo.DB.Create(p).Error
	fill(p)
	return p, err
}
func orGeneral(c string) string {
	if strings.TrimSpace(c) == "" {
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
	out := make([]PublicPost, 0, len(p))
	for _, x := range p {
		out = append(out, toPublic(x))
	}
	response.Success(c, 200, "OK", out)
}
func (h *Handler) PublicDetail(c *gin.Context) {
	p, err := h.S.Repo.BySlug(c.Param("slug"))
	if err != nil {
		response.Fail(c, 404, "Artikel tidak ditemukan")
		return
	}
	response.Success(c, 200, "OK", toPublic(p))
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

// own: ID bukan angka -> 400, tidak ada -> 404, milik orang lain -> 403.
func (h *Handler) own(c *gin.Context) (database.Post, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "ID artikel tidak valid")
		return database.Post{}, false
	}
	p, err := h.S.Repo.ByID(uint(id))
	if err != nil {
		response.Fail(c, http.StatusNotFound, "Artikel tidak ditemukan")
		return p, false
	}
	if p.UserID != c.GetUint("userID") {
		response.Fail(c, http.StatusForbidden, "Anda tidak berhak mengakses artikel ini")
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
	var q UpdateReq
	if c.ShouldBindJSON(&q) != nil ||
		(q.Title != nil && strings.TrimSpace(*q.Title) == "") ||
		(q.Content != nil && strings.TrimSpace(*q.Content) == "") {
		response.Fail(c, 400, "Data tidak valid")
		return
	}
	if q.Title != nil && *q.Title != p.Title {
		p.Title = *q.Title
		p.Slug = h.S.UniqueSlug(p.Title)
	}
	if q.Category != nil {
		p.Category = orGeneral(*q.Category)
	}
	if q.Excerpt != nil {
		p.Excerpt = *q.Excerpt
	}
	if q.Content != nil {
		p.Content = *q.Content
	}
	if q.CoverImage != nil {
		p.CoverImage = *q.CoverImage
	}
	if q.IsPublished != nil {
		setPublish(&p, *q.IsPublished)
	}
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
	if err := h.S.Repo.DB.Save(&p).Error; err != nil {
		response.Error(c, 500, err.Error())
		return
	}
	fill(&p)
	response.Success(c, 200, "Status publikasi diubah", p)
}
func (h *Handler) Delete(c *gin.Context) {
	p, ok := h.own(c)
	if !ok {
		return
	}
	if err := h.S.Repo.DB.Delete(&p).Error; err != nil {
		response.Error(c, 500, err.Error())
		return
	}
	response.Success(c, 200, "Artikel dihapus", nil)
}
func (h *Handler) AdminUsers(c *gin.Context) {
	u, err := h.S.Repo.Users()
	if err != nil {
		response.Error(c, 500, err.Error())
		return
	}
	response.Success(c, 200, "OK", u)
}

// AdminPosts: semua artikel, atau (route /users/:id/posts) milik satu user.
// ID bukan angka -> 400; user tidak ada -> daftar kosong.
func (h *Handler) AdminPosts(c *gin.Context) {
	var uid *uint
	if s := c.Param("id"); s != "" {
		n, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			response.Fail(c, http.StatusBadRequest, "ID pengguna tidak valid")
			return
		}
		v := uint(n)
		uid = &v
	}
	p, err := h.S.Repo.All(uid)
	if err != nil {
		response.Error(c, 500, err.Error())
		return
	}
	for i := range p {
		fill(&p[i])
	}
	response.Success(c, 200, "OK", p)
}
