package auth

import (
	"net/http"
	"time"

	"app/internal/database"
	"app/internal/response"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type RegisterReq struct {
	Username string `json:"username" binding:"required,min=3"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=5"`
}
type LoginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}
type ProfileReq struct {
	Username string `json:"username" binding:"required,min=3"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password"`
}

type Repository struct{ DB *gorm.DB }

func (r *Repository) Create(u *database.User) error { return r.DB.Create(u).Error }
func (r *Repository) ByUsername(n string) (*database.User, error) {
	var u database.User
	return &u, r.DB.Where("username = ?", n).First(&u).Error
}
func (r *Repository) ByID(id uint) (*database.User, error) {
	var u database.User
	return &u, r.DB.First(&u, id).Error
}
func (r *Repository) Save(u *database.User) error { return r.DB.Save(u).Error }

type Service struct {
	Repo *Repository
	Key  string
}

func (s *Service) Register(q RegisterReq) (*database.User, error) {
	h, _ := bcrypt.GenerateFromPassword([]byte(q.Password), bcrypt.DefaultCost)
	u := &database.User{Username: q.Username, Email: q.Email, Password: string(h), Role: "member"}
	return u, s.Repo.Create(u)
}
func (s *Service) Login(q LoginReq) (string, *database.User, error) {
	u, err := s.Repo.ByUsername(q.Username)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(q.Password)) != nil {
		return "", nil, gorm.ErrInvalidData
	}
	t, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": u.ID, "role": u.Role, "exp": time.Now().Add(24 * time.Hour).Unix(),
	}).SignedString([]byte(s.Key))
	return t, u, err
}
func (s *Service) Update(id uint, q ProfileReq) (*database.User, error) {
	u, err := s.Repo.ByID(id)
	if err != nil {
		return nil, err
	}
	u.Username, u.Email = q.Username, q.Email
	if q.Password != "" {
		h, _ := bcrypt.GenerateFromPassword([]byte(q.Password), bcrypt.DefaultCost)
		u.Password = string(h)
	}
	return u, s.Repo.Save(u)
}

type Handler struct{ S *Service }

func (h *Handler) Register(c *gin.Context) {
	var q RegisterReq
	if c.ShouldBindJSON(&q) != nil {
		response.Fail(c, http.StatusBadRequest, "Data tidak valid")
		return
	}
	u, err := h.S.Register(q)
	if err != nil {
		response.Fail(c, http.StatusConflict, "Username atau email sudah terdaftar")
		return
	}
	response.Success(c, http.StatusCreated, "Registrasi berhasil", u)
}
func (h *Handler) Login(c *gin.Context) {
	var q LoginReq
	if c.ShouldBindJSON(&q) != nil {
		response.Fail(c, http.StatusBadRequest, "Data tidak valid")
		return
	}
	t, u, err := h.S.Login(q)
	if err != nil {
		response.Fail(c, http.StatusUnauthorized, "Username atau password salah")
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
	if err != nil {
		response.Fail(c, http.StatusConflict, "Gagal memperbarui profil")
		return
	}
	response.Success(c, http.StatusOK, "Profil diperbarui", u)
}
