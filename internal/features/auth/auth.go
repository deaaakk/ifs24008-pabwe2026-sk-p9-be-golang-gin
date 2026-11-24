package auth

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"app/internal/database"
	"app/internal/response"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// ErrDuplicate dikembalikan bila username/email sudah dipakai.
var ErrDuplicate = errors.New("username atau email sudah terdaftar")

type RegisterReq struct {
	Username string `json:"username" binding:"required,min=3"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// LoginReq menerima username ATAU email sebagai identitas.
type LoginReq struct {
	Username   string `json:"username"`
	Email      string `json:"email"`
	Identifier string `json:"identifier"`
	Password   string `json:"password" binding:"required"`
}

// Identity mengembalikan identitas login yang diisi (username / email).
func (q LoginReq) Identity() string {
	for _, v := range []string{q.Username, q.Email, q.Identifier} {
		if v = strings.TrimSpace(v); v != "" {
			return v
		}
	}
	return ""
}

// ProfileReq: semua field opsional, hanya yang diisi yang diubah.
type ProfileReq struct {
	Username string `json:"username" binding:"omitempty,min=3"`
	Email    string `json:"email" binding:"omitempty,email"`
	Password string `json:"password" binding:"omitempty,min=6"`
}

type Repository struct{ DB *gorm.DB }

func (r *Repository) Create(u *database.User) error { return r.DB.Create(u).Error }
func (r *Repository) ByIdentity(n string) (*database.User, error) {
	var u database.User
	err := r.DB.Where("username = ? OR email = ?", n, n).First(&u).Error
	return &u, err
}
func (r *Repository) ByID(id uint) (*database.User, error) {
	var u database.User
	err := r.DB.First(&u, id).Error
	return &u, err
}
func (r *Repository) Exists(username, email string, exceptID uint) bool {
	var n int64
	r.DB.Model(&database.User{}).
		Where("(username = ? OR email = ?) AND id <> ?", username, email, exceptID).
		Count(&n)
	return n > 0
}
func (r *Repository) Save(u *database.User) error { return r.DB.Save(u).Error }

type Service struct {
	Repo *Repository
	Key  string
}

func (s *Service) Token(u *database.User) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": u.ID, "role": u.Role, "exp": time.Now().Add(24 * time.Hour).Unix(),
	}).SignedString([]byte(s.Key))
}

func (s *Service) Register(q RegisterReq) (*database.User, error) {
	if s.Repo.Exists(q.Username, q.Email, 0) {
		return nil, ErrDuplicate
	}
	h, err := bcrypt.GenerateFromPassword([]byte(q.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	u := &database.User{Username: q.Username, Email: q.Email, Password: string(h), Role: "member"}
	if err := s.Repo.Create(u); err != nil {
		return nil, ErrDuplicate
	}
	return u, nil
}
func (s *Service) Login(q LoginReq) (string, *database.User, error) {
	u, err := s.Repo.ByIdentity(q.Identity())
	if err != nil || bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(q.Password)) != nil {
		return "", nil, gorm.ErrInvalidData
	}
	t, err := s.Token(u)
	return t, u, err
}
func (s *Service) Update(id uint, q ProfileReq) (*database.User, error) {
	u, err := s.Repo.ByID(id)
	if err != nil {
		return nil, err
	}
	if q.Username != "" {
		u.Username = q.Username
	}
	if q.Email != "" {
		u.Email = q.Email
	}
	if s.Repo.Exists(u.Username, u.Email, id) {
		return nil, ErrDuplicate
	}
	if q.Password != "" {
		h, err := bcrypt.GenerateFromPassword([]byte(q.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		u.Password = string(h)
	}
	if err := s.Repo.Save(u); err != nil {
		return nil, ErrDuplicate
	}
	return u, nil
}

type Handler struct{ S *Service }

func (h *Handler) Register(c *gin.Context) {
	var q RegisterReq
	if c.ShouldBindJSON(&q) != nil {
		response.Fail(c, http.StatusBadRequest, "Data tidak valid")
		return
	}
	u, err := h.S.Register(q)
	if errors.Is(err, ErrDuplicate) {
		response.Fail(c, http.StatusBadRequest, "Username atau email sudah terdaftar")
		return
	}
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	t, err := h.S.Token(u)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, http.StatusCreated, "Registrasi berhasil", gin.H{"token": t, "user": u})
}
func (h *Handler) Login(c *gin.Context) {
	var q LoginReq
	if c.ShouldBindJSON(&q) != nil || q.Identity() == "" {
		response.Fail(c, http.StatusBadRequest, "Data tidak valid")
		return
	}
	t, u, err := h.S.Login(q)
	if err != nil {
		response.Fail(c, http.StatusUnauthorized, "Username/email atau password salah")
		return
	}
	response.Success(c, http.StatusOK, "Login berhasil", gin.H{"token": t, "user": u})
}
func (h *Handler) Logout(c *gin.Context) { response.Success(c, http.StatusOK, "Logout berhasil", nil) }
func (h *Handler) Profile(c *gin.Context) {
	u, err := h.S.Repo.ByID(c.GetUint("userID"))
	if err != nil {
		response.Fail(c, http.StatusNotFound, "User tidak ditemukan")
		return
	}
	response.Success(c, http.StatusOK, "OK", u)
}
func (h *Handler) UpdateProfile(c *gin.Context) {
	var q ProfileReq
	if c.ShouldBindJSON(&q) != nil {
		response.Fail(c, http.StatusBadRequest, "Data tidak valid")
		return
	}
	u, err := h.S.Update(c.GetUint("userID"), q)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.Fail(c, http.StatusNotFound, "User tidak ditemukan")
		return
	}
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "Gagal memperbarui profil: username atau email sudah dipakai")
		return
	}
	response.Success(c, http.StatusOK, "Profil diperbarui", u)
}
